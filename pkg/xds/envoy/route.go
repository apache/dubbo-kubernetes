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

package envoy

import (
	envoy_type_matcher_v3 "github.com/envoyproxy/go-control-plane/envoy/type/matcher/v3"
)

import (
	"github.com/apache/dubbo-kubernetes/pkg/core/xds"
)

func TrafficRouteHttpMatchStringMatcherToEnvoyStringMatcher(x xds.TrafficRouteHttpStringMatcher) *envoy_type_matcher_v3.StringMatcher {
	if x == nil {
		return &envoy_type_matcher_v3.StringMatcher{}
	}
	res := &envoy_type_matcher_v3.StringMatcher{}
	switch x.GetType() {
	case xds.EnvoyStringMatchTypePrefix:
		res.MatchPattern = &envoy_type_matcher_v3.StringMatcher_Prefix{Prefix: x.GetValue()}
	case xds.EnvoyStringMatchTypeRegex:
		res.MatchPattern = &envoy_type_matcher_v3.StringMatcher_SafeRegex{&envoy_type_matcher_v3.RegexMatcher{
			EngineType: &envoy_type_matcher_v3.RegexMatcher_GoogleRe2{},
			Regex:      x.GetValue(),
		}}
	case xds.EnvoyStringMatchTypeExact:
		res.MatchPattern = &envoy_type_matcher_v3.StringMatcher_Exact{Exact: x.GetValue()}
	}
	return res
}

type Route struct {
	Match *xds.TrafficRouteHttpMatch
	// modify
	// rateLimit
	Clusters []Cluster
}

func NewRouteFromCluster(cluster Cluster) Route {
	return Route{
		Match:    nil,
		Clusters: []Cluster{cluster},
	}
}

type Routes []Route

func (r Routes) Clusters() []Cluster {
	var clusters []Cluster
	for _, route := range r {
		clusters = append(clusters, route.Clusters...)
	}
	return clusters
}

type NewRouteOpt interface {
	apply(route *Route)
}

type newRouteOptFunc func(route *Route)

func (f newRouteOptFunc) apply(route *Route) {
	f(route)
}

func NewRoute(opts ...NewRouteOpt) Route {
	r := Route{}
	for _, opt := range opts {
		opt.apply(&r)
	}
	return r
}

func WithCluster(cluster Cluster) NewRouteOpt {
	return newRouteOptFunc(func(route *Route) {
		route.Clusters = append(route.Clusters, cluster)
	})
}
