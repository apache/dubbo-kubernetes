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
	envoy_listener "github.com/envoyproxy/go-control-plane/envoy/config/listener/v3"
	envoy_grpc_stats "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/http/grpc_stats/v3"
	envoy_hcm "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/network/http_connection_manager/v3"
)

import (
	util_proto "github.com/apache/dubbo-kubernetes/pkg/util/proto"
)

type GrpcStatsConfigurer struct{}

var _ FilterChainConfigurer = &GrpcStatsConfigurer{}

func (g *GrpcStatsConfigurer) Configure(filterChain *envoy_listener.FilterChain) error {
	config := &envoy_grpc_stats.FilterConfig{
		EmitFilterState: true,
	}
	pbst, err := util_proto.MarshalAnyDeterministic(config)
	if err != nil {
		return err
	}
	return UpdateHTTPConnectionManager(filterChain, func(manager *envoy_hcm.HttpConnectionManager) error {
		manager.HttpFilters = append(manager.HttpFilters,
			&envoy_hcm.HttpFilter{
				Name: "envoy.filters.http.grpc_stats",
				ConfigType: &envoy_hcm.HttpFilter_TypedConfig{
					TypedConfig: pbst,
				},
			})
		return nil
	})
}
