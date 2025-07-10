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
)

import (
	"github.com/apache/dubbo-kubernetes/pkg/xds/envoy/tags"
)

type LbSubsetConfigurer struct {
	TagKeysSets tags.TagKeysSlice
}

var _ ClusterConfigurer = &LbSubsetConfigurer{}

func (e *LbSubsetConfigurer) Configure(c *envoy_cluster.Cluster) error {
	var selectors []*envoy_cluster.Cluster_LbSubsetConfig_LbSubsetSelector
	for _, tagSet := range e.TagKeysSets {
		selectors = append(selectors, &envoy_cluster.Cluster_LbSubsetConfig_LbSubsetSelector{
			Keys: tagSet,
			// if there is a split by "version", and there is no endpoint with such version we should not fallback to all endpoints of the service
			FallbackPolicy: envoy_cluster.Cluster_LbSubsetConfig_LbSubsetSelector_NO_FALLBACK,
		})
	}
	if len(selectors) > 0 {
		// if lb subset is set, but no label (Dubbo's tag) is queried, we should return any endpoint
		c.LbSubsetConfig = &envoy_cluster.Cluster_LbSubsetConfig{
			FallbackPolicy:  envoy_cluster.Cluster_LbSubsetConfig_ANY_ENDPOINT,
			SubsetSelectors: selectors,
		}
	}
	return nil
}
