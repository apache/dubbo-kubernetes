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

	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/registry"
)

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
		if u.URLEqual(url) {
			ac.serviceUrls[key] = append(urls[:i], urls[i+1:]...)
			return
		}
	}
}

func (ac *ApplicationContext) UpdateServiceUrls(interfaceKey string, url *common.URL) {
	urls := ac.serviceUrls[interfaceKey]
	if urls == nil {
		ac.serviceUrls[interfaceKey] = make([]*common.URL, 0)
	}
	for _, oldUrl := range urls {
		if oldUrl.URLEqual(url) {
			return
		}
	}
	urls = append(urls, url)
}

func (ac *ApplicationContext) NewServiceUrls(newServiceUrls map[string][]*common.URL) {
	ac.serviceUrls = newServiceUrls
}

// GetRevisionToMetadata returns the reference to the revisionToMetadata map with read lock
func (ac *ApplicationContext) GetRevisionToMetadata(revision string) *common.MetadataInfo {
	return ac.revisionToMetadata[revision]
}

func (ac *ApplicationContext) UpdateRevisionToMetadata(key string, newKey string, value *common.MetadataInfo) {
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

// SetRevisionToMetadata sets a value in the revisionToMetadata map with write lock
func (ac *ApplicationContext) SetRevisionToMetadata(key string, value *common.MetadataInfo) {
	ac.revisionToMetadata[key] = value
}

func (ac *ApplicationContext) NewRevisionToMetadata(newRevisionToMetadata map[string]*common.MetadataInfo) {
	ac.revisionToMetadata = newRevisionToMetadata
}

// GetAllInstances returns the reference to the allInstances map with read lock
func (ac *ApplicationContext) GetAllInstances() map[string][]registry.ServiceInstance {
	return ac.allInstances
}

func (ac *ApplicationContext) DeleteAllInstances(key string, instance registry.ServiceInstance) {
	instances := ac.allInstances[key]
	for i, serviceInstance := range instances {
		if serviceInstance.GetID() == instance.GetID() {
			if serviceInstance.GetID() == instance.GetID() {
				ac.allInstances[key] = append(instances[:i], instances[i+1:]...)
				return
			}
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
	instances = append(instances, instance)
}

// SetAllInstances sets a value in the allInstances map with write lock
func (ac *ApplicationContext) SetAllInstances(key string, value []registry.ServiceInstance) {
	ac.allInstances[key] = value
}
