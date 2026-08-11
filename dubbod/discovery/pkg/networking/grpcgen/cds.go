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
	discovery "github.com/kdubbo/xds-api/service/discovery/v1"

	"github.com/apache/dubbo-kubernetes/dubbod/discovery/pkg/features"
	"github.com/apache/dubbo-kubernetes/dubbod/discovery/pkg/model"
	"github.com/apache/dubbo-kubernetes/dubbod/discovery/pkg/networking/util"
	"github.com/apache/dubbo-kubernetes/pkg/config/host"
	"github.com/apache/dubbo-kubernetes/pkg/util/sets"
	cluster "github.com/kdubbo/xds-api/cluster/v1"
	core "github.com/kdubbo/xds-api/core/v1"
	tlsv1 "github.com/kdubbo/xds-api/extensions/transport_sockets/tls/v1"
)

type clusterBuilder struct {
	push *model.PushContext
	node *model.Proxy

	defaultClusterName string
	hostname           host.Name
	portNum            int

	svc    *model.Service
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
	svc := push.ServiceForHostname(node, hostname)
	if svc == nil {
		return nil, fmt.Errorf("cds gen for %s: did not find service for cluster %s", node.ID, defaultClusterName)
	}

	_, ok := svc.Ports.GetByPort(portNum)
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

		svc: svc,
	}, nil
}

func (b *clusterBuilder) build() []*cluster.Cluster {
	defaultRequested := b.filter == nil || b.filter.Contains(b.defaultClusterName)
	if !defaultRequested {
		return nil
	}

	defaultCluster := b.edsCluster(b.defaultClusterName)
	defaultCluster.CommonLbConfig = &cluster.Cluster_CommonLbConfig{
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
	if b.requiresPeerAuthenticationMTLS() {
		b.applyPeerAuthenticationMTLS(defaultCluster)
	}
	if b.node != nil && b.node.IsRouter() {
		b.applyBackendTLSPolicy(defaultCluster)
	}
	log.Debugf("generated cluster %s", b.defaultClusterName)
	return []*cluster.Cluster{defaultCluster}
}

func (b *clusterBuilder) requiresPeerAuthenticationMTLS() bool {
	if b.push == nil || b.push.AuthenticationPolicies == nil || b.svc == nil {
		return false
	}
	mode := b.push.AuthenticationPolicies.EffectiveMutualTLSMode(
		b.svc.Attributes.Namespace, nil, uint32(b.portNum),
	)
	return mode == model.MTLSStrict || mode == model.MTLSPermissive
}

func (b *clusterBuilder) applyPeerAuthenticationMTLS(c *cluster.Cluster) {
	if c == nil || c.TransportSocket != nil {
		return
	}
	tlsContext := b.buildUpstreamTLSContext(c)
	if tlsContext == nil {
		log.Warnf("failed to build automatic mTLS context for PeerAuthentication on cluster %s", c.Name)
		return
	}
	c.TransportSocket = &core.TransportSocket{
		Name:       "transport_sockets.tls",
		ConfigType: &core.TransportSocket_TypedConfig{TypedConfig: protoconv.MessageToAny(tlsContext)},
	}
	log.Debugf("applied automatic mTLS to cluster %s for mesh PeerAuthentication", c.Name)
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
		LbPolicy: defaultLbPolicy(),
	}
}

// defaultLbPolicy resolves the mesh-wide default load balancing policy from the
// DUBBO_DEFAULT_LB_POLICY environment variable, falling back to ROUND_ROBIN for
// unknown values.
func defaultLbPolicy() cluster.Cluster_LbPolicy {
	switch strings.ToUpper(features.DefaultLoadBalancerPolicy) {
	case "LEAST_REQUEST":
		return cluster.Cluster_LEAST_REQUEST
	case "RING_HASH":
		return cluster.Cluster_RING_HASH
	case "RANDOM":
		return cluster.Cluster_RANDOM
	case "ROUND_ROBIN", "":
		return cluster.Cluster_ROUND_ROBIN
	default:
		log.Warnf("unknown DUBBO_DEFAULT_LB_POLICY %q, falling back to ROUND_ROBIN", features.DefaultLoadBalancerPolicy)
		return cluster.Cluster_ROUND_ROBIN
	}
}

func (b *clusterBuilder) applyBackendTLSPolicy(c *cluster.Cluster) {
	if c == nil || c.TransportSocket != nil || b.svc == nil || b.push == nil {
		return
	}
	if b.svc.Resolution != model.Alias || b.svc.Attributes.ExternalName == "" {
		return
	}
	settings, found := b.push.BackendTLSForService(b.svc.Attributes.Namespace, b.svc.Attributes.Name)
	if !found || settings.SNI == "" {
		return
	}
	tlsContext := &tlsv1.UpstreamTlsContext{
		CommonTlsContext: &tlsv1.CommonTlsContext{},
		Sni:              settings.SNI,
	}
	c.TransportSocket = &core.TransportSocket{
		Name:       "transport_sockets.tls",
		ConfigType: &core.TransportSocket_TypedConfig{TypedConfig: protoconv.MessageToAny(tlsContext)},
	}
	log.Debugf("applied BackendTLSPolicy simple TLS transport socket to cluster %s (SNI=%s)", c.Name, settings.SNI)
}

// buildUpstreamTLSContext builds an UpstreamTlsContext that conforms to gRPC xDS expectations,
// reusing the common certificate-provider setup from buildCommonTLSContext.
func (b *clusterBuilder) buildUpstreamTLSContext(c *cluster.Cluster) *tlsv1.UpstreamTlsContext {
	// Pin the upstream identity: only certificates whose SAN matches one of the
	// target service's SPIFFE identities are accepted.
	var sans []string
	if b.svc != nil {
		sans = b.push.ServiceAccounts(b.svc.Hostname, b.svc.Attributes.Namespace)
		if len(sans) == 0 && b.hostname != b.svc.Hostname {
			sans = b.push.ServiceAccounts(b.hostname, b.svc.Attributes.Namespace)
		}
		if b.push.ServiceActivationEnabled(b.svc.Attributes.Namespace, b.svc.Attributes.Name) {
			pinned := sets.New(sans...)
			pinned.InsertAll(b.push.ActivationBackendSANs(
				b.svc.Attributes.Namespace,
				b.svc.Attributes.Name,
			)...)
			pinned.InsertAll(b.push.ActivationGatewaySANs(b.svc.Attributes.Namespace)...)
			sans = sets.SortedList(pinned)
		}
		if len(sans) == 0 {
			log.Warnf("no SPIFFE identities found for %s; upstream TLS for cluster %s will not verify peer SAN", b.svc.Hostname, c.Name)
		}
	}
	common := buildCommonTLSContext(sans)
	if common == nil {
		return nil
	}
	applyMeshMinimumTLSVersion(common, b.push.Mesh)

	tlsContext := &tlsv1.UpstreamTlsContext{
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
	// Inherent gRPC always speaks HTTP/2, advertise h2 via ALPN.
	tlsContext.CommonTlsContext.AlpnProtocols = []string{"h2"}
	return tlsContext
}
