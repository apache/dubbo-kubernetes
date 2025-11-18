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
	"fmt"

	"github.com/apache/dubbo-kubernetes/dubbod/planet/pkg/model"
	"github.com/apache/dubbo-kubernetes/dubbod/planet/pkg/networking/util"
	"github.com/apache/dubbo-kubernetes/pkg/config/host"
	"github.com/apache/dubbo-kubernetes/pkg/config/labels"
	"github.com/apache/dubbo-kubernetes/pkg/config/schema/kind"
	dubbolog "github.com/apache/dubbo-kubernetes/pkg/log"
	"github.com/cespare/xxhash/v2"
	core "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	endpoint "github.com/envoyproxy/go-control-plane/envoy/config/endpoint/v3"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

var log = dubbolog.RegisterScope("ads", "ads debugging")

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

var _ model.XdsCacheEntry = &EndpointBuilder{}

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
	// CRITICAL: Log all available services in EndpointIndex for debugging
	allServices := endpointIndex.AllServices()
	if len(allServices) > 0 {
		log.Infof("BuildClusterLoadAssignment: EndpointIndex contains %d services: %v", len(allServices), allServices)
	} else {
		log.Warnf("BuildClusterLoadAssignment: EndpointIndex is empty - no services registered")
	}

	shards, ok := endpointIndex.ShardsForService(string(b.hostname), b.service.Attributes.Namespace)
	if !ok {
		// CRITICAL: Log at INFO level for proxyless gRPC to help diagnose "weighted-target: no targets to pick from" errors
		// Also log what services ARE available in the namespace
		servicesInNamespace := endpointIndex.ServicesInNamespace(b.service.Attributes.Namespace)
		log.Infof("BuildClusterLoadAssignment: no shards found for service %s in namespace %s (cluster=%s, port=%d, svcPort.Name='%s', svcPort.Port=%d). "+
			"This usually means endpoints are not yet available or service is not registered. "+
			"Available services in namespace: %v",
			b.hostname, b.service.Attributes.Namespace, b.clusterName, b.port, svcPort.Name, svcPort.Port, servicesInNamespace)
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
	log.Debugf("BuildClusterLoadAssignment: found shards for service %s (cluster=%s, port=%d, shardCount=%d, totalEndpointsInShards=%d)",
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
	var portNameMismatchCount int
	var unhealthyCount int
	var buildFailedCount int

	// CRITICAL: Log all endpoint ServicePortNames for debugging port name matching issues
	allServicePortNames := make(map[string]int)
	for _, eps := range shards.Shards {
		for _, ep := range eps {
			allServicePortNames[ep.ServicePortName]++
		}
	}
	if len(allServicePortNames) > 0 {
		portNamesList := make([]string, 0, len(allServicePortNames))
		for name, count := range allServicePortNames {
			portNamesList = append(portNamesList, fmt.Sprintf("'%s'(%d)", name, count))
		}
		log.Infof("BuildClusterLoadAssignment: service %s port %d (svcPort.Name='%s'), found endpoints with ServicePortNames: %v",
			b.hostname, b.port, svcPort.Name, portNamesList)
	}

	for _, eps := range shards.Shards {
		for _, ep := range eps {
			totalEndpoints++
			// Filter by port name
			// CRITICAL: According to Istio's implementation, we must match ServicePortName exactly
			// However, if ServicePortName is empty, we should still include the endpoint if there's only one port
			// This handles cases where EndpointSlice doesn't have port name but Service does
			if ep.ServicePortName != svcPort.Name {
				// Special case: if ServicePortName is empty and service has only one port, include it
				if ep.ServicePortName == "" && len(b.service.Ports) == 1 {
					log.Debugf("BuildClusterLoadAssignment: including endpoint %s with empty ServicePortName (service has only one port '%s')",
						ep.FirstAddressOrNil(), svcPort.Name)
					// Continue processing this endpoint
				} else {
					portNameMismatchCount++
					filteredCount++
					log.Warnf("BuildClusterLoadAssignment: filtering out endpoint %s (port name mismatch: ep.ServicePortName='%s' != svcPort.Name='%s', EndpointPort=%d)",
						ep.FirstAddressOrNil(), ep.ServicePortName, svcPort.Name, ep.EndpointPort)
					continue
				}
			}

			// Filter by subset labels if subset is specified
			if b.subsetName != "" && !b.matchesSubset(ep.Labels) {
				continue
			}

			// CRITICAL FIX: Following Istio's implementation, we should ALWAYS include endpoints in EDS,
			// regardless of their health status. The client will decide whether to use them based on
			// OverrideHostStatus in the Cluster configuration.
			//
			// However, if the service explicitly doesn't support unhealthy endpoints (publishNotReadyAddresses=false),
			// we should still include them in EDS but mark them as UNHEALTHY. The client's OverrideHostStatus
			// will determine if they can be used.
			//
			// This is critical for proxyless gRPC - even if endpoints are unhealthy, they should be
			// included in EDS so the client knows they exist and can attempt to connect to them.
			// The client will handle connection failures appropriately.
			//
			// Note: We only filter out unhealthy endpoints if the service explicitly doesn't support them
			// AND we're not in a proxyless gRPC scenario. For proxyless gRPC, always include endpoints.
			// For non-proxyless (Envoy), we follow the service's publishNotReadyAddresses setting.
			// But since this is proxyless gRPC, we should always include endpoints.
			// Actually, let's follow Istio's behavior: always include endpoints, let the client decide.
			// The OverrideHostStatus in Cluster config will control whether unhealthy endpoints can be used.

			// Build LbEndpoint
			lbEp := b.buildLbEndpoint(ep)
			if lbEp == nil {
				buildFailedCount++
				filteredCount++
				log.Debugf("BuildClusterLoadAssignment: buildLbEndpoint returned nil for endpoint %s", ep.FirstAddressOrNil())
				continue
			}
			lbEndpoints = append(lbEndpoints, lbEp)
		}
	}

	if len(lbEndpoints) == 0 {
		// CRITICAL: Log at WARN level for proxyless gRPC to help diagnose "weighted-target: no targets to pick from" errors
		logLevel := log.Debugf
		// For proxyless gRPC, log empty endpoints at INFO level to help diagnose connection issues
		// This helps identify when endpoints are not available vs when they're filtered out
		if totalEndpoints > 0 {
			logLevel = log.Warnf // If endpoints exist but were filtered, this is a warning
		} else {
			logLevel = log.Infof // If no endpoints exist at all, this is informational
		}
		logLevel("BuildClusterLoadAssignment: no endpoints found for cluster %s (hostname=%s, port=%d, svcPort.Name='%s', svcPort.Port=%d, totalEndpoints=%d, filteredCount=%d, portNameMismatch=%d, unhealthy=%d, buildFailed=%d)",
			b.clusterName, b.hostname, b.port, svcPort.Name, svcPort.Port, totalEndpoints, filteredCount, portNameMismatchCount, unhealthyCount, buildFailedCount)
		return buildEmptyClusterLoadAssignment(b.clusterName)
	}

	// Create LocalityLbEndpoints with empty locality (default)
	// CRITICAL: Log endpoint health status for debugging
	healthyCount := 0
	unhealthyInLbCount := 0
	drainingCount := 0
	for _, lbEp := range lbEndpoints {
		switch lbEp.HealthStatus {
		case core.HealthStatus_HEALTHY:
			healthyCount++
		case core.HealthStatus_UNHEALTHY:
			unhealthyInLbCount++
		case core.HealthStatus_DRAINING:
			drainingCount++
		}
	}
	log.Infof("BuildClusterLoadAssignment: cluster %s (hostname=%s, port=%d) - total endpoints: %d (healthy=%d, unhealthy=%d, draining=%d, SupportsUnhealthyEndpoints=%v)",
		b.clusterName, b.hostname, b.port, len(lbEndpoints), healthyCount, unhealthyInLbCount, drainingCount, b.service.SupportsUnhealthyEndpoints())

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
	if b.subsetName == "" {
		return true
	}
	if b.service == nil || b.push == nil {
		return true
	}
	selector := b.push.SubsetLabelsForHost(b.service.Attributes.Namespace, b.hostname, b.subsetName)
	if len(selector) == 0 {
		// No subset labels defined, treat as match-all
		return true
	}
	return selector.SubsetOf(epLabels)
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
	// CRITICAL FIX: Following Istio's pattern, empty ClusterLoadAssignment should have empty Endpoints list
	// This ensures gRPC proxyless clients receive the update and clear their endpoint cache,
	// preventing "weighted-target: no targets to pick from" errors
	return &endpoint.ClusterLoadAssignment{
		ClusterName: clusterName,
		Endpoints:   []*endpoint.LocalityLbEndpoints{}, // Explicitly empty, not nil
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
