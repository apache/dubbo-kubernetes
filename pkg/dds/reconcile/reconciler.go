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
	"sync"
)

import (
	envoy_core "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	envoy_types "github.com/envoyproxy/go-control-plane/pkg/cache/types"
	envoy_cache "github.com/envoyproxy/go-control-plane/pkg/cache/v3"

	"github.com/go-logr/logr"

	"github.com/pkg/errors"

	"golang.org/x/exp/maps"

	"google.golang.org/protobuf/proto"
)

import (
	config_core "github.com/apache/dubbo-kubernetes/pkg/config/core"
	"github.com/apache/dubbo-kubernetes/pkg/core"
	core_model "github.com/apache/dubbo-kubernetes/pkg/core/resources/model"
	"github.com/apache/dubbo-kubernetes/pkg/dds/cache"
	util_dds "github.com/apache/dubbo-kubernetes/pkg/dds/util"
	"github.com/apache/dubbo-kubernetes/pkg/util/xds"
)

func NewReconciler(hasher envoy_cache.NodeHash, cache envoy_cache.SnapshotCache, generator SnapshotGenerator, mode config_core.CpMode, statsCallbacks xds.StatsCallbacks) Reconciler {
	return &reconciler{
		hasher:         hasher,
		cache:          cache,
		generator:      generator,
		mode:           mode,
		statsCallbacks: statsCallbacks,
	}
}

type reconciler struct {
	hasher         envoy_cache.NodeHash
	cache          envoy_cache.SnapshotCache
	generator      SnapshotGenerator
	mode           config_core.CpMode
	statsCallbacks xds.StatsCallbacks

	lock sync.Mutex
}

func (r *reconciler) Clear(ctx context.Context, node *envoy_core.Node) error {
	id := r.hasher.ID(node)
	r.lock.Lock()
	defer r.lock.Unlock()
	snapshot, err := r.cache.GetSnapshot(id)
	if err != nil {
		return nil // GetSnapshot returns an error if there is no snapshot. We don't need to error here
	}
	r.cache.ClearSnapshot(id)
	if snapshot == nil {
		return nil
	}
	for _, typ := range util_dds.GetSupportedTypes() {
		r.statsCallbacks.DiscardConfig(node.Id + typ)
	}
	return nil
}

func (r *reconciler) Reconcile(ctx context.Context, node *envoy_core.Node, changedTypes map[core_model.ResourceType]struct{}, logger logr.Logger) (error, bool) {
	id := r.hasher.ID(node)
	old, _ := r.cache.GetSnapshot(id)

	// construct builder with unchanged types from the old snapshot
	builder := cache.NewSnapshotBuilder()
	if old != nil {
		for _, typ := range util_dds.GetSupportedTypes() {
			resType := core_model.ResourceType(typ)
			if _, ok := changedTypes[resType]; ok {
				continue
			}

			oldRes := old.GetResources(typ)
			if len(oldRes) > 0 {
				builder = builder.With(resType, maps.Values(oldRes))
			}
		}
	}

	new, err := r.generator.GenerateSnapshot(ctx, node, builder, changedTypes)
	if err != nil {
		return err, false
	}
	if new == nil {
		return errors.New("nil snapshot"), false
	}

	new, changed := r.Version(new, old)
	if changed {
		r.logChanges(logger, new, old, node)
		r.meterConfigReadyForDelivery(new, old, node.Id)
		return r.cache.SetSnapshot(ctx, id, new), true
	}
	return nil, false
}

func (r *reconciler) Version(new, old envoy_cache.ResourceSnapshot) (envoy_cache.ResourceSnapshot, bool) {
	if new == nil {
		return nil, false
	}
	changed := false
	newResources := map[core_model.ResourceType]envoy_cache.Resources{}
	for _, typ := range util_dds.GetSupportedTypes() {
		version := new.GetVersion(typ)
		if version != "" {
			// favor a version assigned by resource generator
			continue
		}

		// 如果旧版本不为空, 并且新版本和旧版本的资源是一样的, 那么以旧版本为主
		if old != nil && r.equal(new.GetResources(typ), old.GetResources(typ)) {
			version = old.GetVersion(typ)
		}
		if version == "" {
			version = core.NewUUID()
			changed = true
		}
		if new.GetVersion(typ) == version {
			continue
		}
		n := map[string]envoy_types.ResourceWithTTL{}
		for k, v := range new.GetResourcesAndTTL(typ) {
			n[k] = v
		}
		newResources[core_model.ResourceType(typ)] = envoy_cache.Resources{Version: version, Items: n}
	}
	return &cache.Snapshot{
		Resources: newResources,
	}, changed
}

func (_ *reconciler) equal(new, old map[string]envoy_types.Resource) bool {
	if len(new) != len(old) {
		return false
	}
	for key, newValue := range new {
		if oldValue, hasOldValue := old[key]; !hasOldValue || !proto.Equal(newValue, oldValue) {
			return false
		}
	}
	return true
}

func (r *reconciler) logChanges(logger logr.Logger, new envoy_cache.ResourceSnapshot, old envoy_cache.ResourceSnapshot, node *envoy_core.Node) {
	for _, typ := range util_dds.GetSupportedTypes() {
		if old != nil && old.GetVersion(typ) != new.GetVersion(typ) {
			client := node.Id
			if r.mode == config_core.Zone {
				// we need to override client name because Zone is always a client to Global (on gRPC level)
				client = "global"
			}
			logger.Info("detected changes in the resources. Sending changes to the client.", "resourceType", typ, "client", client) // todo is client needed?
		}
	}
}

func (r *reconciler) meterConfigReadyForDelivery(new envoy_cache.ResourceSnapshot, old envoy_cache.ResourceSnapshot, nodeID string) {
	for _, typ := range util_dds.GetSupportedTypes() {
		if old == nil || old.GetVersion(typ) != new.GetVersion(typ) {
			r.statsCallbacks.ConfigReadyForDelivery(nodeID + typ)
		}
	}
}
