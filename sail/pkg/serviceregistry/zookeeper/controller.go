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

// Controller implements Zookeeper service registry
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

// NewController creates a new Zookeeper service registry controller
func NewController(config *Config, clusterID cluster.ID) *Controller {
	if err := config.Validate(); err != nil {
		klog.Errorf("Invalid Zookeeper config: %v", err)
		return nil
	}

	return &Controller{
		config:      config,
		clusterID:   clusterID,
		servicesMap: make(map[host.Name]*model.Service),
		stopCh:      make(chan struct{}),
	}
}

// Run starts the Zookeeper controller
func (c *Controller) Run(stop <-chan struct{}) {
	klog.Infof("Starting Zookeeper service registry controller for cluster %s", c.clusterID)

	c.running = true
	defer func() {
		if r := recover(); r != nil {
			klog.Errorf("Zookeeper controller for cluster %s panicked: %v", c.clusterID, r)
		}
		c.running = false
		klog.Infof("Zookeeper service registry controller for cluster %s stopped", c.clusterID)
	}()

	// Initialize Zookeeper client connection with retry
	const maxRetries = 3
	var initErr error
	for retry := 0; retry < maxRetries; retry++ {
		if initErr = c.initZookeeperClient(); initErr == nil {
			break
		}
		klog.Warningf("Failed to initialize Zookeeper client (attempt %d/%d): %v", retry+1, maxRetries, initErr)
		if retry < maxRetries-1 {
			select {
			case <-time.After(time.Duration(retry+1) * 2 * time.Second):
				continue
			case <-stop:
				klog.Info("Zookeeper controller stopped during initialization retry")
				return
			}
		}
	}

	if initErr != nil {
		klog.Errorf("Failed to initialize Zookeeper client after %d retries: %v", maxRetries, initErr)
		return
	}

	klog.Infof("Successfully initialized Zookeeper client for cluster %s", c.clusterID)

	// Start service discovery with error recovery
	go func() {
		defer func() {
			if r := recover(); r != nil {
				klog.Errorf("Zookeeper service discovery for cluster %s panicked: %v", c.clusterID, r)
			}
		}()
		c.startServiceDiscovery(stop)
	}()

	<-stop
}

// HasSynced returns true if the controller has synced
func (c *Controller) HasSynced() bool {
	return c.running
}

// Services returns all services from Zookeeper
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
	return provider.Zookeeper
}

// Cluster returns the cluster ID
func (c *Controller) Cluster() cluster.ID {
	return c.clusterID
}

// initZookeeperClient initializes the connection to Zookeeper server
func (c *Controller) initZookeeperClient() error {
	// TODO: implement actual Zookeeper client initialization
	// This would typically involve:
	// 1. Creating Zookeeper connection
	// 2. Setting up authentication if configured
	// 3. Testing connectivity
	// 4. Setting up watchers for service nodes

	klog.Infof("Zookeeper client initialized for servers: %v", c.config.Servers)
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

// syncServices synchronizes services from Zookeeper
func (c *Controller) syncServices() {
	if err := c.updateServicesFromZookeeper(); err != nil {
		klog.Errorf("Failed to sync services from Zookeeper for cluster %s: %v", c.clusterID, err)
	}
}
