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
	util_proto "github.com/apache/dubbo-kubernetes/pkg/util/proto"
	util_xds "github.com/apache/dubbo-kubernetes/pkg/util/xds"
	envoy_common "github.com/apache/dubbo-kubernetes/pkg/xds/envoy"
	envoy_core_v3 "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	envoy_listener "github.com/envoyproxy/go-control-plane/envoy/config/listener/v3"
	envoy_route "github.com/envoyproxy/go-control-plane/envoy/config/route/v3"
	envoy_hcm "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/network/http_connection_manager/v3"
	"google.golang.org/protobuf/types/known/anypb"
)

type DirectResponseConfigurer struct {
	VirtualHostName string
	Endpoints       []DirectResponseEndpoints
}

type DirectResponseEndpoints struct {
	Path       string
	StatusCode uint32
	Response   string
}

var _ FilterChainConfigurer = &DirectResponseConfigurer{}

func (c *DirectResponseConfigurer) Configure(filterChain *envoy_listener.FilterChain) error {
	httpFilters := []*envoy_hcm.HttpFilter{
		{
			Name: "envoy.filters.http.router",
			ConfigType: &envoy_hcm.HttpFilter_TypedConfig{
				TypedConfig: &anypb.Any{
					TypeUrl: "type.googleapis.com/envoy.extensions.filters.http.router.v3.Router",
				},
			},
		},
	}

	var routes []*envoy_route.Route
	for _, endpoint := range c.Endpoints {
		routes = append(routes, &envoy_route.Route{
			Match: &envoy_route.RouteMatch{
				PathSpecifier: &envoy_route.RouteMatch_Prefix{
					Prefix: endpoint.Path,
				},
			},
			Name: envoy_common.AnonymousResource,
			Action: &envoy_route.Route_DirectResponse{
				DirectResponse: &envoy_route.DirectResponseAction{
					Status: endpoint.StatusCode,
					Body: &envoy_core_v3.DataSource{
						Specifier: &envoy_core_v3.DataSource_InlineString{InlineString: endpoint.Response},
					},
				},
			},
		})
	}

	config := &envoy_hcm.HttpConnectionManager{
		StatPrefix:  util_xds.SanitizeMetric(c.VirtualHostName),
		CodecType:   envoy_hcm.HttpConnectionManager_AUTO,
		HttpFilters: httpFilters,
		RouteSpecifier: &envoy_hcm.HttpConnectionManager_RouteConfig{
			RouteConfig: &envoy_route.RouteConfiguration{
				VirtualHosts: []*envoy_route.VirtualHost{{
					Name:    c.VirtualHostName,
					Domains: []string{"*"},
					Routes:  routes,
				}},
			},
		},
	}
	pbst, err := util_proto.MarshalAnyDeterministic(config)
	if err != nil {
		return err
	}

	filterChain.Filters = append(filterChain.Filters, &envoy_listener.Filter{
		Name: "envoy.filters.network.http_connection_manager",
		ConfigType: &envoy_listener.Filter_TypedConfig{
			TypedConfig: pbst,
		},
	})
	return nil
}
