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

package controller

import (
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/apache/dubbo-kubernetes/pkg/cluster"
	"github.com/apache/dubbo-kubernetes/pkg/config/host"
	"github.com/apache/dubbo-kubernetes/pkg/config/mesh"
	"github.com/apache/dubbo-kubernetes/pkg/config/mesh/meshwatcher"
	kubelib "github.com/apache/dubbo-kubernetes/pkg/kube"
	"github.com/apache/dubbo-kubernetes/pkg/kube/krt"
	"github.com/apache/dubbo-kubernetes/pkg/queue"
	"github.com/apache/dubbo-kubernetes/sail/pkg/model"
	"github.com/apache/dubbo-kubernetes/sail/pkg/serviceregistry"
	"github.com/apache/dubbo-kubernetes/sail/pkg/serviceregistry/aggregate"
	"github.com/apache/dubbo-kubernetes/sail/pkg/serviceregistry/kube/annotations"
	"github.com/apache/dubbo-kubernetes/sail/pkg/serviceregistry/provider"
	"go.uber.org/atomic"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/klog/v2"
)

var (
	_ serviceregistry.Instance = &Controller{}
)

type Controller struct {
	opts   Options
	client kubelib.Client
	sync.RWMutex
	servicesMap         map[host.Name]*model.Service
	queue               queue.Instance
	initialSyncTimedout *atomic.Bool

	// k8s Service discovery fields
	serviceInformer   cache.SharedIndexInformer
	endpointsInformer cache.SharedIndexInformer
	k8sServices       map[string]*model.Service
}

type Options struct {
	KubernetesAPIQPS      float32
	KubernetesAPIBurst    int
	DomainSuffix          string
	XDSUpdater            model.XDSUpdater
	MeshNetworksWatcher   mesh.NetworksWatcher
	MeshWatcher           meshwatcher.WatcherCollection
	ClusterID             cluster.ID
	ClusterAliases        map[string]string
	SystemNamespace       string
	MeshServiceController *aggregate.Controller
	KrtDebugger           *krt.DebugHandler
	SyncTimeout           time.Duration

	// k8s Service discovery options
	EnableK8sServiceDiscovery bool
	K8sServiceNamespaces      []string
	DubboAnnotationPrefix     string
}

func (c *Controller) Services() []*model.Service {
	c.RLock()
	out := make([]*model.Service, 0, len(c.servicesMap))
	for _, svc := range c.servicesMap {
		out = append(out, svc)
	}
	c.RUnlock()
	sort.Slice(out, func(i, j int) bool { return out[i].Hostname < out[j].Hostname })
	return out
}

// GetService implements a service catalog operation by hostname specified.
func (c *Controller) GetService(hostname host.Name) *model.Service {
	c.RLock()
	svc := c.servicesMap[hostname]
	c.RUnlock()
	return svc
}

func (c *Controller) Provider() provider.ID {
	return provider.Kubernetes
}

func (c *Controller) Cluster() cluster.ID {
	return c.opts.ClusterID
}

func (c *Controller) Run(stop <-chan struct{}) {
	if c.opts.SyncTimeout != 0 {
		time.AfterFunc(c.opts.SyncTimeout, func() {
			if !c.queue.HasSynced() {
				klog.Warningf("kube controller for %s initial sync timed out", c.opts.ClusterID)
				c.initialSyncTimedout.Store(true)
			}
		})
	}
	st := time.Now()

	kubelib.WaitForCacheSync("kube controller", stop, c.informersSynced)
	klog.Infof("kube controller for %s synced after %v", c.opts.ClusterID, time.Since(st))

	// after the in-order sync we can start processing the queue
	c.queue.Run(stop)
	klog.Infof("Controller terminated")
}

func (c *Controller) HasSynced() bool {
	if c.initialSyncTimedout.Load() {
		return true
	}
	return c.queue.HasSynced()
}

func (c *Controller) informersSynced() bool {
	return false
}

// convertK8sServiceToDubboService converts a Kubernetes Service to Dubbo Service
func (c *Controller) convertK8sServiceToDubboService(k8sService *corev1.Service) *model.Service {
	if k8sService == nil {
		return nil
	}

	// Check if this is a Dubbo service using the annotation parser
	if !annotations.IsDubboService(k8sService) {
		return nil
	}

	// Parse Dubbo annotations
	dubboInfo, err := annotations.ParseDubboAnnotations(k8sService)
	if err != nil {
		klog.Warningf("Failed to parse Dubbo annotations for service %s/%s: %v",
			k8sService.Namespace, k8sService.Name, err)
		return nil
	}

	// Build Dubbo service
	dubboService := &model.Service{
		Hostname:     host.Name(dubboInfo.ServiceName),
		CreationTime: k8sService.CreationTimestamp.Time,
		Attributes: model.ServiceAttributes{
			Name:                dubboInfo.ServiceName,
			Namespace:           k8sService.Namespace,
			ServiceRegistry:     provider.Kubernetes,
			KubernetesService:   k8sService,
			KubernetesNamespace: k8sService.Namespace,
			DubboAnnotations:    make(map[string]string),
		},
	}

	// Set Dubbo-specific annotations
	dubboService.Attributes.DubboAnnotations["version"] = dubboInfo.Version
	dubboService.Attributes.DubboAnnotations["group"] = dubboInfo.Group
	dubboService.Attributes.DubboAnnotations["protocol"] = dubboInfo.Protocol
	if dubboInfo.Port > 0 {
		dubboService.Attributes.DubboAnnotations["port"] = fmt.Sprintf("%d", dubboInfo.Port)
	}

	// Convert service ports
	ports := make(model.PortList, 0, len(k8sService.Spec.Ports))
	for _, port := range k8sService.Spec.Ports {
		dubboPort := &model.Port{
			Name: port.Name,
			Port: int(port.Port),
		}
		ports = append(ports, dubboPort)
	}
	dubboService.Ports = ports

	return dubboService
}

// handleServiceEvent handles Kubernetes Service events
func (c *Controller) handleServiceEvent(eventType string, obj interface{}) {
	service, ok := obj.(*corev1.Service)
	if !ok {
		klog.Warningf("Expected Service object, got %T", obj)
		return
	}

	dubboService := c.convertK8sServiceToDubboService(service)
	if dubboService == nil {
		return // Not a Dubbo service
	}

	c.Lock()
	defer c.Unlock()

	serviceKey := service.Namespace + "/" + service.Name

	switch eventType {
	case "add", "update":
		c.k8sServices[serviceKey] = dubboService
		c.servicesMap[dubboService.Hostname] = dubboService
		klog.Infof("Added/Updated Dubbo service from k8s: %s", dubboService.Hostname)
	case "delete":
		if existingService, exists := c.k8sServices[serviceKey]; exists {
			delete(c.k8sServices, serviceKey)
			delete(c.servicesMap, existingService.Hostname)
			klog.Infof("Deleted Dubbo service from k8s: %s", existingService.Hostname)
		}
	}
}
