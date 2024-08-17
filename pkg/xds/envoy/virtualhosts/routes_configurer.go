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
	"sort"
)

import (
	envoy_config_route_v3 "github.com/envoyproxy/go-control-plane/envoy/config/route/v3"
	envoy_type_matcher_v3 "github.com/envoyproxy/go-control-plane/envoy/type/matcher/v3"
)

import (
	core_xds "github.com/apache/dubbo-kubernetes/pkg/core/xds"
	util_proto "github.com/apache/dubbo-kubernetes/pkg/util/proto"
	envoy_common "github.com/apache/dubbo-kubernetes/pkg/xds/envoy"
)

type RoutesConfigurer struct {
	Routes envoy_common.Routes
}

func (c RoutesConfigurer) Configure(virtualHost *envoy_config_route_v3.VirtualHost) error {
	for i := range c.Routes {
		route := c.Routes[i]
		envoyRoute := &envoy_config_route_v3.Route{
			Match: c.routeMatch(route.Match),
			Name:  envoy_common.AnonymousResource,
			Action: &envoy_config_route_v3.Route_Route{
				Route: c.routeAction(route.Clusters), // need add modify
			},
		}

		// need add (rateLimit in preFilter) (modify operator)

		virtualHost.Routes = append(virtualHost.Routes, envoyRoute)
	}
	return nil
}

func (c RoutesConfigurer) routeMatch(match *core_xds.TrafficRouteHttpMatch) *envoy_config_route_v3.RouteMatch {
	if match == nil {
		return &envoy_config_route_v3.RouteMatch{
			PathSpecifier: &envoy_config_route_v3.RouteMatch_Prefix{
				Prefix: "/",
			},
		}
	}
	envoyMatch := &envoy_config_route_v3.RouteMatch{}

	if match.GetPath() != nil {
		c.setPathMatcher(match.GetPath(), envoyMatch)
	} else {
		// Path match is required on Envoy config so if there is only matching by header in TrafficRoute, we need to place
		// the default route match anyways.
		envoyMatch.PathSpecifier = &envoy_config_route_v3.RouteMatch_Prefix{
			Prefix: "/",
		}
	}

	var headers []string
	for headerName := range match.GetHeaders() {
		headers = append(headers, headerName)
	}
	sort.Strings(headers) // sort for stability of Envoy config
	for _, headerName := range headers {
		envoyMatch.Headers = append(envoyMatch.Headers, c.headerMatcher(headerName, match.Headers[headerName]))
	}

	if match.GetMethod() != nil {
		envoyMatch.Headers = append(envoyMatch.Headers, c.headerMatcher(":method", match.GetMethod()))
	}

	for matcher_k, matcher_v := range match.GetParam() {
		if envoyMatch.QueryParameters == nil {
			envoyMatch.QueryParameters = make([]*envoy_config_route_v3.QueryParameterMatcher, 0, len(match.GetParam()))
		}
		envoyMatch.QueryParameters = append(envoyMatch.QueryParameters, &envoy_config_route_v3.QueryParameterMatcher{
			Name: matcher_k,
			QueryParameterMatchSpecifier: &envoy_config_route_v3.QueryParameterMatcher_StringMatch{
				StringMatch: envoy_common.TrafficRouteHttpMatchStringMatcherToEnvoyStringMatcher(matcher_v),
			},
		})
	}

	return envoyMatch
}

func (c RoutesConfigurer) headerMatcher(name string, matcher core_xds.TrafficRouteHttpStringMatcher) *envoy_config_route_v3.HeaderMatcher {
	headerMatcher := &envoy_config_route_v3.HeaderMatcher{
		Name: name,
	}
	switch matcher.(type) {
	case *core_xds.TrafficRouteHttpMatcherPrefix:
		headerMatcher.HeaderMatchSpecifier = &envoy_config_route_v3.HeaderMatcher_PrefixMatch{
			PrefixMatch: matcher.GetValue(),
		}
	case *core_xds.TrafficRouteHttpMatcherExact:
		stringMatcher := envoy_type_matcher_v3.StringMatcher{
			MatchPattern: &envoy_type_matcher_v3.StringMatcher_Exact{
				Exact: matcher.GetValue(),
			},
		}
		headerMatcher.HeaderMatchSpecifier = &envoy_config_route_v3.HeaderMatcher_StringMatch{
			StringMatch: &stringMatcher,
		}
	case *core_xds.TrafficRouteHttpMatcherRegex:
		headerMatcher.HeaderMatchSpecifier = &envoy_config_route_v3.HeaderMatcher_SafeRegexMatch{
			SafeRegexMatch: &envoy_type_matcher_v3.RegexMatcher{
				Regex: matcher.GetValue(),
			},
		}
	}
	return headerMatcher
}

func (c RoutesConfigurer) setPathMatcher(
	matcher core_xds.TrafficRouteHttpStringMatcher,
	routeMatch *envoy_config_route_v3.RouteMatch,
) {
	switch matcher.(type) {
	case *core_xds.TrafficRouteHttpMatcherPrefix:
		routeMatch.PathSpecifier = &envoy_config_route_v3.RouteMatch_Prefix{
			Prefix: matcher.GetValue(),
		}
	case *core_xds.TrafficRouteHttpMatcherExact:
		routeMatch.PathSpecifier = &envoy_config_route_v3.RouteMatch_Path{
			Path: matcher.GetValue(),
		}
	case *core_xds.TrafficRouteHttpMatcherRegex:
		routeMatch.PathSpecifier = &envoy_config_route_v3.RouteMatch_SafeRegex{
			SafeRegex: &envoy_type_matcher_v3.RegexMatcher{
				Regex: matcher.GetValue(),
			},
		}
	}
}

func (c RoutesConfigurer) hasExternal(clusters []envoy_common.Cluster) bool {
	for _, cluster := range clusters {
		if cluster.IsExternalService() {
			return true
		}
	}
	return false
}

func (c RoutesConfigurer) routeAction(clusters []envoy_common.Cluster) *envoy_config_route_v3.RouteAction {
	routeAction := &envoy_config_route_v3.RouteAction{}
	//if len(clusters) != 0 {
	//	// Timeout can be configured only per outbound listener. So all clusters in the split
	//	// must have the same timeout. That's why we can take the timeout from the first cluster.
	//	cluster := clusters[0].(*envoy_common.ClusterImpl)
	//	routeAction.Timeout = util_proto.Duration(cluster.Timeout().GetHttp().GetRequestTimeout().AsDuration())
	//}
	// here need to add timeout
	if len(clusters) == 1 {
		routeAction.ClusterSpecifier = &envoy_config_route_v3.RouteAction_Cluster{
			Cluster: clusters[0].Name(),
		}
	} else {
		var weightedClusters []*envoy_config_route_v3.WeightedCluster_ClusterWeight
		for _, c := range clusters {
			cluster := c.(*envoy_common.ClusterImpl)
			weightedClusters = append(weightedClusters, &envoy_config_route_v3.WeightedCluster_ClusterWeight{
				Name:   cluster.Name(),
				Weight: util_proto.UInt32(cluster.Weight()),
			})
		}
		routeAction.ClusterSpecifier = &envoy_config_route_v3.RouteAction_WeightedClusters{
			WeightedClusters: &envoy_config_route_v3.WeightedCluster{
				Clusters: weightedClusters,
			},
		}
	}
	if c.hasExternal(clusters) {
		routeAction.HostRewriteSpecifier = &envoy_config_route_v3.RouteAction_AutoHostRewrite{
			AutoHostRewrite: util_proto.Bool(true),
		}
	}
	//need to add modify
	//c.setModifications(routeAction, modify)
	return routeAction
}
