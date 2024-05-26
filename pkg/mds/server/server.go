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

package server

import (
	"context"
	mesh_proto "github.com/apache/dubbo-kubernetes/api/mesh/v1alpha1"
	"github.com/apache/dubbo-kubernetes/pkg/config/dubbo"
	"github.com/apache/dubbo-kubernetes/pkg/core"
	core_mesh "github.com/apache/dubbo-kubernetes/pkg/core/resources/apis/mesh"
	"github.com/apache/dubbo-kubernetes/pkg/core/resources/manager"
	core_model "github.com/apache/dubbo-kubernetes/pkg/core/resources/model"
	core_store "github.com/apache/dubbo-kubernetes/pkg/core/resources/store"
	"github.com/apache/dubbo-kubernetes/pkg/core/runtime/component"
	k8s_common "github.com/apache/dubbo-kubernetes/pkg/plugins/common/k8s"
	"github.com/apache/dubbo-kubernetes/pkg/util/rmkey"
	kube_ctrl "sigs.k8s.io/controller-runtime"
)

var log = core.Log.WithName("mds").WithName("server")

const queueSize = 100

var _ component.Component = &MdsServer{}

type MdsServer struct {
	mesh_proto.MDSSyncServiceServer

	localZone     string
	config        dubbo.DubboConfig
	mappingQueue  chan *RegisterRequest
	metadataQueue chan *RegisterRequest
	converter     k8s_common.Converter

	ctx             context.Context
	manager         kube_ctrl.Manager
	resourceManager manager.ResourceManager
	transactions    core_store.Transactions
	systemNamespace string
}

func (m *MdsServer) Start(stop <-chan struct{}) error {
	go m.debounce(m.mappingQueue, stop, m.mappingRegister)
	go m.debounce(m.metadataQueue, stop, m.metadataRegister)
	return nil
}

func (m *MdsServer) NeedLeaderElection() bool {
	return false
}

func (m *MdsServer) ZoneToDubboInstance(stream mesh_proto.MDSSyncService_ZoneToDubboInstanceServer) error {
	return nil
}

func (m *MdsServer) MetadataRegister(ctx context.Context, req *mesh_proto.MetaDataRegisterRequest) (*mesh_proto.MetaDataRegisterResponse, error) {
	mesh := core_model.DefaultMesh // todo: mesh
	metadata := req.GetMetadata()
	if metadata == nil {
		return &mesh_proto.MetaDataRegisterResponse{
			Success: false,
			Message: "Metadata is nil",
		}, nil
	}

	name := rmkey.GenerateMetadataResourceKey(metadata.App, metadata.Revision, req.GetNamespace())
	registerReq := &RegisterRequest{
		MetadataConfigsUpdated: map[core_model.ResourceKey]*mesh_proto.MetaData{},
	}
	key := core_model.ResourceKey{
		Mesh: mesh,
		Name: name,
	}
	registerReq.MetadataConfigsUpdated[key] = metadata

	// push into queue to debounce, register Metadata Resource
	m.metadataQueue <- registerReq

	// 注射应用名
	dataplane := core_mesh.NewDataplaneResource()
	err := m.resourceManager.Get(m.ctx, dataplane, core_store.GetBy(core_model.ResourceKey{
		Mesh: core_model.DefaultMesh,
		Name: rmkey.GenerateNamespacedName(req.GetPodName(), req.GetNamespace()),
	}))
	if err != nil {
		return nil, err
	}
	if dataplane.Spec.Extensions == nil {
		dataplane.Spec.Extensions = make(map[string]string)
	}
	// 拿到dataplane, 添加extensions, 设置revision
	dataplane.Spec.Extensions[mesh_proto.Revision] = req.GetMetadata().GetRevision()
	dataplane.Spec.Extensions[mesh_proto.Application] = req.GetMetadata().GetApp()

	// 更新dataplane
	err = m.resourceManager.Update(m.ctx, dataplane)
	if err != nil {
		return nil, err
	}

	return &mesh_proto.MetaDataRegisterResponse{
		Success: true,
		Message: "success",
	}, nil
}

func (m *MdsServer) MappingRegister(ctx context.Context, req *mesh_proto.MappingRegisterRequest) (*mesh_proto.MappingRegisterResponse, error) {
	mesh := core_model.DefaultMesh
	interfaces := req.GetInterfaceNames()
	applicationName := req.GetApplicationName()

	registerReq := &RegisterRequest{
		MappingConfigsUpdates: map[core_model.ResourceKey]map[string]struct{}{},
	}
	for _, interfaceName := range interfaces {
		key := core_model.ResourceKey{
			Mesh: mesh,
			Name: interfaceName,
		}
		if _, ok := registerReq.MappingConfigsUpdates[key]; !ok {
			registerReq.MappingConfigsUpdates[key] = make(map[string]struct{})
		}
		registerReq.MappingConfigsUpdates[key][applicationName] = struct{}{}
	}

	m.mappingQueue <- registerReq

	return &mesh_proto.MappingRegisterResponse{
		Success: true,
		Message: "success",
	}, nil
}
