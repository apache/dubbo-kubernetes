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

package universal

import (
	"sync"

	"dubbo.apache.org/dubbo-go/v3/common"
	"github.com/apache/dubbo-kubernetes/pkg/admin/cache"
	"github.com/apache/dubbo-kubernetes/pkg/admin/cache/selector"
	"github.com/apache/dubbo-kubernetes/pkg/admin/constant"
	"github.com/apache/dubbo-kubernetes/pkg/admin/model"
	"github.com/apache/dubbo-kubernetes/pkg/admin/util"
)

var UniversalCacheInstance *UniversalCache

func NewUniversalCache() *UniversalCache {
	return &UniversalCache{
		providers: &cacheMap{
			data: make(map[string]map[string]map[string]*DubboModel),
		},
		consumers: &cacheMap{
			data: make(map[string]map[string]map[string]*DubboModel),
		},
		idCache: sync.Map{},
	}
}

type UniversalCache struct {
	providers *cacheMap
	consumers *cacheMap
	idCache   sync.Map
}

// cacheMap is a cache container for dubbo provider or consumer, before reading or writing its data, you need to lock it.
type cacheMap struct {
	data map[string]map[string]map[string]*DubboModel // application -> serviceKey -> serviceId -> model
	lock sync.RWMutex
}

// set is used to store a DubboModel in cacheMap
func (cm *cacheMap) set(applicationName, serviceKey, serviceId string, model *DubboModel) {
	cm.lock.Lock()
	defer cm.lock.Unlock()
	applicationMap := cm.data
	if _, ok := applicationMap[applicationName]; !ok {
		applicationMap[applicationName] = map[string]map[string]*DubboModel{}
	}
	serviceMap := applicationMap[applicationName]
	if _, ok := serviceMap[serviceKey]; !ok {
		serviceMap[serviceKey] = map[string]*DubboModel{}
	}
	instanceMap := serviceMap[serviceKey]
	instanceMap[serviceId] = model
}

// get is used to get a DubboModel from cacheMap
func (cm *cacheMap) get(applicationName, serviceKey, serviceId string) (actual *DubboModel, ok bool) {
	cm.lock.RLock()
	defer cm.lock.RUnlock()
	applicationMap := cm.data
	if serviceMap, ok := applicationMap[applicationName]; !ok {
		return nil, false
	} else {
		if instanceMap, ok := serviceMap[serviceKey]; !ok {
			return nil, false
		} else {
			return instanceMap[serviceId], true
		}
	}
}

// delete is used to delete a DubboModel from cacheMap
func (cm *cacheMap) delete(applicationName, serviceKey, serviceId string) {
	cm.lock.Lock()
	defer cm.lock.Unlock()
	applicationMap := cm.data
	if serviceMap, ok := applicationMap[applicationName]; ok {
		if instanceMap, ok := serviceMap[serviceKey]; ok {
			delete(instanceMap, serviceId)
		}
	}
}

// GetApplications returns all applications in the registry.
func (uc *UniversalCache) GetApplications(namespace string) ([]*cache.ApplicationModel, error) {
	applicationSet := map[string]struct{}{} // it's used to deduplicate

	uc.providers.lock.RLock()
	for name := range uc.providers.data {
		applicationSet[name] = struct{}{}
	}
	uc.providers.lock.RUnlock()

	uc.consumers.lock.RLock()
	for name := range uc.consumers.data {
		applicationSet[name] = struct{}{}
	}
	uc.consumers.lock.RUnlock()

	applications := make([]*cache.ApplicationModel, 0, len(applicationSet))
	for name := range applicationSet {
		applications = append(applications, &cache.ApplicationModel{Name: name})
	}

	return applications, nil
}

func (uc *UniversalCache) GetWorkloads(namespace string) ([]*cache.WorkloadModel, error) {
	return []*cache.WorkloadModel{}, nil
}

func (uc *UniversalCache) GetWorkloadsWithSelector(namespace string, selector selector.Selector) ([]*cache.WorkloadModel, error) {
	return []*cache.WorkloadModel{}, nil
}

// GetInstances returns all instances in the registry.
//
// An instance is a URL record in the registry, and the key of instance is IP + Port.
func (uc *UniversalCache) GetInstances(namespace string) ([]*cache.InstanceModel, error) {
	res := make([]*cache.InstanceModel, 0)
	instanceSet := map[string]struct{}{} // it's used to deduplicate

	uc.providers.lock.RLock()
	for application, serviceMap := range uc.providers.data {
		for serviceKey, instanceMap := range serviceMap {
			for _, dubboModel := range instanceMap {
				if _, ok := instanceSet[dubboModel.Ip+":"+dubboModel.Port]; ok {
					continue
				} else {
					instanceSet[dubboModel.Ip+":"+dubboModel.Port] = struct{}{}
					res = append(res, &cache.InstanceModel{
						Application: &cache.ApplicationModel{Name: application},
						Workload:    nil,
						Name:        serviceKey + "#" + dubboModel.Ip + ":" + dubboModel.Port,
						Ip:          dubboModel.Ip,
						Port:        dubboModel.Port,
						Status:      "",
						Node:        "",
						Labels:      nil,
					})
				}
			}
		}
	}
	uc.providers.lock.RUnlock()

	uc.consumers.lock.RLock()
	for application, serviceMap := range uc.consumers.data {
		for serviceKey, instanceMap := range serviceMap {
			for _, dubboModel := range instanceMap {
				if _, ok := instanceSet[dubboModel.Ip+":"+dubboModel.Port]; ok {
					continue
				} else {
					instanceSet[dubboModel.Ip+":"+dubboModel.Port] = struct{}{}
					res = append(res, &cache.InstanceModel{
						Application: &cache.ApplicationModel{Name: application},
						Workload:    nil,
						Name:        serviceKey + "#" + dubboModel.Ip + ":" + dubboModel.Port,
						Ip:          dubboModel.Ip,
						Port:        dubboModel.Port,
						Status:      "",
						Node:        "",
						Labels:      nil,
					})
				}
			}
		}
	}
	uc.consumers.lock.RUnlock()

	return res, nil
}

func (uc *UniversalCache) GetInstancesWithSelector(namespace string, selector selector.Selector) ([]*cache.InstanceModel, error) {
	res := make([]*cache.InstanceModel, 0)
	instanceSet := map[string]struct{}{}

	uc.providers.lock.RLock()
	for application, serviceMap := range uc.providers.data {
		if targetApplication, ok := selector.ApplicationOption(); ok && targetApplication != application {
			continue
		} else {
			for serviceKey, instanceMap := range serviceMap {
				for _, dubboModel := range instanceMap {
					if _, ok := instanceSet[dubboModel.Ip+":"+dubboModel.Port]; ok {
						continue
					} else {
						instanceSet[dubboModel.Ip+":"+dubboModel.Port] = struct{}{}
						res = append(res, &cache.InstanceModel{
							Application: &cache.ApplicationModel{Name: application},
							Workload:    nil,
							Name:        serviceKey + "#" + dubboModel.Ip + ":" + dubboModel.Port,
							Ip:          dubboModel.Ip,
							Port:        dubboModel.Port,
							Status:      "",
							Node:        "",
							Labels:      nil,
						})
					}
				}
			}
		}
	}
	uc.providers.lock.RUnlock()

	uc.consumers.lock.RLock()
	for application, serviceMap := range uc.consumers.data {
		if targetApplication, ok := selector.ApplicationOption(); ok && targetApplication != application {
			continue
		} else {
			for serviceKey, instanceMap := range serviceMap {
				for _, dubboModel := range instanceMap {
					if _, ok := instanceSet[dubboModel.Ip+":"+dubboModel.Port]; ok {
						continue
					} else {
						instanceSet[dubboModel.Ip+":"+dubboModel.Port] = struct{}{}
						res = append(res, &cache.InstanceModel{
							Application: &cache.ApplicationModel{Name: application},
							Workload:    nil,
							Name:        serviceKey + "#" + dubboModel.Ip + ":" + dubboModel.Port,
							Ip:          dubboModel.Ip,
							Port:        dubboModel.Port,
							Status:      "",
							Node:        "",
							Labels:      nil,
						})
					}
				}
			}
		}
	}
	uc.consumers.lock.RUnlock()

	return res, nil
}

func (uc *UniversalCache) GetServices(namespace string) ([]*cache.ServiceModel, error) {
	res := make([]*cache.ServiceModel, 0)

	uc.providers.lock.RLock()
	for application, serviceMap := range uc.providers.data {
		for serviceKey := range serviceMap {
			res = append(res, &cache.ServiceModel{
				Application: &cache.ApplicationModel{Name: application},
				Category:    constant.ProviderSide,
				Name:        util.GetInterface(serviceKey),
				Labels:      nil,
				ServiceKey:  serviceKey,
				Group:       util.GetGroup(serviceKey),
				Version:     util.GetVersion(serviceKey),
			})
		}
	}
	uc.providers.lock.RUnlock()

	uc.consumers.lock.RLock()
	for application, serviceMap := range uc.consumers.data {
		for serviceKey := range serviceMap {
			res = append(res, &cache.ServiceModel{
				Application: &cache.ApplicationModel{Name: application},
				Category:    constant.ConsumerSide,
				Name:        util.GetInterface(serviceKey),
				Labels:      nil,
				ServiceKey:  serviceKey,
				Group:       util.GetGroup(serviceKey),
				Version:     util.GetVersion(serviceKey),
			})
		}
	}
	uc.consumers.lock.RUnlock()

	return res, nil
}

func (uc *UniversalCache) GetServicesWithSelector(namespace string, selector selector.Selector) ([]*cache.ServiceModel, error) {
	res := make([]*cache.ServiceModel, 0)

	uc.providers.lock.RLock()
	for application, serviceMap := range uc.providers.data {
		if targetApplication, ok := selector.ApplicationOption(); ok && targetApplication != application {
			continue
		} else {
			for serviceKey := range serviceMap {
				res = append(res, &cache.ServiceModel{
					Application: &cache.ApplicationModel{Name: application},
					Category:    constant.ProviderSide,
					Name:        util.GetInterface(serviceKey),
					Labels:      nil,
					ServiceKey:  serviceKey,
					Group:       util.GetGroup(serviceKey),
					Version:     util.GetVersion(serviceKey),
				})
			}
		}
	}
	uc.providers.lock.RUnlock()

	uc.consumers.lock.RLock()
	for application, serviceMap := range uc.consumers.data {
		if targetApplication, ok := selector.ApplicationOption(); ok && targetApplication != application {
			continue
		} else {
			for serviceKey := range serviceMap {
				res = append(res, &cache.ServiceModel{
					Application: &cache.ApplicationModel{Name: application},
					Category:    constant.ConsumerSide,
					Name:        util.GetInterface(serviceKey),
					Labels:      nil,
					ServiceKey:  serviceKey,
					Group:       util.GetGroup(serviceKey),
					Version:     util.GetVersion(serviceKey),
				})
			}
		}
	}
	uc.consumers.lock.RUnlock()

	return res, nil
}

func (uc *UniversalCache) getId(key string) string {
	id, _ := uc.idCache.LoadOrStore(key, util.Md5_16bit(key))
	return id.(string)
}

func (uc *UniversalCache) store(url *common.URL) {
	if url == nil {
		return
	}
	category := url.GetParam(constant.CategoryKey, "")
	application := url.GetParam(constant.ApplicationKey, "")
	serviceKey := url.ServiceKey()
	serviceId := uc.getId(url.Key())

	dubboModel := &DubboModel{}
	switch category {
	case constant.ProvidersCategory:
		provider := &model.Provider{}
		provider.InitByUrl(serviceId, url)
		dubboModel.InitByProvider(provider, url)
		uc.providers.set(application, serviceKey, serviceId, dubboModel)
	case constant.ConsumersCategory:
		consumer := &model.Consumer{}
		consumer.InitByUrl(serviceId, url)
		dubboModel.InitByConsumer(consumer, url)
		uc.consumers.set(application, serviceKey, serviceId, dubboModel)
	default:
		return
	}
}

func (uc *UniversalCache) delete(url *common.URL) {
	if url == nil {
		return
	}
	category := url.GetParam(constant.CategoryKey, "")
	application := url.GetParam(constant.ApplicationKey, "")
	serviceKey := url.ServiceKey()
	serviceId := uc.getId(url.Key())

	var targetCache *cacheMap
	switch category {
	case constant.ProvidersCategory:
		targetCache = uc.providers
	case constant.ConsumersCategory:
		targetCache = uc.consumers
	}

	// used to determine whether to delete or change the registry type
	isDeleteOrChangeRegisterType := func(actual *DubboModel, deleteRegistryType string) bool {
		if actual.RegistryType == deleteRegistryType {
			return true
		}
		if actual.RegistryType == constant.RegistryAll {
			actual.ToggleRegistryType(deleteRegistryType)
		}
		return false
	}

	registryType := url.GetParam(constant.RegistryType, constant.RegistryInstance)
	group := url.Group()
	version := url.Version()
	if group != constant.AnyValue && version != constant.AnyValue {
		// delete by serviceKey and serviceId
		if actual, ok := targetCache.get(application, serviceKey, serviceId); ok {
			if isDeleteOrChangeRegisterType(actual, registryType) {
				targetCache.delete(application, serviceKey, serviceId)
			}
		}
	} else {
		// support delete by wildcard search
		wildcardDeleteFunc := func(serviceMap map[string]map[string]*DubboModel) {
			for serviceKey := range serviceMap {
				if util.GetInterface(serviceKey) == url.Service() &&
					(group == constant.AnyValue || group == util.GetGroup(serviceKey)) &&
					(version == constant.AnyValue || version == util.GetVersion(serviceKey)) {
					deleteIds := make([]string, 0)
					for id, m := range serviceMap[serviceKey] {
						if isDeleteOrChangeRegisterType(m, registryType) {
							deleteIds = append(deleteIds, id)
						}
					}
					for _, id := range deleteIds {
						targetCache.delete(application, serviceKey, id)
					}
				}
			}
		}

		if application != "" {
			serviceMap := targetCache.data[application]
			wildcardDeleteFunc(serviceMap)
		} else {
			for _, serviceMap := range targetCache.data {
				wildcardDeleteFunc(serviceMap)
			}
		}
	}
}

// DubboModel is a dubbo provider or consumer in registry
type DubboModel struct {
	Application  string
	Category     string // provider or consumer
	ServiceKey   string // service key
	Group        string
	Version      string
	Protocol     string
	Ip           string
	Port         string
	RegistryType string

	Provider *model.Provider
	Consumer *model.Consumer
}

func (m *DubboModel) InitByProvider(provider *model.Provider, url *common.URL) {
	m.Provider = provider

	m.Category = constant.ProviderSide
	m.ServiceKey = provider.Service
	m.Group = url.Group()
	m.Version = url.Version()
	m.Application = provider.Application
	m.Protocol = url.Protocol
	m.Ip = url.Ip
	m.Port = url.Port
	m.RegistryType = url.GetParam(constant.RegistryType, constant.RegistryInstance)
}

func (m *DubboModel) InitByConsumer(consumer *model.Consumer, url *common.URL) {
	m.Consumer = consumer

	m.Category = constant.ConsumerSide
	m.ServiceKey = consumer.Service
	m.Group = url.Group()
	m.Version = url.Version()
	m.Application = consumer.Application
	m.Protocol = url.Protocol
	m.Ip = url.Ip
	m.Port = url.Port
	m.RegistryType = url.GetParam(constant.RegistryType, constant.RegistryInstance)
}

func (m *DubboModel) ToggleRegistryType(deleteType string) {
	if m.RegistryType != constant.RegistryAll {
		return
	}
	if deleteType == constant.RegistryInstance {
		m.RegistryType = constant.RegistryInterface
	} else {
		m.RegistryType = constant.RegistryInstance
	}
}
