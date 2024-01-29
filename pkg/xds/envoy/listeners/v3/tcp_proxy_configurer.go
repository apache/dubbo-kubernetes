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

package v3

import (
	envoy_listener "github.com/envoyproxy/go-control-plane/envoy/config/listener/v3"
	envoy_tcp "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/network/tcp_proxy/v3"
)

import (
	"github.com/apache/dubbo-kubernetes/pkg/util/proto"
	util_xds "github.com/apache/dubbo-kubernetes/pkg/util/xds"
	envoy_common "github.com/apache/dubbo-kubernetes/pkg/xds/envoy"
	envoy_metadata "github.com/apache/dubbo-kubernetes/pkg/xds/envoy/metadata/v3"
)

type TcpProxyConfigurer struct {
	StatsName string
	// Splits to forward traffic to.
	Splits      []envoy_common.Split
	UseMetadata bool
}

func (c *TcpProxyConfigurer) Configure(filterChain *envoy_listener.FilterChain) error {
	if len(c.Splits) == 0 {
		return nil
	}
	tcpProxy := c.tcpProxy()

	pbst, err := proto.MarshalAnyDeterministic(tcpProxy)
	if err != nil {
		return err
	}

	filterChain.Filters = append(filterChain.Filters, &envoy_listener.Filter{
		Name: "envoy.filters.network.tcp_proxy",
		ConfigType: &envoy_listener.Filter_TypedConfig{
			TypedConfig: pbst,
		},
	})
	return nil
}

func (c *TcpProxyConfigurer) tcpProxy() *envoy_tcp.TcpProxy {
	proxy := envoy_tcp.TcpProxy{
		StatPrefix: util_xds.SanitizeMetric(c.StatsName),
	}

	if len(c.Splits) == 1 {
		proxy.ClusterSpecifier = &envoy_tcp.TcpProxy_Cluster{
			Cluster: c.Splits[0].ClusterName(),
		}
		if c.UseMetadata {
			proxy.MetadataMatch = envoy_metadata.LbMetadata(c.Splits[0].LBMetadata())
		}
		return &proxy
	}

	var weightedClusters []*envoy_tcp.TcpProxy_WeightedCluster_ClusterWeight
	for _, split := range c.Splits {
		weightedCluster := &envoy_tcp.TcpProxy_WeightedCluster_ClusterWeight{
			Name:   split.ClusterName(),
			Weight: split.Weight(),
		}
		if c.UseMetadata {
			weightedCluster.MetadataMatch = envoy_metadata.LbMetadata(split.LBMetadata())
		}
		weightedClusters = append(weightedClusters, weightedCluster)
	}
	proxy.ClusterSpecifier = &envoy_tcp.TcpProxy_WeightedClusters{
		WeightedClusters: &envoy_tcp.TcpProxy_WeightedCluster{
			Clusters: weightedClusters,
		},
	}
	return &proxy
}
