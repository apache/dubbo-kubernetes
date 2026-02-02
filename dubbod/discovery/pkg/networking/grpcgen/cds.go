//
// Licensed to the Apache Software Foundation (ASF) under one or more
// contributor license agreements.  See the NOTICE file distributed with
// this work for additional information regarding copyright ownership.
// The ASF licenses this file to You under the Apache License, Version 2.0
// (the "License"); you may not use this file except in compliance with
// the License.  You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package grpcgen

import (
	"fmt"
	"strings"

	"github.com/apache/dubbo-kubernetes/dubbod/discovery/pkg/util/protoconv"
	discovery "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v3"

	networking "github.com/apache/dubbo-kubernetes/api/networking/v1alpha3"
	"github.com/apache/dubbo-kubernetes/dubbod/discovery/pkg/model"
	"github.com/apache/dubbo-kubernetes/dubbod/discovery/pkg/networking/util"
	"github.com/apache/dubbo-kubernetes/pkg/config/host"
	"github.com/apache/dubbo-kubernetes/pkg/util/sets"
	cluster "github.com/envoyproxy/go-control-plane/envoy/config/cluster/v3"
	core "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	tlsv3 "github.com/envoyproxy/go-control-plane/envoy/extensions/transport_sockets/tls/v3"
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

	var dr *networking.DestinationRule
	if b.svc != nil {
		dr = b.push.DestinationRuleForService(b.svc.Attributes.Namespace, b.hostname)
		if dr == nil && b.svc.Hostname != b.hostname {
			dr = b.push.DestinationRuleForService(b.svc.Attributes.Namespace, b.svc.Hostname)
		}
	}
	hasTLSInDR := dr != nil && dr.TrafficPolicy != nil && dr.TrafficPolicy.Tls != nil
	if hasTLSInDR {
		tlsMode := dr.TrafficPolicy.Tls.Mode
		tlsModeStr := dr.TrafficPolicy.Tls.Mode.String()
		hasTLSInDR = (tlsMode == networking.ClientTLSSettings_DUBBO_MUTUAL || tlsModeStr == "DUBBO_MUTUAL")
	}

	// Generate default cluster if requested OR if DestinationRule has DUBBO_MUTUAL TLS
	if defaultRequested || hasTLSInDR {
		defaultCluster = b.edsCluster(b.defaultClusterName)
		// For gRPC proxyless, we need to set CommonLbConfig to handle endpoint health status
		// in OverrideHostStatus so that clients can use them when healthy endpoints are not available.
		// The client will prioritize HEALTHY endpoints but can fall back to UNHEALTHY/DRAINING if needed.
		if defaultCluster.CommonLbConfig == nil {
			defaultCluster.CommonLbConfig = &cluster.Cluster_CommonLbConfig{}
		}
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
		// TLS will be applied in applyDestinationRule after DestinationRule is found
		if hasTLSInDR {
			log.Infof("generated default cluster %s (required for DUBBO_MUTUAL TLS)", b.defaultClusterName)
		} else {
			log.Infof("generated default cluster %s", b.defaultClusterName)
		}
	}

	subsetClusters, newDefaultCluster := b.applyDestinationRule(defaultCluster)
	// If applyDestinationRule generated a new default cluster (because TLS was found but cluster wasn't generated in build()),
	// use it instead of the original defaultCluster
	if newDefaultCluster != nil {
		defaultCluster = newDefaultCluster
	}
	out := make([]*cluster.Cluster, 0, 1+len(subsetClusters))
	if defaultCluster != nil {
		out = append(out, defaultCluster)
	}
	result := append(out, subsetClusters...)
	log.Infof("generated %d clusters total (1 default + %d subsets) for %s",
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
		// For gRPC proxyless, we need to set LbPolicy to ROUND_ROBIN
		// This is the default load balancing policy for gRPC xDS clients
		LbPolicy: cluster.Cluster_ROUND_ROBIN,
	}
}

func (b *clusterBuilder) applyDestinationRule(defaultCluster *cluster.Cluster) (subsetClusters []*cluster.Cluster, newDefaultCluster *cluster.Cluster) {
	if b.svc == nil || b.port == nil {
		log.Warnf("service or port is nil for %s", b.defaultClusterName)
		return nil, nil
	}
	log.Infof("looking for DestinationRule for service %s/%s (hostname=%s, port=%d)",
		b.svc.Attributes.Namespace, b.svc.Attributes.Name, b.hostname, b.portNum)
	dr := b.push.DestinationRuleForService(b.svc.Attributes.Namespace, b.hostname)
	if dr == nil {
		// If not found with b.hostname, try with the service's FQDN hostname
		if b.svc.Hostname != b.hostname {
			dr = b.push.DestinationRuleForService(b.svc.Attributes.Namespace, b.svc.Hostname)
		}
		if dr == nil {
			log.Warnf("no DestinationRule found for %s/%s or %s", b.svc.Attributes.Namespace, b.hostname, b.svc.Hostname)
			return nil, nil
		}
	}

	// Check if DestinationRule has TLS configuration
	hasTLS := dr.TrafficPolicy != nil && dr.TrafficPolicy.Tls != nil
	if hasTLS {
		tlsMode := dr.TrafficPolicy.Tls.Mode
		tlsModeStr := dr.TrafficPolicy.Tls.Mode.String()
		hasTLS = (tlsMode == networking.ClientTLSSettings_DUBBO_MUTUAL || tlsModeStr == "DUBBO_MUTUAL")
	}

	// If no subsets and no TLS, there's nothing to do
	if len(dr.Subsets) == 0 && !hasTLS {
		log.Warnf("DestinationRule found for %s/%s but has no subsets and no TLS policy", b.svc.Attributes.Namespace, b.hostname)
		return nil, nil
	}

	log.Infof("found DestinationRule for %s/%s with %d subsets, defaultCluster requested=%v, hasTLS=%v",
		b.svc.Attributes.Namespace, b.hostname, len(dr.Subsets), defaultCluster != nil, hasTLS)

	// Apply TLS to default cluster if it exists and doesn't have TransportSocket yet
	// This ensures that default cluster gets TLS from the top-level TrafficPolicy in DestinationRule
	// When DestinationRule sets DUBBO_MUTUAL, inbound listener enforces STRICT mTLS, so outbound must also use TLS
	// NOTE: We re-check hasTLS here because firstDestinationRule might have returned a different rule
	// than the one checked in build(), especially when multiple DestinationRules exist and merge failed
	if defaultCluster != nil && defaultCluster.TransportSocket == nil {
		// Re-check TLS in case DestinationRule was found here but not in build()
		recheckTLS := dr != nil && dr.TrafficPolicy != nil && dr.TrafficPolicy.Tls != nil
		if recheckTLS {
			tlsMode := dr.TrafficPolicy.Tls.Mode
			tlsModeStr := dr.TrafficPolicy.Tls.Mode.String()
			recheckTLS = (tlsMode == networking.ClientTLSSettings_DUBBO_MUTUAL || tlsModeStr == "DUBBO_MUTUAL")
		}
		if hasTLS || recheckTLS {
			log.Infof("applying TLS to default cluster %s (DestinationRule has DUBBO_MUTUAL)", b.defaultClusterName)
			b.applyTLSForCluster(defaultCluster, nil)
		} else {
			log.Debugf("skipping TLS for default cluster %s (DestinationRule has no TrafficPolicy or TLS)", b.defaultClusterName)
		}
	} else if defaultCluster == nil && hasTLS {
		// If default cluster was not generated in build() but DestinationRule has TLS,
		// we need to generate it here to ensure TLS is applied
		// This can happen if build() checked the first rule (without TLS) but applyDestinationRule
		// found a different rule (with TLS) via firstDestinationRule's improved logic
		log.Warnf("default cluster was not generated in build() but DestinationRule has TLS, generating it now")
		defaultCluster = b.edsCluster(b.defaultClusterName)
		if defaultCluster.CommonLbConfig == nil {
			defaultCluster.CommonLbConfig = &cluster.Cluster_CommonLbConfig{}
		}
		defaultCluster.CommonLbConfig.OverrideHostStatus = &core.HealthStatusSet{
			Statuses: []core.HealthStatus{
				core.HealthStatus_HEALTHY,
				core.HealthStatus_UNHEALTHY,
				core.HealthStatus_DRAINING,
				core.HealthStatus_UNKNOWN,
				core.HealthStatus_DEGRADED,
			},
		}
		log.Infof("applying TLS to newly generated default cluster %s (DestinationRule has DUBBO_MUTUAL)", b.defaultClusterName)
		b.applyTLSForCluster(defaultCluster, nil)
		return nil, defaultCluster // Return the newly generated default cluster
	}

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

		// Always generate subset clusters if default cluster is requested
		shouldGenerate := true
		if b.filter != nil && !b.filter.Contains(clusterName) {
			// Subset cluster not explicitly requested, but generate it if default cluster was requested
			shouldGenerate = defaultClusterRequested
		}

		if !shouldGenerate {
			log.Debugf("skipping subset cluster %s (not requested and default not requested)", clusterName)
			continue
		}

		log.Infof("generating subset cluster %s for subset %s", clusterName, subset.Name)
		subsetCluster := b.edsCluster(clusterName)
		subsetCluster.CommonLbConfig = commonLbConfig
		b.applyTLSForCluster(subsetCluster, subset)
		subsetClusters = append(subsetClusters, subsetCluster)
	}

	log.Infof("generated %d subset clusters for %s/%s", len(subsetClusters), b.svc.Attributes.Namespace, b.hostname)
	return subsetClusters, nil
}

// applyTLSForCluster attaches a gRPC-compatible TLS transport socket whenever the
// DestinationRule (or subset override) specifies DUBBO_MUTUAL/DUBBO_MUTUAL mode.
func (b *clusterBuilder) applyTLSForCluster(c *cluster.Cluster, subset *networking.Subset) {
	if c == nil || b.svc == nil {
		return
	}

	dr := b.push.DestinationRuleForService(b.svc.Attributes.Namespace, b.hostname)
	if dr == nil && b.svc.Hostname != b.hostname {
		// If not found with b.hostname, try with the service's FQDN hostname
		dr = b.push.DestinationRuleForService(b.svc.Attributes.Namespace, b.svc.Hostname)
	}
	if dr == nil {
		log.Warnf("no DestinationRule found for cluster %s (namespace=%s, hostname=%s, service hostname=%s)",
			c.Name, b.svc.Attributes.Namespace, b.hostname, b.svc.Hostname)
		return
	}

	var policy *networking.TrafficPolicy
	if subset != nil && subset.TrafficPolicy != nil {
		policy = subset.TrafficPolicy
		log.Infof("using TrafficPolicy from subset %s for cluster %s", subset.Name, c.Name)
	} else {
		policy = dr.TrafficPolicy
		if policy != nil {
			log.Infof("using top-level TrafficPolicy for cluster %s", c.Name)
		}
	}

	if policy == nil || policy.Tls == nil {
		if policy == nil {
			log.Warnf("no TrafficPolicy found in DestinationRule for cluster %s", c.Name)
		} else {
			log.Warnf("no TLS settings in TrafficPolicy for cluster %s", c.Name)
		}
		return
	}

	mode := policy.Tls.Mode
	modeStr := policy.Tls.Mode.String()
	if mode != networking.ClientTLSSettings_DUBBO_MUTUAL && modeStr != "DUBBO_MUTUAL" {
		log.Debugf("TLS mode %v (%s) not supported for gRPC proxyless, skipping", mode, modeStr)
		return
	}

	tlsContext := b.buildUpstreamTLSContext(c, policy.Tls)
	if tlsContext == nil {
		log.Warnf("failed to build TLS context for cluster %s", c.Name)
		return
	}

	sni := tlsContext.Sni
	if sni == "" {
		log.Warnf("SNI is empty for cluster %s, this may cause TLS handshake failures", c.Name)
	} else {
		log.Debugf("using SNI=%s for cluster %s", sni, c.Name)
	}

	c.TransportSocket = &core.TransportSocket{
		Name:       "envoy.transport_sockets.tls",
		ConfigType: &core.TransportSocket_TypedConfig{TypedConfig: protoconv.MessageToAny(tlsContext)},
	}
	log.Infof("applied %v TLS transport socket to cluster %s (SNI=%s)", mode, c.Name, sni)
}

// buildUpstreamTLSContext builds an UpstreamTlsContext that conforms to gRPC xDS expectations,
// reusing the common certificate-provider setup from buildCommonTLSContext.
func (b *clusterBuilder) buildUpstreamTLSContext(c *cluster.Cluster, tlsSettings *networking.ClientTLSSettings) *tlsv3.UpstreamTlsContext {
	common := buildCommonTLSContext()
	if common == nil {
		return nil
	}

	tlsContext := &tlsv3.UpstreamTlsContext{
		CommonTlsContext: common,
	}
	// SNI must be the service hostname, not the cluster name
	// Cluster name format: outbound|port|subset|hostname
	// We need to extract the hostname from the cluster name or use the service hostname
	if tlsContext.Sni == "" {
		if b.svc != nil && b.svc.Hostname != "" {
			tlsContext.Sni = string(b.svc.Hostname)
		} else {
			// Fallback: try to extract hostname from cluster name
			// Cluster name format: outbound|port|subset|hostname
			parts := strings.Split(c.Name, "|")
			if len(parts) >= 4 {
				tlsContext.Sni = parts[3]
			} else {
				// Last resort: use cluster name (not ideal but better than empty)
				tlsContext.Sni = c.Name
				log.Warnf("using cluster name as SNI fallback for %s (should be service hostname)", c.Name)
			}
		}
	}
	// Proxyless gRPC always speaks HTTP/2, advertise h2 via ALPN.
	tlsContext.CommonTlsContext.AlpnProtocols = []string{"h2"}
	return tlsContext
}
