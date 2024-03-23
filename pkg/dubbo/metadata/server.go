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
	mesh_proto "github.com/apache/dubbo-kubernetes/api/mesh/v1alpha1"
	"github.com/apache/dubbo-kubernetes/pkg/core"
	"github.com/apache/dubbo-kubernetes/pkg/core/resources/manager"
	core_store "github.com/apache/dubbo-kubernetes/pkg/core/resources/store"
	"github.com/apache/dubbo-kubernetes/pkg/dubbo/pusher"
)

var log = core.Log.WithName("dubbo").WithName("server").WithName("metadata")

const queueSize = 100

type MetadataServer struct {
	mesh_proto.MetadataServiceServer

	queue  chan *RegisterRequest
	pusher pusher.Pusher

	ctx             context.Context
	resourceManager manager.ResourceManager
	transactions    core_store.Transactions
}

func (s *MetadataServer) Start(stop <-chan struct{}) error {
	return nil
}

func (s *MetadataServer) NeedLeaderElection() bool {
	return false
}

func NewMetadataServe(
	ctx context.Context,
	pusher pusher.Pusher,
	resourceManager manager.ResourceManager,
	transactions core_store.Transactions,
) *MetadataServer {
	return &MetadataServer{
		pusher:          pusher,
		ctx:             ctx,
		resourceManager: resourceManager,
		transactions:    transactions,
	}
}

func (m *MetadataServer) MetadataRegister(ctx context.Context, req *mesh_proto.MetaDataRegisterRequest) (*mesh_proto.MetaDataRegisterResponse, error) {
	return &mesh_proto.MetaDataRegisterResponse{
		Success: false,
		Message: "success",
	}, nil
}

func (m MetadataServer) MetadataSync(stream mesh_proto.MetadataService_MetadataSyncServer) error {
	return nil
}

func (s *MetadataServer) debounce(stopCh <-chan struct{}, pushFn func(m *RegisterRequest)) {}
