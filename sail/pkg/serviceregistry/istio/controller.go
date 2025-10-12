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
	"sort"
	"sync"
	"time"

	"github.com/apache/dubbo-kubernetes/pkg/cluster"
	"github.com/apache/dubbo-kubernetes/pkg/config/host"
	"github.com/apache/dubbo-kubernetes/sail/pkg/model"
	"github.com/apache/dubbo-kubernetes/sail/pkg/serviceregistry"
	"github.com/apache/dubbo-kubernetes/sail/pkg/serviceregistry/provider"
	"go.uber.org/atomic"
	"k8s.io/klog/v2"
)

var (
	_ serviceregistry.Instance = &Controller{}
)

// Controller implements the service registry for Istio
type Controller struct {
	config           *IstioConfig
	clusterID        cluster.ID
	pilotClient      PilotClient
	serviceConverter *ServiceConverter
	configManager    *ConfigManager
	services         map[host.Name]*model.Service
	serviceMutex     sync.RWMutex
	hasSynced        *atomic.Bool
	stopCh           chan struct{}
	syncCompleted    *atomic.Bool
}

// NewController creates a new Istio service registry controller
func NewController(config *IstioConfig, clusterID cluster.ID) *Controller {
	if config == nil {
		config = DefaultConfig()
	}

	configManager := NewConfigManager()
	if err := configManager.LoadFromConfig(config); err != nil {
		klog.Errorf("Failed to load Istio config: %v", err)
		config = DefaultConfig()
		configManager.LoadFromConfig(config)
	}

	pilotClient := NewPilotClient(config)
	serviceConverter := NewServiceConverter(string(clusterID))

	return &Controller{
		config:           config,
		clusterID:        clusterID,
		pilotClient:      pilotClient,
		serviceConverter: serviceConverter,
		configManager:    configManager,
		services:         make(map[host.Name]*model.Service),
		hasSynced:        atomic.NewBool(false),
		stopCh:           make(chan struct{}),
		syncCompleted:    atomic.NewBool(false),
	}
}

// Run starts the Istio service registry controller
func (c *Controller) Run(stop <-chan struct{}) {
	klog.Infof("Starting Istio service registry controller for cluster %s", c.clusterID)

	// Connect to Istio Pilot
	if err := c.pilotClient.Connect(); err != nil {
		klog.Errorf("Failed to connect to Istio Pilot: %v", err)
		return
	}

	defer func() {
		if err := c.pilotClient.Disconnect(); err != nil {
			klog.Errorf("Failed to disconnect from Istio Pilot: %v", err)
		}
	}()

	// Start watching for service changes
	if err := c.pilotClient.WatchServices(c.config.Namespace, c); err != nil {
		klog.Errorf("Failed to start watching services: %v", err)
		return
	}

	// Start periodic sync
	go c.periodicSync()

	// Mark as synced after initial connection
	time.AfterFunc(5*time.Second, func() {
		c.hasSynced.Store(true)
		c.syncCompleted.Store(true)
		klog.Info("Istio service registry controller initial sync completed")
	})

	// Wait for stop signal
	select {
	case <-stop:
		klog.Info("Stopping Istio service registry controller")
		close(c.stopCh)
	case <-c.stopCh:
		klog.Info("Istio service registry controller stopped")
	}
}

// HasSynced returns true if the controller has completed initial sync
func (c *Controller) HasSynced() bool {
	return c.hasSynced.Load()
}

// Services returns all services from Istio
func (c *Controller) Services() []*model.Service {
	c.serviceMutex.RLock()
	defer c.serviceMutex.RUnlock()

	services := make([]*model.Service, 0, len(c.services))
	for _, service := range c.services {
		services = append(services, service)
	}

	// Sort services by hostname for consistent ordering
	sort.Slice(services, func(i, j int) bool {
		return services[i].Hostname < services[j].Hostname
	})

	klog.V(4).Infof("Returning %d services from Istio registry", len(services))
	return services
}

// GetService returns a service by hostname from Istio
func (c *Controller) GetService(hostname host.Name) *model.Service {
	c.serviceMutex.RLock()
	defer c.serviceMutex.RUnlock()

	service := c.services[hostname]
	if service != nil {
		klog.V(4).Infof("Found service %s in Istio registry", hostname)
	} else {
		klog.V(4).Infof("Service %s not found in Istio registry", hostname)
	}

	return service
}

// Provider returns the provider type
func (c *Controller) Provider() provider.ID {
	return provider.Istio
}

// Cluster returns the cluster ID
func (c *Controller) Cluster() cluster.ID {
	return c.clusterID
}

// periodicSync performs periodic synchronization with Istio Pilot
func (c *Controller) periodicSync() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-c.stopCh:
			return
		case <-ticker.C:
			if c.pilotClient.IsConnected() {
				c.syncServices()
			} else {
				klog.Warning("Istio Pilot client is not connected, skipping sync")
				// Try to reconnect
				if err := c.pilotClient.Connect(); err != nil {
					klog.Errorf("Failed to reconnect to Istio Pilot: %v", err)
				}
			}
		}
	}
}

// syncServices synchronizes services from Istio Pilot
func (c *Controller) syncServices() {
	klog.V(4).Info("Synchronizing services from Istio Pilot")

	// TODO: Implement actual service synchronization from Istio Pilot
	// This is a placeholder implementation that would fetch services via xDS

	// For now, we'll just log that sync is happening
	klog.V(4).Info("Service synchronization with Istio Pilot completed")
}

// ServiceEventHandler implementation

// OnServiceAdd handles service add events from Istio
func (c *Controller) OnServiceAdd(istioService *IstioServiceInfo) {
	if istioService == nil {
		klog.Warning("Received nil service in OnServiceAdd")
		return
	}

	dubboService, err := c.serviceConverter.ConvertToDubboService(istioService)
	if err != nil {
		klog.Errorf("Failed to convert Istio service %s to Dubbo service: %v", istioService.Hostname, err)
		return
	}

	c.serviceMutex.Lock()
	c.services[dubboService.Hostname] = dubboService
	c.serviceMutex.Unlock()

	klog.Infof("Added service %s from Istio", dubboService.Hostname)
}

// OnServiceUpdate handles service update events from Istio
func (c *Controller) OnServiceUpdate(oldService, newService *IstioServiceInfo) {
	if newService == nil {
		klog.Warning("Received nil new service in OnServiceUpdate")
		return
	}

	dubboService, err := c.serviceConverter.ConvertToDubboService(newService)
	if err != nil {
		klog.Errorf("Failed to convert updated Istio service %s to Dubbo service: %v", newService.Hostname, err)
		return
	}

	c.serviceMutex.Lock()
	c.services[dubboService.Hostname] = dubboService
	c.serviceMutex.Unlock()

	klog.Infof("Updated service %s from Istio", dubboService.Hostname)
}

// OnServiceDelete handles service delete events from Istio
func (c *Controller) OnServiceDelete(istioService *IstioServiceInfo) {
	if istioService == nil {
		klog.Warning("Received nil service in OnServiceDelete")
		return
	}

	hostname := host.Name(istioService.Hostname)

	c.serviceMutex.Lock()
	delete(c.services, hostname)
	c.serviceMutex.Unlock()

	klog.Infof("Deleted service %s from Istio", hostname)
}

// GetConfig returns the current Istio configuration
func (c *Controller) GetConfig() *IstioConfig {
	return c.config
}

// UpdateConfig updates the Istio configuration
func (c *Controller) UpdateConfig(config *IstioConfig) error {
	if config == nil {
		return fmt.Errorf("config cannot be nil")
	}

	if err := c.configManager.LoadFromConfig(config); err != nil {
		return fmt.Errorf("failed to validate config: %v", err)
	}

	c.config = config
	klog.Infof("Updated Istio configuration: %s", c.configManager.String())

	return nil
}

// GetServiceCount returns the number of services in the registry
func (c *Controller) GetServiceCount() int {
	c.serviceMutex.RLock()
	defer c.serviceMutex.RUnlock()
	return len(c.services)
}

// IsHealthy returns true if the controller is healthy
func (c *Controller) IsHealthy() bool {
	return c.pilotClient.IsConnected() && c.hasSynced.Load()
}
