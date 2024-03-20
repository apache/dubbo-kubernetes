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

package rest

import (
	"fmt"
)

import (
	"github.com/pkg/errors"
)

import (
	"github.com/apache/dubbo-kubernetes/pkg/core/resources/model"
)

type Api interface {
	GetResourceApi(model.ResourceType) (ResourceApi, error)
}

type ResourceApi interface {
	List(mesh string) string
	Item(mesh string, name string) string
}

func NewResourceApi(scope model.ResourceScope, path string) ResourceApi {
	switch scope {
	case model.ScopeGlobal:
		return &nonMeshedApi{CollectionPath: path}
	case model.ScopeMesh:
		return &meshedApi{CollectionPath: path}
	default:
		panic("Unsupported scope type")
	}
}

type meshedApi struct {
	CollectionPath string
}

func (r *meshedApi) List(mesh string) string {
	return fmt.Sprintf("/meshes/%s/%s", mesh, r.CollectionPath)
}

func (r meshedApi) Item(mesh string, name string) string {
	return fmt.Sprintf("/meshes/%s/%s/%s", mesh, r.CollectionPath, name)
}

type nonMeshedApi struct {
	CollectionPath string
}

func (r *nonMeshedApi) List(string) string {
	return fmt.Sprintf("/%s", r.CollectionPath)
}

func (r *nonMeshedApi) Item(string, name string) string {
	return fmt.Sprintf("/%s/%s", r.CollectionPath, name)
}

var _ Api = &ApiDescriptor{}

type ApiDescriptor struct {
	Resources map[model.ResourceType]ResourceApi
}

func (m *ApiDescriptor) GetResourceApi(typ model.ResourceType) (ResourceApi, error) {
	mapping, ok := m.Resources[typ]
	if !ok {
		return nil, errors.Errorf("unknown resource type: %q", typ)
	}
	return mapping, nil
}
