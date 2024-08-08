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

import envoy_type_matcher_v3 "github.com/envoyproxy/go-control-plane/envoy/type/matcher/v3"

const (
	EnvoyStringMatchTypePrefix = "prefix"
	EnvoyStringMatchTypeRegex  = "regex"
	EnvoyStringMatchTypeExact  = "exact"
)

type TrafficRouteHttpMatchStringMatcher interface {
	GetValue() string
	GetType() string
}

func TrafficRouteHttpMatchStringMatcherToEnvoyStringMatcher(x TrafficRouteHttpMatchStringMatcher) *envoy_type_matcher_v3.StringMatcher {
	if x == nil {
		return &envoy_type_matcher_v3.StringMatcher{}
	}
	res := &envoy_type_matcher_v3.StringMatcher{}
	switch x.GetType() {
	case EnvoyStringMatchTypePrefix:
		res.MatchPattern = &envoy_type_matcher_v3.StringMatcher_Prefix{Prefix: x.GetValue()}
	case EnvoyStringMatchTypeRegex:
		res.MatchPattern = &envoy_type_matcher_v3.StringMatcher_SafeRegex{&envoy_type_matcher_v3.RegexMatcher{
			EngineType: &envoy_type_matcher_v3.RegexMatcher_GoogleRe2{},
			Regex:      x.GetValue(),
		}}
	case EnvoyStringMatchTypeExact:
		res.MatchPattern = &envoy_type_matcher_v3.StringMatcher_Exact{Exact: x.GetValue()}
	}
	return res
}

type TrafficRouteHttpMatchStringMatcherPrefix struct {
	Value string
}

type TrafficRouteHttpMatchStringMatcherExact struct {
	Value string
}

type TrafficRouteHttpMatchStringMatcherRegex struct {
	Value string
}

func NewTrafficRouteHttpMatchStringMatcherPrefix(val string) TrafficRouteHttpMatchStringMatcherPrefix {
	return TrafficRouteHttpMatchStringMatcherPrefix{Value: val}
}

func (x TrafficRouteHttpMatchStringMatcherPrefix) ToTrafficRouteHttpMatchStringMatcher() TrafficRouteHttpMatchStringMatcher {
	return x
}

func (x TrafficRouteHttpMatchStringMatcherPrefix) GetType() string {
	return EnvoyStringMatchTypePrefix
}

func (x TrafficRouteHttpMatchStringMatcherPrefix) GetValue() string {
	return x.Value
}

func NewTrafficRouteHttpMatchStringMatcherRegex(val string) TrafficRouteHttpMatchStringMatcherRegex {
	return TrafficRouteHttpMatchStringMatcherRegex{Value: val}
}

func (x TrafficRouteHttpMatchStringMatcherRegex) ToTrafficRouteHttpMatchStringMatcher() TrafficRouteHttpMatchStringMatcher {
	return x
}

func (x TrafficRouteHttpMatchStringMatcherRegex) GetType() string {
	return EnvoyStringMatchTypeRegex
}

func (x TrafficRouteHttpMatchStringMatcherRegex) GetValue() string {
	return x.Value
}

func NewTrafficRouteHttpMatchStringMatcherExact(val string) TrafficRouteHttpMatchStringMatcherExact {
	return TrafficRouteHttpMatchStringMatcherExact{Value: val}
}

func (x TrafficRouteHttpMatchStringMatcherExact) ToTrafficRouteHttpMatchStringMatcher() TrafficRouteHttpMatchStringMatcher {
	return x
}

func (x TrafficRouteHttpMatchStringMatcherExact) GetType() string {
	return EnvoyStringMatchTypeExact
}

func (x TrafficRouteHttpMatchStringMatcherExact) GetValue() string {
	return x.Value
}

type TrafficRouteHttpMatch struct {
	Name    string
	Method  TrafficRouteHttpMatchStringMatcher
	Path    TrafficRouteHttpMatchStringMatcher
	Headers map[string]TrafficRouteHttpMatchStringMatcher
	Params  map[string]TrafficRouteHttpMatchStringMatcher
}

func (m TrafficRouteHttpMatch) GetPath() TrafficRouteHttpMatchStringMatcher {
	return m.Path
}

func (m TrafficRouteHttpMatch) GetParam() map[string]TrafficRouteHttpMatchStringMatcher {
	return m.Params
}

func (m TrafficRouteHttpMatch) GetHeaders() map[string]TrafficRouteHttpMatchStringMatcher {
	return m.Headers
}

func (m TrafficRouteHttpMatch) GetMethod() TrafficRouteHttpMatchStringMatcher {
	return m.Method
}

type Route struct {
	Match *TrafficRouteHttpMatch
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
