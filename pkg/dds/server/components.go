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

package server

import (
	"context"
	"github.com/apache/dubbo-kubernetes/pkg/core/resources/model"
	core_runtime "github.com/apache/dubbo-kubernetes/pkg/core/runtime"
	"github.com/apache/dubbo-kubernetes/pkg/dds/reconcile"
	"github.com/apache/dubbo-kubernetes/pkg/events"
	util_watchdog "github.com/apache/dubbo-kubernetes/pkg/util/watchdog"
	util_xds "github.com/apache/dubbo-kubernetes/pkg/util/xds"
	util_xds_v3 "github.com/apache/dubbo-kubernetes/pkg/util/xds/v3"
	envoy_core "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	envoy_cache "github.com/envoyproxy/go-control-plane/pkg/cache/v3"
	envoy_xds "github.com/envoyproxy/go-control-plane/pkg/server/v3"
	"github.com/go-logr/logr"
	"time"
)

func New(
	log logr.Logger,
	rt core_runtime.Runtime,
	providedTypes []model.ResourceType,
	serverID string,
	refresh time.Duration,
	filter reconcile.ResourceFilter,
	mapper reconcile.ResourceMapper,
	nackBackoff time.Duration,
) (Server, error) {
	hasher, cache := newDDSContext(log)
	generator := reconcile.NewSnapshotGenerator(rt.ReadOnlyResourceManager(), filter, mapper)
	reconciler := reconcile.NewReconciler(hasher, cache, generator, rt.GetMode(), nil)
	syncTracker, err := newSyncTracker(
		log,
		reconciler,
		refresh,
		providedTypes,
		rt.EventBus(),
		rt.Extensions(),
	)
	if err != nil {
		return nil, err
	}
	callbacks := util_xds_v3.CallbacksChain{
		&typeAdjustCallbacks{},
		util_xds_v3.NewControlPlaneIdCallbacks(serverID),
		util_xds_v3.AdaptDeltaCallbacks(util_xds.LoggingCallbacks{Log: log}),
		// util_xds_v3.AdaptDeltaCallbacks(NewNackBackoff(nackBackoff)),
		newDdsRetryForcer(log, cache, hasher),
		syncTracker,
	}
	return NewServer(cache, callbacks), nil
}

func newSyncTracker(
	log logr.Logger,
	reconciler reconcile.Reconciler,
	refresh time.Duration,
	providedTypes []model.ResourceType,
	eventBus events.EventBus,
	extensions context.Context,
) (envoy_xds.Callbacks, error) {
	changedTypes := map[model.ResourceType]struct{}{}
	for _, typ := range providedTypes {
		changedTypes[typ] = struct{}{}
	}
	return util_xds_v3.NewWatchdogCallbacks(func(ctx context.Context, node *envoy_core.Node, streamId int64) (util_watchdog.Watchdog, error) {
		return &util_watchdog.SimpleWatchdog{
			NewTicker: func() *time.Ticker {
				return time.NewTicker(refresh)
			},
			OnTick: func(ctx context.Context) error {
				log.V(1).Info("on tick")
				err, _ := reconciler.Reconcile(ctx, node, changedTypes, log)
				return err
			},
			OnError: func(err error) {
				log.Error(err, "OnTick() failed")
			},
			OnStop: func() {
				if err := reconciler.Clear(ctx, node); err != nil {
					log.Error(err, "OnStop() failed")
				}
			},
		}, nil
	}), nil
}

func newDDSContext(log logr.Logger) (envoy_cache.NodeHash, envoy_cache.SnapshotCache) { //nolint:unparam
	hasher := hasher{}
	logger := util_xds.NewLogger(log)
	return hasher, envoy_cache.NewSnapshotCache(false, hasher, logger)
}

type hasher struct{}

func (_ hasher) ID(node *envoy_core.Node) string {
	return node.Id
}
