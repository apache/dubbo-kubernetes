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

package aggregate

import (
	"sync"

	"github.com/apache/dubbo-kubernetes/dubbod/discovery/pkg/model"
	"github.com/apache/dubbo-kubernetes/dubbod/discovery/pkg/serviceregistry"
	"github.com/apache/dubbo-kubernetes/dubbod/discovery/pkg/serviceregistry/provider"
	"github.com/apache/dubbo-kubernetes/pkg/config/host"
	"github.com/apache/dubbo-kubernetes/pkg/config/mesh"
	"github.com/apache/dubbo-kubernetes/pkg/slices"

	dubbolog "github.com/apache/dubbo-kubernetes/pkg/log"
)

var log = dubbolog.RegisterScope("aggregate", "aggregate controller debugging")

var (
	_ model.ServiceDiscovery    = &Controller{}
	_ model.AggregateController = &Controller{}
)

type Controller struct {
	registries []*registryEntry
	storeLock  sync.RWMutex
	running    bool
	meshHolder mesh.Holder
}

type registryEntry struct {
	serviceregistry.Instance
	stop <-chan struct{}
}

type Options struct {
	MeshHolder mesh.Holder
}

func NewController(opt Options) *Controller {
	return &Controller{
		registries: make([]*registryEntry, 0),
		meshHolder: opt.MeshHolder,
		running:    false,
	}
}

func (c *Controller) Run(stop <-chan struct{}) {
	c.storeLock.Lock()
	for _, r := range c.registries {
		registryStop := stop
		if s := r.stop; s != nil {
			registryStop = s
		}
		go r.Run(registryStop)
	}
	c.running = true
	c.storeLock.Unlock()

	<-stop
	log.Info("Registry Aggregator terminated")
}

func (c *Controller) HasSynced() bool {
	for _, r := range c.GetRegistries() {
		if !r.HasSynced() {
			log.Debugf("registry %s is syncing", r.Cluster())
			return false
		}
	}
	return true
}

func (c *Controller) Services() []*model.Service {
	indices := make(map[host.Name]int)
	services := make([]*model.Service, 0)
	for _, registry := range c.GetRegistries() {
		for _, service := range registry.Services() {
			if index, found := indices[service.Hostname]; found {
				if registry.Provider() == provider.External {
					services[index] = services[index].DeepCopy()
					decorateService(services[index], service)
				}
				continue
			}
			indices[service.Hostname] = len(services)
			services = append(services, service)
		}
	}
	return services
}

func (c *Controller) GetService(hostname host.Name) *model.Service {
	var out *model.Service
	for _, registry := range c.GetRegistries() {
		service := registry.GetService(hostname)
		if service == nil {
			continue
		}
		if out == nil {
			out = service.DeepCopy()
		} else if registry.Provider() == provider.External {
			decorateService(out, service)
		}
	}
	return out
}

func decorateService(dst, src *model.Service) {
	accounts := make(map[string]struct{}, len(dst.ServiceAccounts)+len(src.ServiceAccounts))
	for _, account := range dst.ServiceAccounts {
		accounts[account] = struct{}{}
	}
	for _, account := range src.ServiceAccounts {
		if _, found := accounts[account]; found {
			continue
		}
		dst.ServiceAccounts = append(dst.ServiceAccounts, account)
		accounts[account] = struct{}{}
	}
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

func (c *Controller) AddRegistryAndRun(registry serviceregistry.Instance, stop <-chan struct{}) {
	if stop == nil {
		log.Warnf("nil stop channel passed to AddRegistryAndRun for registry %s/%s", registry.Provider(), registry.Cluster())
	}
	c.storeLock.Lock()
	defer c.storeLock.Unlock()
	c.addRegistry(registry, stop)
	if c.running {
		go registry.Run(stop)
	}
}

func (c *Controller) addRegistry(registry serviceregistry.Instance, stop <-chan struct{}) {
	added := false
	if registry.Provider() == provider.Kubernetes {
		for i, r := range c.registries {
			if r.Provider() != provider.Kubernetes {
				// insert the registry in the position of the first non kubernetes registry
				c.registries = slices.Insert(c.registries, i, &registryEntry{Instance: registry, stop: stop})
				added = true
				break
			}
		}
	}
	if !added {
		c.registries = append(c.registries, &registryEntry{Instance: registry, stop: stop})
	}

}

func (c *Controller) GetProxyServiceTargets(node *model.Proxy) []model.ServiceTarget {
	out := make([]model.ServiceTarget, 0)
	for _, r := range c.GetRegistries() {
		instances := r.GetProxyServiceTargets(node)
		if len(instances) > 0 {
			out = append(out, instances...)
		}
	}

	if len(out) == 0 {
		log.Infof("no service targets found for proxy %s", node.ID)
	}

	return out
}
