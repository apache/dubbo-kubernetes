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
	"context"
	"io"
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
	"github.com/apache/dubbo-kubernetes/pkg/core/runtime/component"
	"github.com/apache/dubbo-kubernetes/pkg/dubbo/client"
	"github.com/apache/dubbo-kubernetes/pkg/dubbo/pusher"
)

var log = core.Log.WithName("dubbo").WithName("server").WithName("service-name-mapping")

const queueSize = 100

var _ component.Component = &SnpServer{}

type SnpServer struct {
	mesh_proto.ServiceNameMappingServiceServer

	localZone string
	config    dubbo.DubboConfig
	queue     chan *RegisterRequest
	pusher    pusher.Pusher

	ctx             context.Context
	resourceManager manager.ResourceManager
	transactions    core_store.Transactions
}

func (s *SnpServer) Start(stop <-chan struct{}) error {
	// we start debounce to prevent too many MappingRegisterRequests, we aggregate mapping register information
	go s.debounce(stop, s.register)

	return nil
}

func (s *SnpServer) NeedLeaderElection() bool {
	return false
}

func NewSnpServer(
	ctx context.Context,
	config dubbo.DubboConfig,
	pusher pusher.Pusher,
	resourceManager manager.ResourceManager,
	transactions core_store.Transactions,
	localZone string,
) *SnpServer {
	return &SnpServer{
		localZone:       localZone,
		config:          config,
		pusher:          pusher,
		queue:           make(chan *RegisterRequest, queueSize),
		ctx:             ctx,
		resourceManager: resourceManager,
		transactions:    transactions,
	}
}

func (s *SnpServer) MappingRegister(ctx context.Context, req *mesh_proto.MappingRegisterRequest) (*mesh_proto.MappingRegisterResponse, error) {
	mesh := core_model.DefaultMesh // todo: mesh
	interfaces := req.GetInterfaceNames()
	applicationName := req.GetApplicationName()

	registerReq := &RegisterRequest{ConfigsUpdated: map[core_model.ResourceKey]map[string]struct{}{}}
	for _, interfaceName := range interfaces {
		key := core_model.ResourceKey{
			Mesh: mesh,
			Name: interfaceName,
		}
		if _, ok := registerReq.ConfigsUpdated[key]; !ok {
			registerReq.ConfigsUpdated[key] = make(map[string]struct{})
		}
		registerReq.ConfigsUpdated[key][applicationName] = struct{}{}
	}

	// push into queue to debounce, register Mapping Resource
	s.queue <- registerReq

	return &mesh_proto.MappingRegisterResponse{
		Success: true,
		Message: "success",
	}, nil
}

func (s *SnpServer) MappingSync(stream mesh_proto.ServiceNameMappingService_MappingSyncServer) error {
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
				s.pusher.InvokeCallback(
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

	s.pusher.AddCallback(
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
	defer s.pusher.RemoveCallback(core_mesh.MappingType, mappingSyncClient.ClientID())

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

func (s *SnpServer) debounce(stopCh <-chan struct{}, pushFn func(m *RegisterRequest)) {
	ch := s.queue
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
		if eventDelay >= s.config.Debounce.Max || quietTime >= s.config.Debounce.After {
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
			timeChan = time.After(s.config.Debounce.After - quietTime)
		}
	}

	for {
		select {
		case <-freeCh:
			free = true
			pushWorker()
		case r := <-ch:
			if !s.config.Debounce.Enable {
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

func (s *SnpServer) register(req *RegisterRequest) {
	for key, m := range req.ConfigsUpdated {
		var appNames []string
		for app := range m {
			appNames = append(appNames, app)
		}
		for i := 0; i < 3; i++ {
			if err := s.tryRegister(key.Mesh, key.Name, appNames); err != nil {
				log.Error(err, "register failed", "key", key)
			} else {
				break
			}
		}
	}
}

func (s *SnpServer) tryRegister(mesh, interfaceName string, newApps []string) error {
	err := core_store.InTx(s.ctx, s.transactions, func(ctx context.Context) error {
		key := core_model.ResourceKey{
			Mesh: mesh,
			Name: interfaceName,
		}

		// get Mapping Resource first,
		// if Mapping is not found, create it,
		// else update it.
		mapping := core_mesh.NewMappingResource()
		err := s.resourceManager.Get(s.ctx, mapping, core_store.GetBy(key))
		if err != nil && !core_store.IsResourceNotFound(err) {
			log.Error(err, "get Mapping Resource")
			return err
		}

		if core_store.IsResourceNotFound(err) {
			// create if not found
			mapping.Spec = &mesh_proto.Mapping{
				Zone:             s.localZone,
				InterfaceName:    interfaceName,
				ApplicationNames: newApps,
			}
			err = s.resourceManager.Create(s.ctx, mapping, core_store.CreateBy(key), core_store.CreatedAt(time.Now()))
			if err != nil {
				log.Error(err, "create Mapping Resource failed")
				return err
			}

			log.Info("create Mapping Resource success", "key", key, "applicationNames", newApps)
			return nil
		} else {
			// if found, update it
			previousLen := len(mapping.Spec.ApplicationNames)
			previousAppNames := make(map[string]struct{}, previousLen)
			for _, name := range mapping.Spec.ApplicationNames {
				previousAppNames[name] = struct{}{}
			}
			for _, newApp := range newApps {
				previousAppNames[newApp] = struct{}{}
			}
			if len(previousAppNames) == previousLen {
				log.Info("Mapping not need to register", "interfaceName", interfaceName, "applicationNames", newApps)
				return nil
			}

			mergedApps := make([]string, 0, len(previousAppNames))
			for name := range previousAppNames {
				mergedApps = append(mergedApps, name)
			}
			mapping.Spec = &mesh_proto.Mapping{
				Zone:             s.localZone,
				InterfaceName:    interfaceName,
				ApplicationNames: mergedApps,
			}

			err = s.resourceManager.Update(s.ctx, mapping, core_store.ModifiedAt(time.Now()))
			if err != nil {
				log.Error(err, "update Mapping Resource failed")
				return err
			}

			log.Info("update Mapping Resource success", "key", key, "applicationNames", newApps)
			return nil
		}
	})
	if err != nil {
		log.Error(err, "transactions failed")
		return err
	}

	return nil
}
