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

package servicemapping

import (
	"time"
)

import (
	core_model "github.com/apache/dubbo-kubernetes/pkg/core/resources/model"
)

type resourceMetaObject struct {
	Name             string
	Version          string
	Mesh             string
	CreationTime     time.Time
	ModificationTime time.Time
	Labels           map[string]string
}

var _ core_model.ResourceMeta = &resourceMetaObject{}

func (r *resourceMetaObject) GetName() string {
	return r.Name
}

func (r *resourceMetaObject) GetNameExtensions() core_model.ResourceNameExtensions {
	return core_model.ResourceNameExtensionsUnsupported
}

func (r *resourceMetaObject) GetVersion() string {
	return r.Version
}

func (r *resourceMetaObject) GetMesh() string {
	return r.Mesh
}

func (r *resourceMetaObject) GetCreationTime() time.Time {
	return r.CreationTime
}

func (r *resourceMetaObject) GetModificationTime() time.Time {
	return r.ModificationTime
}

func (r *resourceMetaObject) GetLabels() map[string]string {
	return r.Labels
}
