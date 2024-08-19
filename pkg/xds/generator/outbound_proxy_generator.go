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
	"fmt"
)

import (
	"github.com/pkg/errors"
)

import (
	mesh_proto "github.com/apache/dubbo-kubernetes/api/mesh/v1alpha1"
	"github.com/apache/dubbo-kubernetes/pkg/core"
	core_mesh "github.com/apache/dubbo-kubernetes/pkg/core/resources/apis/mesh"
	"github.com/apache/dubbo-kubernetes/pkg/core/user"
	model "github.com/apache/dubbo-kubernetes/pkg/core/xds"
	"github.com/apache/dubbo-kubernetes/pkg/util/maps"
	util_protocol "github.com/apache/dubbo-kubernetes/pkg/util/protocol"
	xds_context "github.com/apache/dubbo-kubernetes/pkg/xds/context"
	envoy_common "github.com/apache/dubbo-kubernetes/pkg/xds/envoy"
	envoy_clusters "github.com/apache/dubbo-kubernetes/pkg/xds/envoy/clusters"
	envoy_listeners "github.com/apache/dubbo-kubernetes/pkg/xds/envoy/listeners"
	envoy_names "github.com/apache/dubbo-kubernetes/pkg/xds/envoy/names"
	envoy_tags "github.com/apache/dubbo-kubernetes/pkg/xds/envoy/tags"
)

var outboundLog = core.Log.WithName("outbound-proxy-generator")

// OriginOutbound is a marker to indicate by which ProxyGenerator resources were generated.
const OriginOutbound = "outbound"

type OutboundProxyGenerator struct{}

func (g OutboundProxyGenerator) Generator(ctx context.Context, _ *model.ResourceSet, xdsCtx xds_context.Context, proxy *model.Proxy) (*model.ResourceSet, error) {
	outbounds := proxy.Dataplane.Spec.Networking.GetOutbound()
	resources := model.NewResourceSet()
	if len(outbounds) == 0 {
		return resources, nil
	}

	// TODO: implement the logic of tlsReadiness
	tlsReadiness := make(map[string]bool)
	servicesAcc := envoy_common.NewServicesAccumulator(tlsReadiness)

	outboundsMultipleIPs := buildOutboundsWithMultipleIPs(proxy.Dataplane, outbounds)
	for _, outbound := range outboundsMultipleIPs {

		// Determine the list of destination subsets for the given outbound service
		routes := g.determineRoutes(proxy, outbound.Tags)
		clusters := routes.Clusters()

		// Infer the compatible protocol for all the apps for the given service
		protocol := inferProtocol(xdsCtx.Mesh, clusters)

		servicesAcc.Add(clusters...)

		// Generate listener
		listener, err := g.generateLDS(xdsCtx, proxy, routes, outbound, protocol)
		if err != nil {
			return nil, err
		}
		resources.Add(&model.Resource{
			Name:     listener.GetName(),
			Origin:   OriginOutbound,
			Resource: listener,
		})
	}

	services := servicesAcc.Services()

	// Generate clusters. It cannot be generated on the fly with outbound loop because we need to know all subsets of the cluster for every service.
	cdsResources, err := g.generateCDS(xdsCtx, services, proxy)
	if err != nil {
		return nil, err
	}
	resources.AddSet(cdsResources)

	edsResources, err := g.generateEDS(ctx, xdsCtx, services, proxy)
	if err != nil {
		return nil, err
	}
	resources.AddSet(edsResources)
	return resources, nil
}

func (OutboundProxyGenerator) generateLDS(ctx xds_context.Context, proxy *model.Proxy, routes envoy_common.Routes, outbound OutboundWithMultipleIPs, protocol core_mesh.Protocol) (envoy_common.NamedResource, error) {
	oface := outbound.Addresses[0]

	serviceName := outbound.Tags[mesh_proto.ServiceTag]
	outboundListenerName := envoy_names.GetOutboundListenerName(oface.DataplaneIP, oface.DataplanePort)
	filterChainBuilder := func() *envoy_listeners.FilterChainBuilder {
		filterChainBuilder := envoy_listeners.NewFilterChainBuilder(proxy.APIVersion, envoy_common.AnonymousResource)
		switch protocol {
		case core_mesh.ProtocolTriple:
			// TODO: implement the logic of Triple
			// currently, we use the tcp proxy for the triple protocol
			filterChainBuilder.
				Configure(envoy_listeners.TripleConnectionManager()).
				Configure(envoy_listeners.TcpProxyDeprecated(serviceName, routes.Clusters()...))
		case core_mesh.ProtocolGRPC:
			filterChainBuilder.
				Configure(envoy_listeners.HttpConnectionManager(serviceName, false)).
				Configure(envoy_listeners.HttpOutboundRoute(serviceName, routes, proxy.Dataplane.Spec.TagSet())).
				Configure(envoy_listeners.GrpcStats())
		case core_mesh.ProtocolHTTP, core_mesh.ProtocolHTTP2:
			filterChainBuilder.
				Configure(envoy_listeners.HttpConnectionManager(serviceName, false)).
				Configure(envoy_listeners.HttpOutboundRoute(serviceName, routes, proxy.Dataplane.Spec.TagSet()))
		case core_mesh.ProtocolKafka:
			filterChainBuilder.
				Configure(envoy_listeners.Kafka(serviceName)).
				Configure(envoy_listeners.TcpProxyDeprecated(serviceName, routes.Clusters()...))

		case core_mesh.ProtocolTCP:
			fallthrough
		default:
			// configuration for non-HTTP cases
			filterChainBuilder.
				Configure(envoy_listeners.TcpProxyDeprecated(serviceName, routes.Clusters()...))
		}

		return filterChainBuilder
	}()
	listener, err := envoy_listeners.NewOutboundListenerBuilder(proxy.APIVersion, oface.DataplaneIP, oface.DataplanePort, model.SocketAddressProtocolTCP).
		Configure(envoy_listeners.FilterChain(filterChainBuilder)).
		Configure(envoy_listeners.TagsMetadata(envoy_tags.Tags(outbound.Tags).WithoutTags(mesh_proto.MeshTag))).
		Configure(envoy_listeners.AdditionalAddresses(outbound.AdditionalAddresses())).
		Build()
	if err != nil {
		return nil, errors.Wrapf(err, "could not generate listener %s for service %s", outboundListenerName, serviceName)
	}
	return listener, nil
}

func (g OutboundProxyGenerator) generateCDS(ctx xds_context.Context, services envoy_common.Services, proxy *model.Proxy) (*model.ResourceSet, error) {
	resources := model.NewResourceSet()

	for _, serviceName := range services.Sorted() {
		service := services[serviceName]
		protocol := ctx.Mesh.GetServiceProtocol(serviceName)

		for _, c := range service.Clusters() {
			cluster := c.(*envoy_common.ClusterImpl)
			clusterName := cluster.Name()
			edsClusterBuilder := envoy_clusters.NewClusterBuilder(proxy.APIVersion, clusterName)

			// clusterTags := []envoy_tags.Tags{cluster.Tags()}

			if service.HasExternalService() {
				if ctx.Mesh.Resource.ZoneEgressEnabled() {
					edsClusterBuilder.
						Configure(envoy_clusters.EdsCluster())
				} else {
					endpoints := proxy.Routing.ExternalServiceOutboundTargets[serviceName]
					isIPv6 := proxy.Dataplane.IsIPv6()

					edsClusterBuilder.
						Configure(envoy_clusters.ProvidedEndpointCluster(isIPv6, endpoints...))
				}

				switch protocol {
				case core_mesh.ProtocolHTTP:
					edsClusterBuilder.Configure(envoy_clusters.Http())
				case core_mesh.ProtocolHTTP2, core_mesh.ProtocolGRPC:
					edsClusterBuilder.Configure(envoy_clusters.Http2())
				default:
				}
			} else {
				edsClusterBuilder.
					Configure(envoy_clusters.EdsCluster()).
					Configure(envoy_clusters.Http2())
			}

			edsCluster, err := edsClusterBuilder.Build()
			if err != nil {
				return nil, errors.Wrapf(err, "build CDS for cluster %s failed", clusterName)
			}

			resources.Add(&model.Resource{
				Name:     clusterName,
				Origin:   OriginOutbound,
				Resource: edsCluster,
			})
		}
	}

	return resources, nil
}

func (OutboundProxyGenerator) generateEDS(
	ctx context.Context,
	xdsCtx xds_context.Context,
	services envoy_common.Services,
	proxy *model.Proxy,
) (*model.ResourceSet, error) {
	apiVersion := proxy.APIVersion
	resources := model.NewResourceSet()

	for _, serviceName := range services.Sorted() {
		// When no zone egress is present in a mesh Endpoints for ExternalServices
		// are specified in load assignment in DNS Cluster.
		// We are not allowed to add endpoints with DNS names through EDS.
		if !services[serviceName].HasExternalService() || xdsCtx.Mesh.Resource.ZoneEgressEnabled() {
			for _, c := range services[serviceName].Clusters() {
				cluster := c.(*envoy_common.ClusterImpl)
				var endpoints model.EndpointMap
				if cluster.Mesh() != "" {
					// TODO: CrossMeshEndpoints is not implemented yet
				} else {
					endpoints = xdsCtx.Mesh.EndpointMap
				}

				loadAssignment, err := xdsCtx.ControlPlane.CLACache.GetCLA(
					user.Ctx(ctx, user.ControlPlane),
					xdsCtx.Mesh.Resource.Meta.GetName(),
					xdsCtx.Mesh.Hash,
					cluster,
					apiVersion,
					endpoints,
				)
				if err != nil {
					return nil, errors.Wrapf(err, "could not get ClusterLoadAssignment for %s", serviceName)
				}

				resources.Add(&model.Resource{
					Name:     cluster.Name(),
					Origin:   OriginOutbound,
					Resource: loadAssignment,
				})
			}
		}
	}

	return resources, nil
}

func inferProtocol(meshCtx xds_context.MeshContext, clusters []envoy_common.Cluster) core_mesh.Protocol {
	var protocol core_mesh.Protocol = core_mesh.ProtocolUnknown
	for idx, cluster := range clusters {
		serviceName := cluster.Tags()[mesh_proto.ServiceTag]
		serviceProtocol := meshCtx.GetServiceProtocol(serviceName)
		if idx == 0 {
			protocol = serviceProtocol
			continue
		}
		protocol = util_protocol.GetCommonProtocol(serviceProtocol, protocol)
	}
	return protocol
}

func (OutboundProxyGenerator) determineRoutes(
	proxy *model.Proxy,
	outboundTags map[string]string,
) envoy_common.Routes {
	var routes envoy_common.Routes

	type clustersInfo struct {
		info     *mesh_proto.TrafficRoute_Http_Match
		modify   *mesh_proto.TrafficRoute_Http_Modify
		clusters []envoy_common.Cluster
	}

	retriveClusters := func() []clustersInfo {
		var clustersList []clustersInfo
		service := outboundTags[mesh_proto.ServiceTag]

		name, _ := envoy_tags.Tags(outboundTags).DestinationClusterName(nil)

		if mesh, ok := outboundTags[mesh_proto.MeshTag]; ok {
			// The name should be distinct to the service & mesh combination
			name = fmt.Sprintf("%s_%s", name, mesh)
		}

		// We assume that all the targets are either ExternalServices or not
		// therefore we check only the first one
		var isExternalService bool
		if endpoints := proxy.Routing.OutboundTargets[service]; len(endpoints) > 0 {
			isExternalService = endpoints[0].IsExternalService()
		}
		if endpoints := proxy.Routing.ExternalServiceOutboundTargets[service]; len(endpoints) > 0 {
			isExternalService = true
		}
		allTags := envoy_tags.Tags(outboundTags)
		if _, ok := proxy.Routing.OutboundSelector[service]; ok {
			for _, clusterSelectors := range proxy.Routing.OutboundSelector[service] {
				var clusters = make([]envoy_common.Cluster, 0, len(clusterSelectors.EndSelectors))
				for _, endSelector := range clusterSelectors.EndSelectors {
					cluster := envoy_common.NewCluster(
						envoy_common.WithService(service),
						envoy_common.WithName(name),
						envoy_common.WithTags(allTags.WithoutTags(mesh_proto.MeshTag).
							WithTags(envoy_tags.Tags(endSelector.TagSelect).KeyAndValues()...),
						), envoy_common.WithExternalService(isExternalService),
						envoy_common.WithSelector(endSelector),
						envoy_common.WithTimeout(clusterSelectors.ModifyInfo.TimeOut),
					)
					if mesh, ok := outboundTags[mesh_proto.MeshTag]; ok {
						cluster.SetMesh(mesh)
					}
					clusters = append(clusters, cluster)
				}
				clustersList = append(clustersList, clustersInfo{
					modify:   &clusterSelectors.ModifyInfo,
					info:     clusterSelectors.GetMatchInfo(),
					clusters: clusters,
				})
			}
		} else {
			cluster := envoy_common.NewCluster(
				envoy_common.WithService(service),
				envoy_common.WithName(name),
				envoy_common.WithTags(allTags.WithoutTags(mesh_proto.MeshTag)),
				envoy_common.WithExternalService(isExternalService),
			)
			if mesh, ok := outboundTags[mesh_proto.MeshTag]; ok {
				cluster.SetMesh(mesh)
			}
			clustersList = append(clustersList, clustersInfo{
				info:     nil,
				clusters: []envoy_common.Cluster{cluster},
			})
		}
		return clustersList
	}

	appendRoute := func(routes envoy_common.Routes, clustersInfoList []clustersInfo) envoy_common.Routes {
		if len(clustersInfoList) == 0 {
			return routes
		}

		for _, c := range clustersInfoList {
			r := envoy_common.Route{
				Modify:   c.modify,
				Clusters: c.clusters,
				Match:    c.info,
			}
			routes = append(routes, r)
		}
		return routes
	}

	clusters := retriveClusters()
	routes = appendRoute(routes, clusters)

	return routes
}

type OutboundWithMultipleIPs struct {
	Tags      map[string]string
	Addresses []mesh_proto.OutboundInterface
}

func (o OutboundWithMultipleIPs) AdditionalAddresses() []mesh_proto.OutboundInterface {
	if len(o.Addresses) > 1 {
		return o.Addresses[1:]
	}
	return nil
}

func buildOutboundsWithMultipleIPs(dataplane *core_mesh.DataplaneResource, outbounds []*mesh_proto.Dataplane_Networking_Outbound) []OutboundWithMultipleIPs {
	tagsToOutbounds := map[string]OutboundWithMultipleIPs{}
	for _, outbound := range outbounds {
		tags := outbound.GetTags()
		tags[mesh_proto.ServiceTag] = outbound.GetService()
		tagsStr := mesh_proto.SingleValueTagSet(tags).String()
		owmi := tagsToOutbounds[tagsStr]
		owmi.Tags = tags
		address := dataplane.Spec.Networking.ToOutboundInterface(outbound)
		owmi.Addresses = append([]mesh_proto.OutboundInterface{address}, owmi.Addresses...)
		tagsToOutbounds[tagsStr] = owmi
	}

	// return sorted outbounds for a stable XDS config
	var result []OutboundWithMultipleIPs
	for _, key := range maps.SortedKeys(tagsToOutbounds) {
		result = append(result, tagsToOutbounds[key])
	}
	return result
}
