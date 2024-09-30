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
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"sort"
	"strconv"
	"strings"

	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/registry"
	"dubbo.apache.org/dubbo-go/v3/remoting"
	"github.com/apache/dubbo-kubernetes/pkg/core/logger"
	gxset "github.com/dubbogo/gost/container/set"
)

type InterfaceServiceChangedNotifyListener struct {
	listener *NotifyListener
	ctx      *ApplicationContext
	allUrls  *gxset.HashSet
}

type Endpoint struct {
	Port     int    `json:"port,omitempty"`
	Protocol string `json:"protocol,omitempty"`
}

func NewInterfaceServiceChangedNotifyListener(
	listener *NotifyListener,
	ctx *ApplicationContext,
) *InterfaceServiceChangedNotifyListener {
	return &InterfaceServiceChangedNotifyListener{
		listener: listener,
		ctx:      ctx,
		allUrls:  gxset.NewSet(),
	}
}

func (iscnl *InterfaceServiceChangedNotifyListener) Notify(event *registry.ServiceEvent) {
	iscnl.ctx.mu.Lock()
	defer iscnl.ctx.mu.Unlock()

	switch event.Action {
	case remoting.EventTypeAdd:
		iscnl.handleAdd(event.Service)
	case remoting.EventTypeUpdate:
		iscnl.CreateOrUpdateServiceInfo(event.Service)
	case remoting.EventTypeDel:
		iscnl.deleteServiceInfo(event.Service)
	}
}

func (iscnl *InterfaceServiceChangedNotifyListener) NotifyAll(events []*registry.ServiceEvent, f func()) {
	for _, event := range events {
		iscnl.Notify(event)
	}
}

func (iscnl *InterfaceServiceChangedNotifyListener) handleAdd(url *common.URL) {
	urlStr := url.String()
	if iscnl.allUrls.Contains(urlStr) {
		return
	}
	iscnl.allUrls.Add(urlStr)
	iscnl.CreateOrUpdateServiceInfo(url)
}

func (iscnl *InterfaceServiceChangedNotifyListener) CreateOrUpdateServiceInfo(url *common.URL) {
	serviceInfo := common.NewServiceInfoWithURL(url)
	iscnl.ctx.UpdateServiceUrls(serviceInfo.Name, url)

	appName := url.GetParam(constant.ApplicationKey, "")
	instance, oldRevision := iscnl.getInstanceAndRevision(appName, url.Address())

	metadataInfo := iscnl.ctx.GetRevisionToMetadata(oldRevision)
	if metadataInfo == nil {
		metadataInfo = common.NewMetadataInfo(appName, "", make(map[string]*common.ServiceInfo))
	}
	metadataInfo.AddService(serviceInfo)
	revision := resolveRevision(appName, metadataInfo.Services)
	metadataInfo.Revision = revision
	iscnl.ctx.UpdateRevisionToMetadata(oldRevision, revision, metadataInfo)

	if instance == nil {
		port, _ := strconv.Atoi(url.Port)
		instance = &registry.DefaultServiceInstance{
			ID:          url.Address(),
			ServiceName: appName,
			Host:        url.Ip,
			Port:        port,
			Enable:      true,
			Healthy:     true,
			Metadata: map[string]string{
				constant.MetadataStorageTypePropertyName:      constant.DefaultMetadataStorageType,
				constant.TimestampKey:                         url.GetParam(constant.TimestampKey, ""),
				constant.ServiceInstanceEndpoints:             getEndpointsStr(url.Protocol, port),
				constant.MetadataServiceURLParamsPropertyName: getURLParams(serviceInfo),
			},
			Address:   url.Address(),
			GroupName: serviceInfo.Group,
		}
	}
	instance.GetMetadata()[constant.ExportedServicesRevisionPropertyName] = revision
	instance.SetServiceMetadata(metadataInfo)
	iscnl.ctx.UpdateAllInstances(appName, instance)

	iscnl.listener.Notify(&registry.ServiceEvent{
		Action:  remoting.EventTypeAdd,
		Service: url,
	})
}

func (iscnl *InterfaceServiceChangedNotifyListener) deleteServiceInfo(url *common.URL) {
	serviceInfo := common.NewServiceInfoWithURL(url)

	appName := url.GetParam(constant.ApplicationKey, "")
	instance, oldRevision := iscnl.getInstanceAndRevision(appName, url.Address())
	if instance == nil {
		logger.Error("Cannot delete ServiceInfo, instance does not exist")
		return
	}

	var metadataInfo *common.MetadataInfo
	if oldRevision != "" {
		metadataInfo = iscnl.ctx.GetRevisionToMetadata(oldRevision)
		delete(metadataInfo.Services, serviceInfo.GetMatchKey())
		if len(metadataInfo.Services) == 0 {
			iscnl.ctx.DeleteRevisionToMetadata(oldRevision)
			iscnl.ctx.DeleteAllInstance(appName, instance)
			return
		}
		iscnl.ctx.UpdateRevisionToMetadata(oldRevision, resolveRevision(appName, metadataInfo.Services), metadataInfo)
	}

	if metadataInfo != nil {
		instance.SetServiceMetadata(metadataInfo)
		instance.GetMetadata()[constant.ExportedServicesRevisionPropertyName] = metadataInfo.Revision
		iscnl.ctx.UpdateAllInstances(appName, instance)
	}

	iscnl.listener.Notify(&registry.ServiceEvent{
		Action:  remoting.EventTypeDel,
		Service: url,
	})
	iscnl.ctx.DeleteServiceUrl(serviceInfo.Name, url)
}

func (iscnl *InterfaceServiceChangedNotifyListener) getInstanceAndRevision(appName, address string) (registry.ServiceInstance, string) {
	instances, ok := iscnl.ctx.GetAllInstances()[appName]
	if !ok {
		iscnl.ctx.AddAllInstances(appName, make([]registry.ServiceInstance, 0))
		return nil, ""
	}
	var instance registry.ServiceInstance
	var oldRevision string
	for _, elem := range instances {
		if elem.GetAddress() == address {
			instance = elem
			oldRevision = instance.GetMetadata()[constant.ExportedServicesRevisionPropertyName]
			break
		}
	}
	return instance, oldRevision
}

func getEndpointsStr(protocol string, port int) string {
	if port <= 0 {
		return ""
	}
	endpoints := []Endpoint{{Port: port, Protocol: protocol}}
	str, err := json.Marshal(endpoints)
	if err != nil {
		logger.Errorf("Could not convert the endpoints to JSON: %v", err)
		return ""
	}
	return string(str)
}

func getURLParams(serviceInfo *common.ServiceInfo) string {
	urlParams, err := json.Marshal(serviceInfo.Params)
	if err != nil {
		logger.Errorf("Could not convert the URL params to JSON: %v", err)
		return ""
	}
	return string(urlParams)
}

func resolveRevision(appName string, servicesInfo map[string]*common.ServiceInfo) string {
	var sb strings.Builder
	sb.WriteString(appName)
	for _, serviceInfo := range servicesInfo {
		sb.WriteString(ToDescString(serviceInfo))
	}
	return getMd5(sb.String())
}

func ToDescString(s *common.ServiceInfo) string {
	var sb strings.Builder
	sb.WriteString(s.GetMatchKey())
	sb.WriteString(s.URL.Port)
	sb.WriteString(s.Path)

	params := s.GetParams()
	keys := make([]string, 0, len(params))
	for key := range params {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	for _, key := range keys {
		sb.WriteString(key)
		sb.WriteString(params[key][0])
	}

	return sb.String()
}

func getMd5(data string) string {
	hasher := md5.New()
	hasher.Write([]byte(data))
	return hex.EncodeToString(hasher.Sum(nil))
}
