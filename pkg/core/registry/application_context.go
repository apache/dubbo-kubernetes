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
	"strings"
	"sync"

	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/registry"
)

const AppCtx = "ApplicationContext"

type ApplicationContext struct {
	// InterfaceName Urls
	serviceUrls map[string][]*common.URL
	// Revision Metadata
	revisionToMetadata map[string]*common.MetadataInfo
	// AppName Instances
	allInstances map[string][]registry.ServiceInstance
	mu           sync.RWMutex
}

func NewApplicationContext() *ApplicationContext {
	return &ApplicationContext{
		serviceUrls:        make(map[string][]*common.URL),
		revisionToMetadata: make(map[string]*common.MetadataInfo),
		allInstances:       make(map[string][]registry.ServiceInstance),
	}
}

// GetServiceUrls returns the reference to the serviceUrls map with read lock
func (ac *ApplicationContext) GetServiceUrls() map[string][]*common.URL {
	return ac.serviceUrls
}

func (ac *ApplicationContext) DeleteServiceUrl(key string, url *common.URL) {
	urls := ac.serviceUrls[key]
	if urls == nil {
		return
	}

	for i, u := range urls {
		if urlEqual(u, url) {
			ac.serviceUrls[key] = append(urls[:i], urls[i+1:]...)
			return
		}
	}
}

func (ac *ApplicationContext) UpdateServiceUrls(interfaceKey string, url *common.URL) {
	urls := ac.serviceUrls[interfaceKey]
	urls = append(urls, url)
	ac.serviceUrls[interfaceKey] = urls
}

func (ac *ApplicationContext) AddServiceUrls(newServiceUrls map[string][]*common.URL) {
	for k, v := range newServiceUrls {
		urls := ac.serviceUrls[k]

		urlExists := func(newUrl *common.URL) bool {
			for _, url := range urls {
				if urlEqual(url, newUrl) {
					return true
				}
			}
			return false
		}

		if urls == nil {
			ac.serviceUrls[k] = v
		} else {
			for _, newUrl := range v {
				if !urlExists(newUrl) {
					ac.serviceUrls[k] = append(urls, newUrl)
				}
			}
		}
	}
}

// GetRevisionToMetadata returns the reference to the revisionToMetadata map with read lock
func (ac *ApplicationContext) GetRevisionToMetadata(revision string) *common.MetadataInfo {
	return ac.revisionToMetadata[revision]
}

func (ac *ApplicationContext) UpdateRevisionToMetadata(key string, newKey string, value *common.MetadataInfo) {
	if key == newKey {
		return
	}
	if "" != key {
		delete(ac.revisionToMetadata, key)
	}
	ac.revisionToMetadata[newKey] = value
}

func (ac *ApplicationContext) DeleteRevisionToMetadata(key string) {
	if "" != key {
		delete(ac.revisionToMetadata, key)
	}
}

func (ac *ApplicationContext) NewRevisionToMetadata(newRevisionToMetadata map[string]*common.MetadataInfo) {
	ac.revisionToMetadata = newRevisionToMetadata
}

func (ac *ApplicationContext) GetOldRevision(instance registry.ServiceInstance) string {
	for _, elem := range ac.allInstances[instance.GetServiceName()] {
		if instance.GetID() == elem.GetID() {
			return elem.GetMetadata()[constant.ExportedServicesRevisionPropertyName]
		}
	}
	return ""
}

// GetAllInstances returns the reference to the allInstances map with read lock
func (ac *ApplicationContext) GetAllInstances() map[string][]registry.ServiceInstance {
	return ac.allInstances
}

func (ac *ApplicationContext) DeleteAllInstance(key string, instance registry.ServiceInstance) {
	instances := ac.allInstances[key]
	for i, serviceInstance := range instances {
		if serviceInstance.GetID() == instance.GetID() {
			ac.allInstances[key] = append(instances[:i], instances[i+1:]...)
			return
		}
	}
}

func (ac *ApplicationContext) UpdateAllInstances(key string, instance registry.ServiceInstance) {
	instances := ac.allInstances[key]
	for i, serviceInstance := range instances {
		if serviceInstance.GetID() == instance.GetID() {
			instances[i] = serviceInstance
			return
		}
	}
	ac.allInstances[key] = append(instances, instance)
}

func (ac *ApplicationContext) AddAllInstances(key string, value []registry.ServiceInstance) {
	instances, exists := ac.allInstances[key]
	if !exists {
		ac.allInstances[key] = value
		return
	}

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

	ac.allInstances[key] = instances
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
