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

package registry

import (
	"sync"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/registry"

	gxset "github.com/dubbogo/gost/container/set"
)

type InterfaceContext struct {
	// InterfaceName Urls
	serviceUrls map[string][]*common.URL
	// Revision Metadata
	revisionToMetadata map[string]*common.MetadataInfo
	instances          map[string][]registry.ServiceInstance

	mappings map[string]*gxset.HashSet

	mu sync.RWMutex
}

func NewInterfaceContext() *InterfaceContext {
	return &InterfaceContext{
		serviceUrls:        make(map[string][]*common.URL),
		revisionToMetadata: make(map[string]*common.MetadataInfo),
		instances:          make(map[string][]registry.ServiceInstance),
	}
}

func (ac *InterfaceContext) GetMetadata(r string) *common.MetadataInfo {
	return ac.revisionToMetadata[r]
}

func (ac *InterfaceContext) GetMapping() map[string]*gxset.HashSet {
	return ac.mappings
}

func (ac *InterfaceContext) UpdateMapping(inf string, appName string) {
	apps := ac.mappings[inf]
	if apps == nil {
		apps = gxset.NewSet(appName)
	} else {
		apps.Add(appName)
	}
}

func (ac *InterfaceContext) AddInstance(key string, instance *registry.DefaultServiceInstance) {
	instances := ac.instances[key]
	if instances == nil {
		instances = make([]registry.ServiceInstance, 0)
		instances = append(instances, instance)
		ac.instances[key] = instances
		ac.revisionToMetadata[instance.GetID()] = instance.ServiceMetadata
		return
	} else {
		var existing bool
		for _, serviceInstance := range instances {
			if serviceInstance.GetID() == instance.GetID() {
				existing = true
				break
			}
		}
		if !existing {
			instances = append(instances, instance)
			ac.instances[key] = instances
			ac.revisionToMetadata[instance.GetID()] = instance.ServiceMetadata
		}
	}
}

func (ac *InterfaceContext) GetAllInstances() map[string][]registry.ServiceInstance {
	return ac.instances
}

func (ac *InterfaceContext) GetInstances(key string) []registry.ServiceInstance {
	return ac.instances[key]
}

func (ac *InterfaceContext) GetInstance(key string, addr string) (registry.ServiceInstance, bool) {
	instances := ac.instances[key]
	if instances != nil {
		for _, serviceInstance := range instances {
			if serviceInstance.GetID() == addr {
				return serviceInstance, true
			}
		}
	}
	return nil, false
}

func (ac *InterfaceContext) RemoveInstance(key string, addr string) {
	instances := ac.instances[key]
	if instances == nil {
		return
	} else {
		for i, serviceInstance := range instances {
			if serviceInstance.GetID() == addr {
				instances = append(instances[:i], instances[i+1:]...)
				delete(ac.revisionToMetadata, addr)
				break
			}
		}
	}
}

func MergeInstances(insMap1 map[string][]registry.ServiceInstance, insMap2 map[string][]registry.ServiceInstance) map[string][]registry.ServiceInstance {
	instances := make(map[string][]registry.ServiceInstance, 0)
	for app, serviceInstances := range insMap1 {
		newServiceInstances := make([]registry.ServiceInstance, 0, len(serviceInstances))
		newServiceInstances = append(newServiceInstances, serviceInstances...)
		instances[app] = newServiceInstances
	}

	for app, serviceInstances := range insMap2 {
		newServiceInstances := make([]registry.ServiceInstance, 0, len(serviceInstances))
		newServiceInstances = append(newServiceInstances, serviceInstances...)
		existingInstances := instances[app]
		if existingInstances != nil {
			for _, i2 := range newServiceInstances {
				for _, i3 := range existingInstances {
					if i2.GetID() == i3.GetID() {
						i3.GetMetadata()["registry-type"] = "all"
					} else {
						existingInstances = append(existingInstances, i2)
					}
				}
			}
		} else {
			instances[app] = newServiceInstances
		}
	}
	return instances
}

func MergeMapping(mapping1 map[string]*gxset.HashSet, mapping2 map[string]*gxset.HashSet) map[string]*gxset.HashSet {
	return mapping1
}
