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
	"reflect"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	dubboconstant "dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/common/extension"
	"dubbo.apache.org/dubbo-go/v3/metadata/service/local"
	"dubbo.apache.org/dubbo-go/v3/registry"
	"dubbo.apache.org/dubbo-go/v3/remoting"

	gxset "github.com/dubbogo/gost/container/set"
	"github.com/dubbogo/gost/gof/observer"
)

import (
	"github.com/apache/dubbo-kubernetes/pkg/core/consts"
	"github.com/apache/dubbo-kubernetes/pkg/core/logger"
)

// DubboSDNotifyListener The Service Discovery Changed  Event Listener
type DubboSDNotifyListener struct {
	serviceNames *gxset.HashSet
	ctx          *ApplicationContext
	listeners    map[string]registry.NotifyListener
}

func NewDubboSDNotifyListener(services *gxset.HashSet, ctx *ApplicationContext) registry.ServiceInstancesChangedListener {
	return &DubboSDNotifyListener{
		serviceNames: services,
		ctx:          ctx,
	}
}

// OnEvent on ServiceInstancesChangedEvent the service instances change event
func (lstn *DubboSDNotifyListener) OnEvent(e observer.Event) error {
	ce, ok := e.(*registry.ServiceInstancesChangedEvent)
	if !ok {
		return nil
	}
	var err error

	lstn.ctx.mu.Lock()
	defer lstn.ctx.mu.Unlock()

	lstn.ctx.SetAllInstances(ce.ServiceName, ce.Instances)
	revisionToInstances := make(map[string][]registry.ServiceInstance)
	newRevisionToMetadata := make(map[string]*common.MetadataInfo)
	localServiceToRevisions := make(map[*common.ServiceInfo]*gxset.HashSet)
	protocolRevisionsToUrls := make(map[string]map[*gxset.HashSet][]*common.URL)
	newServiceURLs := make(map[string][]*common.URL)

	logger.Infof("Received instance notification event of service %s, instance list size %d", ce.ServiceName, len(ce.Instances))

	for _, instances := range lstn.ctx.GetAllInstances() {
		for _, instance := range instances {
			metadataInstance := ConvertToMetadataInstance(instance)
			if metadataInstance.GetMetadata() == nil {
				logger.Warnf("Instance metadata is nil: %s", metadataInstance.GetHost())
				continue
			}
			revision := metadataInstance.GetMetadata()[dubboconstant.ExportedServicesRevisionPropertyName]
			if "0" == revision {
				logger.Infof("Find instance without valid service metadata: %s", metadataInstance.GetHost())
				continue
			}
			subInstances := revisionToInstances[revision]
			if subInstances == nil {
				subInstances = make([]registry.ServiceInstance, 8)
			}
			revisionToInstances[revision] = append(subInstances, metadataInstance)

			metadataInfo := lstn.ctx.GetRevisionToMetadata(revision)
			if metadataInfo == nil {
				metadataInfo, err = GetMetadataInfo(metadataInstance, revision)
				if err != nil {
					return err
				}
			}
			metadataInstance.SetServiceMetadata(metadataInfo)
			for _, service := range metadataInfo.Services {
				if localServiceToRevisions[service] == nil {
					localServiceToRevisions[service] = gxset.NewSet()
				}
				localServiceToRevisions[service].Add(revision)
			}

			newRevisionToMetadata[revision] = metadataInfo

		}
		lstn.ctx.NewRevisionToMetadata(newRevisionToMetadata)

		for serviceInfo, revisions := range localServiceToRevisions {
			revisionsToUrls := protocolRevisionsToUrls[serviceInfo.Protocol]
			if revisionsToUrls == nil {
				protocolRevisionsToUrls[serviceInfo.Protocol] = make(map[*gxset.HashSet][]*common.URL)
				revisionsToUrls = protocolRevisionsToUrls[serviceInfo.Protocol]
			}
			urls := revisionsToUrls[revisions]
			if urls != nil {
				newServiceURLs[serviceInfo.Name] = urls
			} else {
				urls = make([]*common.URL, 0, 8)
				for _, v := range revisions.Values() {
					r := v.(string)
					for _, i := range revisionToInstances[r] {
						if i != nil {
							urls = append(urls, i.ToURLs(serviceInfo)...)
						}
					}
				}
				revisionsToUrls[revisions] = urls
				newServiceURLs[serviceInfo.Name] = urls
			}
		}
		lstn.ctx.NewServiceUrls(newServiceURLs)

		for key, notifyListener := range lstn.listeners {
			urls := lstn.ctx.GetServiceUrls()[key]
			events := make([]*registry.ServiceEvent, 0, len(urls))
			for _, url := range urls {
				url.SetParam(consts.RegistryType, consts.RegistryInstance)
				events = append(events, &registry.ServiceEvent{
					Action:  remoting.EventTypeAdd,
					Service: url,
				})
			}
			notifyListener.NotifyAll(events, func() {})
		}
	}
	return nil
}

// AddListenerAndNotify add notify listener and notify to listen service event
func (lstn *DubboSDNotifyListener) AddListenerAndNotify(serviceKey string, notify registry.NotifyListener) {
	lstn.listeners[serviceKey] = notify

	urls := lstn.ctx.GetServiceUrls()[serviceKey]
	for _, url := range urls {
		url.SetParam(consts.RegistryType, consts.RegistryInstance)
		notify.Notify(&registry.ServiceEvent{
			Action:  remoting.EventTypeAdd,
			Service: url,
		})
	}
}

// RemoveListener remove notify listener
func (lstn *DubboSDNotifyListener) RemoveListener(serviceKey string) {
	delete(lstn.listeners, serviceKey)
}

// GetServiceNames return all listener service names
func (lstn *DubboSDNotifyListener) GetServiceNames() *gxset.HashSet {
	return lstn.serviceNames
}

// Accept return true if the name is the same
func (lstn *DubboSDNotifyListener) Accept(e observer.Event) bool {
	if ce, ok := e.(*registry.ServiceInstancesChangedEvent); ok {
		return lstn.serviceNames.Contains(ce.ServiceName)
	}
	return false
}

// GetPriority returns -1, it will be the first invoked listener
func (lstn *DubboSDNotifyListener) GetPriority() int {
	return -1
}

// GetEventType returns ServiceInstancesChangedEvent
func (lstn *DubboSDNotifyListener) GetEventType() reflect.Type {
	return reflect.TypeOf(&registry.ServiceInstancesChangedEvent{})
}

// GetMetadataInfo get metadata info when MetadataStorageTypePropertyName is null
func GetMetadataInfo(instance registry.ServiceInstance, revision string) (*common.MetadataInfo, error) {
	var metadataStorageType string
	var metadataInfo *common.MetadataInfo
	if instance.GetMetadata() == nil {
		metadataStorageType = dubboconstant.DefaultMetadataStorageType
	} else {
		metadataStorageType = instance.GetMetadata()[dubboconstant.MetadataStorageTypePropertyName]
	}
	if metadataStorageType == dubboconstant.RemoteMetadataStorageType {
		remoteMetadataServiceImpl, err := extension.GetRemoteMetadataService()
		if err != nil {
			return &common.MetadataInfo{}, err
		}
		metadataInfo, err = remoteMetadataServiceImpl.GetMetadata(instance)
		if err != nil {
			return &common.MetadataInfo{}, err
		}
	} else {
		var err error
		proxyFactory := extension.GetMetadataServiceProxyFactory(dubboconstant.DefaultKey)
		metadataService := proxyFactory.GetProxy(instance)
		defer metadataService.(*local.MetadataServiceProxy).Invoker.Destroy()
		metadataInfo, err = metadataService.GetMetadataInfo(revision)
		if err != nil {
			return &common.MetadataInfo{}, err
		}
	}
	return metadataInfo, nil
}
