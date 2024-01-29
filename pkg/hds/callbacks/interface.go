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

package callbacks

import (
	"context"
	envoy_service_health "github.com/envoyproxy/go-control-plane/envoy/service/health/v3"
)

type Callbacks interface {
	// OnStreamOpen is called once an HDS stream is open with a stream ID and context
	// Returning an error will end processing and close the stream. OnStreamClosed will still be called.
	OnStreamOpen(ctx context.Context, streamID int64) error

	// OnHealthCheckRequest is called when Envoy sends HealthCheckRequest with Node and Capabilities
	OnHealthCheckRequest(streamID int64, request *envoy_service_health.HealthCheckRequest) error

	// OnEndpointHealthResponse is called when there is a response from Envoy with status of endpoints in the cluster
	OnEndpointHealthResponse(streamID int64, response *envoy_service_health.EndpointHealthResponse) error

	// OnStreamClosed is called immediately prior to closing an xDS stream with a stream ID.
	OnStreamClosed(int64)
}
