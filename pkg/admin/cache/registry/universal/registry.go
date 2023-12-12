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
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/extension"
	dubboRegistry "dubbo.apache.org/dubbo-go/v3/registry"
	"dubbo.apache.org/dubbo-go/v3/remoting"
	"github.com/apache/dubbo-kubernetes/pkg/admin/cache"
	"github.com/apache/dubbo-kubernetes/pkg/admin/cache/registry"
	"github.com/apache/dubbo-kubernetes/pkg/admin/config"
	"github.com/apache/dubbo-kubernetes/pkg/admin/constant"
	util2 "github.com/apache/dubbo-kubernetes/pkg/admin/util"
	"github.com/apache/dubbo-kubernetes/pkg/core/logger"
	gxset "github.com/dubbogo/gost/container/set"
	"net/url"
	"strings"
	"sync"
)

var subscribeUrl *common.URL

func init() {
	registry.AddRegistry("universal", func(u *common.URL) (registry.AdminRegistry, error) {
		delegate, err := extension.GetRegistry(u.Protocol, u)
		if err != nil {
			logger.Error("Error initialize registry instance.")
			return nil, err
		}

		sdUrl := u.Clone()
		sdUrl.AddParam("registry", u.Protocol)
		sdUrl.Protocol = "service-discovery"
		sdDelegate, err := extension.GetServiceDiscovery(sdUrl)
		if err != nil {
			logger.Error("Error initialize service discovery instance.")
			return nil, err
		}
		return NewRegistry(delegate, sdDelegate), nil
	})

	queryParams := url.Values{
		constant.InterfaceKey:  {constant.AnyValue},
		constant.GroupKey:      {constant.AnyValue},
		constant.VersionKey:    {constant.AnyValue},
		constant.ClassifierKey: {constant.AnyValue},
		constant.CategoryKey: {constant.ProvidersCategory +
			"," + constant.ConsumersCategory +
			"," + constant.RoutersCategory +
			"," + constant.ConfiguratorsCategory},
		constant.EnabledKey: {constant.AnyValue},
		constant.CheckKey:   {"false"},
	}
	subscribeUrl, _ = common.NewURL(common.GetLocalIp()+":0",
		common.WithProtocol(constant.AdminProtocol),
		common.WithParams(queryParams),
	)
}

type Registry struct {
	delegate   dubboRegistry.Registry
	sdDelegate dubboRegistry.ServiceDiscovery
}

func NewRegistry(delegate dubboRegistry.Registry, sdDelegate dubboRegistry.ServiceDiscovery) *Registry {
	return &Registry{
		delegate:   delegate,
		sdDelegate: sdDelegate,
	}
}

func (r *Registry) Delegate() dubboRegistry.Registry {
	return r.delegate
}

func (r *Registry) Subscribe() error {
	listener := &notifyListener{}
	go func() {
		err := r.delegate.Subscribe(subscribeUrl, listener)
		if err != nil {
			logger.Error("Failed to subscribe to registry, might not be able to show services of the cluster!")
		}
	}()

	getMappingList := func(group string) (map[string]*gxset.HashSet, error) {
		keys, err := config.MetadataReportCenter.GetConfigKeysByGroup(group)
		if err != nil {
			return nil, err
		}

		list := make(map[string]*gxset.HashSet)
		for k := range keys.Items {
			interfaceKey, _ := k.(string)
			if !(interfaceKey == "org.apache.dubbo.mock.api.MockService") {
				rule, err := config.MetadataReportCenter.GetServiceAppMapping(interfaceKey, group, nil)
				if err != nil {
					return nil, err
				}
				list[interfaceKey] = rule
			}
		}
		return list, nil
	}

	go func() {
		mappings, err := getMappingList("mapping")
		if err != nil {
			logger.Error("Failed to get mapping")
		}
		for interfaceKey, oldApps := range mappings {
			mappingListener := NewMappingListener(oldApps, listener)
			apps, _ := config.MetadataReportCenter.GetServiceAppMapping(interfaceKey, "mapping", mappingListener)
			delSDListener := NewDubboSDNotifyListener(apps)
			for appTmp := range apps.Items {
				app := appTmp.(string)
				instances := r.sdDelegate.GetInstances(app)
				logger.Infof("Synchronized instance notification on subscription, instance list size %s", len(instances))
				if len(instances) > 0 {
					err = delSDListener.OnEvent(&dubboRegistry.ServiceInstancesChangedEvent{
						ServiceName: app,
						Instances:   instances,
					})
					if err != nil {
						logger.Warnf("[ServiceDiscoveryRegistry] ServiceInstancesChangedListenerImpl handle error:%v", err)
					}
				}
			}
			delSDListener.AddListenerAndNotify(interfaceKey, listener)
			err = r.sdDelegate.AddListener(delSDListener)
			if err != nil {
				logger.Warnf("Failed to Add Listener")
			}
		}
	}()

	return nil
}

func (r *Registry) Destroy() error {
	return nil
}

type notifyListener struct {
	mu           sync.Mutex
	urlIdsMapper sync.Map
}

func (l *notifyListener) Notify(event *dubboRegistry.ServiceEvent) {
	var interfaceName string
	l.mu.Lock()
	serviceURL := event.Service
	categories := make(map[string]map[string]map[string]*common.URL)
	category := serviceURL.GetParam(constant.CategoryKey, "")
	if len(category) == 0 {
		if constant.ConsumerSide == serviceURL.GetParam(constant.Side, "") ||
			constant.ConsumerProtocol == serviceURL.Protocol {
			category = constant.ConsumersCategory
		} else {
			category = constant.ProvidersCategory
		}
	}
	if event.Action == remoting.EventTypeDel {
		if services, ok := cache.InterfaceRegistryCache.Load(category); ok {
			if services != nil {
				servicesMap, ok := services.(*sync.Map)
				if !ok {
					// servicesMap type error
					logger.Logger().Error("servicesMap type not *sync.Map")
					return
				}
				group := serviceURL.Group()
				version := serviceURL.Version()
				if constant.AnyValue != group && constant.AnyValue != version {
					deleteURL(servicesMap, serviceURL)
				} else {
					// iterator services
					servicesMap.Range(func(key, value interface{}) bool {
						if util2.GetInterface(key.(string)) == serviceURL.Service() &&
							(constant.AnyValue == group || group == util2.GetGroup(key.(string))) &&
							(constant.AnyValue == version || version == util2.GetVersion(key.(string))) {
							deleteURL(servicesMap, serviceURL)
						}
						return true
					})
				}
			}
		}
	} else {
		interfaceName = serviceURL.Service()
		var services map[string]map[string]*common.URL
		if s, ok := categories[category]; ok {
			services = s
		} else {
			services = make(map[string]map[string]*common.URL)
			categories[category] = services
		}
		service := serviceURL.ServiceKey()
		ids, found := services[service]
		if !found {
			ids = make(map[string]*common.URL)
			services[service] = ids
		}
		if md5, ok := l.urlIdsMapper.Load(serviceURL.Key()); ok {
			ids[md5.(string)] = getURLToAdd(nil, serviceURL)
		} else {
			md5 := util2.Md5_16bit(serviceURL.Key())
			ids[md5] = getURLToAdd(nil, serviceURL)
			l.urlIdsMapper.LoadOrStore(serviceURL.Key(), md5)
		}
	}
	// check categories size
	if len(categories) > 0 {
		for category, value := range categories {
			services, ok := cache.InterfaceRegistryCache.Load(category)
			if ok {
				servicesMap, ok := services.(*sync.Map)
				if !ok {
					// servicesMap type error
					logger.Logger().Error("servicesMap type not *sync.Map")
					return
				}
				// iterator services key set
				servicesMap.Range(func(key, inner any) bool {
					ids, ok := value[key.(string)]
					if util2.GetInterface(key.(string)) == interfaceName && !ok {
						servicesMap.Delete(key)
					} else {
						for k, v := range ids {
							inner.(map[string]*common.URL)[k] = getURLToAdd(inner.(map[string]*common.URL)[k], v)
						}
					}
					return true
				})

				for k, v := range value {
					_, ok := services.(*sync.Map).Load(k)
					if !ok {
						services.(*sync.Map).Store(k, v)
					}
				}
			} else {
				services = &sync.Map{}
				cache.InterfaceRegistryCache.Store(category, services)
				for k, v := range value {
					services.(*sync.Map).Store(k, v)
				}
			}
		}
	}

	l.mu.Unlock()
}

func (l *notifyListener) NotifyAll(events []*dubboRegistry.ServiceEvent, f func()) {
	for _, event := range events {
		l.Notify(event)
	}
}
func getURLToAdd(url *common.URL, newURL *common.URL) *common.URL {
	if url == nil {
		newURL = newURL.Clone()
		if currentType, ok := newURL.GetNonDefaultParam(constant.RegistryType); !ok {
			newURL.SetParam(constant.RegistryType, constant.RegistryInterface)
		} else {
			newURL.SetParam(constant.RegistryType, strings.ToUpper(currentType))
		}
	} else {
		currentType, _ := url.GetNonDefaultParam(constant.RegistryType)
		changedType, _ := newURL.GetNonDefaultParam(constant.RegistryType)
		if currentType == constant.RegistryAll || currentType != changedType {
			newURL = newURL.Clone()
			newURL.SetParam(constant.RegistryType, constant.RegistryAll)
		}
	}
	return newURL
}

func deleteURL(servicesMap *sync.Map, serviceURL *common.URL) {
	if innerServices, ok := servicesMap.Load(serviceURL.Service()); ok {
		innerServicesMap := innerServices.(map[string]*common.URL)
		if len(innerServicesMap) > 0 {
			for _, url := range innerServicesMap {
				currentType, _ := url.GetNonDefaultParam(constant.RegistryType)
				changedType := serviceURL.GetParam(constant.RegistryType, constant.RegistryInterface)
				if currentType == constant.RegistryAll {
					if changedType == constant.RegistryInstance {
						url.SetParam(constant.RegistryType, constant.RegistryInterface)
					} else {
						url.SetParam(constant.RegistryType, constant.RegistryInstance)
					}
				} else if currentType == changedType {
					servicesMap.Delete(serviceURL.Service())
				}
			}
		}
	}
}
