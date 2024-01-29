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

package routes

import (
	envoy_config_route_v3 "github.com/envoyproxy/go-control-plane/envoy/config/route/v3"
)

import (
	mesh_proto "github.com/apache/dubbo-kubernetes/api/mesh/v1alpha1"
	v3 "github.com/apache/dubbo-kubernetes/pkg/xds/envoy/routes/v3"
	envoy_virtual_hosts "github.com/apache/dubbo-kubernetes/pkg/xds/envoy/virtualhosts"
)

// ResetTagsHeader adds x-kuma-tags header to the RequestHeadersToRemove list. x-kuma-tags header is planned to be used
// internally, so we don't want to expose it to the destination application.
func ResetTagsHeader() RouteConfigurationBuilderOpt {
	return AddRouteConfigurationConfigurer(&v3.ResetTagsHeaderConfigurer{})
}

func TagsHeader(tags mesh_proto.MultiValueTagSet) RouteConfigurationBuilderOpt {
	return AddRouteConfigurationConfigurer(
		&v3.TagsHeaderConfigurer{
			Tags: tags,
		})
}

func VirtualHost(builder *envoy_virtual_hosts.VirtualHostBuilder) RouteConfigurationBuilderOpt {
	return AddRouteConfigurationConfigurer(
		v3.RouteConfigurationConfigureFunc(func(rc *envoy_config_route_v3.RouteConfiguration) error {
			virtualHost, err := builder.Build()
			if err != nil {
				return err
			}
			rc.VirtualHosts = append(rc.VirtualHosts, virtualHost.(*envoy_config_route_v3.VirtualHost))
			return nil
		}))
}

func CommonRouteConfiguration() RouteConfigurationBuilderOpt {
	return AddRouteConfigurationConfigurer(
		&v3.CommonRouteConfigurationConfigurer{})
}

func IgnorePortInHostMatching() RouteConfigurationBuilderOpt {
	return AddRouteConfigurationConfigurer(
		v3.RouteConfigurationConfigureFunc(func(rc *envoy_config_route_v3.RouteConfiguration) error {
			rc.IgnorePortInHostMatching = true
			return nil
		}),
	)
}
