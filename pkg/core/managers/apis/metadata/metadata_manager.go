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

package metadata

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

type metadataManager struct {
	core_manager.ResourceManager
	store      core_store.ResourceStore
	manager    kube_ctrl.Manager
	deployMode config_core.DeployMode
}

func NewMetadataManager(store core_store.ResourceStore, manage kube_ctrl.Manager, mode config_core.DeployMode) core_manager.ResourceManager {
	return &metadataManager{
		ResourceManager: core_manager.NewResourceManager(store),
		store:           store,
		deployMode:      mode,
		manager:         manage,
	}
}

func (m *metadataManager) Create(ctx context.Context, r core_model.Resource, fs ...core_store.CreateOptionsFunc) error {
	metadata, err := m.metadata(r)
	if err != nil {
		return err
	}
	if metadata.GetSpec().(*mesh_proto.MetaData).GetRevision() == "" {
		logger.Sugar().Error("必须传递revision才能创建metadata资源")
		return nil
	}
	return m.store.Create(ctx, r, append(fs, core_store.CreatedAt(core.Now()))...)
}

func (m *metadataManager) metadata(resource core_model.Resource) (*core_mesh.MetaDataResource, error) {
	metadata, ok := resource.(*core_mesh.MetaDataResource)
	if !ok {
		return nil, errors.Errorf("invalid resource type: expected=%T, got=%T", (*core_mesh.DataplaneResource)(nil), resource)
	}
	return metadata, nil
}
