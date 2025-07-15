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

	"google.golang.org/grpc"

	"google.golang.org/protobuf/proto"
)

import (
	mesh_proto "github.com/apache/dubbo-kubernetes/api/mesh/v1alpha1"
	core_mesh "github.com/apache/dubbo-kubernetes/pkg/core/resources/apis/mesh"
	core_model "github.com/apache/dubbo-kubernetes/pkg/core/resources/model"
)

var _ DubboSyncStream = &stream{}

type stream struct {
	streamClient grpc.ServerStream

	// subscribedInterfaceNames records request's interfaceName in MappingSync Request from data plane.
	subscribedInterfaceNames map[string]struct{}
	// subscribedApplicationNames records request's applicationName in MetaDataSync Request from data plane.
	subscribedApplicationNames map[string]struct{}

	mappingLastNonce  string
	metadataLastNonce string
	mu                sync.RWMutex
}

func NewDubboSyncStream(streamClient grpc.ServerStream) DubboSyncStream {
	return &stream{
		streamClient: streamClient,

		subscribedInterfaceNames:   make(map[string]struct{}),
		subscribedApplicationNames: make(map[string]struct{}),
	}
}

type DubboSyncStream interface {
	Recv() (proto.Message, error)
	Send(resourceList core_model.ResourceList, revision int64) error
	SubscribedInterfaceNames() []string
	SubscribedApplicationNames() []string
}

func (s *stream) Recv() (proto.Message, error) {
	switch s.streamClient.(type) {
	case mesh_proto.MDSSyncService_MappingSyncServer:
		request := &mesh_proto.MappingSyncRequest{}
		err := s.streamClient.RecvMsg(request)
		if err != nil {
			return nil, err
		}
		if s.mappingLastNonce != "" && s.mappingLastNonce != request.GetNonce() {
			return nil, errors.New("mapping sync request's nonce is different to last nonce")
		}

		// subscribe Mapping
		s.mu.Lock()
		interfaceName := request.GetInterfaceName()
		s.subscribedInterfaceNames[interfaceName] = struct{}{}
		s.mu.Lock()

		return request, nil
	case mesh_proto.MDSSyncService_MetadataSyncServer:
		request := &mesh_proto.MetadataSyncRequest{}
		err := s.streamClient.RecvMsg(request)
		if err != nil {
			return nil, err
		}
		if s.metadataLastNonce != "" && s.metadataLastNonce != request.GetNonce() {
			return nil, errors.New("metadata sync request's nonce is different to last nonce")
		}

		// subscribe MetaData
		s.mu.Lock()
		appName := request.GetApplicationName()
		s.subscribedApplicationNames[appName] = struct{}{}
		s.mu.Lock()

		return request, nil
	default:
		return nil, errors.New("unknown type request")
	}
}

func (s *stream) Send(resourceList core_model.ResourceList, revision int64) error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	nonce := uuid.NewString()

	switch resourceList.(type) {
	case *core_mesh.MappingResourceList:
		mappingList := resourceList.(*core_mesh.MappingResourceList)
		mappings := make([]*mesh_proto.Mapping, 0, len(mappingList.Items))
		for _, item := range mappingList.Items {
			mappings = append(mappings, &mesh_proto.Mapping{
				Zone:             item.Spec.Zone,
				InterfaceName:    item.Spec.InterfaceName,
				ApplicationNames: item.Spec.ApplicationNames,
			})
		}

		s.mappingLastNonce = nonce
		response := &mesh_proto.MappingSyncResponse{
			Nonce:    nonce,
			Revision: revision,
			Mappings: mappings,
		}
		return s.streamClient.SendMsg(response)
	case *core_mesh.MetaDataResourceList:
		metadataList := resourceList.(*core_mesh.MetaDataResourceList)
		metaDatum := make([]*mesh_proto.MetaData, 0, len(metadataList.Items))
		for _, item := range metadataList.Items {
			metaDatum = append(metaDatum, &mesh_proto.MetaData{
				App:      item.Spec.GetApp(),
				Revision: item.Spec.Revision,
				Services: item.Spec.GetServices(),
			})
		}

		s.metadataLastNonce = nonce
		response := &mesh_proto.MetadataSyncResponse{
			Nonce:     nonce,
			Revision:  revision,
			MetaDatum: metaDatum,
		}
		return s.streamClient.SendMsg(response)
	default:
		return errors.New("unknown type request")
	}
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

func (s *stream) SubscribedApplicationNames() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]string, 0, len(s.subscribedApplicationNames))
	for appName := range s.subscribedApplicationNames {
		result = append(result, appName)
	}

	return result
}
