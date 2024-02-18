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

package dubbo

import (
	"context"
	"fmt"
	"reflect"
)

import (
	"github.com/pkg/errors"
)

import (
	mesh_proto "github.com/apache/dubbo-kubernetes/api/mesh/v1alpha1"
	"github.com/apache/dubbo-kubernetes/pkg/core"
	core_mesh "github.com/apache/dubbo-kubernetes/pkg/core/resources/apis/mesh"
	core_model "github.com/apache/dubbo-kubernetes/pkg/core/resources/model"
	"github.com/apache/dubbo-kubernetes/pkg/dds/service"
	util_grpc "github.com/apache/dubbo-kubernetes/pkg/util/grpc"
	"github.com/apache/dubbo-kubernetes/pkg/util/k8s"
)

type ddsEnvoyAdminClient struct {
	rpcs     service.EnvoyAdminRPCs
	k8sStore bool
}

func NewDDSEnvoyAdminClient(rpcs service.EnvoyAdminRPCs, k8sStore bool) EnvoyAdminClient {
	return &ddsEnvoyAdminClient{
		rpcs:     rpcs,
		k8sStore: k8sStore,
	}
}

var _ EnvoyAdminClient = &ddsEnvoyAdminClient{}

func (k *ddsEnvoyAdminClient) PostQuit(context.Context, *core_mesh.DataplaneResource) error {
	panic("not implemented")
}

func (k *ddsEnvoyAdminClient) ConfigDump(ctx context.Context, proxy core_model.ResourceWithAddress) ([]byte, error) {
	zone := core_model.ZoneOfResource(proxy)
	nameInZone := resNameInZone(proxy)
	reqId := core.NewUUID()
	tenantZoneID := service.ZoneClientIDFromCtx(ctx, zone)

	err := k.rpcs.XDSConfigDump.Send(tenantZoneID.String(), &mesh_proto.XDSConfigRequest{
		RequestId:    reqId,
		ResourceType: string(proxy.Descriptor().Name),
		ResourceName: nameInZone,                // send the name which without the added prefix
		ResourceMesh: proxy.GetMeta().GetMesh(), // should be empty for ZoneIngress/ZoneEgress
	})
	if err != nil {
		return nil, &DDSTransportError{requestType: "XDSConfigRequest", reason: err.Error()}
	}

	defer k.rpcs.XDSConfigDump.DeleteWatch(tenantZoneID.String(), reqId)
	ch := make(chan util_grpc.ReverseUnaryMessage)
	if err := k.rpcs.XDSConfigDump.WatchResponse(tenantZoneID.String(), reqId, ch); err != nil {
		return nil, errors.Wrapf(err, "could not watch the response")
	}

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case resp := <-ch:
		configResp, ok := resp.(*mesh_proto.XDSConfigResponse)
		if !ok {
			return nil, errors.New("invalid request type")
		}
		if configResp.GetError() != "" {
			return nil, &DDSTransportError{requestType: "XDSConfigRequest", reason: configResp.GetError()}
		}
		return configResp.GetConfig(), nil
	}
}

func (k *ddsEnvoyAdminClient) Stats(ctx context.Context, proxy core_model.ResourceWithAddress) ([]byte, error) {
	zone := core_model.ZoneOfResource(proxy)
	nameInZone := resNameInZone(proxy)
	reqId := core.NewUUID()
	tenantZoneId := service.ZoneClientIDFromCtx(ctx, zone)

	err := k.rpcs.Stats.Send(tenantZoneId.String(), &mesh_proto.StatsRequest{
		RequestId:    reqId,
		ResourceType: string(proxy.Descriptor().Name),
		ResourceName: nameInZone,                // send the name which without the added prefix
		ResourceMesh: proxy.GetMeta().GetMesh(), // should be empty for ZoneIngress/ZoneEgress
	})
	if err != nil {
		return nil, &DDSTransportError{requestType: "StatsRequest", reason: err.Error()}
	}

	defer k.rpcs.Stats.DeleteWatch(tenantZoneId.String(), reqId)
	ch := make(chan util_grpc.ReverseUnaryMessage)
	if err := k.rpcs.Stats.WatchResponse(tenantZoneId.String(), reqId, ch); err != nil {
		return nil, errors.Wrapf(err, "could not watch the response")
	}

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case resp := <-ch:
		statsResp, ok := resp.(*mesh_proto.StatsResponse)
		if !ok {
			return nil, errors.New("invalid request type")
		}
		if statsResp.GetError() != "" {
			return nil, &DDSTransportError{requestType: "StatsRequest", reason: statsResp.GetError()}
		}
		return statsResp.GetStats(), nil
	}
}

func (k *ddsEnvoyAdminClient) Clusters(ctx context.Context, proxy core_model.ResourceWithAddress) ([]byte, error) {
	zone := core_model.ZoneOfResource(proxy)
	nameInZone := resNameInZone(proxy)
	reqId := core.NewUUID()
	tenantZoneID := service.ZoneClientIDFromCtx(ctx, zone)

	err := k.rpcs.Clusters.Send(tenantZoneID.String(), &mesh_proto.ClustersRequest{
		RequestId:    reqId,
		ResourceType: string(proxy.Descriptor().Name),
		ResourceName: nameInZone,                // send the name which without the added prefix
		ResourceMesh: proxy.GetMeta().GetMesh(), // should be empty for ZoneIngress/ZoneEgress
	})
	if err != nil {
		return nil, &DDSTransportError{requestType: "ClustersRequest", reason: err.Error()}
	}

	defer k.rpcs.Clusters.DeleteWatch(tenantZoneID.String(), reqId)
	ch := make(chan util_grpc.ReverseUnaryMessage)
	if err := k.rpcs.Clusters.WatchResponse(tenantZoneID.String(), reqId, ch); err != nil {
		return nil, errors.Wrapf(err, "could not watch the response")
	}

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case resp := <-ch:
		clustersResp, ok := resp.(*mesh_proto.ClustersResponse)
		if !ok {
			return nil, errors.New("invalid request type")
		}
		if clustersResp.GetError() != "" {
			return nil, &DDSTransportError{requestType: "ClustersRequest", reason: clustersResp.GetError()}
		}
		return clustersResp.GetClusters(), nil
	}
}

func resNameInZone(r core_model.Resource) string {
	name := core_model.GetDisplayName(r)
	if ns := r.GetMeta().GetLabels()[mesh_proto.KubeNamespaceTag]; ns != "" {
		name = k8s.K8sNamespacedNameToCoreName(name, ns)
	}
	return name
}

type DDSTransportError struct {
	requestType string
	reason      string
}

func (e *DDSTransportError) Error() string {
	if e.reason == "" {
		return fmt.Sprintf("could not send %s", e.requestType)
	} else {
		return fmt.Sprintf("could not send %s: %s", e.requestType, e.reason)
	}
}

func (e *DDSTransportError) Is(err error) bool {
	return reflect.TypeOf(e) == reflect.TypeOf(err)
}
