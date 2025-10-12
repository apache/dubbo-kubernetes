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

package zookeeper

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/apache/dubbo-kubernetes/pkg/cluster"
	"github.com/apache/dubbo-kubernetes/pkg/config/host"
	"github.com/apache/dubbo-kubernetes/pkg/config/protocol"
	"github.com/apache/dubbo-kubernetes/sail/pkg/model"
	"k8s.io/klog/v2"
)

// ZookeeperService represents a service instance from Zookeeper
type ZookeeperService struct {
	ServiceName string            `json:"serviceName"`
	Group       string            `json:"group"`
	Version     string            `json:"version"`
	IP          string            `json:"ip"`
	Port        int               `json:"port"`
	Protocol    string            `json:"protocol"`
	Path        string            `json:"path"`
	Methods     []string          `json:"methods"`
	Parameters  map[string]string `json:"parameters"`
}

// convertZookeeperServiceToModel converts Zookeeper service data to Dubbo model.Service
func (c *Controller) convertZookeeperServiceToModel(zkService *ZookeeperService) *model.Service {
	hostname := host.Name(zkService.ServiceName)

	// Create service ports based on Zookeeper service data
	ports := make(model.PortList, 0)
	if zkService.Port > 0 {
		// Determine protocol based on Zookeeper service protocol
		protocolInstance := protocol.TCP
		switch strings.ToLower(zkService.Protocol) {
		case "http":
			protocolInstance = protocol.HTTP
		case "https":
			protocolInstance = protocol.HTTPS
		case "grpc":
			protocolInstance = protocol.GRPC
		case "tcp":
			protocolInstance = protocol.TCP
		default:
			protocolInstance = protocol.TCP
		}

		ports = append(ports, &model.Port{
			Name:     zkService.Protocol,
			Port:     zkService.Port,
			Protocol: protocolInstance,
		})
	}

	service := &model.Service{
		Hostname: hostname,
		Ports:    ports,
		Attributes: model.ServiceAttributes{
			Namespace: zkService.Group,
			Name:      zkService.ServiceName,
		},
	}

	// Set cluster VIPs if available
	if zkService.IP != "" {
		// Use cluster ID as the cluster name for VIPs
		service.ClusterVIPs = model.AddressMap{
			Addresses: make(map[cluster.ID][]string),
		}
		service.ClusterVIPs.Addresses[c.clusterID] = []string{zkService.IP}
	}

	klog.V(4).Infof("Converted Zookeeper service %s to model service", zkService.ServiceName)
	return service
}

// fetchServicesFromZookeeper fetches all services from Zookeeper registry
func (c *Controller) fetchServicesFromZookeeper() ([]*ZookeeperService, error) {
	// TODO: Implement actual Zookeeper API calls
	// This would typically involve:
	// 1. Connecting to Zookeeper
	// 2. Querying service nodes under the root path
	// 3. Parsing Dubbo URL format from Zookeeper nodes
	// 4. Converting to ZookeeperService structs

	klog.V(4).Infof("Fetching services from Zookeeper servers: %v", c.config.Servers)

	// Mock implementation - return empty list for now
	// In real implementation, this would:
	// - Use go-zookeeper/zk client
	// - List children of /dubbo/<service-name>/providers
	// - Parse each provider URL to extract service info
	// - Handle Dubbo URL format: dubbo://ip:port/service?params

	return []*ZookeeperService{}, nil
}

// updateServicesFromZookeeper updates the local service cache from Zookeeper
func (c *Controller) updateServicesFromZookeeper() error {
	zkServices, err := c.fetchServicesFromZookeeper()
	if err != nil {
		return fmt.Errorf("failed to fetch services from Zookeeper: %v", err)
	}

	c.servicesLock.Lock()
	defer c.servicesLock.Unlock()

	// Clear existing services
	c.servicesMap = make(map[host.Name]*model.Service)

	// Convert and store Zookeeper services
	for _, zkService := range zkServices {
		modelService := c.convertZookeeperServiceToModel(zkService)
		c.servicesMap[modelService.Hostname] = modelService
	}

	klog.V(2).Infof("Updated %d services from Zookeeper for cluster %s", len(c.servicesMap), c.clusterID)
	return nil
}

// parseDubboURL parses a Dubbo URL from Zookeeper node data
func parseDubboURL(urlStr string) (*ZookeeperService, error) {
	// Dubbo URL format: dubbo://ip:port/service?param1=value1&param2=value2
	u, err := url.Parse(urlStr)
	if err != nil {
		return nil, fmt.Errorf("failed to parse Dubbo URL: %v", err)
	}

	// Extract IP and port
	host := u.Hostname()
	portStr := u.Port()
	port, err := strconv.Atoi(portStr)
	if err != nil {
		return nil, fmt.Errorf("invalid port in Dubbo URL: %s", portStr)
	}

	// Extract service name from path
	serviceName := strings.TrimPrefix(u.Path, "/")

	// Parse query parameters
	params := u.Query()
	group := params.Get("group")
	version := params.Get("version")
	methodsStr := params.Get("methods")

	var methods []string
	if methodsStr != "" {
		methods = strings.Split(methodsStr, ",")
	}

	// Convert query values to map
	parameters := make(map[string]string)
	for key, values := range params {
		if len(values) > 0 {
			parameters[key] = values[0]
		}
	}

	return &ZookeeperService{
		ServiceName: serviceName,
		Group:       group,
		Version:     version,
		IP:          host,
		Port:        port,
		Protocol:    u.Scheme,
		Methods:     methods,
		Parameters:  parameters,
	}, nil
}

// validateZookeeperService validates a Zookeeper service instance
func validateZookeeperService(service *ZookeeperService) error {
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
