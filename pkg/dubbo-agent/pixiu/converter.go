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

package pixiu

import (
	"fmt"
	cluster "github.com/envoyproxy/go-control-plane/envoy/config/cluster/v3"
	endpoint "github.com/envoyproxy/go-control-plane/envoy/config/endpoint/v3"
	listener "github.com/envoyproxy/go-control-plane/envoy/config/listener/v3"
	route "github.com/envoyproxy/go-control-plane/envoy/config/route/v3"
	hcm "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/network/http_connection_manager/v3"
	"gopkg.in/yaml.v3"
)

// ConfigConverter converts xDS configuration to Pixiu configuration
type ConfigConverter struct {
	listeners map[string]*listener.Listener
	routes    map[string]*route.RouteConfiguration
	clusters  map[string]*cluster.Cluster
	endpoints map[string]*endpoint.ClusterLoadAssignment
}

// NewConfigConverter creates a new config converter
func NewConfigConverter() *ConfigConverter {
	return &ConfigConverter{
		listeners: make(map[string]*listener.Listener),
		routes:    make(map[string]*route.RouteConfiguration),
		clusters:  make(map[string]*cluster.Cluster),
		endpoints: make(map[string]*endpoint.ClusterLoadAssignment),
	}
}

// UpdateListener updates listener configuration
func (c *ConfigConverter) UpdateListener(name string, l *listener.Listener) {
	c.listeners[name] = l
}

// UpdateRoute updates route configuration
func (c *ConfigConverter) UpdateRoute(name string, r *route.RouteConfiguration) {
	c.routes[name] = r
}

// UpdateCluster updates cluster configuration
func (c *ConfigConverter) UpdateCluster(name string, cl *cluster.Cluster) {
	c.clusters[name] = cl
}

// UpdateEndpoint updates endpoint configuration
func (c *ConfigConverter) UpdateEndpoint(name string, e *endpoint.ClusterLoadAssignment) {
	c.endpoints[name] = e
}

// ConvertToPixiuConfig converts xDS configuration to Pixiu YAML configuration
func (c *ConfigConverter) ConvertToPixiuConfig() ([]byte, error) {
	pixiuConfig := &PixiuBootstrap{
		StaticResources: PixiuStaticResources{
			Listeners: []*PixiuListener{},
			Clusters:  []*PixiuCluster{},
		},
	}

	// Convert listeners (only HTTP listeners on port 80 for Gateway)
	for name, l := range c.listeners {
		if pixiuListener := c.convertListener(name, l); pixiuListener != nil {
			pixiuConfig.StaticResources.Listeners = append(pixiuConfig.StaticResources.Listeners, pixiuListener)
		}
	}

	// Convert clusters
	for name, cl := range c.clusters {
		if pixiuCluster := c.convertCluster(name, cl); pixiuCluster != nil {
			pixiuConfig.StaticResources.Clusters = append(pixiuConfig.StaticResources.Clusters, pixiuCluster)
		}
	}

	// Marshal to YAML
	data, err := yaml.Marshal(pixiuConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal pixiu config: %v", err)
	}

	return data, nil
}

// convertListener converts Envoy listener to Pixiu listener
func (c *ConfigConverter) convertListener(name string, l *listener.Listener) *PixiuListener {
	// Only process listeners on port 80 (Gateway HTTP listener)
	port := int(l.Address.GetSocketAddress().GetPortValue())
	if port != 80 {
		log.Debugf("Skipping listener %s on port %d (not Gateway port 80)", name, port)
		return nil
	}

	// Extract HTTP connection manager from filter chain
	var hcmFilter *hcm.HttpConnectionManager
	for _, fc := range l.FilterChains {
		for _, f := range fc.Filters {
			if f.Name == "envoy.filters.network.http_connection_manager" {
				hcmAny := f.GetTypedConfig()
				if hcmAny != nil {
					hcmFilter = &hcm.HttpConnectionManager{}
					if err := hcmAny.UnmarshalTo(hcmFilter); err == nil {
						break
					}
				}
			}
		}
		if hcmFilter != nil {
			break
		}
	}

	if hcmFilter == nil {
		log.Debugf("Listener %s does not have HttpConnectionManager filter", name)
		return nil
	}

	routeConfigName := ""
	if rds := hcmFilter.GetRds(); rds != nil {
		routeConfigName = rds.RouteConfigName
	} else if inlineRoute := hcmFilter.GetRouteConfig(); inlineRoute != nil {
		routeConfigName = inlineRoute.Name
	}

	pixiuListener := &PixiuListener{
		Name:        name,
		Address:     PixiuAddress{},
		ProtocolStr: "http",
		FilterChain: PixiuFilterChain{
			Filters: []PixiuFilter{
				{
					Name: "http",
					Config: map[string]interface{}{
						"route_config": map[string]interface{}{
							"name": routeConfigName,
						},
					},
				},
			},
		},
	}

	// Set address
	addr := l.Address.GetSocketAddress()
	if addr != nil {
		pixiuListener.Address = PixiuAddress{
			SocketAddress: PixiuSocketAddress{
				Address:   addr.Address,
				PortValue: int(addr.GetPortValue()),
			},
		}
	}

	return pixiuListener
}

// convertCluster converts Envoy cluster to Pixiu cluster
func (c *ConfigConverter) convertCluster(name string, cl *cluster.Cluster) *PixiuCluster {
	pixiuCluster := &PixiuCluster{
		Name:     name,
		Type:     "EDS", // Pixiu supports EDS, STRICT_DNS, etc.
		LbPolicy: "round_robin",
	}

	// Convert cluster type
	switch cl.GetClusterDiscoveryType().(type) {
	case *cluster.Cluster_Type:
		clusterType := cl.GetType()
		switch clusterType {
		case cluster.Cluster_EDS:
			pixiuCluster.Type = "EDS"
		case cluster.Cluster_STRICT_DNS:
			pixiuCluster.Type = "STRICT_DNS"
		case cluster.Cluster_STATIC:
			pixiuCluster.Type = "STATIC"
		default:
			pixiuCluster.Type = "EDS"
		}
	default:
		// EDS cluster
		if cl.GetEdsClusterConfig() != nil {
			pixiuCluster.Type = "EDS"
		} else {
			pixiuCluster.Type = "EDS"
		}
	}

	// Convert load balancing policy
	switch cl.LbPolicy {
	case cluster.Cluster_ROUND_ROBIN:
		pixiuCluster.LbPolicy = "round_robin"
	case cluster.Cluster_LEAST_REQUEST:
		pixiuCluster.LbPolicy = "least_request"
	case cluster.Cluster_RING_HASH:
		pixiuCluster.LbPolicy = "ring_hash"
	default:
		pixiuCluster.LbPolicy = "round_robin"
	}

	// Convert endpoints
	if cl.LoadAssignment != nil {
		endpoints := c.convertEndpoints(cl.LoadAssignment)
		pixiuCluster.Endpoints = endpoints
	} else if cl.GetClusterType() != nil {
		// EDS cluster - endpoints will be updated separately
		pixiuCluster.Type = "EDS"
	}

	// Convert health check
	if cl.HealthChecks != nil && len(cl.HealthChecks) > 0 {
		hc := cl.HealthChecks[0]
		pixiuCluster.HealthCheck = &PixiuHealthCheck{
			Timeout:            hc.Timeout.AsDuration().String(),
			Interval:           hc.Interval.AsDuration().String(),
			UnhealthyThreshold: int(hc.UnhealthyThreshold.GetValue()),
			HealthyThreshold:   int(hc.HealthyThreshold.GetValue()),
		}
	}

	return pixiuCluster
}

// convertEndpoints converts Envoy endpoints to Pixiu endpoints
func (c *ConfigConverter) convertEndpoints(assignment *endpoint.ClusterLoadAssignment) []PixiuEndpoint {
	var endpoints []PixiuEndpoint

	for _, localityLbEndpoints := range assignment.Endpoints {
		for _, lbEndpoint := range localityLbEndpoints.LbEndpoints {
			if socketAddress := lbEndpoint.GetEndpoint().Address.GetSocketAddress(); socketAddress != nil {
				endpoints = append(endpoints, PixiuEndpoint{
					Address: PixiuAddress{
						SocketAddress: PixiuSocketAddress{
							Address:   socketAddress.Address,
							PortValue: int(socketAddress.GetPortValue()),
						},
					},
				})
			}
		}
	}

	return endpoints
}

// PixiuBootstrap represents Pixiu Bootstrap configuration
type PixiuBootstrap struct {
	StaticResources PixiuStaticResources `yaml:"static_resources" json:"static_resources"`
}

// PixiuStaticResources contains static resources
type PixiuStaticResources struct {
	Listeners []*PixiuListener `yaml:"listeners" json:"listeners"`
	Clusters  []*PixiuCluster  `yaml:"clusters" json:"clusters"`
}

// PixiuListener represents a Pixiu listener
type PixiuListener struct {
	Name        string           `yaml:"name" json:"name"`
	Address     PixiuAddress     `yaml:"address" json:"address"`
	ProtocolStr string           `yaml:"protocol_type" json:"protocol_type"`
	FilterChain PixiuFilterChain `yaml:"filter_chains" json:"filter_chains"`
}

// PixiuAddress represents network address
type PixiuAddress struct {
	SocketAddress PixiuSocketAddress `yaml:"socket_address" json:"socket_address"`
}

// PixiuSocketAddress represents socket address
type PixiuSocketAddress struct {
	Address   string `yaml:"address" json:"address"`
	PortValue int    `yaml:"port_value" json:"port_value"`
}

// PixiuFilterChain represents filter chain
type PixiuFilterChain struct {
	Filters []PixiuFilter `yaml:"filters" json:"filters"`
}

// PixiuFilter represents a filter
type PixiuFilter struct {
	Name   string                 `yaml:"name" json:"name"`
	Config map[string]interface{} `yaml:"config" json:"config"`
}

// PixiuCluster represents a Pixiu cluster
type PixiuCluster struct {
	Name        string            `yaml:"name" json:"name"`
	Type        string            `yaml:"type" json:"type"`
	LbPolicy    string            `yaml:"lb_policy" json:"lb_policy"`
	Endpoints   []PixiuEndpoint   `yaml:"endpoints" json:"endpoints"`
	HealthCheck *PixiuHealthCheck `yaml:"health_check,omitempty" json:"health_check,omitempty"`
}

// PixiuEndpoint represents an endpoint
type PixiuEndpoint struct {
	Address PixiuAddress `yaml:"address" json:"address"`
}

// PixiuHealthCheck represents health check configuration
type PixiuHealthCheck struct {
	Timeout            string `yaml:"timeout" json:"timeout"`
	Interval           string `yaml:"interval" json:"interval"`
	UnhealthyThreshold int    `yaml:"unhealthy_threshold" json:"unhealthy_threshold"`
	HealthyThreshold   int    `yaml:"healthy_threshold" json:"healthy_threshold"`
}
