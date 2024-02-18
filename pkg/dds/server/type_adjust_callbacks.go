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
	"github.com/apache/dubbo-kubernetes/pkg/dds"
	util_xds_v3 "github.com/apache/dubbo-kubernetes/pkg/util/xds/v3"
	envoy_sd "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v3"
)

// We are using go-control-plane's server and cache for DDA exchange.
// We are setting TypeURL for DeltaDiscoveryRequest/DeltaDiscoveryResponse for our resource name like "TrafficRoute" / "Mesh" etc.
// but the actual resource which we are sending is dubbo.mesh.v1alpha1.DubboResource
//
// The function which is marshaling DeltaDiscoveryResponse
// func (r *RawDeltaResponse) GetDeltaDiscoveryResponse() (*discovery.DeltaDiscoveryResponse, error)
// Ignores the TypeURL from marshaling operation and overrides it with TypeURL of the request.
// If we pass wrong TypeURL in envoy_api.DeltaDiscoveryResponse#Resources we won't be able to unmarshall it, therefore we need to adjust the type.
type typeAdjustCallbacks struct {
	util_xds_v3.NoopCallbacks
}

func (c *typeAdjustCallbacks) OnStreamDeltaResponse(streamID int64, req *envoy_sd.DeltaDiscoveryRequest, resp *envoy_sd.DeltaDiscoveryResponse) {
	for _, res := range resp.GetResources() {
		res.Resource.TypeUrl = dds.DubboResource
	}
}
