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

package istio

import (
	"fmt"
	"strconv"

	"github.com/apache/dubbo-kubernetes/pkg/config/host"
	"github.com/apache/dubbo-kubernetes/pkg/config/protocol"
	"github.com/apache/dubbo-kubernetes/sail/pkg/model"
	"github.com/apache/dubbo-kubernetes/sail/pkg/serviceregistry/provider"
	"k8s.io/klog/v2"
)

// ServiceConverter handles conversion between Istio and Dubbo service representations
type ServiceConverter struct {
	clusterID string
}

// NewServiceConverter creates a new service converter
func NewServiceConverter(clusterID string) *ServiceConverter {
	return &ServiceConverter{
		clusterID: clusterID,
	}
}

// ConvertToDubboService converts an IstioServiceInfo to a Dubbo Service
func (sc *ServiceConverter) ConvertToDubboService(istioService *IstioServiceInfo) (*model.Service, error) {
	if istioService == nil {
		return nil, fmt.Errorf("istio service cannot be nil")
	}

	// Create hostname from service info
	hostname := host.Name(istioService.Hostname)

	// Convert ports
	ports := make(model.PortList, 0, len(istioService.Ports))
	for _, istioPort := range istioService.Ports {
		protocol := protocol.Parse(istioPort.Protocol)
		port := &model.Port{
			Name:     istioPort.Name,
			Port:     istioPort.Port,
			Protocol: protocol,
		}
		ports = append(ports, port)
	}

	// Create service attributes
	attributes := model.ServiceAttributes{
		Name:            istioService.Hostname,
		Namespace:       istioService.Namespace,
		ServiceRegistry: provider.Istio,
		Labels:          make(map[string]string),
	}

	// Copy labels from Istio service
	if istioService.Labels != nil {
		for k, v := range istioService.Labels {
			attributes.Labels[k] = v
		}
	}

	// Add Istio-specific labels
	attributes.Labels["istio.io/service"] = istioService.Hostname
	attributes.Labels["istio.io/namespace"] = istioService.Namespace

	// Create the Dubbo service
	dubboService := &model.Service{
		Hostname:   hostname,
		Ports:      ports,
		Attributes: attributes,
	}

	// Add Istio-specific configurations if available
	if istioService.VirtualService != nil {
		sc.applyVirtualServiceConfig(dubboService, istioService.VirtualService)
	}

	if istioService.DestinationRule != nil {
		sc.applyDestinationRuleConfig(dubboService, istioService.DestinationRule)
	}

	klog.V(4).Infof("Converted Istio service %s to Dubbo service with %d ports",
		istioService.Hostname, len(ports))

	return dubboService, nil
}

// ConvertToIstioService converts a Dubbo Service to an IstioServiceInfo
func (sc *ServiceConverter) ConvertToIstioService(dubboService *model.Service) (*IstioServiceInfo, error) {
	if dubboService == nil {
		return nil, fmt.Errorf("dubbo service cannot be nil")
	}

	// Convert ports
	ports := make([]IstioPortInfo, 0, len(dubboService.Ports))
	for _, dubboPort := range dubboService.Ports {
		port := IstioPortInfo{
			Name:     dubboPort.Name,
			Port:     dubboPort.Port,
			Protocol: string(dubboPort.Protocol),
		}
		ports = append(ports, port)
	}

	// Create Istio service info
	istioService := &IstioServiceInfo{
		Hostname:  string(dubboService.Hostname),
		Namespace: dubboService.Attributes.Namespace,
		Ports:     ports,
		Labels:    make(map[string]string),
	}

	// Copy labels from Dubbo service
	if dubboService.Attributes.Labels != nil {
		for k, v := range dubboService.Attributes.Labels {
			istioService.Labels[k] = v
		}
	}

	klog.V(4).Infof("Converted Dubbo service %s to Istio service with %d ports",
		dubboService.Hostname, len(ports))

	return istioService, nil
}

// applyVirtualServiceConfig applies VirtualService configuration to the Dubbo service
func (sc *ServiceConverter) applyVirtualServiceConfig(dubboService *model.Service, vs *VirtualService) {
	if vs == nil || vs.Spec == nil {
		return
	}

	// Add VirtualService metadata to labels
	dubboService.Attributes.Labels["istio.io/virtual-service"] = vs.Name
	dubboService.Attributes.Labels["istio.io/virtual-service-namespace"] = vs.Namespace

	// Process routing rules if available
	if http, ok := vs.Spec["http"].([]interface{}); ok {
		sc.processHTTPRoutes(dubboService, http)
	}

	if tcp, ok := vs.Spec["tcp"].([]interface{}); ok {
		sc.processTCPRoutes(dubboService, tcp)
	}

	klog.V(5).Infof("Applied VirtualService %s/%s configuration to service %s",
		vs.Namespace, vs.Name, dubboService.Hostname)
}

// applyDestinationRuleConfig applies DestinationRule configuration to the Dubbo service
func (sc *ServiceConverter) applyDestinationRuleConfig(dubboService *model.Service, dr *DestinationRule) {
	if dr == nil || dr.Spec == nil {
		return
	}

	// Add DestinationRule metadata to labels
	dubboService.Attributes.Labels["istio.io/destination-rule"] = dr.Name
	dubboService.Attributes.Labels["istio.io/destination-rule-namespace"] = dr.Namespace

	// Process load balancing configuration
	if trafficPolicy, ok := dr.Spec["trafficPolicy"].(map[string]interface{}); ok {
		sc.processTrafficPolicy(dubboService, trafficPolicy)
	}

	// Process subsets configuration
	if subsets, ok := dr.Spec["subsets"].([]interface{}); ok {
		sc.processSubsets(dubboService, subsets)
	}

	klog.V(5).Infof("Applied DestinationRule %s/%s configuration to service %s",
		dr.Namespace, dr.Name, dubboService.Hostname)
}

// processHTTPRoutes processes HTTP routing rules from VirtualService
func (sc *ServiceConverter) processHTTPRoutes(dubboService *model.Service, routes []interface{}) {
	for i, route := range routes {
		if routeMap, ok := route.(map[string]interface{}); ok {
			routeKey := fmt.Sprintf("istio.io/http-route-%d", i)

			// Process match conditions
			if match, ok := routeMap["match"].([]interface{}); ok && len(match) > 0 {
				dubboService.Attributes.Labels[routeKey+"-match"] = "true"
			}

			// Process route destinations
			if routeDestinations, ok := routeMap["route"].([]interface{}); ok {
				dubboService.Attributes.Labels[routeKey+"-destinations"] = strconv.Itoa(len(routeDestinations))
			}
		}
	}
}

// processTCPRoutes processes TCP routing rules from VirtualService
func (sc *ServiceConverter) processTCPRoutes(dubboService *model.Service, routes []interface{}) {
	for i, route := range routes {
		if routeMap, ok := route.(map[string]interface{}); ok {
			routeKey := fmt.Sprintf("istio.io/tcp-route-%d", i)

			// Process route destinations
			if routeDestinations, ok := routeMap["route"].([]interface{}); ok {
				dubboService.Attributes.Labels[routeKey+"-destinations"] = strconv.Itoa(len(routeDestinations))
			}
		}
	}
}

// processTrafficPolicy processes traffic policy from DestinationRule
func (sc *ServiceConverter) processTrafficPolicy(dubboService *model.Service, policy map[string]interface{}) {
	// Process load balancer settings
	if lb, ok := policy["loadBalancer"].(map[string]interface{}); ok {
		if simple, ok := lb["simple"].(string); ok {
			dubboService.Attributes.Labels["istio.io/load-balancer"] = simple
		}
	}

	// Process connection pool settings
	if cp, ok := policy["connectionPool"].(map[string]interface{}); ok {
		dubboService.Attributes.Labels["istio.io/connection-pool"] = "configured"

		if tcp, ok := cp["tcp"].(map[string]interface{}); ok {
			if maxConnections, ok := tcp["maxConnections"].(float64); ok {
				dubboService.Attributes.Labels["istio.io/max-connections"] = strconv.Itoa(int(maxConnections))
			}
		}
	}

	// Process outlier detection settings
	if _, ok := policy["outlierDetection"].(map[string]interface{}); ok {
		dubboService.Attributes.Labels["istio.io/outlier-detection"] = "enabled"
	}
}

// processSubsets processes subsets from DestinationRule
func (sc *ServiceConverter) processSubsets(dubboService *model.Service, subsets []interface{}) {
	dubboService.Attributes.Labels["istio.io/subsets-count"] = strconv.Itoa(len(subsets))

	for i, subset := range subsets {
		if subsetMap, ok := subset.(map[string]interface{}); ok {
			subsetKey := fmt.Sprintf("istio.io/subset-%d", i)

			if name, ok := subsetMap["name"].(string); ok {
				dubboService.Attributes.Labels[subsetKey+"-name"] = name
			}

			if labels, ok := subsetMap["labels"].(map[string]interface{}); ok {
				dubboService.Attributes.Labels[subsetKey+"-labels-count"] = strconv.Itoa(len(labels))
			}
		}
	}
}
