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
	"context"
	"reflect"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	dubboconstant "dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/common/extension"
	"dubbo.apache.org/dubbo-go/v3/metadata/service"
	"dubbo.apache.org/dubbo-go/v3/metadata/service/local"
	triple_api "dubbo.apache.org/dubbo-go/v3/metadata/triple_api/proto"
	"dubbo.apache.org/dubbo-go/v3/registry"
	"dubbo.apache.org/dubbo-go/v3/remoting"

	gxset "github.com/dubbogo/gost/container/set"
	"github.com/dubbogo/gost/gof/observer"

	"github.com/pkg/errors"
)

import (
	"github.com/apache/dubbo-kubernetes/pkg/core/consts"
	"github.com/apache/dubbo-kubernetes/pkg/core/logger"
)

// DubboSDNotifyListener The Service Discovery Changed  Event Listener
type DubboSDNotifyListener struct {
	serviceNames      *gxset.HashSet
	ctx               *ApplicationContext
	listeners         map[string]registry.NotifyListener
	localAllInstances map[string][]registry.ServiceInstance
}

func NewDubboSDNotifyListener(services *gxset.HashSet, ctx *ApplicationContext) registry.ServiceInstancesChangedListener {
	return &DubboSDNotifyListener{
		serviceNames:      services,
		ctx:               ctx,
		listeners:         make(map[string]registry.NotifyListener),
		localAllInstances: make(map[string][]registry.ServiceInstance),
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

	revisionToInstances := make(map[string][]registry.ServiceInstance)
	localServiceToRevisions := make(map[*common.ServiceInfo]*gxset.HashSet)
	protocolRevisionsToUrls := make(map[string]map[*gxset.HashSet][]*common.URL)
	newServiceURLs := make(map[string][]*common.URL)

	logger.Infof("Received instance notification event of service %s, instance list size %d", ce.ServiceName, len(ce.Instances))

	for _, instance := range ce.Instances {
		oldRevision := lstn.ctx.GetOldRevision(instance)
		if instance.GetMetadata() == nil {
			logger.Warnf("Instance metadata is nil: %s", instance.GetHost())
			continue
		}
		revision := instance.GetMetadata()[dubboconstant.ExportedServicesRevisionPropertyName]
		if "0" == revision {
			logger.Infof("Find instance without valid service metadata: %s", instance.GetHost())
			continue
		}
		subInstances := revisionToInstances[revision]
		if subInstances == nil {
			subInstances = make([]registry.ServiceInstance, 8)
		}
		revisionToInstances[revision] = append(subInstances, instance)

		metadataInfo := lstn.ctx.GetRevisionToMetadata(revision)
		if metadataInfo == nil {
			logger.Infof("Start to fetch metadata from remote for app %s instance %s with revision %s ......", instance.GetServiceName(), instance.GetAddress(), revision)
			metadataInfo, err = GetMetadataInfo(instance, revision)
			if err != nil {
				logger.Errorf("Fetch metadata from remote error for revision %s, error detail is %v", revision, err)
				return err
			}
		}
		instance.SetServiceMetadata(metadataInfo)
		for _, service := range metadataInfo.Services {
			if localServiceToRevisions[service] == nil {
				localServiceToRevisions[service] = gxset.NewSet()
			}
			localServiceToRevisions[service].Add(revision)
		}
		lstn.ctx.UpdateRevisionToMetadata(oldRevision, revision, metadataInfo)
	}
	lstn.ctx.AddAllInstances(ce.ServiceName, ce.Instances)

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
		lstn.ctx.AddServiceUrls(newServiceURLs)
	}

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

	for _, instance := range findInstancesToDelete(lstn.localAllInstances[ce.ServiceName], ce.Instances) {
		revision := instance.GetMetadata()[dubboconstant.ExportedServicesRevisionPropertyName]
		metadataInfo := lstn.ctx.revisionToMetadata[revision]
		for _, v := range metadataInfo.Services {
			notifyListener := lstn.listeners[v.Name]
			for _, url := range instance.ToURLs(v) {
				lstn.ctx.DeleteServiceUrl(v.Name, url)
				notifyListener.Notify(&registry.ServiceEvent{
					Action:  remoting.EventTypeDel,
					Service: url,
				})
			}
		}
		lstn.ctx.DeleteRevisionToMetadata(revision)
		lstn.ctx.DeleteAllInstance(ce.ServiceName, instance)
	}

	lstn.localAllInstances[ce.ServiceName] = ce.Instances
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
		if instance.GetMetadata()[constant.MetadataVersion] == constant.MetadataServiceV2Version {
			proxyFactoryV2 := extension.GetMetadataServiceProxyFactoryV2(constant.MetadataServiceV2)
			metadataServiceV2 := proxyFactoryV2.GetProxy(instance)
			if metadataServiceV2 != nil {
				defer destroyInvokerV2(metadataServiceV2)
				var metadataInfoV2 *triple_api.MetadataInfoV2
				metadataInfoV2, err = metadataServiceV2.GetMetadataInfo(context.Background(), &triple_api.MetadataRequest{Revision: revision})
				if err != nil {
					logger.Errorf("get metadata of %s failed, %v", instance.GetHost(), err)
					return &common.MetadataInfo{}, err
				}
				metadataInfo = convertMetadataInfo(metadataInfoV2)
			} else {
				err = errors.New("get remote metadata error please check instance " + instance.GetHost() + " is alive")
			}
		} else {
			proxyFactory := extension.GetMetadataServiceProxyFactory(constant.DefaultKey)
			metadataService := proxyFactory.GetProxy(instance)
			if metadataService != nil {
				defer destroyInvoker(metadataService)
				metadataInfo, err = metadataService.GetMetadataInfo(revision)
				if err != nil {
					logger.Errorf("get metadata of %s failed, %v", instance.GetHost(), err)
					return &common.MetadataInfo{}, err
				}
			} else {
				err = errors.New("get remote metadata error please check instance " + instance.GetHost() + " is alive")
			}
		}
	}
	return metadataInfo, nil
}

func findInstancesToDelete(localAllInstances, newAllInstances []registry.ServiceInstance) []registry.ServiceInstance {
	newInstanceMap := make(map[string]registry.ServiceInstance)
	for _, instance := range newAllInstances {
		newInstanceMap[instance.GetID()] = instance
	}

	var instancesToDelete []registry.ServiceInstance
	for _, instance := range localAllInstances {
		if _, exists := newInstanceMap[instance.GetID()]; !exists {
			instancesToDelete = append(instancesToDelete, instance)
		}
	}

	return instancesToDelete
}

func convertMetadataInfo(v2 *triple_api.MetadataInfoV2) *common.MetadataInfo {
	infos := make(map[string]*common.ServiceInfo, 0)
	for k, v := range v2.Services {
		info := &common.ServiceInfo{
			Name:     v.Name,
			Group:    v.Group,
			Version:  v.Version,
			Protocol: v.Protocol,
			Path:     v.Path,
			Params:   v.Params,
		}
		infos[k] = info
	}

	metadataInfo := &common.MetadataInfo{
		Reported: false,
		App:      v2.App,
		Revision: v2.Version,
		Services: infos,
	}
	return metadataInfo
}

func destroyInvoker(metadataService service.MetadataService) {
	if metadataService == nil {
		return
	}

	proxy := metadataService.(*local.MetadataServiceProxy)
	if proxy.Invoker == nil {
		return
	}

	proxy.Invoker.Destroy()
}

func destroyInvokerV2(metadataService service.MetadataServiceV2) {
	if metadataService == nil {
		return
	}

	proxy := metadataService.(*local.MetadataServiceProxyV2)
	if proxy.Invoker == nil {
		return
	}

	proxy.Invoker.Destroy()
}
