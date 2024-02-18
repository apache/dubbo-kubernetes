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

package envoyadmin

import (
	"context"
	"errors"
	mesh_proto "github.com/apache/dubbo-kubernetes/api/mesh/v1alpha1"
	"github.com/apache/dubbo-kubernetes/pkg/core"
	"github.com/apache/dubbo-kubernetes/pkg/core/resources/manager"
	"github.com/apache/dubbo-kubernetes/pkg/core/resources/model"
	"github.com/apache/dubbo-kubernetes/pkg/core/resources/registry"
	core_store "github.com/apache/dubbo-kubernetes/pkg/core/resources/store"
	"github.com/apache/dubbo-kubernetes/pkg/dubbo"
)

var serverLog = core.Log.WithName("intercp").WithName("catalog").WithName("server")

type server struct {
	adminClient dubbo.EnvoyAdminClient
	resManager  manager.ReadOnlyResourceManager
	mesh_proto.UnimplementedInterCPEnvoyAdminForwardServiceServer
}

var _ mesh_proto.InterCPEnvoyAdminForwardServiceServer = &server{}

func NewServer(adminClient dubbo.EnvoyAdminClient, resManager manager.ReadOnlyResourceManager) mesh_proto.InterCPEnvoyAdminForwardServiceServer {
	return &server{
		adminClient: adminClient,
		resManager:  resManager,
	}
}

func (s *server) XDSConfig(ctx context.Context, req *mesh_proto.XDSConfigRequest) (*mesh_proto.XDSConfigResponse, error) {
	serverLog.V(1).Info("received forwarded request", "operation", "XDSConfig", "request", req)
	resWithAddr, err := s.resWithAddress(ctx, req.ResourceType, req.ResourceName, req.ResourceMesh)
	if err != nil {
		return nil, err
	}
	configDump, err := s.adminClient.ConfigDump(ctx, resWithAddr)
	if err != nil {
		return nil, err
	}
	return &mesh_proto.XDSConfigResponse{
		Result: &mesh_proto.XDSConfigResponse_Config{
			Config: configDump,
		},
	}, nil
}

func (s *server) Stats(ctx context.Context, req *mesh_proto.StatsRequest) (*mesh_proto.StatsResponse, error) {
	serverLog.V(1).Info("received forwarded request", "operation", "Stats", "request", req)
	resWithAddr, err := s.resWithAddress(ctx, req.ResourceType, req.ResourceName, req.ResourceMesh)
	if err != nil {
		return nil, err
	}
	stats, err := s.adminClient.Stats(ctx, resWithAddr)
	if err != nil {
		return nil, err
	}
	return &mesh_proto.StatsResponse{
		Result: &mesh_proto.StatsResponse_Stats{
			Stats: stats,
		},
	}, nil
}

func (s *server) Clusters(ctx context.Context, req *mesh_proto.ClustersRequest) (*mesh_proto.ClustersResponse, error) {
	serverLog.V(1).Info("received forwarded request", "operation", "Clusters", "request", req)
	resWithAddr, err := s.resWithAddress(ctx, req.ResourceType, req.ResourceName, req.ResourceMesh)
	if err != nil {
		return nil, err
	}
	clusters, err := s.adminClient.Clusters(ctx, resWithAddr)
	if err != nil {
		return nil, err
	}
	return &mesh_proto.ClustersResponse{
		Result: &mesh_proto.ClustersResponse_Clusters{
			Clusters: clusters,
		},
	}, nil
}

func (s *server) resWithAddress(ctx context.Context, typ, name, mesh string) (model.ResourceWithAddress, error) {
	obj, err := registry.Global().NewObject(model.ResourceType(typ))
	if err != nil {
		return nil, err
	}
	if err := s.resManager.Get(ctx, obj, core_store.GetByKey(name, mesh)); err != nil {
		return nil, err
	}
	resourceWithAddr, ok := obj.(model.ResourceWithAddress)
	if !ok {
		return nil, errors.New("invalid resource type")
	}
	return resourceWithAddr, nil
}
