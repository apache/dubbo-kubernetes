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

package grpcgen

import (
	"fmt"

	"github.com/apache/dubbo-kubernetes/dubbod/planet/pkg/util/protoconv"
	discovery "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v3"

	"github.com/apache/dubbo-kubernetes/dubbod/planet/pkg/model"
	"github.com/apache/dubbo-kubernetes/dubbod/planet/pkg/networking/util"
	"github.com/apache/dubbo-kubernetes/pkg/config/host"
	"github.com/apache/dubbo-kubernetes/pkg/util/sets"
	cluster "github.com/envoyproxy/go-control-plane/envoy/config/cluster/v3"
	core "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
)

type clusterBuilder struct {
	push *model.PushContext
	node *model.Proxy

	defaultClusterName string
	hostname           host.Name
	portNum            int

	// may not be set
	svc    *model.Service
	port   *model.Port
	filter sets.String
}

func (g *GrpcConfigGenerator) BuildClusters(node *model.Proxy, push *model.PushContext, names []string) model.Resources {
	filter := newClusterFilter(names)
	clusters := make([]*cluster.Cluster, 0, len(names))
	for defaultClusterName, subsetFilter := range filter {
		builder, err := newClusterBuilder(node, push, defaultClusterName, subsetFilter)
		if err != nil {
			log.Warn(err)
			continue
		}
		clusters = append(clusters, builder.build()...)
	}

	resp := make(model.Resources, 0, len(clusters))
	for _, c := range clusters {
		resp = append(resp, &discovery.Resource{
			Name:     c.Name,
			Resource: protoconv.MessageToAny(c),
		})
	}
	if len(resp) == 0 && len(names) == 0 {
		log.Warnf("did not generate any cds for %s; no names provided", node.ID)
	}
	return resp
}

func newClusterFilter(names []string) map[string]sets.String {
	filter := map[string]sets.String{}
	for _, name := range names {
		dir, _, hn, p := model.ParseSubsetKey(name)
		defaultKey := model.BuildSubsetKey(dir, "", hn, p)
		sets.InsertOrNew(filter, defaultKey, name)
	}
	return filter
}

func newClusterBuilder(node *model.Proxy, push *model.PushContext, defaultClusterName string, filter sets.String) (*clusterBuilder, error) {
	_, _, hostname, portNum := model.ParseSubsetKey(defaultClusterName)
	if hostname == "" || portNum == 0 {
		return nil, fmt.Errorf("failed parsing subset key: %s", defaultClusterName)
	}

	// try to resolve the service and port
	var port *model.Port
	svc := push.ServiceForHostname(node, hostname)
	if svc == nil {
		return nil, fmt.Errorf("cds gen for %s: did not find service for cluster %s", node.ID, defaultClusterName)
	}

	port, ok := svc.Ports.GetByPort(portNum)
	if !ok {
		return nil, fmt.Errorf("cds gen for %s: did not find port %d in service for cluster %s", node.ID, portNum, defaultClusterName)
	}

	return &clusterBuilder{
		node: node,
		push: push,

		defaultClusterName: defaultClusterName,
		hostname:           hostname,
		portNum:            portNum,
		filter:             filter,

		svc:  svc,
		port: port,
	}, nil
}

func (b *clusterBuilder) build() []*cluster.Cluster {
	var defaultCluster *cluster.Cluster
	defaultRequested := b.filter == nil || b.filter.Contains(b.defaultClusterName)
	if defaultRequested {
		defaultCluster = b.edsCluster(b.defaultClusterName)
		// CRITICAL: For gRPC proxyless, we need to set CommonLbConfig to handle endpoint health status
		// Following Istio's implementation, we should include UNHEALTHY and DRAINING endpoints
		// in OverrideHostStatus so that clients can use them when healthy endpoints are not available.
		// This prevents "weighted-target: no targets to pick from" errors when all endpoints are unhealthy.
		// The client will prioritize HEALTHY endpoints but can fall back to UNHEALTHY/DRAINING if needed.
		if defaultCluster.CommonLbConfig == nil {
			defaultCluster.CommonLbConfig = &cluster.Cluster_CommonLbConfig{}
		}
		// CRITICAL FIX: Following Istio's implementation, always include UNHEALTHY and DRAINING
		// in OverrideHostStatus. This allows clients to use unhealthy endpoints when healthy ones
		// are not available, preventing "weighted-target: no targets to pick from" errors.
		// The client will still prioritize HEALTHY endpoints, but can fall back to others.
		defaultCluster.CommonLbConfig.OverrideHostStatus = &core.HealthStatusSet{
			Statuses: []core.HealthStatus{
				core.HealthStatus_HEALTHY,
				core.HealthStatus_UNHEALTHY,
				core.HealthStatus_DRAINING,
				core.HealthStatus_UNKNOWN,
				core.HealthStatus_DEGRADED,
			},
		}
		log.Infof("clusterBuilder.build: generated default cluster %s", b.defaultClusterName)
	}

	subsetClusters := b.applyDestinationRule(defaultCluster)
	out := make([]*cluster.Cluster, 0, 1+len(subsetClusters))
	if defaultCluster != nil {
		out = append(out, defaultCluster)
	}
	result := append(out, subsetClusters...)
	log.Infof("clusterBuilder.build: generated %d clusters total (1 default + %d subsets) for %s",
		len(result), len(subsetClusters), b.defaultClusterName)
	return result
}

func (b *clusterBuilder) edsCluster(name string) *cluster.Cluster {
	return &cluster.Cluster{
		Name:                 name,
		AltStatName:          util.DelimitedStatsPrefix(name),
		ClusterDiscoveryType: &cluster.Cluster_Type{Type: cluster.Cluster_EDS},
		EdsClusterConfig: &cluster.Cluster_EdsClusterConfig{
			ServiceName: name,
			EdsConfig: &core.ConfigSource{
				ConfigSourceSpecifier: &core.ConfigSource_Ads{
					Ads: &core.AggregatedConfigSource{},
				},
			},
		},
		// CRITICAL: For gRPC proxyless, we need to set LbPolicy to ROUND_ROBIN
		// This is the default load balancing policy for gRPC xDS clients
		LbPolicy: cluster.Cluster_ROUND_ROBIN,
	}
}

func (b *clusterBuilder) applyDestinationRule(defaultCluster *cluster.Cluster) (subsetClusters []*cluster.Cluster) {
	if b.svc == nil || b.port == nil {
		log.Warnf("applyDestinationRule: service or port is nil for %s", b.defaultClusterName)
		return nil
	}
	log.Infof("applyDestinationRule: looking for DestinationRule for service %s/%s (hostname=%s, port=%d)",
		b.svc.Attributes.Namespace, b.svc.Attributes.Name, b.hostname, b.portNum)
	dr := b.push.DestinationRuleForService(b.svc.Attributes.Namespace, b.hostname)
	if dr == nil {
		log.Warnf("applyDestinationRule: no DestinationRule found for %s/%s", b.svc.Attributes.Namespace, b.hostname)
		return nil
	}
	if len(dr.Subsets) == 0 {
		log.Warnf("applyDestinationRule: DestinationRule found for %s/%s but has no subsets", b.svc.Attributes.Namespace, b.hostname)
		return nil
	}

	log.Infof("applyDestinationRule: found DestinationRule for %s/%s with %d subsets, defaultCluster requested=%v",
		b.svc.Attributes.Namespace, b.hostname, len(dr.Subsets), defaultCluster != nil)

	var commonLbConfig *cluster.Cluster_CommonLbConfig
	if defaultCluster != nil {
		commonLbConfig = defaultCluster.CommonLbConfig
	} else {
		commonLbConfig = &cluster.Cluster_CommonLbConfig{
			OverrideHostStatus: &core.HealthStatusSet{
				Statuses: []core.HealthStatus{
					core.HealthStatus_HEALTHY,
					core.HealthStatus_UNHEALTHY,
					core.HealthStatus_DRAINING,
					core.HealthStatus_UNKNOWN,
					core.HealthStatus_DEGRADED,
				},
			},
		}
	}

	defaultClusterRequested := defaultCluster != nil
	if b.filter != nil {
		defaultClusterRequested = b.filter.Contains(b.defaultClusterName)
	}

	for _, subset := range dr.Subsets {
		if subset == nil || subset.Name == "" {
			continue
		}
		clusterName := model.BuildSubsetKey(model.TrafficDirectionOutbound, subset.Name, b.hostname, b.portNum)

		// CRITICAL: Always generate subset clusters if default cluster is requested
		// This is essential for RDS WeightedCluster to work correctly
		shouldGenerate := true
		if b.filter != nil && !b.filter.Contains(clusterName) {
			// Subset cluster not explicitly requested, but generate it if default cluster was requested
			shouldGenerate = defaultClusterRequested
		}

		if !shouldGenerate {
			log.Debugf("applyDestinationRule: skipping subset cluster %s (not requested and default not requested)", clusterName)
			continue
		}

		log.Infof("applyDestinationRule: generating subset cluster %s for subset %s", clusterName, subset.Name)
		subsetCluster := b.edsCluster(clusterName)
		subsetCluster.CommonLbConfig = commonLbConfig
		subsetClusters = append(subsetClusters, subsetCluster)
	}

	log.Infof("applyDestinationRule: generated %d subset clusters for %s/%s", len(subsetClusters), b.svc.Attributes.Namespace, b.hostname)
	return subsetClusters
}
