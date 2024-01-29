/*
 * Licensed to the Apache Software Foundation (ASF) under one or more
 * contributor license agreements.  See the NOTICE file distributed with
 * this work for additional information regarding copyright ownership.
 * The ASF licenses this file to You under the Apache License, Version 2.0
 * (the "License"); you may not use this file except in compliance with
 * the License.  You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package context

import (
	"bytes"
	"context"
	"encoding/base64"
	"github.com/apache/dubbo-kubernetes/pkg/core/dns/lookup"
	core_mesh "github.com/apache/dubbo-kubernetes/pkg/core/resources/apis/mesh"
	"github.com/apache/dubbo-kubernetes/pkg/core/resources/apis/system"
	"github.com/apache/dubbo-kubernetes/pkg/core/resources/registry"
	core_store "github.com/apache/dubbo-kubernetes/pkg/core/resources/store"
	xds_topology "github.com/apache/dubbo-kubernetes/pkg/xds/topology"
	"hash/fnv"

	"github.com/pkg/errors"
)

import (
	"github.com/apache/dubbo-kubernetes/pkg/core/resources/manager"
	core_model "github.com/apache/dubbo-kubernetes/pkg/core/resources/model"
)

type meshContextBuilder struct {
	rm      manager.ReadOnlyResourceManager
	typeSet map[core_model.ResourceType]struct{}
	ipFunc  lookup.LookupIPFunc
	zone    string
}

type MeshContextBuilder interface {
	// BuildGlobalContextIfChanged builds GlobalContext only if `latest` is nil or hash is different
	// If hash is the same, the return `latest`
	BuildGlobalContextIfChanged(ctx context.Context, latest *GlobalContext) (*GlobalContext, error)

	// BuildIfChanged builds MeshContext only if latestMeshCtx is nil or hash of
	// latestMeshCtx is different.
	// If hash is the same, then the function returns the passed latestMeshCtx.
	// Hash returned in MeshContext can never be empty.
	BuildIfChanged(ctx context.Context, meshName string, latestMeshCtx *MeshContext) (*MeshContext, error)
}

func NewMeshContextBuilder(
	rm manager.ReadOnlyResourceManager,
	types []core_model.ResourceType, // types that should be taken into account when MeshContext is built.
	ipFunc lookup.LookupIPFunc,
	zone string,
) MeshContextBuilder {
	typeSet := map[core_model.ResourceType]struct{}{}
	for _, typ := range types {
		typeSet[typ] = struct{}{}
	}

	return &meshContextBuilder{
		rm:      rm,
		typeSet: typeSet,
		ipFunc:  ipFunc,
		zone:    zone,
	}
}

func (m *meshContextBuilder) BuildGlobalContextIfChanged(ctx context.Context, latest *GlobalContext) (*GlobalContext, error) {
	rmap := ResourceMap{}
	for t := range m.typeSet {
		desc, err := registry.Global().DescriptorFor(t)
		if err != nil {
			return nil, err
		}
		if desc.Scope == core_model.ScopeGlobal && desc.Name != system.ConfigType { // For config we ignore them atm and prefer to rely on more specific filters.
			rmap[t], err = m.fetchResourceList(ctx, t, nil)
			if err != nil {
				return nil, errors.Wrap(err, "failed to build global context")
			}
		}
	}
	newHash := rmap.Hash()
	if latest != nil && bytes.Equal(newHash, latest.hash) {
		return latest, nil
	}
	return &GlobalContext{
		hash:        newHash,
		ResourceMap: rmap,
	}, nil
}

func (m meshContextBuilder) BuildIfChanged(ctx context.Context, meshName string, latestMeshCtx *MeshContext) (*MeshContext, error) {
	globalContext, err := m.BuildGlobalContextIfChanged(ctx, nil)
	if err != nil {
		return nil, err
	}

	resources := NewResources()
	for resType := range m.typeSet {
		rl, ok := globalContext.ResourceMap[resType]
		if ok {
			// Exists in global context take it from there
			resources.MeshLocalResources[resType] = rl
		}
	}

	newHash := base64.StdEncoding.EncodeToString(m.hash(globalContext))
	if latestMeshCtx != nil && newHash == latestMeshCtx.Hash {
		return latestMeshCtx, nil
	}

	dataplanes := resources.Dataplanes().Items
	dataplanesByName := make(map[string]*core_mesh.DataplaneResource, len(dataplanes))
	for _, dp := range dataplanes {
		dataplanesByName[dp.Meta.GetName()] = dp
	}

	endpointMap := xds_topology.BuildEdsEndpoint(m.zone, dataplanes, nil)

	return &MeshContext{
		Hash:             newHash,
		Resources:        resources,
		DataplanesByName: dataplanesByName,
		EndpointMap:      endpointMap,
	}, nil
}

type filterFn = func(rs core_model.Resource) bool

func (m *meshContextBuilder) fetchResourceList(ctx context.Context, resType core_model.ResourceType, filterFn filterFn) (core_model.ResourceList, error) {
	var listOptsFunc []core_store.ListOptionsFunc
	desc, err := registry.Global().DescriptorFor(resType)
	if err != nil {
		return nil, err
	}
	listOptsFunc = append(listOptsFunc, core_store.ListOrdered())
	list := desc.NewList()
	if err := m.rm.List(ctx, list, listOptsFunc...); err != nil {
		return nil, err
	}
	if resType != core_mesh.ZoneIngressType && resType != core_mesh.DataplaneType && filterFn == nil {
		// No post processing stuff so return the list as is
		return list, nil
	}
	list, err = modifyAllEntries(list, func(resource core_model.Resource) (core_model.Resource, error) {
		if filterFn != nil && !filterFn(resource) {
			return nil, nil
		}
		switch resType {
		case core_mesh.DataplaneType:
			list, err = modifyAllEntries(list, func(resource core_model.Resource) (core_model.Resource, error) {
				dp, ok := resource.(*core_mesh.DataplaneResource)
				if !ok {
					return nil, errors.New("entry is not a dataplane this shouldn't happen")
				}
				zi, err := xds_topology.ResolveDataplaneAddress(m.ipFunc, dp)
				if err != nil {
					return nil, nil
				}
				return zi, nil
			})
		}
		return resource, nil
	})
	if err != nil {
		return nil, err
	}
	return list, nil
}

// takes a resourceList and modify it as needed
func modifyAllEntries(list core_model.ResourceList, fn func(resource core_model.Resource) (core_model.Resource, error)) (core_model.ResourceList, error) {
	newList := list.NewItem().Descriptor().NewList()
	for _, v := range list.GetItems() {
		ni, err := fn(v)
		if err != nil {
			return nil, err
		}
		if ni != nil {
			err := newList.AddItem(ni)
			if err != nil {
				return nil, err
			}
		}
	}
	// it is meaningless temporarily
	newList.GetPagination().SetTotal(uint32(len(newList.GetItems())))
	return newList, nil
}

func (m *meshContextBuilder) hash(globalContext *GlobalContext) []byte {
	hasher := fnv.New128a()
	_, _ = hasher.Write(globalContext.hash)
	return hasher.Sum(nil)
}
