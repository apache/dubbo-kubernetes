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

package mesh

import (
	"context"
	"time"
)

import (
	"github.com/pkg/errors"

	kube_ctrl "sigs.k8s.io/controller-runtime"
)

import (
	dubbo_cp "github.com/apache/dubbo-kubernetes/pkg/config/app/dubbo-cp"
	config_core "github.com/apache/dubbo-kubernetes/pkg/config/mode"
	config_store "github.com/apache/dubbo-kubernetes/pkg/config/mode/resources/store"
	core_mesh "github.com/apache/dubbo-kubernetes/pkg/core/resources/apis/mesh"
	core_manager "github.com/apache/dubbo-kubernetes/pkg/core/resources/manager"
	core_model "github.com/apache/dubbo-kubernetes/pkg/core/resources/model"
	core_registry "github.com/apache/dubbo-kubernetes/pkg/core/resources/registry"
	core_store "github.com/apache/dubbo-kubernetes/pkg/core/resources/store"
)

func NewMeshManager(
	store core_store.ResourceStore,
	otherManagers core_manager.ResourceManager,
	registry core_registry.TypeRegistry,
	validator MeshValidator,
	extensions context.Context,
	config dubbo_cp.Config,
	manager kube_ctrl.Manager,
	mode config_core.DeployMode,
) core_manager.ResourceManager {
	meshManager := &meshManager{
		store:         store,
		otherManagers: otherManagers,
		registry:      registry,
		meshValidator: validator,
		unsafeDelete:  config.Store.UnsafeDelete,
		extensions:    extensions,
		manager:       manager,
		deployMode:    mode,
	}
	if config.Store.Type == config_store.KubernetesStore {
		meshManager.k8sStore = true
		meshManager.systemNamespace = config.Store.Kubernetes.SystemNamespace
	}
	return meshManager
}

type meshManager struct {
	store           core_store.ResourceStore
	otherManagers   core_manager.ResourceManager
	registry        core_registry.TypeRegistry
	meshValidator   MeshValidator
	unsafeDelete    bool
	extensions      context.Context
	k8sStore        bool
	systemNamespace string
	manager         kube_ctrl.Manager
	deployMode      config_core.DeployMode
}

func (m *meshManager) Get(ctx context.Context, resource core_model.Resource, fs ...core_store.GetOptionsFunc) error {
	mesh, err := m.mesh(resource)
	if err != nil {
		return err
	}
	return m.store.Get(ctx, mesh, fs...)
}

func (m *meshManager) List(ctx context.Context, list core_model.ResourceList, fs ...core_store.ListOptionsFunc) error {
	meshes, err := m.meshes(list)
	if err != nil {
		return err
	}
	return m.store.List(ctx, meshes, fs...)
}

func (m *meshManager) Create(ctx context.Context, resource core_model.Resource, fs ...core_store.CreateOptionsFunc) error {
	mesh, err := m.mesh(resource)
	if err != nil {
		return err
	}
	if err := core_model.Validate(resource); err != nil {
		return err
	}
	//if err := m.meshValidator.ValidateCreate(ctx, opts.Name, mesh); err != nil {
	//	return err
	//}
	// persist Mesh
	if err := m.store.Create(ctx, mesh, append(fs, core_store.CreatedAt(time.Now()))...); err != nil {
		return err
	}
	return nil
}

func (m *meshManager) Delete(ctx context.Context, resource core_model.Resource, fs ...core_store.DeleteOptionsFunc) error {
	mesh, err := m.mesh(resource)
	if err != nil {
		return err
	}
	//if !m.unsafeDelete {
	//	if err := m.meshValidator.ValidateDelete(ctx, opts.Name); err != nil {
	//		return err
	//	}
	//}
	// delete Mesh first to avoid a state where a Mesh could exist without secrets.
	// even if removal of secrets fails later on, delete operation can be safely tried again.
	var notFoundErr error
	if err := m.store.Delete(ctx, mesh, fs...); err != nil {
		if core_store.IsResourceNotFound(err) {
			notFoundErr = err
		} else { // ignore other errors so we can retry removing other resources
			return err
		}
	}
	// secrets are deleted via owner reference
	return notFoundErr
}

func (m *meshManager) DeleteAll(ctx context.Context, list core_model.ResourceList, fs ...core_store.DeleteAllOptionsFunc) error {
	if _, err := m.meshes(list); err != nil {
		return err
	}
	return core_manager.DeleteAllResources(m, ctx, list, fs...)
}

func (m *meshManager) Update(ctx context.Context, resource core_model.Resource, fs ...core_store.UpdateOptionsFunc) error {
	mesh, err := m.mesh(resource)
	if err != nil {
		return err
	}
	if err := core_model.Validate(resource); err != nil {
		return err
	}

	currentMesh := core_mesh.NewMeshResource()
	if err := m.Get(ctx, currentMesh, core_store.GetBy(core_model.MetaToResourceKey(mesh.GetMeta())), core_store.GetByVersion(mesh.GetMeta().GetVersion())); err != nil {
		return err
	}
	//if err := m.meshValidator.ValidateUpdate(ctx, currentMesh, mesh); err != nil {
	//	return err
	//}
	return m.store.Update(ctx, mesh, append(fs, core_store.ModifiedAt(time.Now()))...)
}

func (m *meshManager) mesh(resource core_model.Resource) (*core_mesh.MeshResource, error) {
	mesh, ok := resource.(*core_mesh.MeshResource)
	if !ok {
		return nil, errors.Errorf("invalid resource type: expected=%T, got=%T", (*core_mesh.MeshResource)(nil), resource)
	}
	return mesh, nil
}

func (m *meshManager) meshes(list core_model.ResourceList) (*core_mesh.MeshResourceList, error) {
	meshes, ok := list.(*core_mesh.MeshResourceList)
	if !ok {
		return nil, errors.Errorf("invalid resource type: expected=%T, got=%T", (*core_mesh.MeshResourceList)(nil), list)
	}
	return meshes, nil
}
