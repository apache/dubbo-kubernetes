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

type Chain []Callbacks

var _ Callbacks = Chain{}

func (chain Chain) OnStreamOpen(ctx context.Context, streamID int64) error {
	for _, cb := range chain {
		if err := cb.OnStreamOpen(ctx, streamID); err != nil {
			return err
		}
	}
	return nil
}

func (chain Chain) OnHealthCheckRequest(streamID int64, request *envoy_service_health.HealthCheckRequest) error {
	for _, cb := range chain {
		if err := cb.OnHealthCheckRequest(streamID, request); err != nil {
			return err
		}
	}
	return nil
}

func (chain Chain) OnEndpointHealthResponse(streamID int64, response *envoy_service_health.EndpointHealthResponse) error {
	for _, cb := range chain {
		if err := cb.OnEndpointHealthResponse(streamID, response); err != nil {
			return err
		}
	}
	return nil
}

func (chain Chain) OnStreamClosed(streamID int64) {
	for i := len(chain) - 1; i >= 0; i-- {
		cb := chain[i]
		cb.OnStreamClosed(streamID)
	}
}
