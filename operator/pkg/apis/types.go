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

package apis

import (
	"encoding/json"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +kubetype-gen
// +kubetype-gen:groupVersion=install.dubbo.io/v1alpha1
// +k8s:deepcopy-gen=true
type DubboOperator struct {
	metav1.TypeMeta `json:",inline"`
	// +optional
	metav1.ObjectMeta `json:"metadata,omitempty"`
	// +optional
	Spec DubboOperatorSpec `json:"spec,omitempty"`
}

type DubboOperatorSpec struct {
	Profile    string              `json:"profile,omitempty"`
	Dashboard  *DubboDashboardSpec `json:"dashboard,omitempty"`
	Components *DubboComponentSpec `json:"components,omitempty"`
	Values     json.RawMessage     `json:"values,omitempty"`
}

type DubboComponentSpec struct {
	Base     *BaseComponentSpec `json:"base,omitempty"`
	Register *RegisterSpec      `json:"register,omitempty"`
}

type DubboDashboardSpec struct {
	Admin *DashboardComponentSpec `json:"admin,omitempty"`
}

type RegisterSpec struct {
	Nacos     *RegisterComponentSpec `json:"nacos,omitempty"`
	Zookeeper *RegisterComponentSpec `json:"zookeeper,omitempty"`
}

type BaseComponentSpec struct {
	Enabled *BoolValue `json:"enabled,omitempty"`
}

type DashboardComponentSpec struct {
	Enabled *BoolValue `json:"enabled,omitempty"`
}

type RegisterComponentSpec struct {
	Enabled   *BoolValue     `json:"enabled,omitempty"`
	Namespace string         `json:"namespace,omitempty"`
	Raw       map[string]any `json:"-"`
}

type MetadataCompSpec struct {
	RegisterComponentSpec
}

type BoolValue struct {
	bool
}

func (b *BoolValue) MarshalJSON() ([]byte, error) {
	return json.Marshal(b.GetValueOrFalse())
}

func (b *BoolValue) UnmarshalJSON(bytes []byte) error {
	bb := false
	if err := json.Unmarshal(bytes, &bb); err != nil {
		return err
	}
	*b = BoolValue{bb}
	return nil
}

func (b *BoolValue) GetValueOrFalse() bool {
	if b == nil {
		return false
	}
	return b.bool
}

func (b *BoolValue) GetValueOrTrue() bool {
	if b == nil {
		return true
	}
	return b.bool
}

var (
	_ json.Unmarshaler = &BoolValue{}
	_ json.Marshaler   = &BoolValue{}
)
