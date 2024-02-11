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

package dataplane

import (
	"context"
)

import (
	"github.com/pkg/errors"
)

import (
	mesh_proto "github.com/apache/dubbo-kubernetes/api/mesh/v1alpha1"
	"github.com/apache/dubbo-kubernetes/pkg/core"
	core_mesh "github.com/apache/dubbo-kubernetes/pkg/core/resources/apis/mesh"
	core_manager "github.com/apache/dubbo-kubernetes/pkg/core/resources/manager"
	core_model "github.com/apache/dubbo-kubernetes/pkg/core/resources/model"
	core_store "github.com/apache/dubbo-kubernetes/pkg/core/resources/store"
)

func NewDataplaneManager(store core_store.ResourceStore, zone string, validator Validator) core_manager.ResourceManager {
	return &dataplaneManager{
		ResourceManager: core_manager.NewResourceManager(store),
		store:           store,
		zone:            zone,
		validator:       validator,
	}
}

type dataplaneManager struct {
	core_manager.ResourceManager
	store     core_store.ResourceStore
	zone      string
	validator Validator
}

func (m *dataplaneManager) Create(ctx context.Context, resource core_model.Resource, fs ...core_store.CreateOptionsFunc) error {
	if err := core_model.Validate(resource); err != nil {
		return err
	}
	dp, err := m.dataplane(resource)
	if err != nil {
		return err
	}

	m.setInboundsClusterTag(dp)

	opts := core_store.NewCreateOptions(fs...)
	owner := core_mesh.NewMeshResource()
	if err := m.store.Get(ctx, owner, core_store.GetByKey(opts.Mesh, core_model.NoMesh)); err != nil {
		return core_manager.MeshNotFound(opts.Mesh)
	}

	key := core_model.ResourceKey{
		Mesh: opts.Mesh,
		Name: opts.Name,
	}
	if err := m.validator.ValidateCreate(ctx, key, dp, owner); err != nil {
		return err
	}

	return m.store.Create(ctx, resource, append(fs, core_store.CreatedAt(core.Now()))...)
}

func (m *dataplaneManager) Update(ctx context.Context, resource core_model.Resource, fs ...core_store.UpdateOptionsFunc) error {
	dp, err := m.dataplane(resource)
	if err != nil {
		return err
	}

	m.setInboundsClusterTag(dp)

	owner := core_mesh.NewMeshResource()
	if err := m.store.Get(ctx, owner, core_store.GetByKey(resource.GetMeta().GetMesh(), core_model.NoMesh)); err != nil {
		return core_manager.MeshNotFound(resource.GetMeta().GetMesh())
	}
	if err := m.validator.ValidateUpdate(ctx, dp, owner); err != nil {
		return err
	}

	return m.ResourceManager.Update(ctx, resource, fs...)
}

func (m *dataplaneManager) dataplane(resource core_model.Resource) (*core_mesh.DataplaneResource, error) {
	dp, ok := resource.(*core_mesh.DataplaneResource)
	if !ok {
		return nil, errors.Errorf("invalid resource type: expected=%T, got=%T", (*core_mesh.DataplaneResource)(nil), resource)
	}
	return dp, nil
}

func (m *dataplaneManager) setInboundsClusterTag(dp *core_mesh.DataplaneResource) {
	if m.zone == "" || dp.Spec.Networking == nil {
		return
	}

	for _, inbound := range dp.Spec.Networking.Inbound {
		if inbound.Tags == nil {
			inbound.Tags = make(map[string]string)
		}
		inbound.Tags[mesh_proto.ZoneTag] = m.zone
	}
}
