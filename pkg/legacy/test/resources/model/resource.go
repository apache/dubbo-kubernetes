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
	"time"
)

import (
	core_model "github.com/apache/dubbo-kubernetes/pkg/core/resources/model"
)

var (
	_ core_model.Resource     = &Resource{}
	_ core_model.ResourceMeta = &ResourceMeta{}
)

type Resource struct {
	Meta           core_model.ResourceMeta
	Spec           core_model.ResourceSpec
	TypeDescriptor core_model.ResourceTypeDescriptor
}

func (r *Resource) SetMeta(meta core_model.ResourceMeta) {
	r.Meta = meta
}

func (r *Resource) SetSpec(spec core_model.ResourceSpec) error {
	r.Spec = spec
	return nil
}

func (r *Resource) GetMeta() core_model.ResourceMeta {
	return r.Meta
}

func (r *Resource) GetSpec() core_model.ResourceSpec {
	return r.Spec
}

func (r *Resource) Descriptor() core_model.ResourceTypeDescriptor {
	return r.TypeDescriptor
}

type ResourceMeta struct {
	Mesh             string
	Name             string
	NameExtensions   core_model.ResourceNameExtensions
	Version          string
	CreationTime     time.Time
	ModificationTime time.Time
	Labels           map[string]string
}

func (m *ResourceMeta) GetMesh() string {
	return m.Mesh
}

func (m *ResourceMeta) GetName() string {
	return m.Name
}

func (m *ResourceMeta) GetNameExtensions() core_model.ResourceNameExtensions {
	return m.NameExtensions
}

func (m *ResourceMeta) GetVersion() string {
	return m.Version
}

func (m *ResourceMeta) GetCreationTime() time.Time {
	return m.CreationTime
}

func (m *ResourceMeta) GetModificationTime() time.Time {
	return m.ModificationTime
}

func (m *ResourceMeta) GetLabels() map[string]string {
	return m.Labels
}
