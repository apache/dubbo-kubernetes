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
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/config_center"
	"net/url"
	"reflect"
	"sync"
	"time"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/metadata/report"
	dubboRegistry "dubbo.apache.org/dubbo-go/v3/registry"

	gxset "github.com/dubbogo/gost/container/set"

	"github.com/go-co-op/gocron"
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
	cfgCenter  config_center.DynamicConfiguration
	ctx        *ApplicationContext
	infCtx     *InterfaceContext
	scheduler  *gocron.Scheduler
}

func NewRegistry(delegate dubboRegistry.Registry, sdDelegate dubboRegistry.ServiceDiscovery, ctx *ApplicationContext, infCtx *InterfaceContext, cfgCenter config_center.DynamicConfiguration) *Registry {
	return &Registry{
		delegate:   delegate,
		sdDelegate: sdDelegate,
		cfgCenter:  cfgCenter,
		ctx:        ctx,
		infCtx:     infCtx,
		scheduler:  gocron.NewScheduler(time.UTC),
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
	out events.Emitter,
	systemNamespace string,
) error {
	listener := NewNotifyListener(resourceManager, cache, out, r.ctx)

	interfaceListener := NewInterfaceServiceChangedNotifyListener(listener, r.infCtx, r)
	r.listenToAllServices(interfaceListener)

	r.subscribeApp(metadataReport, listener, "mapping", out, systemNamespace, r.cfgCenter)

	r.scheduler.StartAsync()

	return nil
}

func (r *Registry) subscribeApp(metadataReport report.MetadataReport, listener *NotifyListener, group string, out events.Emitter,
	systemNamespace string, cfg config_center.DynamicConfiguration,
) {
	_, err := r.scheduler.Every(30 * 60).Second().Do(func() {
		err := doAppSubscribe(r, metadataReport, listener, group, out, systemNamespace, cfg)
		if err != nil {
			logger.Error("Run app subscription scheduling task failed")
		}
	})
	if err != nil {
		logger.Error("Failed to start registry interface services scheduler")
	}
}

func (r *Registry) listenToAllServices(notifyListener *InterfaceServiceChangedNotifyListener) {
	// 构建查询参数
	queryParams := url.Values{
		consts.InterfaceKey:  {consts.AnyValue},
		consts.GroupKey:      {consts.AnyValue},
		consts.VersionKey:    {consts.AnyValue},
		consts.ClassifierKey: {consts.AnyValue},
		consts.CategoryKey: {
			consts.ProvidersCategory + constant.CommaSeparator +
				consts.ConsumersCategory,
		},
		consts.EnabledKey: {consts.AnyValue},
		consts.CheckKey:   {"false"},
	}

	subscribeUrl, _ := common.NewURL(common.GetLocalIp()+":0",
		common.WithProtocol(consts.AdminProtocol),
		common.WithParams(queryParams))

	executeSubscribe := func() {
		err := r.delegate.Subscribe(subscribeUrl, notifyListener)
		if err != nil {
			logger.Error("Failed to subscribe to registry, might not be able to show services of the cluster!")
		}
	}

	if reflect.TypeOf(r.delegate).String() == "*zookeeper.zkRegistry" {
		executeSubscribe()
	} else {
		_, err := r.scheduler.Every(30 * 60).Second().Do(executeSubscribe)
		if err != nil {
			logger.Error("Failed to start registry interface services scheduler")
		}
	}
}

func doAppSubscribe(r *Registry, metadataReport report.MetadataReport, listener *NotifyListener, group string, out events.Emitter,
	systemNamespace string, cfg config_center.DynamicConfiguration,
) error {
	keys, err := cfg.GetConfigKeysByGroup(group)
	if err != nil {
		return err
	}

	mappings := make(map[string]*gxset.HashSet)
	for k := range keys.Items {
		interfaceKey, _ := k.(string)
		if !(interfaceKey == "org.apache.dubbo.mock.api.MockService") {
			rule, err := metadataReport.GetServiceAppMapping(interfaceKey, group, nil)
			if err != nil {
				logger.Error("Failed to get mapping")
				return err
			}
			mappings[interfaceKey] = rule
		}
	}

	r.ctx.UpdateMapping(mappings)

	for interfaceKey, oldApps := range mappings {
		mappingListener := NewMappingListener(interfaceKey, oldApps, listener, out, systemNamespace, r.sdDelegate, r.ctx)
		apps, _ := metadataReport.GetServiceAppMapping(interfaceKey, "mapping", mappingListener)
		delSDListener := NewDubboSDNotifyListener(apps, r.ctx)
		for appTmp := range apps.Items {
			app := appTmp.(string)
			instances := r.sdDelegate.GetInstances(app)
			logger.Infof("Synchronized instance notification for %s on subscription, instance list size %d", app, len(instances))
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

	return nil
}
