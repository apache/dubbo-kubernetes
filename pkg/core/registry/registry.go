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
	"net/url"
	"sync"
	"time"

	"github.com/go-co-op/gocron"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/metadata/report"
	dubboRegistry "dubbo.apache.org/dubbo-go/v3/registry"

	gxset "github.com/dubbogo/gost/container/set"
)

import (
	"github.com/apache/dubbo-kubernetes/pkg/core/consts"
	"github.com/apache/dubbo-kubernetes/pkg/core/logger"
	core_manager "github.com/apache/dubbo-kubernetes/pkg/core/resources/manager"
	"github.com/apache/dubbo-kubernetes/pkg/events"
)

type Registry struct {
	delegate   dubboRegistry.Registry
	sdDelegate dubboRegistry.ServiceDiscovery
	ctx        *ApplicationContext
}

func NewRegistry(delegate dubboRegistry.Registry, sdDelegate dubboRegistry.ServiceDiscovery) *Registry {
	return &Registry{
		delegate:   delegate,
		sdDelegate: sdDelegate,
		ctx:        NewApplicationContext(),
	}
}

func (r *Registry) Destroy() error {
	return nil
}

func (r *Registry) Delegate() dubboRegistry.Registry {
	return r.delegate
}

func (r *Registry) Subscribe(
	metadataReport report.MetadataReport,
	resourceManager core_manager.ResourceManager,
	cache *sync.Map,
	discovery dubboRegistry.ServiceDiscovery,
	out events.Emitter,
	systemNamespace string,
) error {
	queryParams := url.Values{
		consts.InterfaceKey:  {consts.AnyValue},
		consts.GroupKey:      {consts.AnyValue},
		consts.VersionKey:    {consts.AnyValue},
		consts.ClassifierKey: {consts.AnyValue},
		consts.CategoryKey: {consts.ProvidersCategory +
			"," + consts.ConsumersCategory +
			"," + consts.RoutersCategory +
			"," + consts.ConfiguratorsCategory},
		consts.EnabledKey: {consts.AnyValue},
		consts.CheckKey:   {"false"},
	}
	subscribeUrl, _ := common.NewURL(common.GetLocalIp()+":0",
		common.WithProtocol(consts.AdminProtocol),
		common.WithParams(queryParams))

	listener := NewNotifyListener(resourceManager, cache, out, r.ctx)

	interfaceListener := NewInterfaceServiceChangedNotifyListener(listener, r.ctx)
	scheduler := gocron.NewScheduler(time.UTC)
	_, err := scheduler.Every(5).Second().Do(func() {
		err := r.delegate.Subscribe(subscribeUrl, interfaceListener)
		if err != nil {
			logger.Error("Failed to subscribe to registry, might not be able to show services of the cluster!")
		}
	})
	scheduler.StartAsync()
	if err != nil {
		logger.Error("Failed to start registry interface services scheduler")
	}

	getMappingList := func(group string) (map[string]*gxset.HashSet, error) {
		keys, err := metadataReport.GetConfigKeysByGroup(group)
		if err != nil {
			return nil, err
		}

		list := make(map[string]*gxset.HashSet)
		for k := range keys.Items {
			interfaceKey, _ := k.(string)
			if !(interfaceKey == "org.apache.dubbo.mock.api.MockService") {
				rule, err := metadataReport.GetServiceAppMapping(interfaceKey, group, nil)
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
			mappingListener := NewMappingListener(interfaceKey, oldApps, listener, out, systemNamespace, r.sdDelegate, r.ctx)
			apps, _ := metadataReport.GetServiceAppMapping(interfaceKey, "mapping", mappingListener)
			delSDListener := NewDubboSDNotifyListener(apps, r.ctx)
			for appTmp := range apps.Items {
				app := appTmp.(string)
				instances := r.sdDelegate.GetInstances(app)
				logger.Infof("Synchronized instance notification on subscription, instance list size %d", len(instances))
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
