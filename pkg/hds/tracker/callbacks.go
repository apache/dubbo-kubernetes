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
	"sync"
	"time"
)

import (
	envoy_core "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	envoy_service_health "github.com/envoyproxy/go-control-plane/envoy/service/health/v3"

	"github.com/go-logr/logr"

	"github.com/pkg/errors"
)

import (
	mesh_proto "github.com/apache/dubbo-kubernetes/api/mesh/v1alpha1"
	dp_server "github.com/apache/dubbo-kubernetes/pkg/config/dp-server"
	"github.com/apache/dubbo-kubernetes/pkg/core"
	"github.com/apache/dubbo-kubernetes/pkg/core/resources/apis/mesh"
	"github.com/apache/dubbo-kubernetes/pkg/core/resources/manager"
	core_model "github.com/apache/dubbo-kubernetes/pkg/core/resources/model"
	"github.com/apache/dubbo-kubernetes/pkg/core/resources/store"
	"github.com/apache/dubbo-kubernetes/pkg/core/xds"
	hds_callbacks "github.com/apache/dubbo-kubernetes/pkg/hds/callbacks"
	"github.com/apache/dubbo-kubernetes/pkg/util/watchdog"
	util_xds_v3 "github.com/apache/dubbo-kubernetes/pkg/util/xds/v3"
	"github.com/apache/dubbo-kubernetes/pkg/xds/envoy/names"
)

type streams struct {
	watchdogCancel context.CancelFunc
	activeStreams  map[xds.StreamID]bool
}

type tracker struct {
	resourceManager manager.ResourceManager
	config          *dp_server.HdsConfig
	reconciler      *reconciler
	log             logr.Logger

	sync.RWMutex       // protects access to the fields below
	streamsAssociation map[xds.StreamID]core_model.ResourceKey
	dpStreams          map[core_model.ResourceKey]streams
}

func NewCallbacks(
	log logr.Logger,
	resourceManager manager.ResourceManager,
	readOnlyResourceManager manager.ReadOnlyResourceManager,
	cache util_xds_v3.SnapshotCache,
	config *dp_server.HdsConfig,
	hasher util_xds_v3.NodeHash,
	defaultAdminPort uint32,
) hds_callbacks.Callbacks {
	return &tracker{
		resourceManager:    resourceManager,
		streamsAssociation: map[xds.StreamID]core_model.ResourceKey{},
		dpStreams:          map[core_model.ResourceKey]streams{},
		config:             config,
		log:                log,
		reconciler: &reconciler{
			cache:     cache,
			hasher:    hasher,
			versioner: util_xds_v3.SnapshotAutoVersioner{UUID: core.NewUUID},
			generator: NewSnapshotGenerator(readOnlyResourceManager, config, defaultAdminPort),
		},
	}
}

func (t *tracker) OnStreamOpen(ctx context.Context, streamID int64) error {
	return nil
}

func (t *tracker) OnStreamClosed(streamID xds.StreamID) {

	t.Lock()
	defer t.Unlock()

	dp, hasAssociation := t.streamsAssociation[streamID]
	if hasAssociation {
		delete(t.streamsAssociation, streamID)

		streams := t.dpStreams[dp]
		delete(streams.activeStreams, streamID)
		if len(streams.activeStreams) == 0 { // no stream is active, cancel watchdog
			if streams.watchdogCancel != nil {
				streams.watchdogCancel()
			}
			delete(t.dpStreams, dp)
		}
	}
}

func (t *tracker) OnHealthCheckRequest(streamID xds.StreamID, req *envoy_service_health.HealthCheckRequest) error {

	proxyId, err := xds.ParseProxyIdFromString(req.GetNode().GetId())
	if err != nil {
		t.log.Error(err, "failed to parse Dataplane Id out of HealthCheckRequest", "streamid", streamID, "req", req)
		return nil
	}

	dataplaneKey := proxyId.ToResourceKey()

	t.Lock()
	defer t.Unlock()

	streams := t.dpStreams[dataplaneKey]
	if streams.activeStreams == nil {
		streams.activeStreams = map[xds.StreamID]bool{}
	}
	streams.activeStreams[streamID] = true

	if streams.watchdogCancel == nil { // watchdog was not started yet
		stopCh := make(chan struct{})
		streams.watchdogCancel = func() {
			close(stopCh)
		}
		// kick off watchdog for that Dataplane
		go t.newWatchdog(req.Node).Start(stopCh)
		t.log.V(1).Info("started Watchdog for a Dataplane", "streamid", streamID, "proxyId", proxyId, "dataplaneKey", dataplaneKey)
	}
	t.dpStreams[dataplaneKey] = streams
	t.streamsAssociation[streamID] = dataplaneKey
	return nil
}

func (t *tracker) newWatchdog(node *envoy_core.Node) watchdog.Watchdog {
	return &watchdog.SimpleWatchdog{
		NewTicker: func() *time.Ticker {
			return time.NewTicker(t.config.RefreshInterval.Duration)
		},
		OnTick: func(ctx context.Context) error {
			return t.reconciler.Reconcile(ctx, node)
		},
		OnError: func(err error) {
			t.log.Error(err, "OnTick() failed")
		},
		OnStop: func() {
			if err := t.reconciler.Clear(node); err != nil {
				t.log.Error(err, "OnTick() failed")
			}
		},
	}
}

func (t *tracker) OnEndpointHealthResponse(streamID xds.StreamID, resp *envoy_service_health.EndpointHealthResponse) error {

	healthMap := map[uint32]bool{}
	envoyHealth := true // if there is no Envoy HC, assume it's healthy

	for _, clusterHealth := range resp.GetClusterEndpointsHealth() {
		if len(clusterHealth.LocalityEndpointsHealth) == 0 {
			continue
		}
		if len(clusterHealth.LocalityEndpointsHealth[0].EndpointsHealth) == 0 {
			continue
		}
		status := clusterHealth.LocalityEndpointsHealth[0].EndpointsHealth[0].HealthStatus
		health := status == envoy_core.HealthStatus_HEALTHY || status == envoy_core.HealthStatus_UNKNOWN

		if clusterHealth.ClusterName == names.GetEnvoyAdminClusterName() {
			envoyHealth = health
		} else {
			port, err := names.GetPortForLocalClusterName(clusterHealth.ClusterName)
			if err != nil {
				return err
			}
			healthMap[port] = health
		}
	}
	if err := t.updateDataplane(streamID, healthMap, envoyHealth); err != nil {
		return err
	}
	return nil
}

func (t *tracker) updateDataplane(streamID xds.StreamID, healthMap map[uint32]bool, envoyHealth bool) error {
	t.RLock()
	defer t.RUnlock()
	dataplaneKey, hasAssociation := t.streamsAssociation[streamID]
	if !hasAssociation {
		return errors.Errorf("no proxy for streamID = %d", streamID)
	}

	dp := mesh.NewDataplaneResource()
	if err := t.resourceManager.Get(context.Background(), dp, store.GetBy(dataplaneKey)); err != nil {
		return err
	}

	changed := false
	for _, inbound := range dp.Spec.Networking.Inbound {
		intf := dp.Spec.Networking.ToInboundInterface(inbound)
		workloadHealth, exist := healthMap[intf.WorkloadPort]
		if exist {
			workloadHealth = workloadHealth && envoyHealth
		} else {
			workloadHealth = envoyHealth
		}
		if workloadHealth && inbound.State == mesh_proto.Dataplane_Networking_Inbound_NotReady {
			inbound.State = mesh_proto.Dataplane_Networking_Inbound_Ready
			// write health for backwards compatibility with Kuma 2.5 and older
			inbound.Health = &mesh_proto.Dataplane_Networking_Inbound_Health{
				Ready: true,
			}
			changed = true
		} else if !workloadHealth && inbound.State == mesh_proto.Dataplane_Networking_Inbound_Ready {
			inbound.State = mesh_proto.Dataplane_Networking_Inbound_NotReady
			// write health for backwards compatibility with Kuma 2.5 and older
			inbound.Health = &mesh_proto.Dataplane_Networking_Inbound_Health{
				Ready: false,
			}
			changed = true
		}
	}

	if changed {
		t.log.V(1).Info("status updated", "dataplaneKey", dataplaneKey)
		return t.resourceManager.Update(context.Background(), dp)
	}

	return nil
}
