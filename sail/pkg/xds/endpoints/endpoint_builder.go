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

package endpoints

import (
	"github.com/apache/dubbo-kubernetes/pkg/config/host"
	"github.com/apache/dubbo-kubernetes/pkg/config/labels"
	"github.com/apache/dubbo-kubernetes/pkg/config/schema/kind"
	"github.com/apache/dubbo-kubernetes/sail/pkg/model"
	"github.com/apache/dubbo-kubernetes/sail/pkg/networking/util"
	"github.com/cespare/xxhash/v2"
	core "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	endpoint "github.com/envoyproxy/go-control-plane/envoy/config/endpoint/v3"
	"google.golang.org/protobuf/types/known/wrapperspb"
	"k8s.io/klog/v2"
)

var _ model.XdsCacheEntry = &EndpointBuilder{}

// EndpointBuilder builds Envoy endpoints from Dubbo endpoints
type EndpointBuilder struct {
	clusterName string
	proxy       *model.Proxy
	push        *model.PushContext
	hostname    host.Name
	port        int
	subsetName  string
	service     *model.Service
}

// NewEndpointBuilder creates a new EndpointBuilder
func NewEndpointBuilder(clusterName string, proxy *model.Proxy, push *model.PushContext) *EndpointBuilder {
	_, subsetName, hostname, port := model.ParseSubsetKey(clusterName)
	if hostname == "" || port == 0 {
		return nil
	}

	svc := push.ServiceForHostname(proxy, hostname)

	return &EndpointBuilder{
		clusterName: clusterName,
		proxy:       proxy,
		push:        push,
		hostname:    hostname,
		port:        port,
		subsetName:  subsetName,
		service:     svc,
	}
}

// ServiceFound returns whether the service was found
func (b *EndpointBuilder) ServiceFound() bool {
	return b.service != nil
}

// BuildClusterLoadAssignment converts the shards for this EndpointBuilder's Service
// into a ClusterLoadAssignment. Used for EDS.
func (b *EndpointBuilder) BuildClusterLoadAssignment(endpointIndex *model.EndpointIndex) *endpoint.ClusterLoadAssignment {
	if !b.ServiceFound() {
		return buildEmptyClusterLoadAssignment(b.clusterName)
	}

	svcPort := b.servicePort(b.port)
	if svcPort == nil {
		return buildEmptyClusterLoadAssignment(b.clusterName)
	}

	// Get endpoints from the endpoint index
	shards, ok := endpointIndex.ShardsForService(string(b.hostname), b.service.Attributes.Namespace)
	if !ok {
		klog.Warningf("BuildClusterLoadAssignment: no shards found for service %s in namespace %s (cluster=%s, port=%d, svcPort.Name='%s', svcPort.Port=%d)",
			b.hostname, b.service.Attributes.Namespace, b.clusterName, b.port, svcPort.Name, svcPort.Port)
		return buildEmptyClusterLoadAssignment(b.clusterName)
	}

	// CRITICAL: Log shards info before processing
	shards.RLock()
	shardCount := len(shards.Shards)
	totalEndpointsInShards := 0
	for _, eps := range shards.Shards {
		totalEndpointsInShards += len(eps)
	}
	shards.RUnlock()
	klog.V(3).Infof("BuildClusterLoadAssignment: found shards for service %s (cluster=%s, port=%d, shardCount=%d, totalEndpointsInShards=%d)",
		b.hostname, b.clusterName, b.port, shardCount, totalEndpointsInShards)

	// Build port map for filtering
	portMap := map[string]int{}
	for _, port := range b.service.Ports {
		portMap[port.Name] = port.Port
	}
	ports := make(map[int]bool)
	for _, port := range b.service.Ports {
		ports[port.Port] = true
	}

	// Get endpoints from shards
	shards.RLock()
	defer shards.RUnlock()

	// Filter and convert endpoints
	var lbEndpoints []*endpoint.LbEndpoint
	var filteredCount int
	var totalEndpoints int

	for _, eps := range shards.Shards {
		for _, ep := range eps {
			totalEndpoints++
			// Filter by port name
			if ep.ServicePortName != svcPort.Name {
				filteredCount++
				klog.V(3).Infof("BuildClusterLoadAssignment: filtering out endpoint %s (port name mismatch: '%s' != '%s')",
					ep.FirstAddressOrNil(), ep.ServicePortName, svcPort.Name)
				continue
			}

			// Filter by subset labels if subset is specified
			if b.subsetName != "" && !b.matchesSubset(ep.Labels) {
				continue
			}

			// Filter out unhealthy endpoints unless service supports them
			// CRITICAL: Even if endpoint is unhealthy, we should include it if:
			// 1. Service has publishNotReadyAddresses=true (SupportsUnhealthyEndpoints returns true)
			// 2. Or if the endpoint is actually healthy (HealthStatus=Healthy)
			//
			// For gRPC proxyless, we may want to include endpoints even if they're not ready initially,
			// as the client will handle connection failures. However, this is controlled by the service
			// configuration (publishNotReadyAddresses).
			if !b.service.SupportsUnhealthyEndpoints() && ep.HealthStatus == model.UnHealthy {
				klog.V(3).Infof("BuildClusterLoadAssignment: filtering out unhealthy endpoint %s", ep.FirstAddressOrNil())
				filteredCount++
				continue
			}

			// Build LbEndpoint
			lbEp := b.buildLbEndpoint(ep)
			if lbEp == nil {
				klog.V(3).Infof("BuildClusterLoadAssignment: buildLbEndpoint returned nil for endpoint %s", ep.FirstAddressOrNil())
				filteredCount++
				continue
			}
			lbEndpoints = append(lbEndpoints, lbEp)
		}
	}

	if len(lbEndpoints) == 0 {
		klog.V(2).Infof("BuildClusterLoadAssignment: no endpoints found for cluster %s (hostname=%s, port=%d, totalEndpoints=%d, filteredCount=%d)",
			b.clusterName, b.hostname, b.port, totalEndpoints, filteredCount)
		return buildEmptyClusterLoadAssignment(b.clusterName)
	}

	// Create LocalityLbEndpoints with empty locality (default)
	localityLbEndpoints := []*endpoint.LocalityLbEndpoints{
		{
			Locality:    &core.Locality{},
			LbEndpoints: lbEndpoints,
			LoadBalancingWeight: &wrapperspb.UInt32Value{
				Value: uint32(len(lbEndpoints)),
			},
		},
	}

	return &endpoint.ClusterLoadAssignment{
		ClusterName: b.clusterName,
		Endpoints:   localityLbEndpoints,
	}
}

func (b *EndpointBuilder) servicePort(port int) *model.Port {
	if !b.ServiceFound() {
		return nil
	}
	svcPort, ok := b.service.Ports.GetByPort(port)
	if !ok {
		return nil
	}
	return svcPort
}

func (b *EndpointBuilder) matchesSubset(epLabels labels.Instance) bool {
	// TODO: implement subset matching logic based on DestinationRule
	// For now, return true if no subset is specified or if subset is empty
	if b.subsetName == "" {
		return true
	}
	// Simplified subset matching - in real implementation, this should match
	// against DestinationRule subset labels
	return true
}

func (b *EndpointBuilder) buildLbEndpoint(ep *model.DubboEndpoint) *endpoint.LbEndpoint {
	if len(ep.Addresses) == 0 {
		return nil
	}

	address := util.BuildAddress(ep.Addresses[0], ep.EndpointPort)
	if address == nil {
		return nil
	}

	// Convert health status
	healthStatus := core.HealthStatus_HEALTHY
	switch ep.HealthStatus {
	case model.UnHealthy:
		healthStatus = core.HealthStatus_UNHEALTHY
	case model.Draining:
		healthStatus = core.HealthStatus_DRAINING
	case model.Terminating:
		healthStatus = core.HealthStatus_UNHEALTHY
	default:
		healthStatus = core.HealthStatus_HEALTHY
	}

	return &endpoint.LbEndpoint{
		HostIdentifier: &endpoint.LbEndpoint_Endpoint{
			Endpoint: &endpoint.Endpoint{
				Address: address,
			},
		},
		HealthStatus: healthStatus,
		LoadBalancingWeight: &wrapperspb.UInt32Value{
			Value: 1,
		},
	}
}

func buildEmptyClusterLoadAssignment(clusterName string) *endpoint.ClusterLoadAssignment {
	return &endpoint.ClusterLoadAssignment{
		ClusterName: clusterName,
	}
}

// Cacheable implements model.XdsCacheEntry
func (b *EndpointBuilder) Cacheable() bool {
	return b.service != nil
}

// Key implements model.XdsCacheEntry
func (b *EndpointBuilder) Key() any {
	// CRITICAL FIX: EDS cache expects uint64 key, not string
	// Hash the cluster name to uint64 to match the cache type
	return xxhash.Sum64String(b.clusterName)
}

// Type implements model.XdsCacheEntry
func (b *EndpointBuilder) Type() string {
	return "eds" // Must match model.EDSType constant ("eds", not "EDS")
}

// DependentConfigs implements model.XdsCacheEntry
func (b *EndpointBuilder) DependentConfigs() []model.ConfigHash {
	// CRITICAL FIX: Return ServiceEntry ConfigHash so that EDS cache can be properly cleared
	// when endpoints are updated. Without this, cache.Clear() cannot find and remove stale EDS entries.
	if b.service == nil {
		return nil
	}
	// Create ConfigKey for ServiceEntry and return its hash
	configKey := model.ConfigKey{
		Kind:      kind.ServiceEntry,
		Name:      string(b.hostname),
		Namespace: b.service.Attributes.Namespace,
	}
	return []model.ConfigHash{configKey.HashCode()}
}
