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
	"dubbo.apache.org/dubbo-go/v3/metadata/info"
	"dubbo.apache.org/dubbo-go/v3/registry"

	gxset "github.com/dubbogo/gost/container/set"
	"sync"
)

type InterfaceContext struct {
	// InterfaceName []*common.URL
	serviceUrls sync.Map
	// Revision *info.MetadataInfo
	revisionToMetadata sync.Map
	// Service []registry.ServiceInstance
	instances sync.Map
	// Mappings *gxset.HashSet
	mappings sync.Map
}

func NewInterfaceContext() *InterfaceContext {
	return &InterfaceContext{}
}

func (ac *InterfaceContext) GetMetadata(r string) *info.MetadataInfo {
	if v, ok := ac.revisionToMetadata.Load(r); ok {
		return v.(*info.MetadataInfo)
	}
	return nil
}

func (ac *InterfaceContext) GetMapping() map[string]*gxset.HashSet {
	result := make(map[string]*gxset.HashSet)
	ac.mappings.Range(func(key, value interface{}) bool {
		result[key.(string)] = value.(*gxset.HashSet)
		return true
	})
	return result
}

func (ac *InterfaceContext) UpdateMapping(inf string, appName string) {
	apps, _ := ac.mappings.LoadOrStore(inf, gxset.NewSet(appName))
	if set, ok := apps.(*gxset.HashSet); ok {
		set.Add(appName)
	}
}

func (ac *InterfaceContext) AddInstance(key string, instance *registry.DefaultServiceInstance) {
	instancesRaw, _ := ac.instances.LoadOrStore(key, []registry.ServiceInstance{instance})
	instances := instancesRaw.([]registry.ServiceInstance)

	var existing bool
	for _, serviceInstance := range instances {
		if serviceInstance.GetID() == instance.GetID() {
			existing = true
			break
		}
	}

	if !existing {
		instances = append(instances, instance)
		ac.instances.Store(key, instances)
		ac.revisionToMetadata.Store(instance.GetID(), instance.ServiceMetadata)
	}
}

func (ac *InterfaceContext) GetAllInstances() map[string][]registry.ServiceInstance {
	result := make(map[string][]registry.ServiceInstance)
	ac.instances.Range(func(key, value interface{}) bool {
		result[key.(string)] = value.([]registry.ServiceInstance)
		return true
	})
	return result
}

func (ac *InterfaceContext) GetInstances(key string) []registry.ServiceInstance {
	if v, ok := ac.instances.Load(key); ok {
		return v.([]registry.ServiceInstance)
	}
	return nil
}

func (ac *InterfaceContext) GetInstance(key string, addr string) (registry.ServiceInstance, bool) {
	if v, ok := ac.instances.Load(key); ok {
		instances := v.([]registry.ServiceInstance)
		for _, serviceInstance := range instances {
			if serviceInstance.GetID() == addr {
				return serviceInstance, true
			}
		}
	}
	return nil, false
}

func (ac *InterfaceContext) RemoveInstance(key string, addr string) {
	if v, ok := ac.instances.Load(key); ok {
		instances := v.([]registry.ServiceInstance)
		for i, serviceInstance := range instances {
			if serviceInstance.GetID() == addr {
				instances = append(instances[:i], instances[i+1:]...)
				ac.instances.Store(key, instances)
				ac.revisionToMetadata.Delete(addr)
				break
			}
		}
	}
}

func MergeInstances(insMap1 map[string][]registry.ServiceInstance, insMap2 map[string][]registry.ServiceInstance) map[string][]registry.ServiceInstance {
	instances := make(map[string][]registry.ServiceInstance)
	for app, serviceInstances := range insMap1 {
		newServiceInstances := make([]registry.ServiceInstance, 0, len(serviceInstances))
		newServiceInstances = append(newServiceInstances, serviceInstances...)
		instances[app] = newServiceInstances
	}

	for app, serviceInstances := range insMap2 {
		newServiceInstances := make([]registry.ServiceInstance, 0, len(serviceInstances))
		newServiceInstances = append(newServiceInstances, serviceInstances...)
		existingInstances, exists := instances[app]
		if exists {
			for _, i2 := range newServiceInstances {
				find := false
				for _, i3 := range insMap1[app] {
					if i2.GetAddress() == i3.GetAddress() {
						i3.GetMetadata()["registry-type"] = "all"
						find = true
					}
				}

				if !find {
					existingInstances = append(existingInstances, i2)
				}
			}
			instances[app] = existingInstances
		} else {
			instances[app] = newServiceInstances
		}
	}
	return instances
}

func MergeMapping(mapping1 map[string]*gxset.HashSet, mapping2 map[string]*gxset.HashSet) map[string]*gxset.HashSet {
	if len(mapping1) == 0 {
		return mapping2
	}
	if len(mapping2) == 0 {
		return mapping1
	}

	mappings := make(map[string]*gxset.HashSet)
	for k, set := range mapping1 {
		mappings[k] = gxset.NewSet(set.Values()...)
	}
	for k, set := range mapping2 {
		if mappings[k] != nil {
			mappings[k].Add(set.Values()...)
		} else {
			mappings[k] = gxset.NewSet(set.Values()...)
		}
	}
	return mappings
}
