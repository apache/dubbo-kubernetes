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
	"sync"
	"time"

	"github.com/apache/dubbo-kubernetes/pkg/cluster"
	"github.com/apache/dubbo-kubernetes/pkg/config/host"
	"github.com/apache/dubbo-kubernetes/sail/pkg/model"
	"github.com/apache/dubbo-kubernetes/sail/pkg/serviceregistry"
	"github.com/apache/dubbo-kubernetes/sail/pkg/serviceregistry/provider"
	"k8s.io/klog/v2"
)

var (
	_ serviceregistry.Instance = &Controller{}
)

// Controller implements Nacos service registry
type Controller struct {
	config    *Config
	clusterID cluster.ID

	// servicesLock protects servicesMap
	servicesLock sync.RWMutex
	servicesMap  map[host.Name]*model.Service

	// running indicates if the controller is running
	running bool

	// stopCh is used to stop the controller
	stopCh chan struct{}
}

// NewController creates a new Nacos service registry controller
func NewController(config *Config, clusterID cluster.ID) *Controller {
	if err := config.Validate(); err != nil {
		klog.Errorf("Invalid Nacos config: %v", err)
		return nil
	}

	return &Controller{
		config:      config,
		clusterID:   clusterID,
		servicesMap: make(map[host.Name]*model.Service),
		stopCh:      make(chan struct{}),
	}
}

// Run starts the Nacos controller
func (c *Controller) Run(stop <-chan struct{}) {
	klog.Infof("Starting Nacos service registry controller for cluster %s", c.clusterID)

	c.running = true
	defer func() {
		c.running = false
		klog.Infof("Nacos service registry controller for cluster %s stopped", c.clusterID)
	}()

	// Initialize Nacos client connection
	if err := c.initNacosClient(); err != nil {
		klog.Errorf("Failed to initialize Nacos client: %v", err)
		return
	}

	// Start service discovery
	c.startServiceDiscovery(stop)

	<-stop
}

// HasSynced returns true if the controller has synced
func (c *Controller) HasSynced() bool {
	return c.running
}

// Services returns all services from Nacos
func (c *Controller) Services() []*model.Service {
	c.servicesLock.RLock()
	defer c.servicesLock.RUnlock()

	services := make([]*model.Service, 0, len(c.servicesMap))
	for _, service := range c.servicesMap {
		services = append(services, service)
	}
	return services
}

// GetService returns a service by hostname
func (c *Controller) GetService(hostname host.Name) *model.Service {
	c.servicesLock.RLock()
	defer c.servicesLock.RUnlock()

	return c.servicesMap[hostname]
}

// Provider returns the provider type
func (c *Controller) Provider() provider.ID {
	return provider.Nacos
}

// Cluster returns the cluster ID
func (c *Controller) Cluster() cluster.ID {
	return c.clusterID
}

// TODO: Add service and workload event handlers if needed by serviceregistry.Instance interface

// initNacosClient initializes the connection to Nacos server
func (c *Controller) initNacosClient() error {
	// TODO: implement actual Nacos client initialization
	// This would typically involve:
	// 1. Creating Nacos naming client
	// 2. Setting up authentication if configured
	// 3. Testing connectivity

	klog.Infof("Nacos client initialized for servers: %v", c.config.ServerAddrs)
	return nil
}

// startServiceDiscovery starts the service discovery process
func (c *Controller) startServiceDiscovery(stop <-chan struct{}) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	// Initial sync
	c.syncServices()

	for {
		select {
		case <-stop:
			return
		case <-ticker.C:
			c.syncServices()
		}
	}
}

// syncServices synchronizes services from Nacos
func (c *Controller) syncServices() {
	if err := c.updateServicesFromNacos(); err != nil {
		klog.Errorf("Failed to sync services from Nacos for cluster %s: %v", c.clusterID, err)
	}
}
