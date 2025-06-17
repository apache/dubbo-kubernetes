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

package generator

import (
	"context"
	"sort"
)

import (
	envoy_listener_v3 "github.com/envoyproxy/go-control-plane/envoy/config/listener/v3"

	"golang.org/x/exp/maps"
)

import (
	mesh_proto "github.com/apache/dubbo-kubernetes/api/mesh/v1alpha1"
	"github.com/apache/dubbo-kubernetes/pkg/core"
	core_xds "github.com/apache/dubbo-kubernetes/pkg/core/xds"
	model "github.com/apache/dubbo-kubernetes/pkg/core/xds"
	xds_context "github.com/apache/dubbo-kubernetes/pkg/xds/context"
	envoy_listeners "github.com/apache/dubbo-kubernetes/pkg/xds/envoy/listeners"
	"github.com/apache/dubbo-kubernetes/pkg/xds/generator/zoneproxy"
)

var ingressLog = core.Log.WithName("ingress-proxy-generator")

// Ingress is a marker to indicate by which ProxyGenerator resources were generated.
const Ingress = "ingress"

type IngressGenerator struct{}

func (g IngressGenerator) Generator(ctx context.Context, _ *model.ResourceSet, xdsCtx xds_context.Context, proxy *model.Proxy) (*model.ResourceSet, error) {
	resources := core_xds.NewResourceSet()

	networking := proxy.ZoneIngressProxy.ZoneIngressResource.Spec.GetNetworking()
	address, port := networking.GetAddress(), networking.GetPort()
	listenerBuilder := envoy_listeners.NewInboundListenerBuilder(proxy.APIVersion, address, port, core_xds.SocketAddressProtocolTCP).
		Configure(envoy_listeners.TLSInspector())

	availableSvcsByMesh := map[string][]*mesh_proto.ZoneIngress_AvailableService{}
	for _, service := range proxy.ZoneIngressProxy.ZoneIngressResource.Spec.AvailableServices {
		availableSvcsByMesh[service.Mesh] = append(availableSvcsByMesh[service.Mesh], service)
	}

	for _, mr := range proxy.ZoneIngressProxy.MeshResourceList {
		meshName := mr.Mesh.GetMeta().GetName()
		serviceList := maps.Keys(mr.EndpointMap)
		sort.Strings(serviceList)
		dest := zoneproxy.BuildMeshDestinations(
			availableSvcsByMesh[meshName],
			xds_context.Resources{MeshLocalResources: mr.Resources},
		)

		services := zoneproxy.AddFilterChains(availableSvcsByMesh[meshName], proxy.APIVersion, listenerBuilder, dest, mr.EndpointMap)

		cdsResources, err := zoneproxy.GenerateCDS(dest, services, proxy.APIVersion, meshName, Ingress)
		if err != nil {
			return nil, err
		}
		resources.Add(cdsResources...)

		edsResources, err := zoneproxy.GenerateEDS(services, mr.EndpointMap, proxy.APIVersion, meshName, Ingress)
		if err != nil {
			return nil, err
		}
		resources.Add(edsResources...)
	}

	listener, err := listenerBuilder.Build()
	if err != nil {
		return nil, err
	}
	if len(listener.(*envoy_listener_v3.Listener).FilterChains) > 0 {
		resources.Add(&core_xds.Resource{
			Name:     listener.GetName(),
			Origin:   Ingress,
			Resource: listener,
		})
	}

	return resources, nil
}
