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

	kube_core "k8s.io/api/core/v1"

	kube_types "k8s.io/apimachinery/pkg/types"

	kube_ctrl "sigs.k8s.io/controller-runtime"
	kube_controllerutil "sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
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
	k8s_common "github.com/apache/dubbo-kubernetes/pkg/plugins/common/k8s"
	"github.com/apache/dubbo-kubernetes/pkg/util/rmkey"
)

var log = core.Log.WithName("dubbo").WithName("server").WithName("service-name-mapping")

const queueSize = 100

var _ component.Component = &SnpServer{}

type SnpServer struct {
	mesh_proto.ServiceNameMappingServiceServer

	localZone       string
	config          dubbo.DubboConfig
	queue           chan *RegisterRequest
	pusher          pusher.Pusher
	converter       k8s_common.Converter
	resourceManager manager.ResourceManager

	ctx             context.Context
	manager         kube_ctrl.Manager
	transactions    core_store.Transactions
	systemNamespace string
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
	manager kube_ctrl.Manager,
	converter k8s_common.Converter,
	resourceManager manager.ResourceManager,
	transactions core_store.Transactions,
	localZone string,
	systemNamespace string,
) *SnpServer {
	return &SnpServer{
		localZone:       localZone,
		config:          config,
		pusher:          pusher,
		queue:           make(chan *RegisterRequest, queueSize),
		ctx:             ctx,
		resourceManager: resourceManager,
		manager:         manager,
		converter:       converter,
		transactions:    transactions,
		systemNamespace: systemNamespace,
	}
}

func (s *SnpServer) MappingRegister(ctx context.Context, req *mesh_proto.MappingRegisterRequest) (*mesh_proto.MappingRegisterResponse, error) {
	mesh := core_model.DefaultMesh // todo: mesh
	interfaces := req.GetInterfaceNames()
	applicationName := req.GetApplicationName()

	registerReq := &RegisterRequest{ConfigsUpdated: map[core_model.ResourceReq]map[string]struct{}{}}
	for _, interfaceName := range interfaces {
		key := core_model.ResourceReq{
			Mesh:      mesh,
			Name:      interfaceName,
			Namespace: req.GetNamespace(),
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
			if err := s.tryRegister(key, appNames); err != nil {
				log.Error(err, "register failed", "key", key)
			} else {
				break
			}
		}
	}
}

func (s *SnpServer) tryRegister(req core_model.ResourceReq, newApps []string) error {
	err := core_store.InTx(s.ctx, s.transactions, func(ctx context.Context) error {
		interfaceName := req.Name
		// 先获取到对应Name和Namespace的Pod
		kubeClient := s.manager.GetClient()
		pod := &kube_core.Pod{}
		if err := kubeClient.Get(ctx, kube_types.NamespacedName{
			Name:      req.PodName,
			Namespace: req.Namespace,
		}, pod); err != nil {
			return errors.Wrap(err, "unable to get Namespace for Pod")
		}
		key := core_model.ResourceKey{
			Mesh: req.Mesh,
			Name: rmkey.GenerateMappingResourceKey(interfaceName, req.Namespace),
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
		var previousAppNames map[string]struct{}
		previousLen := len(mapping.Spec.ApplicationNames)
		// createOrUpdate
		if core_store.IsResourceNotFound(err) {
			for _, name := range mapping.Spec.ApplicationNames {
				previousAppNames[name] = struct{}{}
			}
		}
		for _, newApp := range newApps {
			previousAppNames[newApp] = struct{}{}
		}
		if len(previousAppNames) == previousLen {
			log.Info("Mapping not need to register", "interfaceName", interfaceName, "applicationNames", newApps)
			return nil
		}

		var mergeApps []string
		for name := range previousAppNames {
			mergeApps = append(mergeApps, name)
		}

		mappingResource := core_mesh.NewMappingResource()
		mappingResource.SetMeta(&resourceMetaObject{
			Name: rmkey.GenerateMappingResourceKey(interfaceName, s.systemNamespace),
			Mesh: core_model.DefaultMesh,
		})
		err = mappingResource.SetSpec(mesh_proto.Mapping{
			Zone:             s.localZone,
			InterfaceName:    interfaceName,
			ApplicationNames: mergeApps,
		})
		if err != nil {
			return err
		}
		mappingObject, err := s.converter.ToKubernetesObject(mappingResource)
		if err != nil {
			return err
		}

		operationResult, err := kube_controllerutil.CreateOrUpdate(ctx, kubeClient, mappingObject, func() error {
			if err := kube_controllerutil.SetControllerReference(pod, mappingObject, s.manager.GetScheme()); err != nil {
				return errors.Wrap(err, "unable to set Metadata's controller reference to Pod")
			}
			return nil
		})
		if err != nil {
			if !errors.Is(err, context.Canceled) {
				log.Error(err, "unable to create/update Metadata", "operationResult", operationResult)
			}

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
