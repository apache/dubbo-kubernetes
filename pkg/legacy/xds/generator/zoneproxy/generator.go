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

package zoneproxy

import (
	mesh_proto "github.com/apache/dubbo-kubernetes/api/mesh/v1alpha1"
	core_xds "github.com/apache/dubbo-kubernetes/pkg/core/xds"
	envoy_common "github.com/apache/dubbo-kubernetes/pkg/xds/envoy"
	envoy_clusters "github.com/apache/dubbo-kubernetes/pkg/xds/envoy/clusters"
	envoy_endpoints "github.com/apache/dubbo-kubernetes/pkg/xds/envoy/endpoints"
	envoy_listeners "github.com/apache/dubbo-kubernetes/pkg/xds/envoy/listeners"
	envoy_names "github.com/apache/dubbo-kubernetes/pkg/xds/envoy/names"
	envoy_tags "github.com/apache/dubbo-kubernetes/pkg/xds/envoy/tags"
)

func GenerateCDS(
	destinationsPerService map[string][]envoy_tags.Tags,
	services envoy_common.Services,
	apiVersion core_xds.APIVersion,
	meshName string,
	origin string,
) ([]*core_xds.Resource, error) {
	matchAllDestinations := destinationsPerService[mesh_proto.MatchAllTag]

	var resources []*core_xds.Resource
	for _, service := range services.Sorted() {
		clusters := services[service]

		var tagsSlice envoy_tags.TagsSlice
		for _, cluster := range clusters.Clusters() {
			tagsSlice = append(tagsSlice, cluster.Tags())
		}
		tagSlice := append(tagsSlice, matchAllDestinations...)

		tagKeySlice := tagSlice.ToTagKeysSlice().Transform(
			envoy_tags.Without(mesh_proto.ServiceTag),
		)

		clusterName := envoy_names.GetMeshClusterName(meshName, service)
		edsCluster, err := envoy_clusters.NewClusterBuilder(apiVersion, clusterName).
			Configure(envoy_clusters.EdsCluster()).
			Configure(envoy_clusters.LbSubset(tagKeySlice)).
			Build()
		if err != nil {
			return nil, err
		}
		resources = append(resources, &core_xds.Resource{
			Name:     clusterName,
			Origin:   origin,
			Resource: edsCluster,
		})
	}

	return resources, nil
}

func GenerateEDS(
	services envoy_common.Services,
	endpointMap core_xds.EndpointMap,
	apiVersion core_xds.APIVersion,
	meshName string,
	origin string,
) ([]*core_xds.Resource, error) {
	var resources []*core_xds.Resource

	for _, service := range services.Sorted() {
		endpoints := endpointMap[service]

		clusterName := envoy_names.GetMeshClusterName(meshName, service)
		cla, err := envoy_endpoints.CreateClusterLoadAssignment(clusterName, endpoints, apiVersion)
		if err != nil {
			return nil, err
		}
		resources = append(resources, &core_xds.Resource{
			Name:     clusterName,
			Origin:   origin,
			Resource: cla,
		})
	}

	return resources, nil
}

// AddFilterChains adds filter chains to a listener. We generate
// FilterChainsMatcher for each unique destination. This approach has
// a limitation: additional tags on outbound in Universal mode won't work across
// different zones.
func AddFilterChains(
	availableServices []*mesh_proto.ZoneIngress_AvailableService,
	apiVersion core_xds.APIVersion,
	listenerBuilder *envoy_listeners.ListenerBuilder,
	destinationsPerService map[string][]envoy_tags.Tags,
	endpointMap core_xds.EndpointMap,
) envoy_common.Services {
	servicesAcc := envoy_common.NewServicesAccumulator(nil)

	for _, service := range availableServices {
		serviceName := service.Tags[mesh_proto.ServiceTag]
		destinations := destinationsPerService[serviceName]
		destinations = append(destinations, destinationsPerService[mesh_proto.MatchAllTag]...)
		clusterName := envoy_names.GetMeshClusterName(service.Mesh, serviceName)
		serviceEndpoints := endpointMap[serviceName]

		for _, destination := range destinations {

			// relevantTags is a set of tags for which it actually makes sense to do LB split on.
			// If the endpoint list is the same with or without the tag, we should just not do the split.
			// This solves the problem that Envoy deduplicate endpoints of the same address and different metadata.
			// example 1:
			// Ingress1 (10.0.0.1) supports service:a,version:1 and service:a,version:2
			// Ingress2 (10.0.0.2) supports service:a,version:1 and service:a,version:2
			// If we want to split by version, we don't need to do LB subset on version.
			//
			// example 2:
			// Ingress1 (10.0.0.1) supports service:a,version:1
			// Ingress2 (10.0.0.2) supports service:a,version:2
			// If we want to split by version, we need LB subset.
			relevantTags := envoy_tags.Tags{}
			for key, value := range destination {
				matchedTargets := map[string]struct{}{}
				allTargets := map[string]struct{}{}
				for _, endpoint := range serviceEndpoints {
					address := endpoint.Address()
					if endpoint.Tags[key] == value || value == mesh_proto.MatchAllTag {
						matchedTargets[address] = struct{}{}
					}
					allTargets[address] = struct{}{}
				}
				if len(matchedTargets) < len(allTargets) {
					relevantTags[key] = value
				}
			}

			cluster := envoy_common.NewCluster(
				envoy_common.WithName(clusterName),
				envoy_common.WithService(serviceName),
				envoy_common.WithTags(relevantTags),
			)
			cluster.SetMesh(service.Mesh)

			filterChain := envoy_listeners.FilterChain(
				envoy_listeners.NewFilterChainBuilder(apiVersion, envoy_common.AnonymousResource).Configure(
					envoy_listeners.TcpProxyDeprecatedWithMetadata(
						clusterName,
						cluster,
					),
				),
			)

			listenerBuilder.Configure(filterChain)

			servicesAcc.Add(cluster)
		}
	}

	return servicesAcc.Services()
}
