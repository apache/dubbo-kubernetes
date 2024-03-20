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
	"net"
)

import (
	envoy_cluster "github.com/envoyproxy/go-control-plane/envoy/config/cluster/v3"

	"github.com/pkg/errors"
)

import (
	"github.com/apache/dubbo-kubernetes/pkg/core/xds"
	envoy_endpoints "github.com/apache/dubbo-kubernetes/pkg/xds/envoy/endpoints/v3"
)

type ProvidedEndpointClusterConfigurer struct {
	Name      string
	Endpoints []xds.Endpoint
	HasIPv6   bool
}

var _ ClusterConfigurer = &ProvidedEndpointClusterConfigurer{}

func (e *ProvidedEndpointClusterConfigurer) Configure(c *envoy_cluster.Cluster) error {
	if len(e.Endpoints) == 0 {
		return errors.New("cluster must have at least 1 endpoint")
	}
	if len(e.Endpoints) > 1 {
		c.LbPolicy = envoy_cluster.Cluster_ROUND_ROBIN
	}
	var nonIpEndpoints []xds.Endpoint
	var ipEndpoints []xds.Endpoint
	for _, endpoint := range e.Endpoints {
		if net.ParseIP(endpoint.Target) != nil || endpoint.UnixDomainPath != "" {
			ipEndpoints = append(ipEndpoints, endpoint)
		} else {
			nonIpEndpoints = append(nonIpEndpoints, endpoint)
		}
	}
	if len(nonIpEndpoints) > 0 && len(ipEndpoints) > 0 {
		return errors.New("cluster is a mix of ips and hostnames, can't generate envoy config.")
	}
	if len(nonIpEndpoints) > 0 {
		c.ClusterDiscoveryType = &envoy_cluster.Cluster_Type{Type: envoy_cluster.Cluster_STRICT_DNS}
		if e.HasIPv6 {
			c.DnsLookupFamily = envoy_cluster.Cluster_AUTO
		} else {
			c.DnsLookupFamily = envoy_cluster.Cluster_V4_ONLY
		}
	} else {
		c.ClusterDiscoveryType = &envoy_cluster.Cluster_Type{Type: envoy_cluster.Cluster_STATIC}
	}
	c.LoadAssignment = envoy_endpoints.CreateClusterLoadAssignment(e.Name, e.Endpoints)
	return nil
}
