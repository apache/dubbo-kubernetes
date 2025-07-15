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
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/metadata/info"
	"dubbo.apache.org/dubbo-go/v3/registry"
	gxset "github.com/dubbogo/gost/container/set"
	"strings"
	"sync"
)

type ApplicationContext struct {
	// InterfaceName []*common.URL
	serviceUrls sync.Map
	// Revision *info.MetadataInfo
	revisionToMetadata sync.Map
	// AppName []registry.ServiceInstance
	allInstances sync.Map
	// Mappings *gxset.HashSet
	mappings sync.Map
}

func NewApplicationContext() *ApplicationContext {
	return &ApplicationContext{}
}

// GetServiceUrls returns the reference to the serviceUrls map with read lock
func (ac *ApplicationContext) GetServiceUrls() map[string][]*common.URL {
	result := make(map[string][]*common.URL)
	ac.serviceUrls.Range(func(key, value interface{}) bool {
		result[key.(string)] = value.([]*common.URL)
		return true
	})
	return result
}

func (ac *ApplicationContext) DeleteServiceUrl(key string, url *common.URL) {
	if v, ok := ac.serviceUrls.Load(key); ok {
		urls := v.([]*common.URL)
		for i, u := range urls {
			if urlEqual(u, url) {
				urls = append(urls[:i], urls[i+1:]...)
				ac.serviceUrls.Store(key, urls)
				return
			}
		}
	}
}

func (ac *ApplicationContext) UpdateServiceUrls(interfaceKey string, url *common.URL) {
	v, _ := ac.serviceUrls.LoadOrStore(interfaceKey, []*common.URL{})
	urls := v.([]*common.URL)
	urls = append(urls, url)
	ac.serviceUrls.Store(interfaceKey, urls)
}

func (ac *ApplicationContext) AddServiceUrls(newServiceUrls map[string][]*common.URL) {
	for k, v := range newServiceUrls {
		var urls []*common.URL
		if existingV, ok := ac.serviceUrls.Load(k); ok {
			urls = existingV.([]*common.URL)
		}

		urlExists := func(newUrl *common.URL) bool {
			for _, url := range urls {
				if urlEqual(url, newUrl) {
					return true
				}
			}
			return false
		}

		if urls == nil {
			ac.serviceUrls.Store(k, v)
		} else {
			for _, newUrl := range v {
				if !urlExists(newUrl) {
					urls = append(urls, newUrl)
				}
			}
			ac.serviceUrls.Store(k, urls)
		}
	}
}

// GetRevisionToMetadata returns the reference to the revisionToMetadata map with read lock
func (ac *ApplicationContext) GetRevisionToMetadata(revision string) *info.MetadataInfo {
	if v, ok := ac.revisionToMetadata.Load(revision); ok {
		return v.(*info.MetadataInfo)
	}
	return nil
}

func (ac *ApplicationContext) UpdateRevisionToMetadata(key string, newKey string, value *info.MetadataInfo) {
	if key == newKey {
		return
	}
	if key != "" {
		ac.revisionToMetadata.Delete(key)
	}
	ac.revisionToMetadata.Store(newKey, value)
}

func (ac *ApplicationContext) DeleteRevisionToMetadata(key string) {
	if key != "" {
		ac.revisionToMetadata.Delete(key)
	}
}

func (ac *ApplicationContext) NewRevisionToMetadata(newRevisionToMetadata map[string]*info.MetadataInfo) {
	for k, v := range newRevisionToMetadata {
		ac.revisionToMetadata.Store(k, v)
	}
}

func (ac *ApplicationContext) GetOldRevision(instance registry.ServiceInstance) string {
	if v, ok := ac.allInstances.Load(instance.GetServiceName()); ok {
		for _, elem := range v.([]registry.ServiceInstance) {
			if instance.GetID() == elem.GetID() {
				return elem.GetMetadata()[constant.ExportedServicesRevisionPropertyName]
			}
		}
	}
	return ""
}

// GetAllInstances returns the reference to the allInstances map with read lock
func (ac *ApplicationContext) GetAllInstances() map[string][]registry.ServiceInstance {
	result := make(map[string][]registry.ServiceInstance)
	ac.allInstances.Range(func(key, value interface{}) bool {
		result[key.(string)] = value.([]registry.ServiceInstance)
		return true
	})
	return result
}

func (ac *ApplicationContext) DeleteAllInstance(key string, instance registry.ServiceInstance) {
	if v, ok := ac.allInstances.Load(key); ok {
		instances := v.([]registry.ServiceInstance)
		for i, serviceInstance := range instances {
			if serviceInstance.GetID() == instance.GetID() {
				instances = append(instances[:i], instances[i+1:]...)
				ac.allInstances.Store(key, instances)
				return
			}
		}
	}
}

func (ac *ApplicationContext) UpdateAllInstances(key string, instance registry.ServiceInstance) {
	if v, ok := ac.allInstances.Load(key); ok {
		instances := v.([]registry.ServiceInstance)
		for i, serviceInstance := range instances {
			if serviceInstance.GetID() == instance.GetID() {
				instances[i] = serviceInstance
				ac.allInstances.Store(key, instances)
				return
			}
		}
	}
	v, _ := ac.allInstances.LoadOrStore(key, []registry.ServiceInstance{})
	instances := v.([]registry.ServiceInstance)
	instances = append(instances, instance)
	ac.allInstances.Store(key, instances)
}

func (ac *ApplicationContext) AddAllInstances(key string, value []registry.ServiceInstance) {
	v, _ := ac.allInstances.LoadOrStore(key, []registry.ServiceInstance{})
	instances := v.([]registry.ServiceInstance)

	instanceExists := func(instance registry.ServiceInstance) bool {
		for i, inst := range instances {
			if inst.GetID() == instance.GetID() {
				instances[i] = instance
				return true
			}
		}
		return false
	}

	for _, inst := range value {
		if !instanceExists(inst) {
			instances = append(instances, inst)
		}
	}

	ac.allInstances.Store(key, instances)
}

func (ac *ApplicationContext) UpdateMapping(mapping map[string]*gxset.HashSet) {
	for k, v := range mapping {
		ac.mappings.Store(k, v)
	}
}

func (ac *ApplicationContext) GetMapping() map[string]*gxset.HashSet {
	result := make(map[string]*gxset.HashSet)
	ac.mappings.Range(func(key, value interface{}) bool {
		result[key.(string)] = value.(*gxset.HashSet)
		return true
	})
	return result
}

func urlEqual(url *common.URL, c *common.URL) bool {
	tmpC := c.Clone()
	tmpURL := url.Clone()

	cGroup := tmpC.GetParam(constant.GroupKey, "")
	urlGroup := tmpURL.GetParam(constant.GroupKey, "")
	cKey := tmpC.Key()
	urlKey := tmpURL.Key()

	if cGroup == constant.AnyValue {
		cKey = strings.Replace(cKey, "group=*", "group="+urlGroup, 1)
	} else if urlGroup == constant.AnyValue {
		urlKey = strings.Replace(urlKey, "group=*", "group="+cGroup, 1)
	}

	// 1. protocol, username, password, ip, port, service name, group, version should be equal
	if cKey != urlKey {
		return false
	}

	// 2. if URL contains enabled key, should be true, or *
	if tmpURL.GetParam(constant.EnabledKey, "true") != "true" && tmpURL.GetParam(constant.EnabledKey, "") != constant.AnyValue {
		return false
	}

	return isMatchCategory(tmpURL.GetParam(constant.CategoryKey, constant.DefaultCategory), tmpC.GetParam(constant.CategoryKey, constant.DefaultCategory))
}

func isMatchCategory(category1 string, category2 string) bool {
	if len(category2) == 0 {
		return category1 == constant.DefaultCategory
	} else if strings.Contains(category2, constant.AnyValue) {
		return true
	} else if strings.Contains(category2, constant.RemoveValuePrefix) {
		return !strings.Contains(category2, constant.RemoveValuePrefix+category1)
	} else {
		return strings.Contains(category2, category1)
	}
}
