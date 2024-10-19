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
	"sort"
	"strconv"
)

import (
	"github.com/apache/dubbo-kubernetes/api/mesh/v1alpha1"
	_ "github.com/apache/dubbo-kubernetes/api/mesh/v1alpha1"
	"github.com/apache/dubbo-kubernetes/pkg/admin/model"
	"github.com/apache/dubbo-kubernetes/pkg/core/resources/apis/mesh"
	core_model "github.com/apache/dubbo-kubernetes/pkg/core/resources/model"
	"github.com/apache/dubbo-kubernetes/pkg/core/resources/store"
	core_runtime "github.com/apache/dubbo-kubernetes/pkg/core/runtime"
)

func GetServiceTabDistribution(rt core_runtime.Runtime, req *model.ServiceTabDistributionReq) ([]*model.ServiceTabDistributionResp, error) {
	manager := rt.ResourceManager()
	mappingList := &mesh.MappingResourceList{}

	serviceName := req.ServiceName

	if err := manager.List(rt.AppContext(), mappingList); err != nil {
		return nil, err
	}

	res := make([]*model.ServiceTabDistributionResp, 0)

	for _, mapping := range mappingList.Items {
		// 找到对应serviceName的appNames
		if mapping.Spec.InterfaceName == serviceName {
			for _, appName := range mapping.Spec.ApplicationNames {
				dataplaneList := &mesh.DataplaneResourceList{}
				// 每拿到一个appName，都将对应的实例数据填充进dataplaneList, 再通过dataplane拿到这个appName对应的所有实例
				if err := manager.List(rt.AppContext(), dataplaneList, store.ListByApplication(appName)); err != nil {
					return nil, err
				}

				// 拿到了appName，接下来从dataplane取实例信息
				for _, dataplane := range dataplaneList.Items {
					metadata := &mesh.MetaDataResource{
						Spec: &v1alpha1.MetaData{},
					}
					if err := manager.Get(rt.AppContext(), metadata, store.GetByRevision(dataplane.Spec.GetExtensions()[v1alpha1.Revision]), store.GetByType(dataplane.Spec.GetExtensions()["registry-type"])); err != nil {
						return nil, err
					}
					respItem := &model.ServiceTabDistributionResp{}
					res = append(res, respItem.FromServiceDataplaneResource(dataplane, metadata, appName, req))
				}

			}
		}
	}
	return res, nil
}

func GetSearchServices(rt core_runtime.Runtime, req *model.ServiceSearchReq) (*model.SearchPaginationResult, error) {
	res := make([]*model.ServiceSearchResp, 0)

	serviceMap := make(map[string]*model.ServiceSearch)

	manager := rt.ResourceManager()

	dataplaneList := &mesh.DataplaneResourceList{}
	if err := manager.List(rt.AppContext(), dataplaneList); err != nil {
		return nil, err
	}
	// 通过dataplane extension字段获取所有revision
	revisions := make(map[string]struct{}, 0)
	for _, dataplane := range dataplaneList.Items {
		rev, ok := dataplane.Spec.GetExtensions()[v1alpha1.Revision]
		if ok {
			revisions[rev] = struct{}{}
		}
	}

	// 遍历 revisions
	for rev := range revisions {
		metadata := &mesh.MetaDataResource{
			Spec: &v1alpha1.MetaData{},
		}
		err := manager.Get(rt.AppContext(), metadata, store.GetByRevision(rev))
		if err != nil {
			return nil, err
		}
		for _, serviceInfo := range metadata.Spec.Services {
			if _, ok := serviceMap[serviceInfo.Name]; ok {
				serviceMap[serviceInfo.Name].FromServiceInfo(serviceInfo)
			} else {
				serviceSearch := model.NewServiceSearch(serviceInfo.Name)
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

	pagedRes := ToServiceSearchPaginationResult(res, req.PageReq)
	return pagedRes, nil
}

func ToServiceSearchPaginationResult(services []*model.ServiceSearchResp, req model.PageReq) *model.SearchPaginationResult {
	res := model.NewSearchPaginationResult()

	list := make([]*model.ServiceSearchResp, 0)

	sort.Sort(model.ByServiceName{})
	lenFilteredItems := len(services)
	pageSize := lenFilteredItems
	pageSize = req.PageSize
	offset := req.PageOffset

	for i := offset; i < offset+pageSize && i < lenFilteredItems; i++ {
		list = append(list, services[i])
	}

	nextOffset := ""
	if offset+pageSize < lenFilteredItems { // set new offset only if we did not reach the end of the collection
		nextOffset = strconv.Itoa(offset + req.PageSize)
	}

	res.List = list
	res.PageInfo = &core_model.Pagination{
		Total:      uint32(lenFilteredItems),
		NextOffset: nextOffset,
	}

	return res
}

func BannerSearchServices(rt core_runtime.Runtime, req *model.SearchReq) ([]*model.ServiceSearchResp, error) {
	res := make([]*model.ServiceSearchResp, 0)

	manager := rt.ResourceManager()
	mappingList := &mesh.MappingResourceList{}

	if req.Keywords != "" {
		if err := manager.List(rt.AppContext(), mappingList, store.ListByNameContains(req.Keywords)); err != nil {
			return nil, err
		}

		for _, mapping := range mappingList.Items {
			serviceSearchResp := model.NewServiceSearchResp()
			serviceSearchResp.ServiceName = mapping.GetMeta().GetName()
			res = append(res, serviceSearchResp)
		}
	}

	return res, nil
}
