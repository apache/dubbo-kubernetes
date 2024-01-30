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

package virtualhosts

import (
	envoy_config_core_v3 "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	envoy_config_route_v3 "github.com/envoyproxy/go-control-plane/envoy/config/route/v3"
)

import (
	envoy_common "github.com/apache/dubbo-kubernetes/pkg/xds/envoy"
)

func DomainNames(domainNames ...string) VirtualHostBuilderOpt {
	if len(domainNames) == 0 {
		return VirtualHostBuilderOptFunc(nil)
	}

	return AddVirtualHostConfigurer(
		VirtualHostMustConfigureFunc(func(vh *envoy_config_route_v3.VirtualHost) {
			vh.Domains = domainNames
		}),
	)
}

func Routes(routes envoy_common.Routes) VirtualHostBuilderOpt {
	return AddVirtualHostConfigurer(
		&RoutesConfigurer{
			Routes: routes,
		})
}

// Redirect for paths that match to matchPath returns 301 status code with new port and path
func Redirect(matchPath, newPath string, allowGetOnly bool, port uint32) VirtualHostBuilderOpt {
	return AddVirtualHostConfigurer(&RedirectConfigurer{
		MatchPath:    matchPath,
		NewPath:      newPath,
		Port:         port,
		AllowGetOnly: allowGetOnly,
	})
}

// RequireTLS specifies that this virtual host must only accept TLS connections.
func RequireTLS() VirtualHostBuilderOpt {
	return AddVirtualHostConfigurer(
		VirtualHostMustConfigureFunc(func(vh *envoy_config_route_v3.VirtualHost) {
			vh.RequireTls = envoy_config_route_v3.VirtualHost_ALL
		}),
	)
}

// SetResponseHeader unconditionally sets the named response header to the given value.
func SetResponseHeader(name string, value string) VirtualHostBuilderOpt {
	return AddVirtualHostConfigurer(
		VirtualHostMustConfigureFunc(func(vh *envoy_config_route_v3.VirtualHost) {
			hsts := &envoy_config_core_v3.HeaderValueOption{
				AppendAction: envoy_config_core_v3.HeaderValueOption_OVERWRITE_IF_EXISTS_OR_ADD,
				Header: &envoy_config_core_v3.HeaderValue{
					Key:   name,
					Value: value,
				},
			}

			vh.ResponseHeadersToAdd = append(vh.ResponseHeadersToAdd, hsts)
		}),
	)
}

func Route(matchPath, newPath, cluster string, allowGetOnly bool) VirtualHostBuilderOpt {
	return AddVirtualHostConfigurer(
		&VirtualHostRouteConfigurer{
			MatchPath:    matchPath,
			NewPath:      newPath,
			Cluster:      cluster,
			AllowGetOnly: allowGetOnly,
		})
}
