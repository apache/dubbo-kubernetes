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

package tags

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
)

const TagsHeaderName = "x-dubbo-tags"

const splitClusterFmtString = "%s-%x"

var splitClusterRegex = regexp.MustCompile("(.*)-[[:xdigit:]]{16}$")

type Tags map[string]string

type (
	TagsSlice    []Tags
	TagKeys      []string
	TagKeysSlice []TagKeys
)

func (t Tags) String() string {
	var pairs []string
	for _, key := range t.Keys() {
		pairs = append(pairs, fmt.Sprintf("%s=%s", key, t[key]))
	}
	return strings.Join(pairs, ",")
}

func (t Tags) Keys() TagKeys {
	var keys []string
	for key := range t {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func ServiceFromClusterName(name string) string {
	matchedGroups := splitClusterRegex.FindStringSubmatch(name)
	if len(matchedGroups) == 0 {
		return name
	}
	return matchedGroups[1]
}

func (t Tags) WithTags(keysAndValues ...string) Tags {
	result := Tags{}
	for tagName, tagValue := range t {
		result[tagName] = tagValue
	}
	for i := 0; i < len(keysAndValues); {
		key, value := keysAndValues[i], keysAndValues[i+1]
		result[key] = value
		i += 2
	}
	return result
}
