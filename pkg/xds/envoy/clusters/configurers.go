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

package clusters

import (
	core_mesh "github.com/apache/dubbo-kubernetes/pkg/core/resources/apis/mesh"
	envoy_cluster "github.com/envoyproxy/go-control-plane/envoy/config/cluster/v3"

	"google.golang.org/protobuf/types/known/wrapperspb"
)

import (
	core_xds "github.com/apache/dubbo-kubernetes/pkg/core/xds"
	v3 "github.com/apache/dubbo-kubernetes/pkg/xds/envoy/clusters/v3"
	envoy_tags "github.com/apache/dubbo-kubernetes/pkg/xds/envoy/tags"
)

func EdsCluster() ClusterBuilderOpt {
	return ClusterBuilderOptFunc(func(builder *ClusterBuilder) {
		builder.AddConfigurer(&v3.EdsClusterConfigurer{})
		builder.AddConfigurer(&v3.AltStatNameConfigurer{})
	})
}

func DefaultTimeout() ClusterBuilderOpt {
	return ClusterBuilderOptFunc(func(builder *ClusterBuilder) {
		builder.AddConfigurer(&v3.TimeoutConfigurer{
			Protocol: core_mesh.ProtocolTCP,
		})
	})
}

// ProvidedEndpointCluster sets the cluster with the defined endpoints, this is useful when endpoints are not discovered using EDS, so we don't use EdsCluster
func ProvidedEndpointCluster(hasIPv6 bool, endpoints ...core_xds.Endpoint) ClusterBuilderOpt {
	return ClusterBuilderOptFunc(func(builder *ClusterBuilder) {
		builder.AddConfigurer(&v3.ProvidedEndpointClusterConfigurer{
			Name:      builder.name,
			Endpoints: endpoints,
			HasIPv6:   hasIPv6,
		})
		builder.AddConfigurer(&v3.AltStatNameConfigurer{})
	})
}

// LbSubset is required for MetadataMatch in Weighted Cluster in TCP Proxy to work.
// Subset loadbalancing is used in two use cases
//  1. TrafficRoute for splitting traffic. Example: TrafficRoute that splits 10% of the traffic to version 1 of the service backend and 90% traffic to version 2 of the service backend
//  2. Multiple outbound sections with the same service
//     Example:
//     type: Dataplane
//     networking:
//     outbound:
//     - port: 1234
//     tags:
//     dubbo.io/service: backend
//     - port: 1234
//     tags:
//     dubbo.io/service: backend
//     version: v1
//     Only one cluster "backend" is generated for such dataplane, but with lb subset by version.
func LbSubset(tagSets envoy_tags.TagKeysSlice) ClusterBuilderOptFunc {
	return func(builder *ClusterBuilder) {
		builder.AddConfigurer(&v3.LbSubsetConfigurer{
			TagKeysSets: tagSets,
		})
	}
}

func PassThroughCluster() ClusterBuilderOpt {
	return ClusterBuilderOptFunc(func(builder *ClusterBuilder) {
		builder.AddConfigurer(&v3.PassThroughClusterConfigurer{})
		builder.AddConfigurer(&v3.AltStatNameConfigurer{})
	})
}

func UpstreamBindConfig(address string, port uint32) ClusterBuilderOpt {
	return ClusterBuilderOptFunc(func(builder *ClusterBuilder) {
		builder.AddConfigurer(&v3.UpstreamBindConfigConfigurer{
			Address: address,
			Port:    port,
		})
	})
}

func ConnectionBufferLimit(bytes uint32) ClusterBuilderOpt {
	return ClusterBuilderOptFunc(func(builder *ClusterBuilder) {
		builder.AddConfigurer(v3.ClusterMustConfigureFunc(func(c *envoy_cluster.Cluster) {
			c.PerConnectionBufferLimitBytes = wrapperspb.UInt32(bytes)
		}))
	})
}

func Http2() ClusterBuilderOpt {
	return ClusterBuilderOptFunc(func(builder *ClusterBuilder) {
		builder.AddConfigurer(&v3.Http2Configurer{})
	})
}

func Http2FromEdge() ClusterBuilderOpt {
	return ClusterBuilderOptFunc(func(builder *ClusterBuilder) {
		builder.AddConfigurer(&v3.Http2Configurer{EdgeProxyWindowSizes: true})
	})
}

func Http() ClusterBuilderOpt {
	return ClusterBuilderOptFunc(func(builder *ClusterBuilder) {
		builder.AddConfigurer(&v3.HttpConfigurer{})
	})
}
