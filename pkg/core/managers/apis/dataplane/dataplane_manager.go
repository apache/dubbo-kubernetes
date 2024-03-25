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

	kube_ctrl "sigs.k8s.io/controller-runtime"
)

import (
	mesh_proto "github.com/apache/dubbo-kubernetes/api/mesh/v1alpha1"
	config_core "github.com/apache/dubbo-kubernetes/pkg/config/core"
	"github.com/apache/dubbo-kubernetes/pkg/core"
	"github.com/apache/dubbo-kubernetes/pkg/core/logger"
	core_mesh "github.com/apache/dubbo-kubernetes/pkg/core/resources/apis/mesh"
	core_manager "github.com/apache/dubbo-kubernetes/pkg/core/resources/manager"
	core_model "github.com/apache/dubbo-kubernetes/pkg/core/resources/model"
	core_store "github.com/apache/dubbo-kubernetes/pkg/core/resources/store"
)

type dataplaneManager struct {
	core_manager.ResourceManager
	store      core_store.ResourceStore
	zone       string
	manager    kube_ctrl.Manager
	deployMode config_core.DeployMode
}

func NewDataplaneManager(store core_store.ResourceStore, zone string, manager kube_ctrl.Manager, mode config_core.DeployMode) core_manager.ResourceManager {
	return &dataplaneManager{
		ResourceManager: core_manager.NewResourceManager(store),
		store:           store,
		zone:            zone,
		manager:         manager,
		deployMode:      mode,
	}
}

func (m *dataplaneManager) Create(ctx context.Context, r core_model.Resource, fs ...core_store.CreateOptionsFunc) error {
	return m.store.Create(ctx, r, append(fs, core_store.CreatedAt(core.Now()))...)
}

func (m *dataplaneManager) Update(ctx context.Context, r core_model.Resource, fs ...core_store.UpdateOptionsFunc) error {
	return m.ResourceManager.Update(ctx, r, fs...)
}

func (m *dataplaneManager) Get(ctx context.Context, r core_model.Resource, opts ...core_store.GetOptionsFunc) error {
	dataplane, err := m.dataplane(r)
	if err != nil {
		return err
	}
	options := core_store.NewGetOptions(opts...)
	if options.Labels[mesh_proto.Application] == "" || options.Labels[mesh_proto.Revision] == "" {
		logger.Sugar().Error("需要携带application和revision才能查询")
		return nil
	}
	if err := m.store.Get(ctx, dataplane, opts...); err != nil {
		return err
	}
	m.setInboundsClusterTag(dataplane)
	m.setHealth(dataplane)
	if m.deployMode != config_core.UniversalMode {
		m.setExtensions(dataplane)
	}
	return nil
}

func (m *dataplaneManager) List(ctx context.Context, r core_model.ResourceList, opts ...core_store.ListOptionsFunc) error {
	dataplanes, err := m.dataplanes(r)
	if err != nil {
		return err
	}
	if err := m.store.List(ctx, dataplanes, opts...); err != nil {
		return err
	}
	for _, item := range dataplanes.Items {
		m.setHealth(item)
		m.setInboundsClusterTag(item)
		if m.deployMode != config_core.UniversalMode {
			m.setExtensions(item)
		}
	}
	return nil
}

func (m *dataplaneManager) dataplane(resource core_model.Resource) (*core_mesh.DataplaneResource, error) {
	dp, ok := resource.(*core_mesh.DataplaneResource)
	if !ok {
		return nil, errors.Errorf("invalid resource type: expected=%T, got=%T", (*core_mesh.DataplaneResource)(nil), resource)
	}
	return dp, nil
}

func (m *dataplaneManager) dataplanes(resources core_model.ResourceList) (*core_mesh.DataplaneResourceList, error) {
	dp, ok := resources.(*core_mesh.DataplaneResourceList)
	if !ok {
		return nil, errors.Errorf("invalid resource type: expected=%T, got=%T", (*core_mesh.DataplaneResourceList)(nil), resources)
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

func (m *dataplaneManager) setHealth(dp *core_mesh.DataplaneResource) {
	if m.zone == "" || dp.Spec.Networking == nil {
		return
	}

	for _, inbound := range dp.Spec.Networking.Inbound {
		if inbound.ServiceProbe != nil {
			inbound.State = mesh_proto.Dataplane_Networking_Inbound_NotReady
			// write health for backwards compatibility with Kuma 2.5 and older
			inbound.Health = &mesh_proto.Dataplane_Networking_Inbound_Health{
				Ready: false,
			}
		}
	}
}

func (m *dataplaneManager) setExtensions(dp *core_mesh.DataplaneResource) {
	if m.zone == "" || dp.Spec.Networking == nil {
		return
	}

}
