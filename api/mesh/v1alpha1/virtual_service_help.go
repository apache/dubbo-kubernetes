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

package v1alpha1

import (
	"regexp"
	"strings"
)

import (
	"github.com/golang/protobuf/proto"

	"golang.org/x/exp/slices"

	"google.golang.org/protobuf/types/known/durationpb"
)

import (
	"github.com/apache/dubbo-kubernetes/pkg/util/sets"
)

type TrafficRoute_Http_Match struct {
	Authority *StringMatch
	Method    *StringMatch
	Path      *StringMatch
	Headers   map[string]*StringMatch
	Params    map[string]*StringMatch
}

func (m TrafficRoute_Http_Match) GetPath() *StringMatch {
	return m.Path
}

func (m TrafficRoute_Http_Match) GetHeaders() map[string]*StringMatch {
	return m.Headers
}

func (m TrafficRoute_Http_Match) GetMethod() *StringMatch {
	return m.Method
}

func (m TrafficRoute_Http_Match) GetAuthority() *StringMatch {
	return m.Authority
}

func (m TrafficRoute_Http_Match) GetParam() map[string]*StringMatch {
	return m.Params
}

type TrafficRoute_Http_Modify struct {
	TimeOut         *durationpb.Duration
	Retries         *HTTPRetry
	Path            *TrafficRoute_Http_Modify_Path
	Host            *TrafficRoute_Http_Modify_Host
	RequestHeaders  *TrafficRoute_Http_Modify_Headers
	ResponseHeaders *TrafficRoute_Http_Modify_Headers
}

type isTrafficRoute_Http_Modify_Path_Type interface {
	isTrafficRoute_Http_Modify_Path_Type()
}

func (x *TrafficRoute_Http_Modify) Reset() {
	*x = TrafficRoute_Http_Modify{}
}

func (*TrafficRoute_Http_Modify) ProtoMessage() {}

func (x *TrafficRoute_Http_Modify) GetPath() *TrafficRoute_Http_Modify_Path {
	if x != nil {
		return x.Path
	}
	return nil
}

func (x *TrafficRoute_Http_Modify) GetHost() *TrafficRoute_Http_Modify_Host {
	if x != nil {
		return x.Host
	}
	return nil
}

func (x *TrafficRoute_Http_Modify) GetRequestHeaders() *TrafficRoute_Http_Modify_Headers {
	if x != nil {
		return x.RequestHeaders
	}
	return nil
}

func (x *TrafficRoute_Http_Modify) GetResponseHeaders() *TrafficRoute_Http_Modify_Headers {
	if x != nil {
		return x.ResponseHeaders
	}
	return nil
}

type TrafficRoute_Http_Modify_Path struct {
	Type isTrafficRoute_Http_Modify_Path_Type
}

func (x *TrafficRoute_Http_Modify_Path) Reset() {
	*x = TrafficRoute_Http_Modify_Path{}
}

func (m *TrafficRoute_Http_Modify_Path) GetType() isTrafficRoute_Http_Modify_Path_Type {
	if m != nil {
		return m.Type
	}
	return nil
}

func (x *TrafficRoute_Http_Modify_Path) GetRewritePrefix() string {
	if x, ok := x.GetType().(*TrafficRoute_Http_Modify_Path_RewritePrefix); ok {
		return x.RewritePrefix
	}
	return ""
}

func (x *TrafficRoute_Http_Modify_Path) GetRegex() *TrafficRoute_Http_Modify_RegexReplace {
	if x, ok := x.GetType().(*TrafficRoute_Http_Modify_Path_Regex); ok {
		return x.Regex
	}
	return nil
}

type TrafficRoute_Http_Modify_Host struct {
	Type isTrafficRoute_Http_Modify_Host_Type
}

type isTrafficRoute_Http_Modify_Host_Type interface {
	isTrafficRoute_Http_Modify_Host_Type()
}

func (x *TrafficRoute_Http_Modify_Host) Reset() {
	*x = TrafficRoute_Http_Modify_Host{}
}

func (m *TrafficRoute_Http_Modify_Host) GetType() isTrafficRoute_Http_Modify_Host_Type {
	if m != nil {
		return m.Type
	}
	return nil
}

type TrafficRoute_Http_Modify_Host_Value struct {
	Value string
}

func (*TrafficRoute_Http_Modify_Host_Value) isTrafficRoute_Http_Modify_Host_Type() {}

func (x *TrafficRoute_Http_Modify_Host) GetValue() string {
	if x, ok := x.GetType().(*TrafficRoute_Http_Modify_Host_Value); ok {
		return x.Value
	}
	return ""
}

func (x *TrafficRoute_Http_Modify_Host) GetFromPath() *TrafficRoute_Http_Modify_RegexReplace {
	if x, ok := x.GetType().(*TrafficRoute_Http_Modify_Host_FromPath); ok {
		return x.FromPath
	}
	return nil
}

type TrafficRoute_Http_Modify_Host_FromPath struct {
	FromPath *TrafficRoute_Http_Modify_RegexReplace
}

func (*TrafficRoute_Http_Modify_Host_FromPath) isTrafficRoute_Http_Modify_Host_Type() {}

type TrafficRoute_Http_Modify_Headers struct {
	Add    []*TrafficRoute_Http_Modify_Headers_Add
	Remove []*TrafficRoute_Http_Modify_Headers_Remove
}

func (x *TrafficRoute_Http_Modify_Headers) Reset() {
	*x = TrafficRoute_Http_Modify_Headers{}
}

func (x *TrafficRoute_Http_Modify_Headers) GetAdd() []*TrafficRoute_Http_Modify_Headers_Add {
	if x != nil {
		return x.Add
	}
	return nil
}

func (x *TrafficRoute_Http_Modify_Headers) GetRemove() []*TrafficRoute_Http_Modify_Headers_Remove {
	if x != nil {
		return x.Remove
	}
	return nil
}

type TrafficRoute_Http_Modify_Headers_Add struct {
	Name   string
	Value  string
	Append bool
}

func (x *TrafficRoute_Http_Modify_Headers_Add) Reset() {
	*x = TrafficRoute_Http_Modify_Headers_Add{}
}

func (x *TrafficRoute_Http_Modify_Headers_Add) GetName() string {
	if x != nil {
		return x.Name
	}
	return ""
}

func (x *TrafficRoute_Http_Modify_Headers_Add) GetValue() string {
	if x != nil {
		return x.Value
	}
	return ""
}

func (x *TrafficRoute_Http_Modify_Headers_Add) GetAppend() bool {
	if x != nil {
		return x.Append
	}
	return false
}

type TrafficRoute_Http_Modify_Headers_Remove struct {
	Name string
}

func (x *TrafficRoute_Http_Modify_Headers_Remove) Reset() {
	*x = TrafficRoute_Http_Modify_Headers_Remove{}
}

func (x *TrafficRoute_Http_Modify_Headers_Remove) GetName() string {
	if x != nil {
		return x.Name
	}
	return ""
}

type TrafficRoute_Http_Modify_Path_RewritePrefix struct {
	RewritePrefix string
}

func (*TrafficRoute_Http_Modify_Path_RewritePrefix) isTrafficRoute_Http_Modify_Path_Type() {}

type TrafficRoute_Http_Modify_RegexReplace struct {
	Pattern      string
	Substitution string
}

func (x *TrafficRoute_Http_Modify_RegexReplace) Reset() {
	*x = TrafficRoute_Http_Modify_RegexReplace{}
}

func (x *TrafficRoute_Http_Modify_RegexReplace) GetPattern() string {
	if x != nil {
		return x.Pattern
	}
	return ""
}

func (x *TrafficRoute_Http_Modify_RegexReplace) GetSubstitution() string {
	if x != nil {
		return x.Substitution
	}
	return ""
}

type TrafficRoute_Http_Modify_Path_Regex struct {
	Regex *TrafficRoute_Http_Modify_RegexReplace
}

func (*TrafficRoute_Http_Modify_Path_Rewrite) isTrafficRoute_Http_Modify_Path_Type() {}

func (x *TrafficRoute_Http_Modify_Path_Rewrite) Reset() {
	*x = TrafficRoute_Http_Modify_Path_Rewrite{}
}

func (x *TrafficRoute_Http_Modify_Path_Rewrite) GetRewrite() string {
	if x != nil {
		return x.Rewrite
	}
	return ""
}

type TrafficRoute_Http_Modify_Path_Rewrite struct {
	Rewrite string
}

func (*TrafficRoute_Http_Modify_Path_Regex) isTrafficRoute_Http_Modify_Path_Type() {}

func (x *StringMatch) Match(target string) bool {
	if x == nil {
		return true
	}
	switch x.MatchType.(type) {
	case *StringMatch_Exact:
		return x.GetExact() == target
	case *StringMatch_Prefix:
		return strings.HasPrefix(target, x.GetPrefix())
	case *StringMatch_Regex:
		i, _ := regexp.Match(x.GetRegex(), []byte(target))
		return i
	}
	return true
}

func (x *VirtualService) httpConfigDeduplicateAndMerge() {
	uniqueMap := make(map[string]*HTTPRoute)
	uniqueMatchMap := make(map[string]sets.String)
	res := make([]*HTTPRoute, 0, len(x.Http))
	getHash := func(msg proto.Message) string {
		bt, _ := proto.Marshal(msg)
		return string(bt)
	}
	for _, route := range x.Http {
		m := route.Match
		route.Match = nil
		h := getHash(route)
		if h == "" {
			continue
		}
		if uniqueMap[h] == nil {
			uniqueMap[h] = route
		}
		if uniqueMatchMap[h] == nil {
			uniqueMatchMap[h] = sets.String{}
		}
		for _, m := range m {
			if mh := getHash(m); mh != "" && !uniqueMatchMap[h].Contains(mh) {
				uniqueMatchMap[h].Insert(mh)
				uniqueMap[h].Match = append(uniqueMap[h].Match, m)
			}
		}
	}
	x.Http = res
}

// Split the match configuration into separate
func (x *VirtualService) preDealHttpConfig() {
	res := make([]*HTTPRoute, 0, len(x.Http))
	for i := len(x.Http) - 1; i >= 0; i-- {
		msg := x.Http[i]
		m := msg.Match
		msg.Match = nil
		for _, match := range m {
			n := proto.Clone(msg).(*HTTPRoute)
			n.Match = []*HTTPMatchRequest{match}
			res = append(res, n)
		}
	}
	x.Http = res
}

// RangeHTTPReferenceUpdate it ensure one piece httpRoute only has one match field in range, and merge match field after range done
func (x *VirtualService) RangeHTTPReferenceUpdate(
	ref_range func(index int, ref *HTTPRoute) (insert *HTTPRoute, remove bool, stop bool),
) {
	if ref_range == nil {
		return
	}
	var (
		insert             *HTTPRoute
		remove             bool
		stop               bool
		newhttpReverselist = make([]*HTTPRoute, 0, len(x.Http)*2)
	)
	x.preDealHttpConfig()
	// reverse range for ensure config priority in appending
	for i := len(x.Http) - 1; i >= 0; i-- {
		if !stop {
			insert, remove, stop = ref_range(i, x.Http[i])
			if !remove {
				newhttpReverselist = append(newhttpReverselist, x.Http[i])
			}
			if insert != nil {
				newhttpReverselist = append(newhttpReverselist, insert)
			}
		} else {
			newhttpReverselist = append(newhttpReverselist, x.Http[i])
		}
	}
	slices.Reverse(newhttpReverselist)
	x.Http = newhttpReverselist
	x.httpConfigDeduplicateAndMerge()
}

func (x *HTTPMatchRequest) WithoutURIEmpty() bool {
	if x.Scheme != nil {
		return false
	} else if x.Method != nil {
		return false
	} else if x.Authority != nil {
		return false
	} else if x.Headers != nil {
		return false
	} else if x.Port != 0 {
		return false
	} else if x.SourceLabels != nil {
		return false
	} else if x.Gateways != nil {
		return false
	} else if x.QueryParams != nil {
		return false
	} else if x.IgnoreUriCase != false {
		return false
	} else if x.WithoutHeaders != nil {
		return false
	} else if x.SourceNamespace != "" {
		return false
	} else if x.StatPrefix != "" {
		return false
	}
	return true
}
