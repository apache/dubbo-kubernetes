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
	"github.com/apache/dubbo-kubernetes/pkg/admin/model"
	"github.com/apache/dubbo-kubernetes/pkg/core/resources/apis/mesh"
	core_model "github.com/apache/dubbo-kubernetes/pkg/core/resources/model"
	"github.com/apache/dubbo-kubernetes/pkg/core/resources/store"
	core_runtime "github.com/apache/dubbo-kubernetes/pkg/core/runtime"
	"strconv"
)

func GetServiceTabDistribution(rt core_runtime.Runtime, req *model.ServiceTabDistributionReq) ([]*model.ServiceTabDistributionResp, error) {
	manager := rt.ResourceManager()
	dataplaneList := &mesh.DataplaneResourceList{}

	if err := manager.List(rt.AppContext(), dataplaneList, store.ListByNameContains(req.AppName)); err != nil {
		return nil, err
	}

	res := make([]*model.ServiceTabDistributionResp, 0, len(dataplaneList.Items))
	for i, dataplane := range dataplaneList.Items {
		res[i] = &model.ServiceTabDistributionResp{}
		res[i] = res[i].FromServiceDataplaneResource(dataplane)
	}

	return res, nil
}

func GetSearchServices(rt core_runtime.Runtime, req *model.ServiceSearchReq) ([]*model.ServiceSearchResp, *core_model.Pagination, error) {
	manager := rt.ResourceManager()
	dataplaneList := &mesh.DataplaneResourceList{}

	if err := manager.List(rt.AppContext(), dataplaneList, store.ListByNameContains(req.AppName), store.ListByPage(req.PageSize, strconv.Itoa(req.CurPage))); err != nil {
		return nil, nil, err
	}

	res := make([]*model.ServiceSearchResp, 0, len(dataplaneList.Items))
	for i, dataplane := range dataplaneList.Items {
		res[i] = &model.ServiceSearchResp{}
		res[i] = res[i].FromServiceDataplaneResource(dataplane)
	}

	return res, &dataplaneList.Pagination, nil
}
