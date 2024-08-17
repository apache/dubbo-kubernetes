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

package xds

import envoy_tags "github.com/apache/dubbo-kubernetes/pkg/xds/envoy/tags"

const (
	EnvoyStringMatchTypePrefix = "prefix"
	EnvoyStringMatchTypeRegex  = "regex"
	EnvoyStringMatchTypeExact  = "exact"
)

type TrafficRouteHttpStringMatcher interface {
	GetValue() string
	GetType() string
}

type TrafficRouteHttpMatcherPrefix struct {
	Value string
}

type TrafficRouteHttpMatcherExact struct {
	Value string
}

type TrafficRouteHttpMatcherRegex struct {
	Value string
}

func NewTrafficRouteHttpMatchStringMatcherPrefix(val string) TrafficRouteHttpMatcherPrefix {
	return TrafficRouteHttpMatcherPrefix{Value: val}
}

func (x TrafficRouteHttpMatcherPrefix) ToTrafficRouteHttpMatchStringMatcher() TrafficRouteHttpStringMatcher {
	return x
}

func (x TrafficRouteHttpMatcherPrefix) GetType() string {
	return EnvoyStringMatchTypePrefix
}

func (x TrafficRouteHttpMatcherPrefix) GetValue() string {
	return x.Value
}

func NewTrafficRouteHttpMatchStringMatcherRegex(val string) TrafficRouteHttpMatcherRegex {
	return TrafficRouteHttpMatcherRegex{Value: val}
}

func (x TrafficRouteHttpMatcherRegex) ToTrafficRouteHttpMatchStringMatcher() TrafficRouteHttpStringMatcher {
	return x
}

func (x TrafficRouteHttpMatcherRegex) GetType() string {
	return EnvoyStringMatchTypeRegex
}

func (x TrafficRouteHttpMatcherRegex) GetValue() string {
	return x.Value
}

func NewTrafficRouteHttpMatcherExact(val string) TrafficRouteHttpMatcherExact {
	return TrafficRouteHttpMatcherExact{Value: val}
}

func (x TrafficRouteHttpMatcherExact) ToTrafficRouteHttpMatchStringMatcher() TrafficRouteHttpStringMatcher {
	return x
}

func (x TrafficRouteHttpMatcherExact) GetType() string {
	return EnvoyStringMatchTypeExact
}

func (x TrafficRouteHttpMatcherExact) GetValue() string {
	return x.Value
}

type TrafficRouteHttpMatch struct {
	Method  TrafficRouteHttpStringMatcher
	Path    TrafficRouteHttpStringMatcher
	Headers map[string]TrafficRouteHttpStringMatcher
	Params  map[string]TrafficRouteHttpStringMatcher
}

func (m TrafficRouteHttpMatch) GetPath() TrafficRouteHttpStringMatcher {
	return m.Path
}

func (m TrafficRouteHttpMatch) GetParam() map[string]TrafficRouteHttpStringMatcher {
	return m.Params
}

func (m TrafficRouteHttpMatch) GetHeaders() map[string]TrafficRouteHttpStringMatcher {
	return m.Headers
}

func (m TrafficRouteHttpMatch) GetMethod() TrafficRouteHttpStringMatcher {
	return m.Method
}

func (m *TrafficRouteHttpMatch) ShouldMerge(other *TrafficRouteHttpMatch) bool {
	if (m.Method == nil) != (other.Method == nil) || (m.Path == nil) != (other.Path == nil) {
		return false
	}

	if m.Method != nil && (m.Method.GetType() != other.Method.GetType() || m.Method.GetValue() != other.Method.GetValue()) {
		return false
	}

	if m.Path != nil && (m.Path.GetType() != other.Path.GetType() || m.Path.GetValue() != other.Path.GetValue()) {
		return false
	}

	return true
}

func (m *TrafficRouteHttpMatch) Merge(other *TrafficRouteHttpMatch) TrafficRouteHttpMatch {
	res := TrafficRouteHttpMatch{
		Method:  m.Method,
		Path:    m.Path,
		Headers: make(map[string]TrafficRouteHttpStringMatcher, len(m.Headers)+len(other.Headers)),
		Params:  make(map[string]TrafficRouteHttpStringMatcher, len(m.Params)+len(other.Params)),
	}

	for k, matcher := range m.Headers {
		res.Headers[k] = matcher
	}
	for k, matcher := range other.Headers {
		res.Headers[k] = matcher
	}

	for k, matcher := range m.Params {
		res.Params[k] = matcher
	}
	for k, matcher := range other.Params {
		res.Params[k] = matcher
	}

	return res
}

type TrafficRouteConfig struct {
	ExternalTags envoy_tags.Tags
	Weight       int64
}

func MergeClusterSelectorList(x, y []ClusterSelectorList) []ClusterSelectorList {
	if x == nil {
		return y
	}
	if y == nil {
		return x
	}
	res := make([]ClusterSelectorList, 0, len(x)*len(y))
	for _, xlist := range x {
		for _, ylist := range y {
			if xlist.GetMatchInfo().ShouldMerge(ylist.GetMatchInfo()) {
				res = append(res, xlist)
				res = append(res, ylist)
			} else {
				tc := ClusterSelectorList{
					MatchInfo:    xlist.GetMatchInfo().Merge(ylist.GetMatchInfo()),
					EndSelectors: make([]ClusterSelector, 0, len(xlist.EndSelectors)*len(ylist.EndSelectors)),
				}
				for _, xselector := range xlist.EndSelectors {
					for _, yselector := range ylist.EndSelectors {
						tc.EndSelectors = append(tc.EndSelectors, ClusterSelector{
							ConfigInfo: TrafficRouteConfig{
								ExternalTags: xselector.ConfigInfo.ExternalTags.Copy().
									WithTags(yselector.ConfigInfo.ExternalTags.KeyAndValues()...),
								Weight: func() int64 {
									xweight, yweight := xselector.ConfigInfo.Weight, yselector.ConfigInfo.Weight
									if xweight == 0 {
										xweight = 100
									}
									if yweight == 0 {
										yweight = 100
									}
									return xweight * yweight / 100
								}(),
							},
							SelectFunc: func(endpoint EndpointList) EndpointList {
								if xselector.SelectFunc != nil {
									endpoint = xselector.Select(endpoint)
								}
								if yselector.SelectFunc != nil {
									endpoint = yselector.Select(endpoint)
								}
								return endpoint
							},
						})
					}
				}
			}
		}
	}
	return res
}
