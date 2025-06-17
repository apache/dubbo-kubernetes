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

package sync

import (
	"context"
)

import (
	"github.com/apache/dubbo-kubernetes/pkg/core/dns/lookup"
	core_mesh "github.com/apache/dubbo-kubernetes/pkg/core/resources/apis/mesh"
	"github.com/apache/dubbo-kubernetes/pkg/core/resources/manager"
	core_model "github.com/apache/dubbo-kubernetes/pkg/core/resources/model"
	core_store "github.com/apache/dubbo-kubernetes/pkg/core/resources/store"
	core_xds "github.com/apache/dubbo-kubernetes/pkg/core/xds"
	xds_context "github.com/apache/dubbo-kubernetes/pkg/xds/context"
	"github.com/apache/dubbo-kubernetes/pkg/xds/ingress"
	xds_topology "github.com/apache/dubbo-kubernetes/pkg/xds/topology"
)

type IngressProxyBuilder struct {
	ResManager manager.ResourceManager
	LookupIP   lookup.LookupIPFunc

	apiVersion        core_xds.APIVersion
	zone              string
	ingressTagFilters []string
}

func (p *IngressProxyBuilder) Build(
	ctx context.Context,
	key core_model.ResourceKey,
	aggregatedMeshCtxs xds_context.AggregatedMeshContexts,
) (*core_xds.Proxy, error) {
	zoneIngress, err := p.getZoneIngress(ctx, key, aggregatedMeshCtxs)
	if err != nil {
		return nil, err
	}

	zoneIngress, err = xds_topology.ResolveZoneIngressPublicAddress(p.LookupIP, zoneIngress)
	if err != nil {
		return nil, err
	}

	proxy := &core_xds.Proxy{
		Id:               core_xds.FromResourceKey(key),
		APIVersion:       p.apiVersion,
		Zone:             p.zone,
		ZoneIngressProxy: p.buildZoneIngressProxy(zoneIngress, aggregatedMeshCtxs),
	}

	return proxy, nil
}

func (p *IngressProxyBuilder) buildZoneIngressProxy(
	zoneIngress *core_mesh.ZoneIngressResource,
	aggregatedMeshCtxs xds_context.AggregatedMeshContexts,
) *core_xds.ZoneIngressProxy {
	var meshResourceList []*core_xds.MeshIngressResources

	for _, mesh := range aggregatedMeshCtxs.Meshes {
		meshName := mesh.GetMeta().GetName()
		meshCtx := aggregatedMeshCtxs.MustGetMeshContext(meshName)

		meshResources := &core_xds.MeshIngressResources{
			Mesh: mesh,
			EndpointMap: ingress.BuildEndpointMap(
				ingress.BuildDestinationMap(meshName, zoneIngress),
				meshCtx.Resources.Dataplanes().Items,
			),
			Resources: meshCtx.Resources.MeshLocalResources,
		}

		meshResourceList = append(meshResourceList, meshResources)
	}

	return &core_xds.ZoneIngressProxy{
		ZoneIngressResource: zoneIngress,
		MeshResourceList:    meshResourceList,
	}
}

func (p *IngressProxyBuilder) getZoneIngress(
	ctx context.Context,
	key core_model.ResourceKey,
	aggregatedMeshCtxs xds_context.AggregatedMeshContexts,
) (*core_mesh.ZoneIngressResource, error) {
	zoneIngress := core_mesh.NewZoneIngressResource()
	if err := p.ResManager.Get(ctx, zoneIngress, core_store.GetBy(key)); err != nil {
		return nil, err
	}
	// Update Ingress' Available Services
	// This was placed as an operation of DataplaneWatchdog out of the convenience.
	// Consider moving to the outside of this component
	if err := p.updateIngress(ctx, zoneIngress, aggregatedMeshCtxs); err != nil {
		return nil, err
	}
	return zoneIngress, nil
}

func (p *IngressProxyBuilder) updateIngress(
	ctx context.Context, zoneIngress *core_mesh.ZoneIngressResource,
	aggregatedMeshCtxs xds_context.AggregatedMeshContexts,
) error {
	// Update Ingress' Available Services
	// This was placed as an operation of DataplaneWatchdog out of the convenience.
	// Consider moving to the outside of this component (follow the pattern of updating VIP outbounds)
	return ingress.UpdateAvailableServices(
		ctx,
		p.ResManager,
		zoneIngress,
		aggregatedMeshCtxs.AllDataplanes(),
		p.ingressTagFilters,
	)
}
