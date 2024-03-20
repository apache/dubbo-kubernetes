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

package mapping

import (
	"context"
)

import (
	"github.com/apache/dubbo-kubernetes/pkg/core"
	core_mesh "github.com/apache/dubbo-kubernetes/pkg/core/resources/apis/mesh"
	core_manager "github.com/apache/dubbo-kubernetes/pkg/core/resources/manager"
	core_model "github.com/apache/dubbo-kubernetes/pkg/core/resources/model"
	core_store "github.com/apache/dubbo-kubernetes/pkg/core/resources/store"
)

type mappingManager struct {
	core_manager.ResourceManager
	store core_store.ResourceStore
}

func NewMappingManager(store core_store.ResourceStore) core_manager.ResourceManager {
	return &mappingManager{
		ResourceManager: core_manager.NewResourceManager(store),
		store:           store,
	}
}

func (m *mappingManager) Create(ctx context.Context, r core_model.Resource, fs ...core_store.CreateOptionsFunc) error {
	if err := core_model.Validate(r); err != nil {
		return err
	}
	opts := core_store.NewCreateOptions(fs...)
	owner := core_mesh.NewMeshResource()
	if err := m.store.Get(ctx, owner, core_store.GetByKey(opts.Mesh, core_model.NoMesh)); err != nil {
		return core_manager.MeshNotFound(opts.Mesh)
	}

	return m.store.Create(ctx, r, append(fs, core_store.CreatedAt(core.Now()))...)
}

func (m *mappingManager) Update(ctx context.Context, r core_model.Resource, fs ...core_store.UpdateOptionsFunc) error {
	owner := core_mesh.NewMeshResource()
	if err := m.store.Get(ctx, owner, core_store.GetByKey(r.GetMeta().GetMesh(), core_model.NoMesh)); err != nil {
		return core_manager.MeshNotFound(r.GetMeta().GetMesh())
	}
	return m.ResourceManager.Update(ctx, r, fs...)
}
