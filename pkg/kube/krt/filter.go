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

package krt

import (
	"fmt"
	"github.com/apache/dubbo-kubernetes/pkg/config/labels"
	"github.com/apache/dubbo-kubernetes/pkg/util/smallset"
	"reflect"
)

type filter struct {
	keys            smallset.Set[string]
	index           *indexFilter
	selects         map[string]string
	selectsNonEmpty map[string]string
	labels          map[string]string
	generic         func(any) bool
}

type indexFilter struct {
	filterUID    collectionUID
	list         func() any
	indexMatches func(any) bool
	extractKeys  objectKeyExtractor
	key          string
}

type objectKeyExtractor = func(o any) []string

func FilterKey(k string) FetchOption {
	return func(h *dependency) {
		h.filter.keys = smallset.New(k)
	}
}

func getKeyExtractor(o any) []string {
	return []string{GetKey(o)}
}

func getLabelSelector(a any) map[string]string {
	ak, ok := a.(LabelSelectorer)
	if ok {
		return ak.GetLabelSelector()
	}
	val := reflect.ValueOf(a)

	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}

	specField := val.FieldByName("Spec")
	if !specField.IsValid() {
		panic(fmt.Sprintf("obj %T has no Spec", a))
	}

	labelsField := specField.FieldByName("Selector")
	if !labelsField.IsValid() {
		panic(fmt.Sprintf("obj %T has no Selector", a))
	}

	switch s := labelsField.Interface().(type) {
	case map[string]string:
		return s
	default:
		panic(fmt.Sprintf("obj %T has unknown Selector", s))
	}
}

func (f *filter) reverseIndexKey() ([]string, indexedDependencyType, objectKeyExtractor, collectionUID, bool) {
	if f.keys.Len() > 0 {
		if f.index != nil {
			panic("cannot filter by index and key")
		}
		return f.keys.List(), getKeyType, getKeyExtractor, 0, true
	}
	if f.index != nil {
		return []string{f.index.key}, indexType, f.index.extractKeys, f.index.filterUID, true
	}
	return nil, unknownIndexType, nil, 0, false
}

func (f *filter) Matches(object any, forList bool) bool {
	// Check each of our defined filters to see if the object matches
	// This function is called very often and is important to keep fast
	// Cheaper checks should come earlier to avoid additional work and short circuit early

	// If we are listing, we already did this. Do not redundantly check.
	if !forList {
		// First, lookup directly by key. This is cheap
		// an empty set will match none
		if !f.keys.IsNil() && !f.keys.Contains(GetKey[any](object)) {
			return false
		}
		// Index is also cheap, and often used to filter namespaces out. Make sure we do this early
		if f.index != nil {
			if !f.index.indexMatches(object) {
				return false
			}
		}
	}

	// Rest is expensive
	if f.selects != nil && !labels.Instance(getLabelSelector(object)).SubsetOf(f.selects) {
		return false
	}
	if f.selectsNonEmpty != nil && !labels.Instance(getLabelSelector(object)).Match(f.selectsNonEmpty) {
		return false
	}
	if f.labels != nil && !labels.Instance(f.labels).SubsetOf(getLabels(object)) {
		return false
	}
	if f.generic != nil && !f.generic(object) {
		return false
	}
	return true
}
