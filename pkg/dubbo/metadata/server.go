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
	"io"
	"strings"
	"time"
)

import (
	"github.com/google/uuid"

	"github.com/pkg/errors"

	"google.golang.org/grpc/codes"

	"google.golang.org/grpc/status"
)

import (
	mesh_proto "github.com/apache/dubbo-kubernetes/api/mesh/v1alpha1"
	"github.com/apache/dubbo-kubernetes/pkg/config/dubbo"
	"github.com/apache/dubbo-kubernetes/pkg/core"
	core_mesh "github.com/apache/dubbo-kubernetes/pkg/core/resources/apis/mesh"
	"github.com/apache/dubbo-kubernetes/pkg/core/resources/manager"
	core_model "github.com/apache/dubbo-kubernetes/pkg/core/resources/model"
	core_store "github.com/apache/dubbo-kubernetes/pkg/core/resources/store"
	"github.com/apache/dubbo-kubernetes/pkg/dubbo/client"
	"github.com/apache/dubbo-kubernetes/pkg/dubbo/pusher"
	"github.com/apache/dubbo-kubernetes/pkg/util/rmkey"
)

var log = core.Log.WithName("dubbo").WithName("server").WithName("metadata")

const queueSize = 100

type MetadataServer struct {
	mesh_proto.MetadataServiceServer

	config dubbo.DubboConfig
	queue  chan *RegisterRequest
	pusher pusher.Pusher

	ctx             context.Context
	resourceManager manager.ResourceManager
	transactions    core_store.Transactions
}

func (m *MetadataServer) Start(stop <-chan struct{}) error {
	// we start debounce to prevent too many MetadataRegisterRequests, we aggregate metadata register information
	go m.debounce(stop, m.register)

	return nil
}

func (m *MetadataServer) NeedLeaderElection() bool {
	return false
}

func NewMetadataServe(
	ctx context.Context,
	config dubbo.DubboConfig,
	pusher pusher.Pusher,
	resourceManager manager.ResourceManager,
	transactions core_store.Transactions,
) *MetadataServer {
	return &MetadataServer{
		config:          config,
		pusher:          pusher,
		queue:           make(chan *RegisterRequest, queueSize),
		ctx:             ctx,
		resourceManager: resourceManager,
		transactions:    transactions,
	}
}

func (m *MetadataServer) MetadataRegister(ctx context.Context, req *mesh_proto.MetaDataRegisterRequest) (*mesh_proto.MetaDataRegisterResponse, error) {
	mesh := core_model.DefaultMesh // todo: mesh
	podName := req.GetPodName()
	metadata := req.GetMetadata()
	namespace := req.GetNamespace()
	if metadata == nil {
		return &mesh_proto.MetaDataRegisterResponse{
			Success: false,
			Message: "Metadata is nil",
		}, nil
	}

	name := rmkey.GenerateMetadataResourceKey(metadata.App, metadata.Revision, req.GetNamespace())
	registerReq := &RegisterRequest{ConfigsUpdated: map[core_model.ResourceReq]*mesh_proto.MetaData{}}
	key := core_model.ResourceReq{
		Mesh:      mesh,
		Name:      name,
		PodName:   podName,
		Namespace: namespace,
	}
	registerReq.ConfigsUpdated[key] = metadata

	// push into queue to debounce, register Metadata Resource
	m.queue <- registerReq

	return &mesh_proto.MetaDataRegisterResponse{
		Success: true,
		Message: "success",
	}, nil
}

func (m *MetadataServer) MetadataSync(stream mesh_proto.MetadataService_MetadataSyncServer) error {
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

func (m *MetadataServer) debounce(stopCh <-chan struct{}, pushFn func(m *RegisterRequest)) {
	ch := m.queue
	var timeChan <-chan time.Time
	var startDebounce time.Time
	var lastConfigUpdateTime time.Time

	pushCounter := 0
	debouncedEvents := 0

	var req *RegisterRequest

	free := true
	freeCh := make(chan struct{}, 1)

	push := func(req *RegisterRequest) {
		pushFn(req)
		freeCh <- struct{}{}
	}

	pushWorker := func() {
		eventDelay := time.Since(startDebounce)
		quietTime := time.Since(lastConfigUpdateTime)
		if eventDelay >= m.config.Debounce.Max || quietTime >= m.config.Debounce.After {
			if req != nil {
				pushCounter++

				if req.ConfigsUpdated != nil {
					log.Info("debounce stable[%d] %d for config %s: %v since last change, %v since last push",
						pushCounter, debouncedEvents, configsUpdated(req),
						quietTime, eventDelay)
				}
				free = false
				go push(req)
				req = nil
				debouncedEvents = 0
			}
		} else {
			timeChan = time.After(m.config.Debounce.After - quietTime)
		}
	}

	for {
		select {
		case <-freeCh:
			free = true
			pushWorker()
		case r := <-ch:
			if !m.config.Debounce.Enable {
				go push(r)
				req = nil
				continue
			}

			lastConfigUpdateTime = time.Now()
			if debouncedEvents == 0 {
				timeChan = time.After(200 * time.Millisecond)
				startDebounce = lastConfigUpdateTime
			}
			debouncedEvents++

			req = req.merge(r)
		case <-timeChan:
			if free {
				pushWorker()
			}
		case <-stopCh:
			return
		}
	}
}

func (m *MetadataServer) register(req *RegisterRequest) {
	for key, metadata := range req.ConfigsUpdated {
		for i := 0; i < 3; i++ {
			if err := m.tryRegister(key, metadata); err != nil {
				log.Error(err, "register failed", "key", key)
			} else {
				break
			}
		}
	}
}

func (m *MetadataServer) tryRegister(key core_model.ResourceReq, newMetadata *mesh_proto.MetaData) error {
	err := core_store.InTx(m.ctx, m.transactions, func(ctx context.Context) error {
		// get Metadata Resource first,
		// if Metadata is not found, create it,
		// else update it.
		metadata := core_mesh.NewMetaDataResource()
		err := m.resourceManager.Get(m.ctx, metadata, core_store.GetBy(core_model.ResourceKey{
			Mesh: key.Mesh,
			Name: key.Name,
		}))
		if err != nil && !core_store.IsResourceNotFound(err) {
			log.Error(err, "get Metadata Resource")
			return err
		}

		if core_store.IsResourceNotFound(err) {
			// create if not found
			metadata.Spec = newMetadata
			err = m.resourceManager.Create(m.ctx, metadata, core_store.CreateBy(core_model.ResourceKey{
				Mesh: key.Mesh,
				Name: key.Name,
			}), core_store.CreatedAt(time.Now()))
			if err != nil {
				log.Error(err, "create Metadata Resource failed")
				return err
			}

			log.Info("create Metadata Resource success", "key", key, "metadata", newMetadata)
		} else {
			// if found, update it
			metadata.Spec = newMetadata

			err = m.resourceManager.Update(m.ctx, metadata, core_store.ModifiedAt(time.Now()))
			if err != nil {
				log.Error(err, "update Metadata Resource failed")
				return err
			}

			log.Info("update Metadata Resource success", "key", key, "metadata", newMetadata)
		}

		// 更新dataplane资源
		// 根据podName Get到dataplane资源
		dataplane := core_mesh.NewDataplaneResource()
		err = m.resourceManager.Get(m.ctx, dataplane, core_store.GetBy(core_model.ResourceKey{
			Mesh: core_model.DefaultMesh,
			Name: rmkey.GenerateNamespacedName(key.PodName, key.Namespace),
		}))
		if err != nil {
			return err
		}
		if dataplane.Spec.Extensions == nil {
			dataplane.Spec.Extensions = make(map[string]string)
		}
		// 拿到dataplane, 添加extensions, 设置revision
		dataplane.Spec.Extensions[mesh_proto.Revision] = metadata.Spec.Revision
		dataplane.Spec.Extensions[mesh_proto.Application] = metadata.Spec.App

		// 更新dataplane
		err = m.resourceManager.Update(m.ctx, dataplane)
		if err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		log.Error(err, "transactions failed")
		return err
	}

	return nil
}
