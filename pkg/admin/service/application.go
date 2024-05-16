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
		var applicationDetail *model.ApplicationDetail
		if _, ok := appMap[appName]; ok {
			applicationDetail = appMap[appName]
		} else {
			applicationDetail = model.NewApplicationDetail()
		}
		applicationDetail.Merge(dataplane)
		appMap[appName] = applicationDetail
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
		res = append(res, resItem.FromDataplaneResource(dataplane))
	}

	return res, nil
}
