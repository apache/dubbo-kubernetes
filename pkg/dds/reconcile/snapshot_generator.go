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

package reconcile

import (
	"context"
	"strings"
)

import (
	envoy_core "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	envoy_types "github.com/envoyproxy/go-control-plane/pkg/cache/types"
	envoy_cache "github.com/envoyproxy/go-control-plane/pkg/cache/v3"
)

import (
	config_store "github.com/apache/dubbo-kubernetes/pkg/config/core/resources/store"
	core_manager "github.com/apache/dubbo-kubernetes/pkg/core/resources/manager"
	"github.com/apache/dubbo-kubernetes/pkg/core/resources/model"
	core_model "github.com/apache/dubbo-kubernetes/pkg/core/resources/model"
	"github.com/apache/dubbo-kubernetes/pkg/core/resources/registry"
	"github.com/apache/dubbo-kubernetes/pkg/dds"
	cache_dds "github.com/apache/dubbo-kubernetes/pkg/dds/cache"
	"github.com/apache/dubbo-kubernetes/pkg/dds/util"
)

type (
	ResourceFilter func(ctx context.Context, clusterID string, features dds.Features, r core_model.Resource) bool
	ResourceMapper func(features dds.Features, r core_model.Resource) (core_model.Resource, error)
)

func NoopResourceMapper(_ dds.Features, r model.Resource) (model.Resource, error) {
	return r, nil
}

func Any(context.Context, string, dds.Features, model.Resource) bool {
	return true
}

func TypeIs(rtype core_model.ResourceType) func(core_model.Resource) bool {
	return func(r core_model.Resource) bool {
		return r.Descriptor().Name == rtype
	}
}

func IsKubernetes(storeType config_store.StoreType) func(core_model.Resource) bool {
	return func(_ core_model.Resource) bool {
		return storeType == config_store.KubernetesStore
	}
}

func ScopeIs(s core_model.ResourceScope) func(core_model.Resource) bool {
	return func(r core_model.Resource) bool {
		return r.Descriptor().Scope == s
	}
}

func NameHasPrefix(prefix string) func(core_model.Resource) bool {
	return func(r core_model.Resource) bool {
		return strings.HasPrefix(r.GetMeta().GetName(), prefix)
	}
}

func Not(f func(core_model.Resource) bool) func(core_model.Resource) bool {
	return func(r core_model.Resource) bool {
		return !f(r)
	}
}

func And(fs ...func(core_model.Resource) bool) func(core_model.Resource) bool {
	return func(r core_model.Resource) bool {
		for _, f := range fs {
			if !f(r) {
				return false
			}
		}
		return true
	}
}

func If(condition func(core_model.Resource) bool, m ResourceMapper) ResourceMapper {
	return func(features dds.Features, r core_model.Resource) (core_model.Resource, error) {
		if condition(r) {
			return m(features, r)
		}
		return r, nil
	}
}

func NewSnapshotGenerator(resourceManager core_manager.ReadOnlyResourceManager, filter ResourceFilter, mapper ResourceMapper) SnapshotGenerator {
	return &snapshotGenerator{
		resourceManager: resourceManager,
		resourceFilter:  filter,
		resourceMapper:  mapper,
	}
}

type snapshotGenerator struct {
	resourceManager core_manager.ReadOnlyResourceManager
	resourceFilter  ResourceFilter
	resourceMapper  ResourceMapper
}

func (s *snapshotGenerator) GenerateSnapshot(
	ctx context.Context,
	node *envoy_core.Node,
	builder cache_dds.SnapshotBuilder,
	resTypes map[model.ResourceType]struct{},
) (envoy_cache.ResourceSnapshot, error) {
	for typ := range resTypes {
		resources, err := s.getResources(ctx, typ, node)
		if err != nil {
			return nil, err
		}
		builder = builder.With(typ, resources)
	}

	return builder.Build(""), nil
}

func (s *snapshotGenerator) getResources(ctx context.Context, typ model.ResourceType, node *envoy_core.Node) ([]envoy_types.Resource, error) {
	rlist, err := registry.Global().NewList(typ)
	if err != nil {
		return nil, err
	}
	if err := s.resourceManager.List(ctx, rlist); err != nil {
		return nil, err
	}

	resources, err := s.mapper(s.filter(ctx, rlist, node), node)
	if err != nil {
		return nil, err
	}

	return util.ToEnvoyResources(resources)
}

func (s *snapshotGenerator) filter(ctx context.Context, rs model.ResourceList, node *envoy_core.Node) model.ResourceList {
	features := getFeatures(node)

	rv := registry.Global().MustNewList(rs.GetItemType())
	for _, r := range rs.GetItems() {
		if s.resourceFilter(ctx, node.GetId(), features, r) {
			_ = rv.AddItem(r)
		}
	}
	return rv
}

func (s *snapshotGenerator) mapper(rs model.ResourceList, node *envoy_core.Node) (model.ResourceList, error) {
	features := getFeatures(node)

	rv := registry.Global().MustNewList(rs.GetItemType())
	for _, r := range rs.GetItems() {
		resource, err := s.resourceMapper(features, r)
		if err != nil {
			return nil, err
		}

		if err := rv.AddItem(resource); err != nil {
			return nil, err
		}
	}

	return rv, nil
}

func getFeatures(node *envoy_core.Node) dds.Features {
	features := dds.Features{}
	for _, value := range node.GetMetadata().GetFields()[dds.MetadataFeatures].GetListValue().GetValues() {
		features[value.GetStringValue()] = true
	}
	return features
}
