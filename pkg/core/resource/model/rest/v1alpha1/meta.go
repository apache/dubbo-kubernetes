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

package v1alpha1

import (
	"time"

	"github.com/apache/dubbo-kubernetes/pkg/core/resource/model"
)

type ResourceMeta struct {
	Type             string            `json:"type"`
	Mesh             string            `json:"mesh,omitempty"`
	Name             string            `json:"name"`
	CreationTime     time.Time         `json:"creationTime"`
	ModificationTime time.Time         `json:"modificationTime"`
	Labels           map[string]string `json:"labels,omitempty"`
}

var _ model.ResourceMeta = ResourceMeta{}

func (r ResourceMeta) GetName() string {
	return r.Name
}

func (r ResourceMeta) GetNameExtensions() model.ResourceNameExtensions {
	return model.ResourceNameExtensionsUnsupported
}

func (r ResourceMeta) GetVersion() string {
	return ""
}

func (r ResourceMeta) GetMesh() string {
	return r.Mesh
}

func (r ResourceMeta) GetCreationTime() time.Time {
	return r.CreationTime
}

func (r ResourceMeta) GetModificationTime() time.Time {
	return r.ModificationTime
}

func (r ResourceMeta) GetLabels() map[string]string {
	return r.Labels
}
