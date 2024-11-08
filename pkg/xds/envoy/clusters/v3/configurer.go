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
	envoy_listener "github.com/envoyproxy/go-control-plane/envoy/config/listener/v3"
)

// ClusterConfigurer is responsible for configuring a single aspect of the entire Envoy cluster,
// such as filter chain, transparent proxying, etc.
type ClusterConfigurer interface {
	// Configure configures a single aspect on a given Envoy cluster.
	Configure(cluster *envoy_cluster.Cluster) error
}

type FilterChainConfigurer interface {
	// Configure configures a single aspect on a given Envoy filter chain.
	Configure(filterChain *envoy_listener.FilterChain) error
}

// ClusterMustConfigureFunc adapts a configuration function that never
// fails to the ListenerConfigurer interface.
type ClusterMustConfigureFunc func(cluster *envoy_cluster.Cluster)

func (f ClusterMustConfigureFunc) Configure(cluster *envoy_cluster.Cluster) error {
	if f != nil {
		f(cluster)
	}

	return nil
}
