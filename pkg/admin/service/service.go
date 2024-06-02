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
	mesh_proto "github.com/apache/dubbo-kubernetes/api/mesh/v1alpha1"
	"github.com/apache/dubbo-kubernetes/pkg/admin/constants"
	"github.com/apache/dubbo-kubernetes/pkg/admin/model"
	"github.com/apache/dubbo-kubernetes/pkg/core/resources/apis/mesh"
	"github.com/apache/dubbo-kubernetes/pkg/core/resources/store"
	core_runtime "github.com/apache/dubbo-kubernetes/pkg/core/runtime"
)

func GetServiceTabDistribution(rt core_runtime.Runtime, req *model.ServiceTabDistributionReq) ([]*model.ServiceTabDistributionResp, error) {
	manager := rt.ResourceManager()
	dataplaneList := &mesh.DataplaneResourceList{}
	mappingList := &mesh.MappingResourceList{}
	metadataList := &mesh.MetaDataResourceList{}

	// 伪造 dataplaneList
	dataplaneList = &mesh.DataplaneResourceList{
		Items: []*mesh.DataplaneResource{
			// 添加一些模拟的 dataplane
			{
				Spec: &mesh_proto.Dataplane{
					Extensions: map[string]string{"app1": "appName1"},
					Networking: &mesh_proto.Dataplane_Networking{
						Inbound: []*mesh_proto.Dataplane_Networking_Inbound{
							{
								Port:        8080,
								ServicePort: 8080,
								Address:     "127.0.0.1",
							},
						},
						AdvertisedAddress: "127.0.0.1",
					},
				},
			},
			{
				Spec: &mesh_proto.Dataplane{
					Extensions: map[string]string{"app2": "appName2"},
					Networking: &mesh_proto.Dataplane_Networking{
						Inbound: []*mesh_proto.Dataplane_Networking_Inbound{
							{
								Port:        8080,
								ServicePort: 8080,
								Address:     "127.0.0.1",
							},
						},
						AdvertisedAddress: "127.0.0.1",
					},
				},
			},
			{
				Spec: &mesh_proto.Dataplane{
					Extensions: map[string]string{"app3": "appName3"},
					Networking: &mesh_proto.Dataplane_Networking{
						Inbound: []*mesh_proto.Dataplane_Networking_Inbound{
							{
								Port:        8080,
								ServicePort: 8080,
								Address:     "127.0.0.1",
							},
						},
						AdvertisedAddress: "127.0.0.1",
					},
				},
			},
			{
				Spec: &mesh_proto.Dataplane{
					Extensions: map[string]string{"app4": "appName4"},
					Networking: &mesh_proto.Dataplane_Networking{
						Inbound: []*mesh_proto.Dataplane_Networking_Inbound{
							{
								Port:        8080,
								ServicePort: 8080,
								Address:     "127.0.0.1",
							},
						},
						AdvertisedAddress: "127.0.0.1",
					},
				},
			},
		},
	}

	metadataList = &mesh.MetaDataResourceList{
		Items: []*mesh.MetaDataResource{
			// 添加一些模拟的 metadata
			{Spec: &mesh_proto.MetaData{Services: map[string]*v1alpha1.ServiceInfo{
				"service1": {Name: "service1", Group: "servicegroup1", Version: "v1", Params: map[string]string{constants.RetriesKey: "2", constants.TimeoutKey: "3000"}},
			}}},
			{Spec: &mesh_proto.MetaData{Services: map[string]*v1alpha1.ServiceInfo{
				"service2": {Name: "service2", Group: "servicegroup1", Version: "v1", Params: map[string]string{constants.RetriesKey: "2", constants.TimeoutKey: "3000"}},
			}}},
		},
	}

	// 伪造 mappingList
	mappingList = &mesh.MappingResourceList{
		Items: []*mesh.MappingResource{
			// 添加一些模拟的 mapping
			{
				Spec: &mesh_proto.Mapping{
					Zone:             "zone1",
					InterfaceName:    "service1",
					ApplicationNames: []string{"appName1", "appName2"},
				},
			},
			{
				Spec: &mesh_proto.Mapping{
					Zone:             "zone2",
					InterfaceName:    "service2",
					ApplicationNames: []string{"appName3", "appName4"},
				},
			},
		},
	}

	serviceName := req.ServiceName

	if err := manager.List(rt.AppContext(), mappingList); err != nil {
		return nil, err
	}

	res := make([]*model.ServiceTabDistributionResp, 0)

	for _, mapping := range mappingList.Items {
		//找到对应serviceName的appNames
		if mapping.Spec.InterfaceName == serviceName {
			for _, appName := range mapping.Spec.ApplicationNames {
				//每拿到一个appName，都将对应的实例数据填充进dataplaneList, 再通过dataplane拿到这个appName对应的所有实例
				if err := manager.List(rt.AppContext(), dataplaneList, store.ListByNameContains(appName)); err != nil {
					return nil, err
				}
				if err := manager.List(rt.AppContext(), metadataList, store.ListByNameContains(appName)); err != nil {
					return nil, err
				}
				//拿到了appName，接下来从dataplane取实例信息
				for _, dataplane := range dataplaneList.Items {
					respItem := &model.ServiceTabDistributionResp{}
					res = append(res, respItem.FromServiceDataplaneResource(dataplane, metadataList, appName, req))
				}
			}
		}
	}
	return res, nil
}

func GetSearchServices(rt core_runtime.Runtime) ([]*model.ServiceSearchResp, error) {
	manager := rt.ResourceManager()

	dataplaneList := &mesh.DataplaneResourceList{}
	metadataList := &mesh.MetaDataResourceList{}

	if err := manager.List(rt.AppContext(), dataplaneList); err != nil {
		return nil, err
	}

	dataplaneList = &mesh.DataplaneResourceList{
		Items: []*mesh.DataplaneResource{
			// 添加一些模拟的 dataplane
			{Spec: &mesh_proto.Dataplane{Extensions: map[string]string{"app1": "appName1"}}},
			{Spec: &mesh_proto.Dataplane{Extensions: map[string]string{"app2": "appName2"}}},
		},
	}

	metadataList = &mesh.MetaDataResourceList{
		Items: []*mesh.MetaDataResource{
			// 添加一些模拟的 metadata
			{Spec: &mesh_proto.MetaData{Services: map[string]*v1alpha1.ServiceInfo{
				"service1": {Name: "service1", Group: "servicegroup1", Version: "v1"},
			}}},
			{Spec: &mesh_proto.MetaData{Services: map[string]*v1alpha1.ServiceInfo{
				"service2": {Name: "service2", Group: "servicegroup1", Version: "v1"},
			}}},
		},
	}

	//通过dataplane的extension字段获取所有appName
	appNames := make(map[string]struct{}, 0)

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

	}

	res := make([]*model.ServiceSearchResp, 0)

	serviceMap := make(map[string]*model.ServiceSearch)
	for _, metadata := range metadataList.Items {
		for _, serviceInfo := range metadata.Spec.Services {
			serviceSearch := model.NewServiceSearch(serviceInfo.Name)
			if _, ok := serviceMap[serviceInfo.Name]; ok {
				serviceMap[serviceInfo.Name].FromServiceInfo(serviceInfo)
			} else {
				serviceSearch.FromServiceInfo(serviceInfo)
				serviceMap[serviceInfo.Name] = serviceSearch
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
