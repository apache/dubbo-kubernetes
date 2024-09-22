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
	"io"
	"strings"
)

import (
	"github.com/google/uuid"

	"github.com/pkg/errors"

	"google.golang.org/grpc/codes"

	"google.golang.org/grpc/status"

	kube_ctrl "sigs.k8s.io/controller-runtime"
)

import (
	mesh_proto "github.com/apache/dubbo-kubernetes/api/mesh/v1alpha1"
	"github.com/apache/dubbo-kubernetes/pkg/config/dubbo"
	"github.com/apache/dubbo-kubernetes/pkg/core"
	core_mesh "github.com/apache/dubbo-kubernetes/pkg/core/resources/apis/mesh"
	"github.com/apache/dubbo-kubernetes/pkg/core/resources/manager"
	core_model "github.com/apache/dubbo-kubernetes/pkg/core/resources/model"
	core_store "github.com/apache/dubbo-kubernetes/pkg/core/resources/store"
	"github.com/apache/dubbo-kubernetes/pkg/core/runtime/component"
	"github.com/apache/dubbo-kubernetes/pkg/mds/client"
	"github.com/apache/dubbo-kubernetes/pkg/mds/pusher"
	k8s_common "github.com/apache/dubbo-kubernetes/pkg/plugins/common/k8s"
	"github.com/apache/dubbo-kubernetes/pkg/util/rmkey"
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
	pusher        pusher.Pusher
	converter     k8s_common.Converter

	ctx             context.Context
	manager         kube_ctrl.Manager
	resourceManager manager.ResourceManager
	transactions    core_store.Transactions
	systemNamespace string
}

func NewMdsServer(
	ctx context.Context,
	config dubbo.DubboConfig,
	pusher pusher.Pusher,
	manager kube_ctrl.Manager,
	converter k8s_common.Converter,
	resourceManager manager.ResourceManager,
	transactions core_store.Transactions,
	localZone string,
	systemNamespace string,
) *MdsServer {
	return &MdsServer{
		localZone:       localZone,
		config:          config,
		pusher:          pusher,
		mappingQueue:    make(chan *RegisterRequest, queueSize),
		metadataQueue:   make(chan *RegisterRequest, queueSize),
		ctx:             ctx,
		resourceManager: resourceManager,
		manager:         manager,
		converter:       converter,
		transactions:    transactions,
		systemNamespace: systemNamespace,
	}
}

func (m *MdsServer) Start(stop <-chan struct{}) error {
	go m.debounce(m.mappingQueue, stop, m.mappingRegister)
	go m.debounce(m.metadataQueue, stop, m.metadataRegister)
	return nil
}

func (m *MdsServer) NeedLeaderElection() bool {
	return false
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

func (m *MdsServer) MetadataSync(stream mesh_proto.MDSSyncService_MetadataSyncServer) error {
	mesh := core_model.DefaultMesh // todo: mesh
	errChan := make(chan error)

	clientID := uuid.NewString()
	metadataSyncStream := client.NewDubboSyncStream(stream)
	// DubboSyncClient is to handle MetaSyncRequest from data plane
	metadataSyncClient := client.NewDubboSyncClient(
		log.WithName("client"),
		clientID,
		metadataSyncStream,
		&client.Callbacks{
			OnMetadataSyncRequestReceived: func(request *mesh_proto.MetadataSyncRequest) error {
				// when received request, invoke callback
				m.pusher.InvokeCallback(
					core_mesh.MetaDataType,
					clientID,
					request,
					// filter by req.ApplicationName (and req.Revision)
					func(rawRequest interface{}, resourceList core_model.ResourceList) core_model.ResourceList {
						req := rawRequest.(*mesh_proto.MetadataSyncRequest)
						metadataList := resourceList.(*core_mesh.MetaDataResourceList)

						// only response the target MetaData Resource by application name or revision
						respMetadataList := &core_mesh.MetaDataResourceList{}
						for _, item := range metadataList.Items {
							// MetaData.Name = AppName.Revision, so we need to check MedaData.Name has prefix of AppName
							if item.Spec != nil && strings.HasPrefix(item.Spec.App, req.ApplicationName) {
								if req.Revision != "" {
									// revision is not empty, response the Metadata with application name and target revision
									if req.Revision == item.Spec.Revision {
										_ = respMetadataList.AddItem(item)
									}
								} else {
									// revision is empty, response the Metadata with target application name
									_ = respMetadataList.AddItem(item)
								}
							}
						}

						return respMetadataList
					},
				)
				return nil
			},
		})

	// Add callback and call by following resourceType and metadataSyncClient.ClientID()
	m.pusher.AddCallback(
		core_mesh.MetaDataType,
		metadataSyncClient.ClientID(),
		func(items pusher.PushedItems) {
			resourceList := items.ResourceList()
			revision := items.Revision()
			metadataList, ok := resourceList.(*core_mesh.MetaDataResourceList)
			if !ok {
				return
			}

			err := metadataSyncClient.Send(metadataList, revision)
			if err != nil {
				if errors.Is(err, io.EOF) {
					log.Info("DubboSyncClient finished gracefully")
					errChan <- nil
					return
				}

				log.Error(err, "send metadata sync response failed", "metadataList", metadataList, "revision", revision)
				errChan <- errors.Wrap(err, "DubboSyncClient send with an error")
			}
		},
		// filtered which resources client focus on
		func(resourceList core_model.ResourceList) core_model.ResourceList {
			if resourceList.GetItemType() != core_mesh.MetaDataType {
				return nil
			}

			// only send Metadata which client subscribed
			newResourceList := &core_mesh.MeshResourceList{}
			for _, resource := range resourceList.GetItems() {
				expected := false
				metaData := resource.(*core_mesh.MetaDataResource)
				for _, applicationName := range metadataSyncStream.SubscribedApplicationNames() {
					// MetaData.Name = AppName.Revision, so we need to check MedaData.Name has prefix of AppName
					if strings.HasPrefix(metaData.Spec.GetApp(), applicationName) && mesh == resource.GetMeta().GetMesh() {
						expected = true
						break
					}
				}

				if expected {
					// find
					_ = newResourceList.AddItem(resource)
				}
			}

			return newResourceList
		},
	)

	// in the end, remove callback of this client
	defer m.pusher.RemoveCallback(core_mesh.MetaDataType, metadataSyncClient.ClientID())

	go func() {
		// Handle requests from client
		err := metadataSyncClient.HandleReceive()
		if errors.Is(err, io.EOF) {
			log.Info("DubboSyncClient finished gracefully")
			errChan <- nil
			return
		}

		log.Error(err, "DubboSyncClient finished with an error")
		errChan <- errors.Wrap(err, "DubboSyncClient finished with an error")
	}()

	for {
		select {
		case err := <-errChan:
			if err == nil {
				log.Info("MetadataSync finished gracefully")
				return nil
			}

			log.Error(err, "MetadataSync finished with an error")
			return status.Error(codes.Internal, err.Error())
		}
	}
}

func (m *MdsServer) MappingSync(stream mesh_proto.MDSSyncService_MappingSyncServer) error {
	mesh := core_model.DefaultMesh // todo: mesh
	errChan := make(chan error)

	clientID := uuid.NewString()
	mappingSyncStream := client.NewDubboSyncStream(stream)
	// DubboSyncClient is to handle MappingSyncRequest from data plane
	mappingSyncClient := client.NewDubboSyncClient(
		log.WithName("client"),
		clientID,
		mappingSyncStream,
		&client.Callbacks{
			OnMappingSyncRequestReceived: func(request *mesh_proto.MappingSyncRequest) error {
				// when received request, invoke callback
				m.pusher.InvokeCallback(
					core_mesh.MappingType,
					clientID,
					request,
					func(rawRequest interface{}, resourceList core_model.ResourceList) core_model.ResourceList {
						req := rawRequest.(*mesh_proto.MappingSyncRequest)
						mappingList := resourceList.(*core_mesh.MappingResourceList)

						// only response the target Mapping Resource by interface name
						respMappingList := &core_mesh.MappingResourceList{}
						for _, item := range mappingList.Items {
							if item.Spec != nil && req.InterfaceName == item.Spec.InterfaceName {
								_ = respMappingList.AddItem(item)
							}
						}

						return respMappingList
					},
				)
				return nil
			},
		})

	m.pusher.AddCallback(
		core_mesh.MappingType,
		mappingSyncClient.ClientID(),
		func(items pusher.PushedItems) {
			resourceList := items.ResourceList()
			revision := items.Revision()
			mappingList, ok := resourceList.(*core_mesh.MappingResourceList)
			if !ok {
				return
			}

			err := mappingSyncClient.Send(mappingList, revision)
			if err != nil {
				if errors.Is(err, io.EOF) {
					log.Info("DubboSyncClient finished gracefully")
					errChan <- nil
					return
				}

				log.Error(err, "send mapping sync response failed", "mappingList", mappingList, "revision", revision)
				errChan <- errors.Wrap(err, "DubboSyncClient send with an error")
			}
		},
		func(resourceList core_model.ResourceList) core_model.ResourceList {
			if resourceList.GetItemType() != core_mesh.MappingType {
				return nil
			}

			// only send Mapping which client subscribed
			newResourceList := &core_mesh.MeshResourceList{}
			for _, resource := range resourceList.GetItems() {
				expected := false
				for _, interfaceName := range mappingSyncStream.SubscribedInterfaceNames() {
					if interfaceName == resource.GetMeta().GetName() && mesh == resource.GetMeta().GetMesh() {
						expected = true
						break
					}
				}

				if expected {
					// find
					_ = newResourceList.AddItem(resource)
				}
			}

			return newResourceList
		},
	)

	// in the end, remove callback of this client
	defer m.pusher.RemoveCallback(core_mesh.MappingType, mappingSyncClient.ClientID())

	go func() {
		// Handle requests from client
		err := mappingSyncClient.HandleReceive()
		if errors.Is(err, io.EOF) {
			log.Info("DubboSyncClient finished gracefully")
			errChan <- nil
			return
		}

		log.Error(err, "DubboSyncClient finished with an error")
		errChan <- errors.Wrap(err, "DubboSyncClient finished with an error")
	}()

	for {
		select {
		case err := <-errChan:
			if err == nil {
				log.Info("MappingSync finished gracefully")
				return nil
			}

			log.Error(err, "MappingSync finished with an error")
			return status.Error(codes.Internal, err.Error())
		}
	}
}
