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
	envoy_config_core_v3 "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	envoy_config_route_v3 "github.com/envoyproxy/go-control-plane/envoy/config/route/v3"
	envoy_type_matcher_v3 "github.com/envoyproxy/go-control-plane/envoy/type/matcher/v3"
)

import (
	mesh_proto "github.com/apache/dubbo-kubernetes/api/mesh/v1alpha1"
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
			Name:  envoy_common.AnonymousResource,
			Match: c.routeMatch(route.Match),
			Action: &envoy_config_route_v3.Route_Route{
				Route: c.routeAction(route.Clusters, route.Modify), // need add modify
			},
		}

		c.setHeadersModifications(envoyRoute, route.Modify)

		virtualHost.Routes = append(virtualHost.Routes, envoyRoute)
	}
	return nil
}

func (c RoutesConfigurer) routeMatch(match *mesh_proto.TrafficRoute_Http_Match) *envoy_config_route_v3.RouteMatch {
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

	for match_k, match_v := range match.GetHeaders() {
		if envoyMatch.Headers == nil {
			envoyMatch.Headers = make([]*envoy_config_route_v3.HeaderMatcher, 0, len(match.GetHeaders()))
		}
		hm := c.headerMatcher(match_k, match_v)
		envoyMatch.Headers = append(envoyMatch.Headers, hm)
	}

	for matcher_k, matcher_v := range match.GetParam() {
		if envoyMatch.QueryParameters == nil {
			envoyMatch.QueryParameters = make([]*envoy_config_route_v3.QueryParameterMatcher, 0, len(match.GetParam()))
		}
		envoyMatch.QueryParameters = append(envoyMatch.QueryParameters, &envoy_config_route_v3.QueryParameterMatcher{
			Name: matcher_k,
			QueryParameterMatchSpecifier: &envoy_config_route_v3.QueryParameterMatcher_StringMatch{
				StringMatch: TrafficRouteHttpMatchStringMatcherToEnvoyStringMatcher(matcher_v),
			},
		})
	}

	return envoyMatch
}

func (c RoutesConfigurer) headerMatcher(name string, matcher *mesh_proto.StringMatch) *envoy_config_route_v3.HeaderMatcher {
	headerMatcher := &envoy_config_route_v3.HeaderMatcher{
		Name: name,
	}
	switch matcher.GetMatchType().(type) {
	case *mesh_proto.StringMatch_Prefix:
		headerMatcher.HeaderMatchSpecifier = &envoy_config_route_v3.HeaderMatcher_PrefixMatch{
			PrefixMatch: matcher.GetPrefix(),
		}
	case *mesh_proto.StringMatch_Exact:
		stringMatcher := envoy_type_matcher_v3.StringMatcher{
			MatchPattern: &envoy_type_matcher_v3.StringMatcher_Exact{
				Exact: matcher.GetExact(),
			},
		}
		headerMatcher.HeaderMatchSpecifier = &envoy_config_route_v3.HeaderMatcher_StringMatch{
			StringMatch: &stringMatcher,
		}
	case *mesh_proto.StringMatch_Regex:
		headerMatcher.HeaderMatchSpecifier = &envoy_config_route_v3.HeaderMatcher_SafeRegexMatch{
			SafeRegexMatch: &envoy_type_matcher_v3.RegexMatcher{
				Regex: matcher.GetRegex(),
			},
		}
	}
	return headerMatcher
}

func (c RoutesConfigurer) setPathMatcher(
	matcher *mesh_proto.StringMatch,
	routeMatch *envoy_config_route_v3.RouteMatch,
) {
	switch matcher.GetMatchType().(type) {
	case *mesh_proto.StringMatch_Prefix:
		routeMatch.PathSpecifier = &envoy_config_route_v3.RouteMatch_Prefix{
			Prefix: matcher.GetPrefix(),
		}
	case *mesh_proto.StringMatch_Exact:
		routeMatch.PathSpecifier = &envoy_config_route_v3.RouteMatch_Path{
			Path: matcher.GetExact(),
		}
	case *mesh_proto.StringMatch_Regex:
		routeMatch.PathSpecifier = &envoy_config_route_v3.RouteMatch_SafeRegex{
			SafeRegex: &envoy_type_matcher_v3.RegexMatcher{
				Regex: matcher.GetRegex(),
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

func (c RoutesConfigurer) routeAction(clusters []envoy_common.Cluster, modify *mesh_proto.TrafficRoute_Http_Modify) *envoy_config_route_v3.RouteAction {
	routeAction := &envoy_config_route_v3.RouteAction{}
	if modify.TimeOut != nil {
		routeAction.Timeout = modify.TimeOut
	} else if len(clusters) != 0 {
		// Timeout can be configured only per outbound listener. So all clusters in the split
		// must have the same timeout. That's why we can take the timeout from the first cluster.
		cluster := clusters[0].(*envoy_common.ClusterImpl)
		routeAction.Timeout = util_proto.Duration(cluster.Timeout())
	}
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
	c.setModifications(routeAction, modify)
	return routeAction
}

func (c RoutesConfigurer) setModifications(action *envoy_config_route_v3.RouteAction, modify *mesh_proto.TrafficRoute_Http_Modify) {
	if modify.GetPath() != nil {
		switch modify.GetPath().Type.(type) {
		case *mesh_proto.TrafficRoute_Http_Modify_Path_RewritePrefix:
			action.PrefixRewrite = modify.GetPath().GetRewritePrefix()
		case *mesh_proto.TrafficRoute_Http_Modify_Path_Regex:
			action.RegexRewrite = &envoy_type_matcher_v3.RegexMatchAndSubstitute{
				Pattern: &envoy_type_matcher_v3.RegexMatcher{
					Regex: modify.GetPath().GetRegex().GetPattern(),
				},
				Substitution: modify.GetPath().GetRegex().GetSubstitution(),
			}
		}
	}

	if modify.GetHost() != nil {
		switch modify.GetHost().Type.(type) {
		case *mesh_proto.TrafficRoute_Http_Modify_Host_Value:
			action.HostRewriteSpecifier = &envoy_config_route_v3.RouteAction_HostRewriteLiteral{
				HostRewriteLiteral: modify.GetHost().GetValue(),
			}
		case *mesh_proto.TrafficRoute_Http_Modify_Host_FromPath:
			action.HostRewriteSpecifier = &envoy_config_route_v3.RouteAction_HostRewritePathRegex{
				HostRewritePathRegex: &envoy_type_matcher_v3.RegexMatchAndSubstitute{
					Pattern: &envoy_type_matcher_v3.RegexMatcher{
						Regex: modify.GetHost().GetFromPath().GetPattern(),
					},
					Substitution: modify.GetHost().GetFromPath().GetSubstitution(),
				},
			}
		}
	}
}

func (c RoutesConfigurer) setHeadersModifications(route *envoy_config_route_v3.Route, modify *mesh_proto.TrafficRoute_Http_Modify) {
	for _, add := range modify.GetRequestHeaders().GetAdd() {
		appendAction := envoy_config_core_v3.HeaderValueOption_OVERWRITE_IF_EXISTS_OR_ADD
		if add.Append {
			appendAction = envoy_config_core_v3.HeaderValueOption_APPEND_IF_EXISTS_OR_ADD
		}
		route.RequestHeadersToAdd = append(route.RequestHeadersToAdd, &envoy_config_core_v3.HeaderValueOption{
			Header: &envoy_config_core_v3.HeaderValue{
				Key:   add.Name,
				Value: add.Value,
			},
			AppendAction: appendAction,
		})
	}
	for _, remove := range modify.GetRequestHeaders().GetRemove() {
		route.RequestHeadersToRemove = append(route.RequestHeadersToRemove, remove.Name)
	}

	for _, add := range modify.GetResponseHeaders().GetAdd() {
		appendAction := envoy_config_core_v3.HeaderValueOption_OVERWRITE_IF_EXISTS_OR_ADD
		if add.Append {
			appendAction = envoy_config_core_v3.HeaderValueOption_APPEND_IF_EXISTS_OR_ADD
		}
		route.ResponseHeadersToAdd = append(route.ResponseHeadersToAdd, &envoy_config_core_v3.HeaderValueOption{
			Header: &envoy_config_core_v3.HeaderValue{
				Key:   add.Name,
				Value: add.Value,
			},
			AppendAction: appendAction,
		})
	}
	for _, remove := range modify.GetResponseHeaders().GetRemove() {
		route.ResponseHeadersToRemove = append(route.ResponseHeadersToRemove, remove.Name)
	}
}

func TrafficRouteHttpMatchStringMatcherToEnvoyStringMatcher(x *mesh_proto.StringMatch) *envoy_type_matcher_v3.StringMatcher {
	if x == nil {
		return &envoy_type_matcher_v3.StringMatcher{}
	}
	res := &envoy_type_matcher_v3.StringMatcher{}
	switch x.GetMatchType().(type) {
	case *mesh_proto.StringMatch_Prefix:
		res.MatchPattern = &envoy_type_matcher_v3.StringMatcher_Prefix{Prefix: x.GetPrefix()}
	case *mesh_proto.StringMatch_Regex:
		res.MatchPattern = &envoy_type_matcher_v3.StringMatcher_SafeRegex{&envoy_type_matcher_v3.RegexMatcher{
			EngineType: &envoy_type_matcher_v3.RegexMatcher_GoogleRe2{},
			Regex:      x.GetRegex(),
		}}
	case *mesh_proto.StringMatch_Exact:
		res.MatchPattern = &envoy_type_matcher_v3.StringMatcher_Exact{Exact: x.GetExact()}
	}
	return res
}
