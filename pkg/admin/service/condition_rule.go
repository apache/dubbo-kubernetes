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
	"github.com/apache/dubbo-kubernetes/pkg/core/consts"
	"github.com/apache/dubbo-kubernetes/pkg/core/logger"
	"github.com/apache/dubbo-kubernetes/pkg/core/resources/apis/mesh"
	res_model "github.com/apache/dubbo-kubernetes/pkg/core/resources/model"
	"github.com/apache/dubbo-kubernetes/pkg/core/resources/store"
	core_runtime "github.com/apache/dubbo-kubernetes/pkg/core/runtime"
	"strconv"
)

func SearchConditionRules(rt core_runtime.Runtime, req *model.SearchConditionRuleReq) (*model.SearchPaginationResult, error) {
	ruleList := &mesh.ConditionRouteResourceList{}
	if req.Keywords == "" {
		if err := rt.ResourceManager().List(rt.AppContext(), ruleList, store.ListByPage(req.PageSize, strconv.Itoa(req.PageOffset))); err != nil {
			return nil, err
		}
	} else {
		if err := rt.ResourceManager().List(rt.AppContext(), ruleList, store.ListByNameContains(req.Keywords), store.ListByPage(req.PageSize, strconv.Itoa(req.PageOffset))); err != nil {
			return nil, err
		}
	}

	var respList []model.ConditionRuleSearchResp
	for _, item := range ruleList.Items {
		if v3 := item.Spec.ToConditionRouteV3(); v3 != nil {
			respList = append(respList, model.ConditionRuleSearchResp{
				RuleName:   item.Meta.GetName(),
				Scope:      v3.GetScope(),
				CreateTime: item.Meta.GetCreationTime().String(),
				Enabled:    v3.GetEnabled(),
			})
		} else if v3x1 := item.Spec.ToConditionRouteV3x1(); v3x1 != nil {
			respList = append(respList, model.ConditionRuleSearchResp{
				RuleName:   item.Meta.GetName(),
				Scope:      v3x1.GetScope(),
				CreateTime: item.Meta.GetCreationTime().String(),
				Enabled:    v3x1.GetEnabled(),
			})
		} else {
			logger.Errorf("Invalid condition route %v", item)
		}
	}
	result := model.NewSearchPaginationResult()
	result.List = respList
	result.PageInfo = &ruleList.Pagination
	return result, nil
}

func GetConditionRule(rt core_runtime.Runtime, name string) (*mesh.ConditionRouteResource, error) {
	res := &mesh.ConditionRouteResource{Spec: &mesh_proto.ConditionRoute{}}
	if err := rt.ResourceManager().Get(rt.AppContext(), res,
		// here `name` may be service-name or app-name, set *ByApplication(`name`) is ok.
		store.GetByApplication(name), store.GetByKey(name+consts.ConditionRuleSuffix, res_model.DefaultMesh)); err != nil {
		logger.Warnf("get %s condition failed with error: %s", name, err.Error())
		return nil, err
	}
	return res, nil
}

func UpdateConditionRule(rt core_runtime.Runtime, name string, res *mesh.ConditionRouteResource) error {
	if err := rt.ResourceManager().Update(rt.AppContext(), res,
		// here `name` may be service-name or app-name, set *ByApplication(`name`) is ok.
		store.UpdateByApplication(name), store.UpdateByKey(name+consts.ConditionRuleSuffix, res_model.DefaultMesh)); err != nil {
		logger.Warnf("update %s condition failed with error: %s", name, err.Error())
		return err
	}
	return nil
}

func CreateConditionRule(rt core_runtime.Runtime, name string, res *mesh.ConditionRouteResource) error {
	if err := rt.ResourceManager().Create(rt.AppContext(), res,
		// here `name` may be service-name or app-name, set *ByApplication(`name`) is ok.
		store.CreateByApplication(name), store.CreateByKey(name+consts.ConditionRuleSuffix, res_model.DefaultMesh)); err != nil {
		logger.Warnf("create %s condition failed with error: %s", name, err.Error())
		return err
	}
	return nil
}

func DeleteConditionRule(rt core_runtime.Runtime, name string, res *mesh.ConditionRouteResource) error {
	if err := rt.ResourceManager().Delete(rt.AppContext(), res,
		// here `name` may be service-name or app-name, set *ByApplication(`name`) is ok.
		store.DeleteByApplication(name), store.DeleteByKey(name+consts.ConditionRuleSuffix, res_model.DefaultMesh)); err != nil {
		logger.Warnf("delete %s condition failed with error: %s", name, err.Error())
		return err
	}
	return nil
}
