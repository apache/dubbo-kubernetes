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

package tracker

import (
	"context"
	"github.com/apache/dubbo-kubernetes/pkg/hds/cache"
	util_xds_v3 "github.com/apache/dubbo-kubernetes/pkg/util/xds/v3"
	envoy_config_core_v3 "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
)

type reconciler struct {
	hasher    util_xds_v3.NodeHash
	cache     util_xds_v3.SnapshotCache
	generator *SnapshotGenerator
	versioner util_xds_v3.SnapshotVersioner
}

func (r *reconciler) Reconcile(ctx context.Context, node *envoy_config_core_v3.Node) error {
	new, err := r.generator.GenerateSnapshot(ctx, node)
	if err != nil {
		return err
	}
	if err := new.Consistent(); err != nil {
		return err
	}
	id := r.hasher.ID(node)
	old, _ := r.cache.GetSnapshot(id)
	new = r.versioner.Version(new, old)
	return r.cache.SetSnapshot(id, new)
}

func (r *reconciler) Clear(node *envoy_config_core_v3.Node) error {
	// cache.Clear() operation does not push a new (empty) configuration to Envoy.
	// That is why instead of calling cache.Clear() we set configuration to an empty Snapshot.
	// This fake value will be removed from cache on Envoy disconnect.
	return r.cache.SetSnapshot(r.hasher.ID(node), &cache.Snapshot{})
}
