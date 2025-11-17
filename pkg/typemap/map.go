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

package typemap

import (
	"github.com/apache/dubbo-kubernetes/pkg/util/ptr"
	"reflect"
)

type TypeMap struct {
	inner map[reflect.Type]any
}

func NewTypeMap() TypeMap {
	return TypeMap{make(map[reflect.Type]any)}
}

func Set[T any](t TypeMap, v T) {
	interfaceType := reflect.TypeOf((*T)(nil)).Elem()
	t.inner[interfaceType] = v
}

func Get[T any](t TypeMap) *T {
	v, f := t.inner[reflect.TypeFor[T]()]
	if f {
		return ptr.Of(v.(T))
	}
	return nil
}
