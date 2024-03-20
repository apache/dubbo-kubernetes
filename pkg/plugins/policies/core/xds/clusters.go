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

package xds

import (
	envoy_cluster "github.com/envoyproxy/go-control-plane/envoy/config/cluster/v3"
	envoy_resource "github.com/envoyproxy/go-control-plane/pkg/resource/v3"
)

import (
	mesh_proto "github.com/apache/dubbo-kubernetes/api/mesh/v1alpha1"
	core_xds "github.com/apache/dubbo-kubernetes/pkg/core/xds"
	"github.com/apache/dubbo-kubernetes/pkg/xds/envoy/tags"
	"github.com/apache/dubbo-kubernetes/pkg/xds/generator"
)

type Clusters struct {
	Inbound       map[string]*envoy_cluster.Cluster
	Outbound      map[string]*envoy_cluster.Cluster
	OutboundSplit map[string][]*envoy_cluster.Cluster
}

func GatherClusters(rs *core_xds.ResourceSet) Clusters {
	clusters := Clusters{
		Inbound:       map[string]*envoy_cluster.Cluster{},
		Outbound:      map[string]*envoy_cluster.Cluster{},
		OutboundSplit: map[string][]*envoy_cluster.Cluster{},
	}
	for _, res := range rs.Resources(envoy_resource.ClusterType) {
		cluster := res.Resource.(*envoy_cluster.Cluster)

		switch res.Origin {
		case generator.OriginOutbound:
			serviceName := tags.ServiceFromClusterName(cluster.Name)
			if serviceName != cluster.Name {
				// first group is service name and second split number
				clusters.OutboundSplit[serviceName] = append(clusters.OutboundSplit[serviceName], cluster)
			} else {
				clusters.Outbound[cluster.Name] = cluster
			}
		case generator.OriginInbound:
			clusters.Inbound[cluster.Name] = cluster
		default:
			continue
		}
	}
	return clusters
}

func GatherTargetedClusters(
	outbounds []*mesh_proto.Dataplane_Networking_Outbound,
	outboundSplitClusters map[string][]*envoy_cluster.Cluster,
	outboundClusters map[string]*envoy_cluster.Cluster,
) map[*envoy_cluster.Cluster]string {
	targetedClusters := map[*envoy_cluster.Cluster]string{}
	for _, outbound := range outbounds {
		serviceName := outbound.GetService()
		for _, splitCluster := range outboundSplitClusters[serviceName] {
			targetedClusters[splitCluster] = serviceName
		}

		cluster, ok := outboundClusters[serviceName]
		if ok {
			targetedClusters[cluster] = serviceName
		}
	}

	return targetedClusters
}
