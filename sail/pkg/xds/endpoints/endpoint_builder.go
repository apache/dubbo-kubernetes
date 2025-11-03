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
	"github.com/apache/dubbo-kubernetes/sail/pkg/model"
	"github.com/apache/dubbo-kubernetes/sail/pkg/networking/util"
	core "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	endpoint "github.com/envoyproxy/go-control-plane/envoy/config/endpoint/v3"
	"google.golang.org/protobuf/types/known/wrapperspb"
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
		return buildEmptyClusterLoadAssignment(b.clusterName)
	}

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
	for _, eps := range shards.Shards {
		for _, ep := range eps {
			// Filter by port name
			if ep.ServicePortName != svcPort.Name {
				continue
			}

			// Filter by subset labels if subset is specified
			if b.subsetName != "" && !b.matchesSubset(ep.Labels) {
				continue
			}

			// Filter out unhealthy endpoints unless service supports them
			if !b.service.SupportsUnhealthyEndpoints() && ep.HealthStatus == model.UnHealthy {
				continue
			}

			// Build LbEndpoint
			lbEp := b.buildLbEndpoint(ep)
			if lbEp != nil {
				lbEndpoints = append(lbEndpoints, lbEp)
			}
		}
	}

	if len(lbEndpoints) == 0 {
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
	// Simple key based on cluster name for now
	// TODO: implement proper hashing if needed for cache optimization
	return b.clusterName
}

// Type implements model.XdsCacheEntry
func (b *EndpointBuilder) Type() string {
	return "EDS"
}

// DependentConfigs implements model.XdsCacheEntry
func (b *EndpointBuilder) DependentConfigs() []model.ConfigHash {
	// Return empty slice for now - can be enhanced to return ServiceEntry config hash if needed
	return nil
}
