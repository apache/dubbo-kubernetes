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

package client

import (
	"sync"
)

import (
	"github.com/google/uuid"

	"github.com/pkg/errors"
)

import (
	mesh_proto "github.com/apache/dubbo-kubernetes/api/mesh/v1alpha1"
	core_mesh "github.com/apache/dubbo-kubernetes/pkg/core/resources/apis/mesh"
)

var _ MappingSyncStream = &stream{}

type stream struct {
	streamClient mesh_proto.ServiceNameMappingService_MappingSyncServer

	// subscribedInterfaceNames records request's interfaceName in Mapping Request from data plane.
	subscribedInterfaceNames map[string]struct{}
	lastNonce                string
	mu                       sync.RWMutex
}

func NewMappingSyncStream(streamClient mesh_proto.ServiceNameMappingService_MappingSyncServer) MappingSyncStream {
	return &stream{
		streamClient: streamClient,

		subscribedInterfaceNames: make(map[string]struct{}),
	}
}

type MappingSyncStream interface {
	Recv() (*mesh_proto.MappingSyncRequest, error)
	Send(mappingList *core_mesh.MappingResourceList, revision int64) error
	SubscribedInterfaceNames() []string
}

func (s *stream) Recv() (*mesh_proto.MappingSyncRequest, error) {
	request, err := s.streamClient.Recv()
	if err != nil {
		return nil, err
	}

	if s.lastNonce != "" && s.lastNonce != request.GetNonce() {
		return nil, errors.New("request's nonce is different to last nonce")
	}

	// subscribe Mapping
	s.mu.Lock()
	interfaceName := request.GetInterfaceName()
	s.subscribedInterfaceNames[interfaceName] = struct{}{}
	s.mu.Lock()

	return request, nil
}

func (s *stream) Send(mappingList *core_mesh.MappingResourceList, revision int64) error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	nonce := uuid.NewString()
	mappings := make([]*mesh_proto.Mapping, 0, len(mappingList.Items))
	for _, item := range mappingList.Items {
		mappings = append(mappings, &mesh_proto.Mapping{
			Zone:             item.Spec.Zone,
			InterfaceName:    item.Spec.InterfaceName,
			ApplicationNames: item.Spec.ApplicationNames,
		})
	}

	s.lastNonce = nonce
	response := &mesh_proto.MappingSyncResponse{
		Nonce:    nonce,
		Revision: revision,
		Mappings: mappings,
	}
	return s.streamClient.Send(response)
}

func (s *stream) SubscribedInterfaceNames() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]string, 0, len(s.subscribedInterfaceNames))
	for interfaceName := range s.subscribedInterfaceNames {
		result = append(result, interfaceName)
	}

	return result
}
