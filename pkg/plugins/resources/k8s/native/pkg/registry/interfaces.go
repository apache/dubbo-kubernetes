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
	core_model "github.com/apache/dubbo-kubernetes/pkg/core/resources/model"
	"github.com/apache/dubbo-kubernetes/pkg/plugins/resources/k8s/native/pkg/model"
)

type ResourceType = core_model.ResourceSpec

type TypeRegistry interface {
	RegisterObjectType(ResourceType, model.KubernetesObject) error
	RegisterObjectTypeIfAbsent(ResourceType, model.KubernetesObject)
	RegisterListType(ResourceType, model.KubernetesList) error
	RegisterListTypeIfAbsent(ResourceType, model.KubernetesList)

	NewObject(ResourceType) (model.KubernetesObject, error)
	NewList(ResourceType) (model.KubernetesList, error)
}
