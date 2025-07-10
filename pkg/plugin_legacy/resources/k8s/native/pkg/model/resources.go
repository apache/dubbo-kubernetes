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

package model

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/apache/dubbo-kubernetes/pkg/core/resources/model"
)

type Scope string

const (
	ScopeNamespace Scope = "namespace"
	ScopeCluster   Scope = "cluster"
)

type KubernetesObject interface {
	client.Object

	GetObjectMeta() *metav1.ObjectMeta
	SetObjectMeta(*metav1.ObjectMeta)
	GetMesh() string
	SetMesh(string)
	GetSpec() (model.ResourceSpec, error)
	SetSpec(model.ResourceSpec)
	Scope() Scope
}

type KubernetesList interface {
	client.ObjectList

	GetItems() []KubernetesObject
	GetContinue() string
}

// RawMessage is a carrier for an untyped JSON payload.
type RawMessage map[string]interface{}

// DeepCopy ...
func (in RawMessage) DeepCopy() RawMessage {
	return runtime.DeepCopyJSON(in)
}
