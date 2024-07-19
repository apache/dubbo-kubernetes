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
	dubboRegistry "dubbo.apache.org/dubbo-go/v3/registry"
	"github.com/apache/dubbo-kubernetes/pkg/core/consts"
	"github.com/apache/dubbo-kubernetes/pkg/core/logger"
	gxset "github.com/dubbogo/gost/container/set"
)

type GeneralInterfaceNotifyListener struct {
	ctx            *ApplicationContext
	notifyListener *InterfaceServiceChangedNotifyListener
	registry       dubboRegistry.Registry

	allUrls *gxset.HashSet
	mutex   sync.Mutex
}

func NewGeneralInterfaceNotifyListener(
	ctx *ApplicationContext,
	notifyListener *InterfaceServiceChangedNotifyListener,
	allUrls *gxset.HashSet,
	registry dubboRegistry.Registry,
) *GeneralInterfaceNotifyListener {
	return &GeneralInterfaceNotifyListener{
		ctx:            ctx,
		allUrls:        allUrls,
		notifyListener: notifyListener,
		registry:       registry,
		mutex:          sync.Mutex{},
	}
}

func (gilstn *GeneralInterfaceNotifyListener) Notify(event *dubboRegistry.ServiceEvent) {
	url := event.Service
	urlStr := url.String()

	if gilstn.allUrls.Contains(urlStr) {
		return
	}

	gilstn.mutex.Lock()
	defer gilstn.mutex.Unlock()

	go func() {
		subscribeUrl := buildServiceSubscribeUrl(common.NewServiceInfoWithURL(url))
		err := gilstn.registry.Subscribe(subscribeUrl, gilstn.notifyListener)
		if err != nil {
			logger.Error("Failed to subscribe to registry, might not be able to show services of the cluster!")
		}
	}()

	gilstn.allUrls.Add(urlStr)
}

func (gilstn *GeneralInterfaceNotifyListener) NotifyAll(events []*dubboRegistry.ServiceEvent, f func()) {
	for _, event := range events {
		gilstn.Notify(event)
	}
}

func buildServiceSubscribeUrl(serviceInfo *common.ServiceInfo) *common.URL {
	subscribeUrl, _ := common.NewURL(common.GetLocalIp()+":0",
		common.WithProtocol(consts.AdminProtocol),
		common.WithParams(serviceInfo.GetParams()))
	return subscribeUrl
}
