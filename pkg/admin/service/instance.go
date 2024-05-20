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
	"strconv"

	"github.com/apache/dubbo-kubernetes/pkg/admin/model"
	"github.com/apache/dubbo-kubernetes/pkg/core/resources/apis/mesh"
	core_model "github.com/apache/dubbo-kubernetes/pkg/core/resources/model"
	"github.com/apache/dubbo-kubernetes/pkg/core/resources/store"
	core_runtime "github.com/apache/dubbo-kubernetes/pkg/core/runtime"
)

func SearchInstances(rt core_runtime.Runtime, req *model.SearchInstanceReq) ([]*model.SearchInstanceResp, *core_model.Pagination, error) {
	manager := rt.ResourceManager()
	dataplaneList := &mesh.DataplaneResourceList{}

	if err := manager.List(rt.AppContext(), dataplaneList, store.ListByNameContains(req.AppName), store.ListByPage(req.PageSize, strconv.Itoa(req.CurPage))); err != nil {
		return nil, nil, err
	}

	res := make([]*model.SearchInstanceResp, len(dataplaneList.Items))
	for i, item := range dataplaneList.Items {
		res[i] = &model.SearchInstanceResp{}
		res[i] = res[i].FromDataplaneResource(item)
	}

	return res, &dataplaneList.Pagination, nil
}

func GetInstanceDetail(rt core_runtime.Runtime, req *model.InstanceDetailReq) ([]*model.InstanceDetailResp, error) {
	manager := rt.ResourceManager()
	dataplaneList := &mesh.DataplaneResourceList{}

	if err := manager.List(rt.AppContext(), dataplaneList, store.ListByNameContains(req.InstanceName)); err != nil {
		return nil, err
	}

	instMap := make(map[string]*model.InstanceDetail)
	for _, dataplane := range dataplaneList.Items {
		instName := dataplane.Meta.GetLabels()[mesh_proto.InstanceTag]
		var instanceDetail *model.InstanceDetail
		if _, ok := instMap[instName]; ok {
			instanceDetail = instMap[instName]
		} else {
			instanceDetail = model.NewInstanceDetail()
		}
		instanceDetail.Merge(dataplane) //convert dataplane info to instance detail
		instMap[instName] = instanceDetail
	}

	resp := make([]*model.InstanceDetailResp, 0, len(instMap))
	for _, instDetail := range instMap {
		respItem := &model.InstanceDetailResp{}
		resp = append(resp, respItem.FromInstanceDetail(instDetail))
	}

	return resp, nil
}
