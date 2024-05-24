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
	mesh_proto "github.com/apache/dubbo-kubernetes/api/mesh/v1alpha1"
	"github.com/apache/dubbo-kubernetes/pkg/admin/constants"
	"github.com/apache/dubbo-kubernetes/pkg/admin/model"
	"github.com/apache/dubbo-kubernetes/pkg/core/resources/apis/mesh"
	"github.com/apache/dubbo-kubernetes/pkg/core/resources/store"
	core_runtime "github.com/apache/dubbo-kubernetes/pkg/core/runtime"
)

func GetApplicationDetail(rt core_runtime.Runtime, req *model.ApplicationDetailReq) ([]*model.ApplicationDetailResp, error) {
	manager := rt.ResourceManager()
	dataplaneList := &mesh.DataplaneResourceList{}

	if err := manager.List(rt.AppContext(), dataplaneList, store.ListByNameContains(req.AppName)); err != nil {
		return nil, err
	}

	appMap := make(map[string]*model.ApplicationDetail)
	for _, dataplane := range dataplaneList.Items {
		appName := dataplane.Meta.GetLabels()[mesh_proto.AppTag]
		//appName := dataplane.Spec.Networking.Inbound[0].GetTags()[mesh_proto.AppTag]
		var applicationDetail *model.ApplicationDetail
		if _, ok := appMap[appName]; ok {
			applicationDetail = appMap[appName]
		} else {
			applicationDetail = model.NewApplicationDetail()
		}

		applicationDetail.MergeDatapalne(dataplane)
		applicationDetail.GetRegistry(rt)
		appMap[appName] = applicationDetail
	}

	metadataList := &mesh.MetaDataResourceList{}
	if err := manager.List(rt.AppContext(), metadataList, store.ListByNameContains(req.AppName)); err != nil {
		return nil, err
	}

	//get data from metadata
	for _, metadata := range metadataList.Items {
		if appDetail, ok := appMap[metadata.Spec.App]; ok {
			appDetail.MergeMetaData(metadata)
		}
	}

	resp := make([]*model.ApplicationDetailResp, 0, len(appMap))
	for appName, appDetail := range appMap {
		respItem := &model.ApplicationDetailResp{
			AppName: appName,
		}
		resp = append(resp, respItem.FromApplicationDetail(appDetail))
	}

	return resp, nil
}

func GetApplicationTabInstanceInfo(rt core_runtime.Runtime, req *model.ApplicationTabInstanceInfoReq) ([]*model.ApplicationTabInstanceInfoResp, error) {
	manager := rt.ResourceManager()
	dataplaneList := &mesh.DataplaneResourceList{}

	if err := manager.List(rt.AppContext(), dataplaneList, store.ListByNameContains(req.AppName)); err != nil {
		return nil, err
	}

	res := make([]*model.ApplicationTabInstanceInfoResp, 0, len(dataplaneList.Items))
	for _, dataplane := range dataplaneList.Items {
		resItem := &model.ApplicationTabInstanceInfoResp{}
		resItem.FromDataplaneResource(dataplane)
		resItem.GetRegistry(rt)
		res = append(res, resItem)
	}

	return res, nil
}

func GetApplicationServiceFormInfo(rt core_runtime.Runtime, req *model.ApplicationServiceFormReq) ([]*model.ApplicationServiceFormResp, error) {
	manager := rt.ResourceManager()
	dataplaneList := &mesh.DataplaneResourceList{}
	if err := manager.List(rt.AppContext(), dataplaneList, store.ListByNameContains(req.AppName)); err != nil {
		return nil, err
	}

	metadataList := &mesh.MetaDataResourceList{}
	if err := manager.List(rt.AppContext(), metadataList, store.ListByNameContains(req.AppName)); err != nil {
		return nil, err
	}

	return getApplicationServiceFormInfoBySide(req.Side, metadataList)
}

func getApplicationServiceFormInfoBySide(side string, metadataList *mesh.MetaDataResourceList) ([]*model.ApplicationServiceFormResp, error) {
	res := make([]*model.ApplicationServiceFormResp, 0)
	serviceMap := make(map[string]*model.ApplicationServiceForm)
	for _, metadata := range metadataList.Items {
		for _, serviceInfo := range metadata.Spec.Services {
			if serviceInfo.Params[constants.ServiceInfoSide] == side {
				applicationServiceForm := model.NewApplicationServiceForm(serviceInfo.Name)
				if _, ok := serviceMap[serviceInfo.Name]; ok {
					serviceMap[serviceInfo.Name].FromServiceInfo(serviceInfo)
				} else {
					applicationServiceForm.FromServiceInfo(serviceInfo)
					serviceMap[serviceInfo.Name] = applicationServiceForm
				}
			}
		}
	}

	for _, applicationServiceForm := range serviceMap {
		applicationServiceFormResp := model.NewApplicationServiceFormResp()
		applicationServiceFormResp.FromApplicationServiceForm(applicationServiceForm)
		res = append(res, applicationServiceFormResp)
	}
	return res, nil
}

func GetApplicationSearchInfo(rt core_runtime.Runtime) ([]*model.ApplicationSearchResp, error) {
	manager := rt.ResourceManager()
	dataplaneList := &mesh.DataplaneResourceList{}
	if err := manager.List(rt.AppContext(), dataplaneList); err != nil {
		return nil, err
	}

	res := make([]*model.ApplicationSearchResp, 0)
	appMap := make(map[string]*model.ApplicationSearch)
	for _, dataplane := range dataplaneList.Items {
		appName := dataplane.Meta.GetLabels()[mesh_proto.AppTag]
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
