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

package registry

import (
	"github.com/apache/dubbo-kubernetes/pkg/plugins/resources/k8s/native/pkg/model"
)

var global = NewTypeRegistry()

func Global() TypeRegistry {
	return global
}

func RegisterObjectType(typ ResourceType, obj model.KubernetesObject) {
	if err := global.RegisterObjectType(typ, obj); err != nil {
		panic(err)
	}
}

func RegisterObjectTypeIfAbsent(typ ResourceType, obj model.KubernetesObject) {
	global.RegisterObjectTypeIfAbsent(typ, obj)
}

func RegisterListType(typ ResourceType, obj model.KubernetesList) {
	if err := global.RegisterListType(typ, obj); err != nil {
		panic(err)
	}
}

func RegisterListTypeIfAbsent(typ ResourceType, obj model.KubernetesList) {
	global.RegisterListTypeIfAbsent(typ, obj)
}
