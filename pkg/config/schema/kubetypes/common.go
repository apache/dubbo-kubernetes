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

package kubetypes

import (
	"github.com/apache/dubbo-kubernetes/pkg/config"
	"github.com/apache/dubbo-kubernetes/pkg/config/schema/gvk"
	"github.com/apache/dubbo-kubernetes/pkg/ptr"
	"github.com/apache/dubbo-kubernetes/pkg/typemap"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func MustGVKFromType[T runtime.Object]() (cfg config.GroupVersionKind) {
	if gvk, ok := getGvk(ptr.Empty[T]()); ok {
		return gvk
	}
	if rp := typemap.Get[RegisterType[T]](registeredTypes); rp != nil {
		return (*rp).GetGVK()
	}
	panic("unknown kind: " + cfg.String())
}

func MustToGVR[T runtime.Object](cfg config.GroupVersionKind) schema.GroupVersionResource {
	if r, ok := gvk.ToGVR(cfg); ok {
		return r
	}
	if rp := typemap.Get[RegisterType[T]](registeredTypes); rp != nil {
		return (*rp).GetGVR()
	}
	panic("unknown kind: " + cfg.String())
}

var registeredTypes = typemap.NewTypeMap()

type RegisterType[T runtime.Object] interface {
	GetGVK() config.GroupVersionKind
	GetGVR() schema.GroupVersionResource
	Object() T
}
