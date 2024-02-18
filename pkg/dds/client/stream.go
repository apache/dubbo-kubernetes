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
	"fmt"
	"strings"
)

import (
	envoy_core "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	envoy_sd "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v3"

	"google.golang.org/genproto/googleapis/rpc/status"

	"google.golang.org/protobuf/types/known/structpb"
)

import (
	core_model "github.com/apache/dubbo-kubernetes/pkg/core/resources/model"
	core_runtime "github.com/apache/dubbo-kubernetes/pkg/core/runtime"
	"github.com/apache/dubbo-kubernetes/pkg/dds"
	"github.com/apache/dubbo-kubernetes/pkg/dds/util"
)

var _ DeltaDDSStream = &stream{}

type latestReceived struct {
	nonce         string
	nameToVersion util.NameToVersion
}

type stream struct {
	streamClient   DDSSyncServiceStream
	latestACKed    map[core_model.ResourceType]string
	latestReceived map[core_model.ResourceType]*latestReceived
	clientId       string
	cpConfig       string
	runtimeInfo    core_runtime.RuntimeInfo
}

type DDSSyncServiceStream interface {
	Send(*envoy_sd.DeltaDiscoveryRequest) error
	Recv() (*envoy_sd.DeltaDiscoveryResponse, error)
}

func NewDeltaDDSStream(s DDSSyncServiceStream, clientId string, runtimeInfo core_runtime.RuntimeInfo, cpConfig string) DeltaDDSStream {
	return &stream{
		streamClient:   s,
		runtimeInfo:    runtimeInfo,
		latestACKed:    make(map[core_model.ResourceType]string),
		latestReceived: make(map[core_model.ResourceType]*latestReceived),
		clientId:       clientId,
		cpConfig:       cpConfig,
	}
}

func (s *stream) DeltaDiscoveryRequest(resourceType core_model.ResourceType) error {

	req := &envoy_sd.DeltaDiscoveryRequest{
		ResponseNonce: "",
		Node: &envoy_core.Node{
			Id: s.clientId,
			Metadata: &structpb.Struct{
				Fields: map[string]*structpb.Value{
					dds.MetadataFieldConfig:    {Kind: &structpb.Value_StringValue{StringValue: s.cpConfig}},
					dds.MetadataControlPlaneId: {Kind: &structpb.Value_StringValue{StringValue: s.runtimeInfo.GetInstanceId()}},
					dds.MetadataFeatures: {Kind: &structpb.Value_ListValue{ListValue: &structpb.ListValue{
						Values: []*structpb.Value{
							{Kind: &structpb.Value_StringValue{StringValue: dds.FeatureZoneToken}},
							{Kind: &structpb.Value_StringValue{StringValue: dds.FeatureHashSuffix}},
						},
					}}},
				},
			},
		},
		ResourceNamesSubscribe: []string{"*"},
		TypeUrl:                string(resourceType),
	}
	return s.streamClient.Send(req)
}

func (s *stream) Receive() (UpstreamResponse, error) {
	resp, err := s.streamClient.Recv()
	if err != nil {
		return UpstreamResponse{}, err
	}
	rs, nameToVersion, err := util.ToDeltaCoreResourceList(resp)
	if err != nil {
		return UpstreamResponse{}, err
	}
	// when there isn't nonce it means it's the first request
	isInitialRequest := true
	if _, found := s.latestACKed[rs.GetItemType()]; found {
		isInitialRequest = false
	}
	s.latestReceived[rs.GetItemType()] = &latestReceived{
		nonce:         resp.Nonce,
		nameToVersion: nameToVersion,
	}
	return UpstreamResponse{
		ControlPlaneId:      resp.GetControlPlane().GetIdentifier(),
		Type:                rs.GetItemType(),
		AddedResources:      rs,
		RemovedResourcesKey: s.mapRemovedResources(resp.RemovedResources),
		IsInitialRequest:    isInitialRequest,
	}, nil
}

func (s *stream) ACK(resourceType core_model.ResourceType) error {
	latestReceived := s.latestReceived[resourceType]
	if latestReceived == nil {
		return nil
	}
	err := s.streamClient.Send(&envoy_sd.DeltaDiscoveryRequest{
		ResponseNonce: latestReceived.nonce,
		Node: &envoy_core.Node{
			Id: s.clientId,
		},
		TypeUrl: string(resourceType),
	})
	if err == nil {
		s.latestACKed[resourceType] = latestReceived.nonce
	}
	return err
}

func (s *stream) NACK(resourceType core_model.ResourceType, err error) error {
	latestReceived, found := s.latestReceived[resourceType]
	if !found {
		return nil
	}
	return s.streamClient.Send(&envoy_sd.DeltaDiscoveryRequest{
		ResponseNonce:          latestReceived.nonce,
		ResourceNamesSubscribe: []string{"*"},
		TypeUrl:                string(resourceType),
		Node: &envoy_core.Node{
			Id: s.clientId,
		},
		ErrorDetail: &status.Status{
			Message: fmt.Sprintf("%s", err),
		},
	})
}

// go-contro-plane cache keeps them as a <resource_name>.<mesh_name>
func (s *stream) mapRemovedResources(removedResourceNames []string) []core_model.ResourceKey {
	removed := []core_model.ResourceKey{}
	for _, resourceName := range removedResourceNames {
		index := strings.LastIndex(resourceName, ".")
		var rk core_model.ResourceKey
		if index != -1 {
			rk = core_model.WithMesh(resourceName[index+1:], resourceName[:index])
		} else {
			rk = core_model.WithoutMesh(resourceName)
		}
		removed = append(removed, rk)
	}
	return removed
}
