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
	envoy_common "github.com/apache/dubbo-kubernetes/pkg/xds/envoy"
	envoy_config_route_v3 "github.com/envoyproxy/go-control-plane/envoy/config/route/v3"
	envoy_type_matcher_v3 "github.com/envoyproxy/go-control-plane/envoy/type/matcher/v3"
)

type VirtualHostRouteConfigurer struct {
	MatchPath    string
	NewPath      string
	Cluster      string
	AllowGetOnly bool
}

func (c VirtualHostRouteConfigurer) Configure(virtualHost *envoy_config_route_v3.VirtualHost) error {
	var headersMatcher []*envoy_config_route_v3.HeaderMatcher
	if c.AllowGetOnly {
		matcher := envoy_type_matcher_v3.StringMatcher{
			MatchPattern: &envoy_type_matcher_v3.StringMatcher_Exact{
				Exact: "GET",
			},
		}
		headersMatcher = []*envoy_config_route_v3.HeaderMatcher{
			{
				Name: ":method",
				HeaderMatchSpecifier: &envoy_config_route_v3.HeaderMatcher_StringMatch{
					StringMatch: &matcher,
				},
			},
		}
	}
	virtualHost.Routes = append(virtualHost.Routes, &envoy_config_route_v3.Route{
		Match: &envoy_config_route_v3.RouteMatch{
			PathSpecifier: &envoy_config_route_v3.RouteMatch_Path{
				Path: c.MatchPath,
			},
			Headers: headersMatcher,
		},
		Name: envoy_common.AnonymousResource,
		Action: &envoy_config_route_v3.Route_Route{
			Route: &envoy_config_route_v3.RouteAction{
				RegexRewrite: &envoy_type_matcher_v3.RegexMatchAndSubstitute{
					Pattern: &envoy_type_matcher_v3.RegexMatcher{
						EngineType: &envoy_type_matcher_v3.RegexMatcher_GoogleRe2{
							GoogleRe2: &envoy_type_matcher_v3.RegexMatcher_GoogleRE2{},
						},
						Regex: `.*`,
					},
					Substitution: c.NewPath,
				},
				ClusterSpecifier: &envoy_config_route_v3.RouteAction_Cluster{
					Cluster: c.Cluster,
				},
			},
		},
	})
	return nil
}
