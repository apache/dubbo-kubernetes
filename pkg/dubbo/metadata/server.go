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
	"time"
)

import (
	mesh_proto "github.com/apache/dubbo-kubernetes/api/mesh/v1alpha1"
	"github.com/apache/dubbo-kubernetes/pkg/config/dubbo"
	"github.com/apache/dubbo-kubernetes/pkg/core"
	core_mesh "github.com/apache/dubbo-kubernetes/pkg/core/resources/apis/mesh"
	"github.com/apache/dubbo-kubernetes/pkg/core/resources/manager"
	core_model "github.com/apache/dubbo-kubernetes/pkg/core/resources/model"
	core_store "github.com/apache/dubbo-kubernetes/pkg/core/resources/store"
	"github.com/apache/dubbo-kubernetes/pkg/dubbo/pusher"
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
	if metadata == nil {
		return &mesh_proto.MetaDataRegisterResponse{
			Success: false,
			Message: "Metadata is nil",
		}, nil
	}

	// MetaData name = podName.revision
	name := podName + "." + metadata.Revision
	registerReq := &RegisterRequest{ConfigsUpdated: map[core_model.ResourceKey]*mesh_proto.MetaData{}}
	key := core_model.ResourceKey{
		Mesh: mesh,
		Name: name,
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
	return nil
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

func (m *MetadataServer) tryRegister(key core_model.ResourceKey, newMetadata *mesh_proto.MetaData) error {
	err := core_store.InTx(m.ctx, m.transactions, func(ctx context.Context) error {

		// get Metadata Resource first,
		// if Metadata is not found, create it,
		// else update it.
		metadata := core_mesh.NewMetaDataResource()
		err := m.resourceManager.Get(m.ctx, metadata, core_store.GetBy(key))
		if err != nil && !core_store.IsResourceNotFound(err) {
			log.Error(err, "get Metadata Resource")
			return err
		}

		if core_store.IsResourceNotFound(err) {
			// create if not found
			metadata.Spec = newMetadata
			err = m.resourceManager.Create(m.ctx, metadata, core_store.CreateBy(key), core_store.CreatedAt(time.Now()))
			if err != nil {
				log.Error(err, "create Metadata Resource failed")
				return err
			}

			log.Info("create Metadata Resource success", "key", key, "metadata", newMetadata)
			return nil
		} else {
			// if found, update it
			metadata.Spec = newMetadata

			err = m.resourceManager.Update(m.ctx, metadata, core_store.ModifiedAt(time.Now()))
			if err != nil {
				log.Error(err, "update Metadata Resource failed")
				return err
			}

			log.Info("update Metadata Resource success", "key", key, "metadata", newMetadata)
			return nil
		}
	})
	if err != nil {
		log.Error(err, "transactions failed")
		return err
	}

	return nil
}
