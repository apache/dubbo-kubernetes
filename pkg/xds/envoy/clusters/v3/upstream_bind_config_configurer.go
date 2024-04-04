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
	envoy_cluster "github.com/envoyproxy/go-control-plane/envoy/config/cluster/v3"
	envoy_core "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
)

type UpstreamBindConfigConfigurer struct {
	Address string
	Port    uint32
}

var _ ClusterConfigurer = &UpstreamBindConfigConfigurer{}

func (u *UpstreamBindConfigConfigurer) Configure(c *envoy_cluster.Cluster) error {
	c.UpstreamBindConfig = &envoy_core.BindConfig{
		SourceAddress: &envoy_core.SocketAddress{
			Address: u.Address,
			PortSpecifier: &envoy_core.SocketAddress_PortValue{
				PortValue: u.Port,
			},
		},
	}
	return nil
}
