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
)

import (
	core_xds "github.com/apache/dubbo-kubernetes/pkg/core/xds"
	envoy_common "github.com/apache/dubbo-kubernetes/pkg/xds/envoy"
	envoy_names "github.com/apache/dubbo-kubernetes/pkg/xds/envoy/names"
	envoy_routes "github.com/apache/dubbo-kubernetes/pkg/xds/envoy/routes"
	envoy_virtual_hosts "github.com/apache/dubbo-kubernetes/pkg/xds/envoy/virtualhosts"
)

type HttpInboundRouteConfigurer struct {
	Service string
	Routes  envoy_common.Routes
}

var _ FilterChainConfigurer = &HttpInboundRouteConfigurer{}

func (c *HttpInboundRouteConfigurer) Configure(filterChain *envoy_listener.FilterChain) error {
	routeName := envoy_names.GetInboundRouteName(c.Service)

	static := HttpStaticRouteConfigurer{
		Builder: envoy_routes.NewRouteConfigurationBuilder(core_xds.APIVersion(envoy_common.APIV3), routeName).
			Configure(envoy_routes.CommonRouteConfiguration()).
			Configure(envoy_routes.ResetTagsHeader()).
			Configure(envoy_routes.VirtualHost(envoy_virtual_hosts.NewVirtualHostBuilder(core_xds.APIVersion(envoy_common.APIV3), c.Service).
				Configure(envoy_virtual_hosts.Routes(c.Routes)))),
	}

	return static.Configure(filterChain)
}
