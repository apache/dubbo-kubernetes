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

package aggregate

import (
	"github.com/apache/dubbo-kubernetes/pkg/cluster"
	"github.com/apache/dubbo-kubernetes/pkg/config/host"
	"github.com/apache/dubbo-kubernetes/pkg/config/mesh"
	"github.com/apache/dubbo-kubernetes/sail/pkg/model"
	"github.com/apache/dubbo-kubernetes/sail/pkg/serviceregistry"
	"github.com/apache/dubbo-kubernetes/sail/pkg/serviceregistry/provider"
	"k8s.io/klog/v2"
	"sync"
)

var (
	_ model.ServiceDiscovery    = &Controller{}
	_ model.AggregateController = &Controller{}
)

type Controller struct {
	registries      []*registryEntry
	storeLock       sync.RWMutex
	running         bool
	meshHolder      mesh.Holder
	configClusterID cluster.ID
}

type registryEntry struct {
	serviceregistry.Instance
	// stop if not nil is the per-registry stop chan. If null, the server stop chan should be used to Run the registry.
	stop <-chan struct{}
}

type Options struct {
	MeshHolder      mesh.Holder
	ConfigClusterID cluster.ID
}

func NewController(opt Options) *Controller {
	return &Controller{
		registries:      make([]*registryEntry, 0),
		configClusterID: opt.ConfigClusterID,
		meshHolder:      opt.MeshHolder,
		running:         false,
	}
}

// Run starts all the controllers
func (c *Controller) Run(stop <-chan struct{}) {
	c.storeLock.Lock()
	for _, r := range c.registries {
		// prefer the per-registry stop channel
		registryStop := stop
		if s := r.stop; s != nil {
			registryStop = s
		}
		go r.Run(registryStop)
	}
	c.running = true
	c.storeLock.Unlock()

	<-stop
	klog.Info("Registry Aggregator terminated")
}

func (c *Controller) HasSynced() bool {
	for _, r := range c.GetRegistries() {
		if !r.HasSynced() {
			klog.V(2).Infof("registry %s is syncing", r.Cluster())
			return false
		}
	}
	return true
}

func (c *Controller) Services() []*model.Service {
	// smap is a map of hostname (string) to service index, used to identify services that
	// are installed in multiple clusters.
	smap := make(map[host.Name]int)
	index := 0
	services := make([]*model.Service, 0)
	// Locking Registries list while walking it to prevent inconsistent results
	for _, r := range c.GetRegistries() {
		svcs := r.Services()
		if r.Provider() != provider.Kubernetes {
			index += len(svcs)
			services = append(services, svcs...)
		} else {
			for _, s := range svcs {
				previous, ok := smap[s.Hostname]
				if !ok {
					// First time we see a service. The result will have a single service per hostname
					// The first cluster will be listed first, so the services in the primary cluster
					// will be used for default settings. If a service appears in multiple clusters,
					// the order is less clear.
					smap[s.Hostname] = index
					index++
					services = append(services, s)
				} else {
					// We must deepcopy before merge, and after merging, the ClusterVips length will be >= 2.
					// This is an optimization to prevent deepcopy multi-times
					if services[previous].ClusterVIPs.Len() < 2 {
						// Deep copy before merging, otherwise there is a case
						// a service in remote cluster can be deleted, but the ClusterIP left.
						services[previous] = services[previous].DeepCopy()
					}
					// If it is seen second time, that means it is from a different cluster, update cluster VIPs.
					mergeService(services[previous], s, r)
				}
			}
		}
	}
	return services
}

func (c *Controller) GetService(hostname host.Name) *model.Service {
	var out *model.Service
	for _, r := range c.GetRegistries() {
		service := r.GetService(hostname)
		if service == nil {
			continue
		}
		if r.Provider() != provider.Kubernetes {
			return service
		}
		if out == nil {
			out = service.DeepCopy()
		} else {
			// If we are seeing the service for the second time, it means it is available in multiple clusters.
			mergeService(out, service, r)
		}
	}
	return out
}

func (c *Controller) GetRegistries() []serviceregistry.Instance {
	c.storeLock.RLock()
	defer c.storeLock.RUnlock()

	// copy registries to prevent race, no need to deep copy here.
	out := make([]serviceregistry.Instance, len(c.registries))
	for i := range c.registries {
		out[i] = c.registries[i]
	}
	return out
}

func mergeService(dst, src *model.Service, srcRegistry serviceregistry.Instance) {
	if !src.Ports.Equals(dst.Ports) {
		klog.V(2).Infof("service %s defined from cluster %s is different from others", src.Hostname, srcRegistry.Cluster())
	}
	// Prefer the k8s HostVIPs where possible
	clusterID := srcRegistry.Cluster()
	if len(dst.ClusterVIPs.GetAddressesFor(clusterID)) == 0 {
		newAddresses := src.ClusterVIPs.GetAddressesFor(clusterID)
		dst.ClusterVIPs.SetAddressesFor(clusterID, newAddresses)
	}
}
