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
	"net/url"

	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/extension"
	dubboRegistry "dubbo.apache.org/dubbo-go/v3/registry"
	"dubbo.apache.org/dubbo-go/v3/remoting"
	"github.com/apache/dubbo-kubernetes/pkg/admin/cache/registry"
	"github.com/apache/dubbo-kubernetes/pkg/admin/config"
	"github.com/apache/dubbo-kubernetes/pkg/admin/constant"
	"github.com/apache/dubbo-kubernetes/pkg/core/kubeclient/client"
	"github.com/apache/dubbo-kubernetes/pkg/core/logger"
	gxset "github.com/dubbogo/gost/container/set"
)

var subscribeUrl *common.URL

func init() {
	registry.AddRegistry("universal", func(u *common.URL, _ *client.KubeClient) (registry.AdminRegistry, error) {
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
		UniversalCacheInstance = NewUniversalCache() // init cache instance before start registry
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

type notifyListener struct{}

func (l *notifyListener) Notify(event *dubboRegistry.ServiceEvent) {
	switch event.Action {
	case remoting.EventTypeAdd, remoting.EventTypeUpdate:
		UniversalCacheInstance.store(event.Service)
	case remoting.EventTypeDel:
		UniversalCacheInstance.delete(event.Service)
	}
}

func (l *notifyListener) NotifyAll(events []*dubboRegistry.ServiceEvent, f func()) {
	for _, event := range events {
		l.Notify(event)
	}
}
