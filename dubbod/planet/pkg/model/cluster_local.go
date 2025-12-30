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

package model

import (
	"strings"
	"sync"

	"github.com/apache/dubbo-kubernetes/pkg/config/host"
)

var (
	defaultClusterLocalNamespaces = []string{"kube-system"}
	defaultClusterLocalServices   = []string{"kubernetes.default.svc"}
)

type clusterLocalProvider struct {
	mutex sync.RWMutex
	hosts ClusterLocalHosts
}

type ClusterLocalHosts struct {
	specific map[host.Name]bool
	wildcard map[host.Name]bool
}

type ClusterLocalProvider interface {
	GetClusterLocalHosts() ClusterLocalHosts
}

func NewClusterLocalProvider(e *Environment) ClusterLocalProvider {
	c := &clusterLocalProvider{}

	// Register a handler to update the environment when the mesh config is updated.
	e.AddMeshHandler(func() {
		c.onMeshUpdated(e)
	})

	// Update the cluster-local hosts now.
	c.onMeshUpdated(e)
	return c
}

func (c *clusterLocalProvider) GetClusterLocalHosts() ClusterLocalHosts {
	c.mutex.RLock()
	out := c.hosts
	c.mutex.RUnlock()
	return out
}

func (c *clusterLocalProvider) onMeshUpdated(e *Environment) {
	// Create the default list of cluster-local hosts.
	domainSuffix := e.DomainSuffix
	defaultClusterLocalHosts := make([]host.Name, 0)
	for _, n := range defaultClusterLocalNamespaces {
		defaultClusterLocalHosts = append(defaultClusterLocalHosts, host.Name("*."+n+".svc."+domainSuffix))
	}
	for _, s := range defaultClusterLocalServices {
		defaultClusterLocalHosts = append(defaultClusterLocalHosts, host.Name(s+"."+domainSuffix))
	}

	if discoveryHost, _, err := e.GetDiscoveryAddress(); err != nil {
		log.Errorf("failed to make discoveryAddress cluster-local: %v", err)
	} else {
		if !strings.HasSuffix(string(discoveryHost), domainSuffix) {
			discoveryHost += host.Name("." + domainSuffix)
		}
		defaultClusterLocalHosts = append(defaultClusterLocalHosts, discoveryHost)
	}

	// Collect the cluster-local hosts.
	hosts := ClusterLocalHosts{
		specific: make(map[host.Name]bool),
		wildcard: make(map[host.Name]bool),
	}

	// ServiceSettings field is not available in current MeshGlobalConfig
	// If needed, this functionality should be implemented when the field is added
	// for _, serviceSettings := range e.Mesh().ServiceSettings {
	// 	isClusterLocal := serviceSettings.GetSettings().GetClusterLocal()
	// 	for _, h := range serviceSettings.GetHosts() {
	// 		// If clusterLocal false, check to see if we should remove a default clusterLocal host.
	// 		if !isClusterLocal {
	// 			for i, defaultClusterLocalHost := range defaultClusterLocalHosts {
	// 				if len(defaultClusterLocalHost) > 0 {
	// 					if h == string(defaultClusterLocalHost) ||
	// 						(defaultClusterLocalHost.IsWildCarded() &&
	// 							strings.HasSuffix(h, string(defaultClusterLocalHost[1:]))) {
	// 						// This default was explicitly overridden, so remove it.
	// 						defaultClusterLocalHosts[i] = ""
	// 					}
	// 				}
	// 			}
	// 		}
	// 	}
	//
	// 	// Add hosts with their clusterLocal setting to sets.
	// 	for _, h := range serviceSettings.GetHosts() {
	// 		hostname := host.Name(h)
	// 		if hostname.IsWildCarded() {
	// 			hosts.wildcard[hostname] = isClusterLocal
	// 		} else {
	// 			hosts.specific[hostname] = isClusterLocal
	// 		}
	// 	}
	// }

	// Add any remaining defaults to the end of the list.
	for _, defaultClusterLocalHost := range defaultClusterLocalHosts {
		if len(defaultClusterLocalHost) > 0 {
			if defaultClusterLocalHost.IsWildCarded() {
				hosts.wildcard[defaultClusterLocalHost] = true
			} else {
				hosts.specific[defaultClusterLocalHost] = true
			}
		}
	}

	c.mutex.Lock()
	c.hosts = hosts
	c.mutex.Unlock()
}
