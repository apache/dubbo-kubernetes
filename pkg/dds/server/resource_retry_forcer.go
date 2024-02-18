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
	"errors"
	"github.com/apache/dubbo-kubernetes/pkg/core/resources/model"
	"github.com/apache/dubbo-kubernetes/pkg/core/xds"
	"github.com/apache/dubbo-kubernetes/pkg/dds/cache"
	util_xds_v3 "github.com/apache/dubbo-kubernetes/pkg/util/xds/v3"
	envoy_core "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	envoy_sd "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v3"
	envoy_cache "github.com/envoyproxy/go-control-plane/pkg/cache/v3"
	envoy_xds "github.com/envoyproxy/go-control-plane/pkg/server/v3"
	"github.com/go-logr/logr"
	"sync"
)

type ddsRetryForcer struct {
	util_xds_v3.NoopCallbacks
	hasher  envoy_cache.NodeHash
	cache   envoy_cache.SnapshotCache
	log     logr.Logger
	nodeIDs map[xds.StreamID]string

	sync.Mutex
}

func newDdsRetryForcer(log logr.Logger, cache envoy_cache.SnapshotCache, hasher envoy_cache.NodeHash) *ddsRetryForcer {
	return &ddsRetryForcer{
		cache:   cache,
		hasher:  hasher,
		log:     log,
		nodeIDs: map[xds.StreamID]string{},
	}
}

var _ envoy_xds.Callbacks = &ddsRetryForcer{}

func (r *ddsRetryForcer) OnDeltaStreamClosed(streamID int64, _ *envoy_core.Node) {
	r.Lock()
	defer r.Unlock()
	delete(r.nodeIDs, streamID)
}

func (r *ddsRetryForcer) OnStreamDeltaRequest(streamID xds.StreamID, request *envoy_sd.DeltaDiscoveryRequest) error {
	if request.ResponseNonce == "" {
		return nil // initial request, no need to force warming
	}

	if request.ErrorDetail == nil {
		return nil // not NACK, no need to retry
	}

	r.Lock()
	nodeID := r.nodeIDs[streamID]
	if nodeID == "" {
		nodeID = r.hasher.ID(request.Node) // request.Node can be set only on first request therefore we need to save it
		r.nodeIDs[streamID] = nodeID
	}
	r.Unlock()
	r.log.Info("received NACK", "nodeID", nodeID, "type", request.TypeUrl, "err", request.GetErrorDetail().GetMessage())
	snapshot, err := r.cache.GetSnapshot(nodeID)
	if err != nil {
		return nil // GetSnapshot returns an error if there is no snapshot. We don't need to force on a new snapshot
	}
	cacheSnapshot, ok := snapshot.(*cache.Snapshot)
	if !ok {
		return errors.New("couldn't convert snapshot from cache to envoy Snapshot")
	}
	for resourceName := range cacheSnapshot.VersionMap[model.ResourceType(request.TypeUrl)] {
		cacheSnapshot.VersionMap[model.ResourceType(request.TypeUrl)][resourceName] = ""
	}

	r.log.V(1).Info("forced the new verion of resources", "nodeID", nodeID, "type", request.TypeUrl)
	return nil
}
