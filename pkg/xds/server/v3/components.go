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

package v3

import (
	"context"
	"time"
)

import (
	envoy_service_discovery "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v3"
	envoy_server "github.com/envoyproxy/go-control-plane/pkg/server/v3"
)

import (
	"github.com/apache/dubbo-kubernetes/pkg/core"
	core_model "github.com/apache/dubbo-kubernetes/pkg/core/resources/model"
	core_runtime "github.com/apache/dubbo-kubernetes/pkg/core/runtime"
	core_xds "github.com/apache/dubbo-kubernetes/pkg/core/xds"
	util_xds "github.com/apache/dubbo-kubernetes/pkg/util/xds"
	util_xds_v3 "github.com/apache/dubbo-kubernetes/pkg/util/xds/v3"
	xds_context "github.com/apache/dubbo-kubernetes/pkg/xds/context"
	"github.com/apache/dubbo-kubernetes/pkg/xds/envoy"
	"github.com/apache/dubbo-kubernetes/pkg/xds/generator"
	xds_callbacks "github.com/apache/dubbo-kubernetes/pkg/xds/server/callbacks"
	xds_sync "github.com/apache/dubbo-kubernetes/pkg/xds/sync"
)

var xdsServerLog = core.Log.WithName("xds-server")

func RegisterXDS(
	statsCallbacks util_xds.StatsCallbacks,
	envoyCpCtx *xds_context.ControlPlaneContext,
	rt core_runtime.Runtime,
) error {
	xdsContext := NewXdsContext()
	metadataTracker := xds_callbacks.NewDataplaneMetadataTracker()
	reconciler := DefaultReconciler(rt, xdsContext, statsCallbacks)
	ingressReconciler := DefaultIngressReconciler(rt, xdsContext, statsCallbacks)
	watchdogFactory, err := xds_sync.DefaultDataplaneWatchdogFactory(rt, metadataTracker, reconciler, ingressReconciler, nil, envoyCpCtx, core_xds.APIVersion(envoy.APIV3))
	if err != nil {
		return err
	}

	callbacks := util_xds_v3.CallbacksChain{
		util_xds_v3.NewControlPlaneIdCallbacks(rt.GetInstanceId()),
		util_xds_v3.AdaptCallbacks(statsCallbacks),
		util_xds_v3.AdaptCallbacks(xds_callbacks.DataplaneCallbacksToXdsCallbacks(metadataTracker)),
		util_xds_v3.AdaptCallbacks(xds_callbacks.DataplaneCallbacksToXdsCallbacks(xds_callbacks.NewDataplaneSyncTracker(watchdogFactory.New))),
		util_xds_v3.AdaptCallbacks(xds_callbacks.DataplaneCallbacksToXdsCallbacks(
			xds_callbacks.NewDataplaneLifecycle(rt.AppContext(), rt.ResourceManager(), rt.Config().XdsServer.DataplaneDeregistrationDelay.Duration, rt.GetInstanceId())),
		),
		util_xds_v3.AdaptCallbacks(DefaultDataplaneStatusTracker(rt)),
		util_xds_v3.AdaptCallbacks(xds_callbacks.NewNackBackoff(10)),
	}

	if cb := rt.XDS().ServerCallbacks; cb != nil {
		callbacks = append(callbacks, util_xds_v3.AdaptCallbacks(cb))
	}

	srv := envoy_server.NewServer(context.Background(), xdsContext.Cache(), callbacks)
	xdsServerLog.Info("registering Aggregated Discovery Service V3 in Dataplane Server")
	envoy_service_discovery.RegisterAggregatedDiscoveryServiceServer(rt.DpServer().GrpcServer(), srv)
	return nil
}

func DefaultReconciler(
	rt core_runtime.Runtime,
	xdsContext XdsContext,
	statsCallbacks util_xds.StatsCallbacks,
) xds_sync.SnapshotReconciler {
	return &reconciler{
		generator: &TemplateSnapshotGenerator{
			[]string{
				generator.DefaultProxy,
			},
		},
		cacher:         &simpleSnapshotCacher{xdsContext.Hasher(), xdsContext.Cache()},
		statsCallbacks: statsCallbacks,
	}
}

func DefaultIngressReconciler(
	rt core_runtime.Runtime,
	xdsContext XdsContext,
	statsCallbacks util_xds.StatsCallbacks,
) xds_sync.SnapshotReconciler {
	return &reconciler{
		generator:      &TemplateSnapshotGenerator{[]string{generator.IngressProxy}},
		cacher:         &simpleSnapshotCacher{xdsContext.Hasher(), xdsContext.Cache()},
		statsCallbacks: nil,
	}
}

func DefaultDataplaneStatusTracker(rt core_runtime.Runtime) xds_callbacks.DataplaneStatusTracker {
	return xds_callbacks.NewDataplaneStatusTracker(rt,
		func(dataplaneType core_model.ResourceType, accessor xds_callbacks.SubscriptionStatusAccessor) xds_callbacks.DataplaneInsightSink {
			return xds_callbacks.NewDataplaneInsightSink(
				dataplaneType,
				accessor,
				func() *time.Ticker {
					return time.NewTicker(rt.Config().XdsServer.DataplaneStatusFlushInterval.Duration)
				},
				func() *time.Ticker {
					return nil
				},
				rt.Config().XdsServer.DataplaneStatusFlushInterval.Duration/10,
				xds_callbacks.NewDataplaneInsightStore(rt.ResourceManager()),
			)
		})
}
