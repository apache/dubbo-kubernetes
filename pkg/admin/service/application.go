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
	"strconv"
)

import (
	mesh_proto "github.com/apache/dubbo-kubernetes/api/mesh/v1alpha1"
	"github.com/apache/dubbo-kubernetes/pkg/admin/constants"
	"github.com/apache/dubbo-kubernetes/pkg/admin/model"
	"github.com/apache/dubbo-kubernetes/pkg/core/resources/apis/mesh"
	"github.com/apache/dubbo-kubernetes/pkg/core/resources/store"
	core_runtime "github.com/apache/dubbo-kubernetes/pkg/core/runtime"
)

func GetApplicationDetail(rt core_runtime.Runtime, req *model.ApplicationDetailReq) (*model.ApplicationDetailResp, error) {
	manager := rt.ResourceManager()
	dataplaneList := &mesh.DataplaneResourceList{}

	if err := manager.List(rt.AppContext(), dataplaneList, store.ListByApplication(req.AppName)); err != nil {
		return nil, err
	}

	revisions := make(map[string]*mesh.MetaDataResource, 0)
	applicationDetail := model.NewApplicationDetail()
	for _, dataplane := range dataplaneList.Items {
		rev, ok := dataplane.Spec.GetExtensions()[mesh_proto.Application]
		if ok {
			if metadata, cached := revisions[rev]; !cached {
				metadata = &mesh.MetaDataResource{
					Spec: &mesh_proto.MetaData{},
				}
				if err := manager.Get(rt.AppContext(), metadata, store.GetByRevision(rev), store.GetByType(dataplane.Spec.GetExtensions()["registry-type"])); err != nil {
					return nil, err
				}
				revisions[rev] = metadata
				applicationDetail.MergeMetaData(metadata)
			}
		}
		applicationDetail.MergeDatapalne(dataplane)
		applicationDetail.GetRegistry(rt)
	}

	respItem := &model.ApplicationDetailResp{
		AppName: req.AppName,
	}
	respItem = respItem.FromApplicationDetail(applicationDetail)

	return respItem, nil
}

func GetApplicationTabInstanceInfo(rt core_runtime.Runtime, req *model.ApplicationTabInstanceInfoReq) (*model.SearchPaginationResult, error) {
	manager := rt.ResourceManager()
	dataplaneList := &mesh.DataplaneResourceList{}

	if err := manager.List(rt.AppContext(), dataplaneList, store.ListByApplication(req.AppName), store.ListByPage(req.PageSize, strconv.Itoa(req.PageOffset))); err != nil {
		return nil, err
	}

	res := model.NewSearchPaginationResult()
	list := make([]*model.ApplicationTabInstanceInfoResp, 0, len(dataplaneList.Items))
	for _, dataplane := range dataplaneList.Items {
		resItem := &model.ApplicationTabInstanceInfoResp{}
		resItem.FromDataplaneResource(dataplane)
		resItem.GetRegistry(rt)
		list = append(list, resItem)
	}

	res.List = list
	res.PageInfo = &dataplaneList.Pagination

	return res, nil
}

func GetApplicationServiceFormInfo(rt core_runtime.Runtime, req *model.ApplicationServiceFormReq) (*model.SearchPaginationResult, error) {
	manager := rt.ResourceManager()
	dataplaneList := &mesh.DataplaneResourceList{}

	if err := manager.List(rt.AppContext(), dataplaneList, store.ListByApplication(req.AppName)); err != nil {
		return nil, err
	}

	res := make([]*model.ApplicationServiceFormResp, 0)
	serviceMap := make(map[string]*model.ApplicationServiceForm)
	revisions := make(map[string]*mesh.MetaDataResource, 0)
	for _, dataplane := range dataplaneList.Items {
		rev, ok := dataplane.Spec.GetExtensions()[mesh_proto.Revision]
		if ok {
			metadata, cached := revisions[rev]
			if !cached {
				metadata = &mesh.MetaDataResource{
					Spec: &mesh_proto.MetaData{},
				}
				if err := manager.Get(rt.AppContext(), metadata, store.GetByRevision(rev), store.GetByType(dataplane.Spec.GetExtensions()["registry-type"])); err != nil {
					return nil, err
				}
				revisions[rev] = metadata
			}

			for _, serviceInfo := range metadata.Spec.Services {
				if serviceInfo.Params[constants.ServiceInfoSide] == req.Side {
					applicationServiceForm := model.NewApplicationServiceForm(serviceInfo.Name)
					if _, ok := serviceMap[serviceInfo.Name]; ok {
						if err := serviceMap[serviceInfo.Name].FromServiceInfo(serviceInfo); err != nil {
							return nil, err
						}
					} else {
						if err := applicationServiceForm.FromServiceInfo(serviceInfo); err != nil {
							return nil, err
						}
						serviceMap[serviceInfo.Name] = applicationServiceForm
					}
				}
			}
		}
	}

	for _, applicationServiceForm := range serviceMap {
		applicationServiceFormResp := model.NewApplicationServiceFormResp()
		if err := applicationServiceFormResp.FromApplicationServiceForm(applicationServiceForm); err != nil {
			return nil, err
		}
		res = append(res, applicationServiceFormResp)
	}

	pagedRes := ToSearchPaginationResult(res, model.ByAppServiceFormName(res), req.PageReq)

	return pagedRes, nil
}

func GetApplicationSearchInfo(rt core_runtime.Runtime, req *model.ApplicationSearchReq) (*model.SearchPaginationResult, error) {
	manager := rt.ResourceManager()
	dataplaneList := &mesh.DataplaneResourceList{}

	if req.Keywords != "" {
		if err := manager.List(rt.AppContext(), dataplaneList, store.ListByApplicationContains(req.Keywords)); err != nil {
			return nil, err
		}
	} else {
		if err := manager.List(rt.AppContext(), dataplaneList); err != nil {
			return nil, err
		}
	}

	res := make([]*model.ApplicationSearchResp, 0)
	appMap := make(map[string]*model.ApplicationSearch)
	for _, dataplane := range dataplaneList.Items {
		appName := dataplane.Spec.GetExtensions()[mesh_proto.Application]
		if _, ok := appMap[appName]; !ok {
			appMap[appName] = model.NewApplicationSearch(appName)
		}
		appMap[appName].MergeDataplane(dataplane)
		appMap[appName].GetRegistry(rt)
	}

	for appName, search := range appMap {
		applicationSearchResp := &model.ApplicationSearchResp{
			AppName: appName,
		}
		res = append(res, applicationSearchResp.FromApplicationSearch(search))
	}

	pagedRes := ToSearchPaginationResult(res, model.ByAppName(res), req.PageReq)
	return pagedRes, nil
}

func BannerSearchApplications(rt core_runtime.Runtime, req *model.SearchReq) ([]*model.ApplicationSearchResp, error) {
	manager := rt.ResourceManager()
	dataplaneList := &mesh.DataplaneResourceList{}
	if req.Keywords != "" {
		if err := manager.List(rt.AppContext(), dataplaneList, store.ListByApplicationContains(req.Keywords)); err != nil {
			return nil, err
		}
	} else {
		if err := manager.List(rt.AppContext(), dataplaneList); err != nil {
			return nil, err
		}
	}

	res := make([]*model.ApplicationSearchResp, 0)
	appMap := make(map[string]*model.ApplicationSearch)
	for _, dataplane := range dataplaneList.Items {
		appName := dataplane.Spec.GetExtensions()[mesh_proto.Application]
		if _, ok := appMap[appName]; !ok {
			appMap[appName] = model.NewApplicationSearch(appName)
		}
		appMap[appName].MergeDataplane(dataplane)
		appMap[appName].GetRegistry(rt)
	}

	for appName, search := range appMap {
		applicationSearchResp := &model.ApplicationSearchResp{
			AppName: appName,
		}
		res = append(res, applicationSearchResp.FromApplicationSearch(search))
	}
	return res, nil
}
