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

package mux

import (
	envoy_api_v2 "github.com/envoyproxy/go-control-plane/envoy/api/v2"
	envoy_core_v3 "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	envoy_sd_v3 "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v3"
)

const (
	DDSVersionHeaderKey = "dds-version"
	DDSVersionV3        = "v3"
)

func DiscoveryRequestV3(request *envoy_api_v2.DiscoveryRequest) *envoy_sd_v3.DiscoveryRequest {
	return &envoy_sd_v3.DiscoveryRequest{
		VersionInfo: request.VersionInfo,
		Node: &envoy_core_v3.Node{
			Id:       request.Node.Id,
			Metadata: request.Node.Metadata,
		},
		ResourceNames: request.ResourceNames,
		TypeUrl:       request.TypeUrl,
		ResponseNonce: request.ResponseNonce,
		ErrorDetail:   request.ErrorDetail,
	}
}

func DiscoveryResponseV3(response *envoy_api_v2.DiscoveryResponse) *envoy_sd_v3.DiscoveryResponse {
	return &envoy_sd_v3.DiscoveryResponse{
		VersionInfo: response.VersionInfo,
		Resources:   response.Resources,
		TypeUrl:     response.TypeUrl,
		Nonce:       response.Nonce,
		ControlPlane: &envoy_core_v3.ControlPlane{
			Identifier: response.ControlPlane.Identifier,
		},
	}
}
