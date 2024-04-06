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
	"math/rand"
	"time"
)

import (
	envoy_core "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	envoy_cache "github.com/envoyproxy/go-control-plane/pkg/cache/v3"
	envoy_xds "github.com/envoyproxy/go-control-plane/pkg/server/v3"

	"github.com/go-logr/logr"
)

import (
	dubbo_cp "github.com/apache/dubbo-kubernetes/pkg/config/app/dubbo-cp"
	"github.com/apache/dubbo-kubernetes/pkg/core/resources/model"
	core_runtime "github.com/apache/dubbo-kubernetes/pkg/core/runtime"
	"github.com/apache/dubbo-kubernetes/pkg/dds/reconcile"
	"github.com/apache/dubbo-kubernetes/pkg/events"
	dubbo_log "github.com/apache/dubbo-kubernetes/pkg/log"
	util_watchdog "github.com/apache/dubbo-kubernetes/pkg/util/watchdog"
	util_xds "github.com/apache/dubbo-kubernetes/pkg/util/xds"
	util_xds_v3 "github.com/apache/dubbo-kubernetes/pkg/util/xds/v3"
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
		rt.Config().DDSEventBasedWatchdog,
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
	watchdogCfg dubbo_cp.DDSEventBasedWatchdog,
	extensions context.Context,
) (envoy_xds.Callbacks, error) {
	changedTypes := map[model.ResourceType]struct{}{}
	for _, typ := range providedTypes {
		changedTypes[typ] = struct{}{}
	}
	return util_xds_v3.NewWatchdogCallbacks(func(ctx context.Context, node *envoy_core.Node, streamId int64) (util_watchdog.Watchdog, error) {
		log := log.WithValues("streamID", streamId, "nodeID", node.Id)
		log = dubbo_log.AddFieldsFromCtx(log, ctx, extensions)
		return &EventBasedWatchdog{
			Ctx:           ctx,
			Node:          node,
			EventBus:      eventBus,
			Reconciler:    reconciler,
			ProvidedTypes: changedTypes,
			Log:           log,
			NewFlushTicker: func() *time.Ticker {
				return time.NewTicker(watchdogCfg.FlushInterval.Duration)
			},
			NewFullResyncTicker: func() *time.Ticker {
				if watchdogCfg.DelayFullResync {
					// To ensure an even distribution of connections over time, we introduce a random delay within
					// the full resync interval. This prevents clustering connections within a short timeframe
					// and spreads them evenly across the entire interval. After the initial trigger, we reset
					// the ticker, returning it to its full resync interval.
					// #nosec G404 - math rand is enough
					delay := time.Duration(watchdogCfg.FullResyncInterval.Duration.Seconds()*rand.Float64()) * time.Second
					ticker := time.NewTicker(watchdogCfg.FullResyncInterval.Duration + delay)
					go func() {
						<-time.After(delay)
						ticker.Reset(watchdogCfg.FullResyncInterval.Duration)
					}()
					return ticker
				} else {
					return time.NewTicker(watchdogCfg.FullResyncInterval.Duration)
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
