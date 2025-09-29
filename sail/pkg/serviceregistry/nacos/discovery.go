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

package nacos

import (
	"fmt"
	"strings"

	"github.com/apache/dubbo-kubernetes/pkg/cluster"
	"github.com/apache/dubbo-kubernetes/pkg/config/host"
	"github.com/apache/dubbo-kubernetes/pkg/config/protocol"
	"github.com/apache/dubbo-kubernetes/sail/pkg/model"
	"k8s.io/klog/v2"
)

// NacosService represents a service instance from Nacos
type NacosService struct {
	ServiceName string            `json:"serviceName"`
	GroupName   string            `json:"groupName"`
	ClusterName string            `json:"clusterName"`
	IP          string            `json:"ip"`
	Port        uint64            `json:"port"`
	Weight      float64           `json:"weight"`
	Healthy     bool              `json:"healthy"`
	Enable      bool              `json:"enable"`
	Ephemeral   bool              `json:"ephemeral"`
	Metadata    map[string]string `json:"metadata"`
}

// convertNacosServiceToModel converts Nacos service data to Dubbo model.Service
func (c *Controller) convertNacosServiceToModel(nacosService *NacosService) *model.Service {
	hostname := host.Name(nacosService.ServiceName)

	// Create service ports based on Nacos service data
	ports := make(model.PortList, 0)
	if nacosService.Port > 0 {
		ports = append(ports, &model.Port{
			Name:     "http",
			Port:     int(nacosService.Port),
			Protocol: protocol.HTTP,
		})
	}

	service := &model.Service{
		Hostname: hostname,
		Ports:    ports,
		Attributes: model.ServiceAttributes{
			Namespace: nacosService.GroupName,
			Name:      nacosService.ServiceName,
		},
	}

	// Set cluster VIPs if available
	if nacosService.ClusterName != "" {
		// Initialize ClusterVIPs with empty AddressMap
		service.ClusterVIPs = model.AddressMap{
			Addresses: make(map[cluster.ID][]string),
		}
		// Add the IP address for this cluster
		service.ClusterVIPs.Addresses[cluster.ID(nacosService.ClusterName)] = []string{nacosService.IP}
	}

	klog.V(4).Infof("Converted Nacos service %s to model service", nacosService.ServiceName)
	return service
}

// fetchServicesFromNacos fetches all services from Nacos registry
func (c *Controller) fetchServicesFromNacos() ([]*NacosService, error) {
	// TODO: Implement actual Nacos API calls
	// This would typically involve:
	// 1. Making HTTP/gRPC calls to Nacos server
	// 2. Querying all services in the configured namespace/group
	// 3. Parsing response and converting to NacosService structs

	klog.V(4).Infof("Fetching services from Nacos servers: %v", c.config.ServerAddrs)

	// Mock implementation - return empty list for now
	// In real implementation, this would call Nacos client APIs:
	// - naming_client.GetAllServicesInfo()
	// - naming_client.GetService() for each service
	// - Convert Nacos service instances to NacosService structs

	return []*NacosService{}, nil
}

// updateServicesFromNacos updates the local service cache from Nacos
func (c *Controller) updateServicesFromNacos() error {
	nacosServices, err := c.fetchServicesFromNacos()
	if err != nil {
		return fmt.Errorf("failed to fetch services from Nacos: %v", err)
	}

	c.servicesLock.Lock()
	defer c.servicesLock.Unlock()

	// Clear existing services
	c.servicesMap = make(map[host.Name]*model.Service)

	// Convert and store Nacos services
	for _, nacosService := range nacosServices {
		if !nacosService.Healthy || !nacosService.Enable {
			continue // Skip unhealthy or disabled services
		}

		modelService := c.convertNacosServiceToModel(nacosService)
		c.servicesMap[modelService.Hostname] = modelService
	}

	klog.V(2).Infof("Updated %d services from Nacos for cluster %s", len(c.servicesMap), c.clusterID)
	return nil
}

// validateNacosService validates a Nacos service instance
func validateNacosService(service *NacosService) error {
	if service.ServiceName == "" {
		return fmt.Errorf("service name cannot be empty")
	}
	if service.IP == "" {
		return fmt.Errorf("service IP cannot be empty")
	}
	if service.Port == 0 {
		return fmt.Errorf("service port cannot be zero")
	}
	return nil
}

// parseNacosServiceKey parses a Nacos service key into components
func parseNacosServiceKey(serviceKey string) (serviceName, groupName string, err error) {
	parts := strings.Split(serviceKey, "@@")
	if len(parts) < 2 {
		return "", "", fmt.Errorf("invalid Nacos service key format: %s", serviceKey)
	}
	return parts[0], parts[1], nil
}

// buildNacosServiceKey builds a Nacos service key from components
func buildNacosServiceKey(serviceName, groupName string) string {
	return fmt.Sprintf("%s@@%s", serviceName, groupName)
}
