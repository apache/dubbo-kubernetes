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

import envoy_config_route_v3 "github.com/envoyproxy/go-control-plane/envoy/config/route/v3"

// RouteConfigurationConfigurer is responsible for configuring a single aspect of the entire Envoy RouteConfiguration,
// such as VirtualHost, HTTP headers to add or remove, etc.
type RouteConfigurationConfigurer interface {
	// Configure configures a single aspect on a given Envoy RouteConfiguration.
	Configure(routeConfiguration *envoy_config_route_v3.RouteConfiguration) error
}

// RouteConfigurationConfigureFunc adapts a configuration function to the
// RouteConfigurationConfigurer interface.
type RouteConfigurationConfigureFunc func(rc *envoy_config_route_v3.RouteConfiguration) error

func (f RouteConfigurationConfigureFunc) Configure(rc *envoy_config_route_v3.RouteConfiguration) error {
	if f != nil {
		return f(rc)
	}

	return nil
}

// RouteConfigurationMustConfigureFunc adapts a configuration function that
// never fails to the RouteConfigurationConfigurer interface.
type RouteConfigurationMustConfigureFunc func(rc *envoy_config_route_v3.RouteConfiguration)

func (f RouteConfigurationMustConfigureFunc) Configure(rc *envoy_config_route_v3.RouteConfiguration) error {
	if f != nil {
		f(rc)
	}

	return nil
}
