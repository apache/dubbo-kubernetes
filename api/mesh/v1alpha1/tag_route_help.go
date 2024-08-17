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
	"path/filepath"
	"regexp"
	"strings"
)

func (x *TagRoute) ListUnGenConfigs() []*Tag {
	res := make([]*Tag, 0, len(x.Tags)/2+1)
	for _, tag := range x.Tags {
		if !tag.XGenerateByCp {
			res = append(res, tag)
		}
	}
	return res
}

func (x *TagRoute) ListGenConfigs() []*Tag {
	res := make([]*Tag, 0, len(x.Tags)/2+1)
	for _, tag := range x.Tags {
		if tag.XGenerateByCp {
			res = append(res, tag)
		}
	}
	return res
}

func (x *TagRoute) RangeTags(f func(*Tag) (isStop bool)) {
	if f == nil {
		return
	}
	for _, tag := range x.Tags {
		if f(tag) {
			break
		}
	}
}

func (x *TagRoute) RangeTagsToRemove(f func(*Tag) (IsRemove bool)) {
	if f == nil {
		return
	}
	i := make([]*Tag, 0, len(x.Tags)/2+1)
	for _, tag := range x.Tags {
		if !f(tag) {
			i = append(i, tag)
		}
	}
	x.Tags = i
}

func (x *StringMatch) Match(target string) bool {
	if x.Exact != "" {
		return x.Exact == target
	}
	if x.Prefix != "" {
		return strings.HasPrefix(target, x.Prefix)
	}
	if x.Regex != "" {
		ok, _ := regexp.Match(x.Regex, []byte(target))
		return ok
	}
	if x.Noempty != "" {
		return target != ""
	}
	if x.Empty != "" {
		return target == ""
	}
	if x.Wildcard != "" {
		ok, _ := filepath.Match(x.Wildcard, target)
		return ok
	}
	return true
}
