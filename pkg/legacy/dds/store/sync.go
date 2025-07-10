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

package store

import (
	"context"
	"fmt"
	"maps"
	"strings"
	"time"
)

import (
	"github.com/go-logr/logr"

	"github.com/pkg/errors"

	"github.com/prometheus/client_golang/prometheus"
)

import (
	mesh_proto "github.com/apache/dubbo-kubernetes/api/mesh/v1alpha1"
	config_core "github.com/apache/dubbo-kubernetes/pkg/config/mode"
	"github.com/apache/dubbo-kubernetes/pkg/core"
	core_mesh "github.com/apache/dubbo-kubernetes/pkg/core/resources/apis/mesh"
	"github.com/apache/dubbo-kubernetes/pkg/core/resources/apis/system"
	core_manager "github.com/apache/dubbo-kubernetes/pkg/core/resources/manager"
	"github.com/apache/dubbo-kubernetes/pkg/core/resources/model"
	core_model "github.com/apache/dubbo-kubernetes/pkg/core/resources/model"
	"github.com/apache/dubbo-kubernetes/pkg/core/resources/registry"
	"github.com/apache/dubbo-kubernetes/pkg/core/resources/store"
	"github.com/apache/dubbo-kubernetes/pkg/dds"
	"github.com/apache/dubbo-kubernetes/pkg/dds/client"
	"github.com/apache/dubbo-kubernetes/pkg/dds/util"
	resources_k8s "github.com/apache/dubbo-kubernetes/pkg/plugins/resources/k8s"
	k8s_model "github.com/apache/dubbo-kubernetes/pkg/plugins/resources/k8s/native/pkg/model"
)

// ResourceSyncer allows to synchronize resources in Store
type ResourceSyncer interface {
	// Sync method takes 'upstream' as a basis and synchronize underlying store.
	// It deletes resources that were removed in the upstream, creates new resources that
	// are not represented in store yet and updates the rest.
	// Using 'PrefilterBy' option Sync allows to select scope of resources that will be
	// affected by Sync
	//
	// Sync takes into account only 'Name' and 'Mesh' when it comes to upstream's Meta.
	// 'Version', 'CreationTime' and 'ModificationTime' are managed by downstream store.
	Sync(ctx context.Context, upstream client.UpstreamResponse, fs ...SyncOptionFunc) error
}

type SyncOption struct {
	Predicate func(r core_model.Resource) bool
	Zone      string
}

type SyncOptionFunc func(*SyncOption)

func NewSyncOptions(fs ...SyncOptionFunc) *SyncOption {
	opts := &SyncOption{}
	for _, f := range fs {
		f(opts)
	}
	return opts
}

func Zone(name string) SyncOptionFunc {
	return func(opts *SyncOption) {
		opts.Zone = name
	}
}

func PrefilterBy(predicate func(r core_model.Resource) bool) SyncOptionFunc {
	return func(opts *SyncOption) {
		opts.Predicate = predicate
	}
}

type syncResourceStore struct {
	log             logr.Logger
	resourceManager core_manager.ResourceManager
	transactions    store.Transactions
	metric          prometheus.Histogram
	extensions      context.Context
}

func NewResourceSyncer(
	log logr.Logger,
	resourceStore core_manager.ResourceManager,
	transactions store.Transactions,
	extensions context.Context,
) (ResourceSyncer, error) {
	return &syncResourceStore{
		log:             log,
		resourceManager: resourceStore,
		transactions:    transactions,
		extensions:      extensions,
	}, nil
}

type OnUpdate struct {
	r    core_model.Resource
	opts []store.UpdateOptionsFunc
}

func (s *syncResourceStore) Sync(syncCtx context.Context, upstreamResponse client.UpstreamResponse, fs ...SyncOptionFunc) error {
	now := core.Now()
	defer func() {
		s.metric.Observe(float64(time.Since(now).Milliseconds()) / 1000)
	}()
	opts := NewSyncOptions(fs...)
	log := s.log.WithValues("type", upstreamResponse.Type)
	upstream := upstreamResponse.AddedResources
	downstream, err := registry.Global().NewList(upstreamResponse.Type)
	if err != nil {
		return err
	}
	if upstreamResponse.IsInitialRequest {
		if err := s.resourceManager.List(syncCtx, downstream); err != nil {
			return err
		}
	} else {
		upstreamChangeKeys := append(core_model.ResourceListToResourceKeys(upstream), upstreamResponse.RemovedResourcesKey...)
		if err := s.resourceManager.List(syncCtx, downstream, store.ListByResourceKeys(upstreamChangeKeys)); err != nil {
			return err
		}
	}
	log.V(1).Info("before filtering", "downstream", downstream, "upstream", upstream)

	if opts.Predicate != nil {
		if filtered, err := filter(downstream, opts.Predicate); err != nil {
			return err
		} else {
			downstream = filtered
		}
		if filtered, err := filter(upstream, opts.Predicate); err != nil {
			return err
		} else {
			upstream = filtered
		}
	}
	log.V(1).Info("after filtering", "downstream", downstream, "upstream", upstream)

	indexedDownstream := newIndexed(downstream)
	indexedUpstream := newIndexed(upstream)

	var onDelete []core_model.Resource
	// 1. delete resources which were removed from the upstream
	// on the first request when the control-plane starts we want to sync
	// whole the resources in the store. In this case we do not check removed
	// resources because we want to make stores synced. When we already
	// have resources in the map, we are going to receive only updates
	// so we don't want to remove resources haven't changed.
	if upstreamResponse.IsInitialRequest {
		for _, r := range downstream.GetItems() {
			if indexedUpstream.get(core_model.MetaToResourceKey(r.GetMeta())) == nil {
				onDelete = append(onDelete, r)
			}
		}
	} else {
		for _, rk := range upstreamResponse.RemovedResourcesKey {
			// check if we are adding and removing the resource at the same time
			if r := indexedUpstream.get(rk); r != nil {
				// it isn't remove but update
				continue
			}
			if r := indexedDownstream.get(rk); r != nil {
				onDelete = append(onDelete, r)
			}
		}
	}

	// 2. create resources which are not represented in 'downstream' and update the rest of them
	var (
		onCreate []core_model.Resource
		onUpdate []OnUpdate
	)
	for _, r := range upstream.GetItems() {
		existing := indexedDownstream.get(core_model.MetaToResourceKey(r.GetMeta()))
		if existing == nil {
			onCreate = append(onCreate, r)
			continue
		}
		newLabels := r.GetMeta().GetLabels()
		if !core_model.Equal(existing.GetSpec(), r.GetSpec()) || !maps.Equal(existing.GetMeta().GetLabels(), newLabels) {
			// we have to use meta of the current Store during update, because some Stores (Kubernetes, Memory)
			// expect to receive ResourceMeta of own type.
			r.SetMeta(existing.GetMeta())
			onUpdate = append(onUpdate, OnUpdate{r: r, opts: []store.UpdateOptionsFunc{store.UpdateWithLabels(newLabels)}})
		}
	}

	zone := system.NewZoneResource()
	if opts.Zone != "" && len(onCreate) > 0 {
		if err := s.resourceManager.Get(syncCtx, zone, store.GetByKey(opts.Zone, core_model.NoMesh)); err != nil {
			return err
		}
	}

	return store.InTx(syncCtx, s.transactions, func(ctx context.Context) error {
		for _, r := range onDelete {
			rk := core_model.MetaToResourceKey(r.GetMeta())
			log.Info("deleting a resource since it's no longer available in the upstream", "name", r.GetMeta().GetName(), "mesh", r.GetMeta().GetMesh())
			if err := s.resourceManager.Delete(ctx, r, store.DeleteBy(rk)); err != nil {
				return err
			}
		}

		for _, r := range onCreate {
			rk := core_model.MetaToResourceKey(r.GetMeta())
			log.Info("creating a new resource from upstream", "name", r.GetMeta().GetName(), "mesh", r.GetMeta().GetMesh())

			createOpts := []store.CreateOptionsFunc{
				store.CreateBy(rk),
				store.CreatedAt(core.Now()),
				store.CreateWithLabels(r.GetMeta().GetLabels()),
			}
			if opts.Zone != "" {
				createOpts = append(createOpts, store.CreateWithOwner(zone))
			}

			// some Stores try to cast ResourceMeta to own Store type that's why we have to set meta to nil
			r.SetMeta(nil)

			if err := s.resourceManager.Create(ctx, r, createOpts...); err != nil {
				return err
			}
		}

		for _, upd := range onUpdate {
			log.V(1).Info("updating a resource", "name", upd.r.GetMeta().GetName(), "mesh", upd.r.GetMeta().GetMesh())
			now := time.Now()
			// some stores manage ModificationTime time on they own (Kubernetes), in order to be consistent
			// we set ModificationTime when we add to downstream store. This time is almost the same with ModificationTime
			// from upstream store, because we update downstream only when resource have changed in upstream
			if err := s.resourceManager.Update(ctx, upd.r, append(upd.opts, store.ModifiedAt(now))...); err != nil {
				return err
			}
		}
		return nil
	})
}

func filter(rs core_model.ResourceList, predicate func(r core_model.Resource) bool) (core_model.ResourceList, error) {
	rv, err := registry.Global().NewList(rs.GetItemType())
	if err != nil {
		return nil, err
	}
	for _, r := range rs.GetItems() {
		if predicate(r) {
			if err := rv.AddItem(r); err != nil {
				return nil, err
			}
		}
	}
	return rv, nil
}

type indexed struct {
	indexByResourceKey map[core_model.ResourceKey]core_model.Resource
}

func (i *indexed) get(rk core_model.ResourceKey) core_model.Resource {
	return i.indexByResourceKey[rk]
}

func newIndexed(rs core_model.ResourceList) *indexed {
	idxByRk := map[core_model.ResourceKey]core_model.Resource{}
	for _, r := range rs.GetItems() {
		idxByRk[core_model.MetaToResourceKey(r.GetMeta())] = r
	}
	return &indexed{indexByResourceKey: idxByRk}
}

func ZoneSyncCallback(ctx context.Context, configToSync map[string]bool, syncer ResourceSyncer, k8sStore bool, localZone string, kubeFactory resources_k8s.KubeFactory, systemNamespace string) *client.Callbacks {
	return &client.Callbacks{
		OnResourcesReceived: func(upstream client.UpstreamResponse) error {
			if k8sStore && upstream.Type != system.ConfigType && upstream.Type != system.SecretType {
				if err := addNamespaceSuffix(kubeFactory, upstream, systemNamespace); err != nil {
					return err
				}
			}

			switch {
			case upstream.Type == system.ConfigType:
				return syncer.Sync(ctx, upstream, PrefilterBy(func(r core_model.Resource) bool {
					return configToSync[r.GetMeta().GetName()]
				}))
			}

			return syncer.Sync(ctx, upstream, PrefilterBy(func(r core_model.Resource) bool {
				if zi, ok := r.(*core_mesh.ZoneIngressResource); ok {
					return zi.IsRemoteIngress(localZone)
				}

				if m, ok := r.(*core_mesh.MappingResource); ok {
					// we do not sync Mapping Resource in local zone
					return m.IsRemoteMapping(localZone)
				}

				if m, ok := r.(*core_mesh.MetaDataResource); ok {
					return m.IsRemotingMetadata(localZone)
				}

				return !core_model.IsLocallyOriginated(config_core.Zone, r) || !isExpectedOnZoneCP(r.Descriptor())
			}))
		},
	}
}

// isExpectedOnZoneCP returns true if it's possible for the resource type to be on Zone CP. Some resource types
// (i.e. Mesh, Secret) are allowed on non-federated Zone CPs, but after transition to federated Zone CP they're moved
// to Global and must be replaced during the DDS sync.
func isExpectedOnZoneCP(desc core_model.ResourceTypeDescriptor) bool {
	return desc.DDSFlags.Has(core_model.ZoneToGlobalFlag)
}

func GlobalSyncCallback(
	ctx context.Context,
	syncer ResourceSyncer,
	k8sStore bool,
	kubeFactory resources_k8s.KubeFactory,
	systemNamespace string,
) *client.Callbacks {
	supportsHashSuffixes := dds.ContextHasFeature(ctx, dds.FeatureHashSuffix)

	return &client.Callbacks{
		OnResourcesReceived: func(upstream client.UpstreamResponse) error {
			if !supportsHashSuffixes {
				// todo: remove in 2 releases after 2.6.x
				upstream.RemovedResourcesKey = util.AddPrefixToResourceKeyNames(upstream.RemovedResourcesKey, upstream.ControlPlaneId)
				util.AddPrefixToNames(upstream.AddedResources.GetItems(), upstream.ControlPlaneId)
			}

			for _, r := range upstream.AddedResources.GetItems() {
				r.SetMeta(util.CloneResourceMeta(r.GetMeta(),
					util.WithLabel(mesh_proto.ZoneTag, upstream.ControlPlaneId),
					util.WithLabel(mesh_proto.ResourceOriginLabel, string(mesh_proto.ZoneResourceOrigin)),
				))
			}

			if k8sStore {
				if err := addNamespaceSuffix(kubeFactory, upstream, systemNamespace); err != nil {
					return err
				}
			}

			switch upstream.Type {
			case core_mesh.ZoneIngressType:
				for _, zi := range upstream.AddedResources.(*core_mesh.ZoneIngressResourceList).Items {
					zi.Spec.Zone = upstream.ControlPlaneId
				}
			case core_mesh.ZoneEgressType:
				for _, ze := range upstream.AddedResources.(*core_mesh.ZoneEgressResourceList).Items {
					ze.Spec.Zone = upstream.ControlPlaneId
				}
			case core_mesh.MappingType:
				for _, m := range upstream.AddedResources.(*core_mesh.MappingResourceList).Items {
					m.Spec.Zone = upstream.ControlPlaneId
				}
			case core_mesh.MetaDataType:
				for _, m := range upstream.AddedResources.(*core_mesh.MetaDataResourceList).Items {
					m.Spec.Zone = upstream.ControlPlaneId
				}
			}

			return syncer.Sync(ctx, upstream, PrefilterBy(func(r model.Resource) bool {
				if !supportsHashSuffixes {
					// todo: remove in 2 releases after 2.6.x
					return strings.HasPrefix(r.GetMeta().GetName(), fmt.Sprintf("%s.", upstream.ControlPlaneId))
				}
				return r.GetMeta().GetLabels()[mesh_proto.ZoneTag] == upstream.ControlPlaneId
			}), Zone(upstream.ControlPlaneId))
		},
	}
}

func addNamespaceSuffix(kubeFactory resources_k8s.KubeFactory, upstream client.UpstreamResponse, ns string) error {
	// if type of Store is Kubernetes then we want to store upstream resources in dedicated Namespace.
	// KubernetesStore parses Name and considers substring after the last dot as a Namespace's Name.
	kubeObject, err := kubeFactory.NewObject(upstream.AddedResources.NewItem())
	if err != nil {
		return errors.Wrap(err, "could not convert object")
	}
	if kubeObject.Scope() == k8s_model.ScopeNamespace {
		util.AddSuffixToNames(upstream.AddedResources.GetItems(), ns)
		upstream.RemovedResourcesKey = util.AddSuffixToResourceKeyNames(upstream.RemovedResourcesKey, ns)
	}
	return nil
}
