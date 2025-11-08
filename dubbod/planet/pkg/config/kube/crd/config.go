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

package crd

import (
	"encoding/json"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

type DubboKind struct {
	metav1.TypeMeta
	metav1.ObjectMeta `json:"metadata"`
	Spec              json.RawMessage  `json:"spec"`
	Status            *json.RawMessage `json:"status,omitempty"`
}

func (in *DubboKind) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}

	return nil
}

func (in *DubboKind) DeepCopy() *DubboKind {
	if in == nil {
		return nil
	}
	out := new(DubboKind)
	in.DeepCopyInto(out)
	return out
}

func (in *DubboKind) DeepCopyInto(out *DubboKind) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	out.Spec = in.Spec
	out.Status = in.Status
}

func (in *DubboKind) GetObjectMeta() metav1.ObjectMeta {
	return in.ObjectMeta
}

func (in *DubboKind) GetSpec() json.RawMessage {
	return in.Spec
}

func (in *DubboKind) GetStatus() *json.RawMessage {
	return in.Status
}

type DubboObject interface {
	runtime.Object
	GetSpec() json.RawMessage
	GetStatus() *json.RawMessage
	GetObjectMeta() metav1.ObjectMeta
}
