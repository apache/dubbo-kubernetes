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

package service

import (
	"context"
	"time"
)

import (
	"github.com/pkg/errors"
)

import (
	mesh_proto "github.com/apache/dubbo-kubernetes/api/mesh/v1alpha1"
	core_manager "github.com/apache/dubbo-kubernetes/pkg/core/resources/manager"
	core_model "github.com/apache/dubbo-kubernetes/pkg/core/resources/model"
	"github.com/apache/dubbo-kubernetes/pkg/core/resources/registry"
	core_store "github.com/apache/dubbo-kubernetes/pkg/core/resources/store"
)

type EnvoyAdminProcessor interface {
	StartProcessingXDSConfigs(stream mesh_proto.GlobalDDSService_StreamXDSConfigsClient, errorCh chan error)
	StartProcessingStats(stream mesh_proto.GlobalDDSService_StreamStatsClient, errorCh chan error)
	StartProcessingClusters(stream mesh_proto.GlobalDDSService_StreamClustersClient, errorCh chan error)
}

type EnvoyAdminFn = func(ctx context.Context, proxy core_model.ResourceWithAddress) ([]byte, error)

type envoyAdminProcessor struct {
	resManager core_manager.ReadOnlyResourceManager

	configDumpFn EnvoyAdminFn
	statsFn      EnvoyAdminFn
	clustersFn   EnvoyAdminFn
}

func (e *envoyAdminProcessor) StartProcessingXDSConfigs(stream mesh_proto.GlobalDDSService_StreamXDSConfigsClient, errorCh chan error) {
	for {
		req, err := stream.Recv()
		if err != nil {
			errorCh <- err
			return
		}
		go func() { // schedule in the background to be able to quickly process more requests
			config, err := e.executeAdminFn(stream.Context(), req.ResourceType, req.ResourceName, req.ResourceMesh, e.configDumpFn)

			resp := &mesh_proto.XDSConfigResponse{
				RequestId: req.RequestId,
			}
			if len(config) > 0 {
				resp.Result = &mesh_proto.XDSConfigResponse_Config{
					Config: config,
				}
			}
			if err != nil { // send the error to the client instead of terminating stream.
				resp.Result = &mesh_proto.XDSConfigResponse_Error{
					Error: err.Error(),
				}
			}
			if err := stream.Send(resp); err != nil {
				errorCh <- err
				return
			}
		}()
	}
}

func (e *envoyAdminProcessor) StartProcessingStats(stream mesh_proto.GlobalDDSService_StreamStatsClient, errorCh chan error) {
	for {
		req, err := stream.Recv()
		if err != nil {
			errorCh <- err
			return
		}
		go func() { // schedule in the background to be able to quickly process more requests
			stats, err := e.executeAdminFn(stream.Context(), req.ResourceType, req.ResourceName, req.ResourceMesh, e.statsFn)

			resp := &mesh_proto.StatsResponse{
				RequestId: req.RequestId,
			}
			if len(stats) > 0 {
				resp.Result = &mesh_proto.StatsResponse_Stats{
					Stats: stats,
				}
			}
			if err != nil { // send the error to the client instead of terminating stream.
				resp.Result = &mesh_proto.StatsResponse_Error{
					Error: err.Error(),
				}
			}
			if err := stream.Send(resp); err != nil {
				errorCh <- err
				return
			}
		}()
	}
}

func (e *envoyAdminProcessor) StartProcessingClusters(stream mesh_proto.GlobalDDSService_StreamClustersClient, errorCh chan error) {
	for {
		req, err := stream.Recv()
		if err != nil {
			errorCh <- err
			return
		}
		go func() { // schedule in the background to be able to quickly process more requests
			clusters, err := e.executeAdminFn(stream.Context(), req.ResourceType, req.ResourceName, req.ResourceMesh, e.clustersFn)

			resp := &mesh_proto.ClustersResponse{
				RequestId: req.RequestId,
			}
			if len(clusters) > 0 {
				resp.Result = &mesh_proto.ClustersResponse_Clusters{
					Clusters: clusters,
				}
			}
			if err != nil { // send the error to the client instead of terminating stream.
				resp.Result = &mesh_proto.ClustersResponse_Error{
					Error: err.Error(),
				}
			}
			if err := stream.Send(resp); err != nil {
				errorCh <- err
				return
			}
		}()
	}
}

func (s *envoyAdminProcessor) executeAdminFn(
	ctx context.Context,
	resType string,
	resName string,
	resMesh string,
	adminFn EnvoyAdminFn,
) ([]byte, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	res, err := registry.Global().NewObject(core_model.ResourceType(resType))
	if err != nil {
		return nil, err
	}
	if err := s.resManager.Get(ctx, res, core_store.GetByKey(resName, resMesh)); err != nil {
		return nil, err
	}

	resWithAddr, ok := res.(core_model.ResourceWithAddress)
	if !ok {
		return nil, errors.Errorf("invalid type %T", resWithAddr)
	}

	return adminFn(ctx, resWithAddr)
}

var _ EnvoyAdminProcessor = &envoyAdminProcessor{}

func NewEnvoyAdminProcessor(
	resManager core_manager.ReadOnlyResourceManager,
	configDumpFn EnvoyAdminFn,
	statsFn EnvoyAdminFn,
	clustersFn EnvoyAdminFn,
) EnvoyAdminProcessor {
	return &envoyAdminProcessor{
		resManager:   resManager,
		configDumpFn: configDumpFn,
		statsFn:      statsFn,
		clustersFn:   clustersFn,
	}
}
