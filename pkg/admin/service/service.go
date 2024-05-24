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

package service

import (
	"github.com/apache/dubbo-kubernetes/api/mesh/v1alpha1"
	_ "github.com/apache/dubbo-kubernetes/api/mesh/v1alpha1"
	"github.com/apache/dubbo-kubernetes/pkg/admin/model"
	"github.com/apache/dubbo-kubernetes/pkg/core/resources/apis/mesh"
	_ "github.com/apache/dubbo-kubernetes/pkg/core/resources/model"
	"github.com/apache/dubbo-kubernetes/pkg/core/resources/store"
	core_runtime "github.com/apache/dubbo-kubernetes/pkg/core/runtime"
)

func GetServiceTabDistribution(rt core_runtime.Runtime, req *model.ServiceTabDistributionReq) ([]*model.ServiceTabDistributionResp, error) {
	manager := rt.ResourceManager()
	dataplaneList := &mesh.DataplaneResourceList{}
	mappingList := &mesh.MappingResourceList{}

	serviceName := req.ServiceName

	if err := manager.List(rt.AppContext(), dataplaneList, store.ListByNameContains(req.ServiceName)); err != nil {
		return nil, err
	}

	if err := manager.List(rt.AppContext(), mappingList); err != nil {
		return nil, err
	}

	res := make([]*model.ServiceTabDistributionResp, 0)

	for _, mapping := range mappingList.Items {
		//找到对应serviceName的appNames
		if mapping.Spec.InterfaceName == serviceName {
			for _, appName := range mapping.Spec.ApplicationNames {
				serviceDistributionResp := model.NewServiceDistributionResp()
				serviceDistributionResp.FromServiceMappingResource(dataplaneList, appName)
				res = append(res, serviceDistributionResp)
			}

		}
	}

	return res, nil
}

func GetSearchServices(rt core_runtime.Runtime) ([]*model.ServiceSearchResp, error) {
	manager := rt.ResourceManager()

	dataplaneList := &mesh.DataplaneResourceList{}
	metadataList := &mesh.MetaDataResourceList{}
	mappingList := &mesh.MappingResourceList{}

	if err := manager.List(rt.AppContext(), dataplaneList); err != nil {
		return nil, err
	}

	//通过dataplane的extension字段获取所有appName
	appNames := make(map[string]struct{}, 0)
	serviceNames := make([]string, 0)

	for _, dataplane := range dataplaneList.Items {
		appName, ok := dataplane.Spec.GetExtensions()[v1alpha1.Application]
		if ok {
			appNames[appName] = struct{}{}
		}
	}

	// 遍历 appName
	for appName := range appNames {
		//获取metadataList
		err := manager.List(rt.AppContext(), metadataList, store.ListByNameContains(appName))
		if err != nil {
			return nil, err
		}

		//获取mappingList
		err = manager.List(rt.AppContext(), mappingList, store.ListByNameContains(appName))
		if err != nil {
			return nil, err
		}

		//通过mapping与AppName查询InterfaceName（serviceName）
		for _, mapping := range mappingList.Items {
			for _, appNameItem := range mapping.Spec.ApplicationNames {
				if appNameItem == appName {
					serviceNames = append(serviceNames, mapping.Spec.InterfaceName)
				}
			}
		}
	}

	res := make([]*model.ServiceSearchResp, 0)

	serviceMap := make(map[string]*model.ServiceSearch)
	for _, metadata := range metadataList.Items {
		for _, serviceInfo := range metadata.Spec.Services {
			if contains(serviceNames, serviceInfo.Name) {
				serviceSearch := model.NewServiceSearch(serviceInfo.Name)
				if _, ok := serviceMap[serviceInfo.Name]; ok {
					serviceMap[serviceInfo.Name].FromServiceInfo(serviceInfo)
				} else {
					serviceSearch.FromServiceInfo(serviceInfo)
					serviceMap[serviceInfo.Name] = serviceSearch
				}
			}
		}
	}

	for _, serviceSearch := range serviceMap {
		serviceSearchResp := model.NewServiceSearchResp()
		serviceSearchResp.FromServiceSearch(serviceSearch)
		res = append(res, serviceSearchResp)
	}

	return res, nil
}

func contains(names []string, name string) bool {
	for _, item := range names {
		if item == name {
			return true
		}
	}
	return false
}
