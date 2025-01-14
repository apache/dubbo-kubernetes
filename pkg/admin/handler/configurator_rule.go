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

package handler

import (
	"fmt"
	"github.com/apache/dubbo-kubernetes/pkg/admin/service"
	"net/http"
	"strconv"
	"strings"
)

import (
	"github.com/gin-gonic/gin"
)

import (
	mesh_proto "github.com/apache/dubbo-kubernetes/api/mesh/v1alpha1"
	"github.com/apache/dubbo-kubernetes/pkg/admin/model"
	"github.com/apache/dubbo-kubernetes/pkg/core/consts"
	"github.com/apache/dubbo-kubernetes/pkg/core/resources/apis/mesh"
	"github.com/apache/dubbo-kubernetes/pkg/core/resources/store"
	core_runtime "github.com/apache/dubbo-kubernetes/pkg/core/runtime"
)

func ConfiguratorSearch(rt core_runtime.Runtime) gin.HandlerFunc {
	return func(c *gin.Context) {
		req := model.NewSearchConfiguratorReq()
		if err := c.ShouldBindQuery(req); err != nil {
			c.JSON(http.StatusBadRequest, model.NewErrorResp(err.Error()))
			return
		}
		ruleList := &mesh.DynamicConfigResourceList{}
		var respList []model.ConfiguratorSearchResp
		if req.Keywords == "" {
			if err := rt.ResourceManager().List(rt.AppContext(), ruleList, store.ListByPage(req.PageSize, strconv.Itoa(req.PageOffset))); err != nil {
				c.JSON(http.StatusBadRequest, model.NewErrorResp(err.Error()))
				return
			}
		} else {
			if err := rt.ResourceManager().List(rt.AppContext(), ruleList, store.ListByNameContains(req.Keywords), store.ListByPage(req.PageSize, strconv.Itoa(req.PageOffset))); err != nil {
				c.JSON(http.StatusBadRequest, model.NewErrorResp(err.Error()))
				return
			}
		}
		for _, item := range ruleList.Items {
			respList = append(respList, model.ConfiguratorSearchResp{
				RuleName:   item.Meta.GetName(),
				Scope:      item.Spec.GetScope(),
				CreateTime: item.Meta.GetCreationTime().String(),
				Enabled:    item.Spec.GetEnabled(),
			})
		}
		result := model.NewSearchPaginationResult()
		result.List = respList
		result.PageInfo = &ruleList.Pagination
		c.JSON(http.StatusOK, model.NewSuccessResp(result))
	}
}

func GetConfiguratorWithRuleName(rt core_runtime.Runtime) gin.HandlerFunc {
	return func(c *gin.Context) {
		var name string
		ruleName := c.Param("ruleName")
		if strings.HasSuffix(ruleName, consts.ConfiguratorRuleSuffix) {
			name = ruleName[:len(ruleName)-len(consts.ConfiguratorRuleSuffix)]
		} else {
			c.JSON(http.StatusBadRequest, model.NewErrorResp(fmt.Sprintf("ruleName must end with %s", consts.ConfiguratorRuleSuffix)))
			return
		}
		res, err := service.GetConfigurator(rt, name)
		if err != nil {
			c.JSON(http.StatusBadRequest, model.NewErrorResp(err.Error()))
			return
		}
		c.JSON(http.StatusOK, model.GenDynamicConfigToResp(res.Spec))
	}
}

func PutConfiguratorWithRuleName(rt core_runtime.Runtime) gin.HandlerFunc {
	return func(c *gin.Context) {
		var name string
		ruleName := c.Param("ruleName")
		if strings.HasSuffix(ruleName, consts.ConfiguratorRuleSuffix) {
			name = ruleName[:len(ruleName)-len(consts.ConfiguratorRuleSuffix)]
		} else {
			c.JSON(http.StatusBadRequest, model.NewErrorResp(fmt.Sprintf("ruleName must end with %s", consts.ConfiguratorRuleSuffix)))
			return
		}
		res := &mesh.DynamicConfigResource{
			Meta: nil,
			Spec: &mesh_proto.DynamicConfig{},
		}
		err := c.Bind(res.Spec)
		if err != nil {
			c.JSON(http.StatusBadRequest, model.NewErrorResp(err.Error()))
			return
		}
		if err = service.UpdateConfigurator(rt, name, res); err != nil {
			c.JSON(http.StatusBadRequest, model.NewErrorResp(err.Error()))
			return
		} else {
			c.JSON(http.StatusOK, model.GenDynamicConfigToResp(res.Spec))
		}
	}
}

func PostConfiguratorWithRuleName(rt core_runtime.Runtime) gin.HandlerFunc {
	return func(c *gin.Context) {
		var name string
		ruleName := c.Param("ruleName")
		if strings.HasSuffix(ruleName, consts.ConfiguratorRuleSuffix) {
			name = ruleName[:len(ruleName)-len(consts.ConfiguratorRuleSuffix)]
		} else {
			c.JSON(http.StatusBadRequest, model.NewErrorResp(fmt.Sprintf("ruleName must end with %s", consts.ConfiguratorRuleSuffix)))
			return
		}
		res := &mesh.DynamicConfigResource{
			Meta: nil,
			Spec: &mesh_proto.DynamicConfig{},
		}
		err := c.Bind(res.Spec)
		if err != nil {
			c.JSON(http.StatusBadRequest, model.NewErrorResp(err.Error()))
			return
		}
		if err = service.CreateConfigurator(rt, name, res); err != nil {
			c.JSON(http.StatusBadRequest, model.NewErrorResp(err.Error()))
			return
		} else {
			c.JSON(http.StatusOK, model.GenDynamicConfigToResp(res.Spec))
		}
	}
}

func DeleteConfiguratorWithRuleName(rt core_runtime.Runtime) gin.HandlerFunc {
	return func(c *gin.Context) {
		var name string
		ruleName := c.Param("ruleName")
		res := &mesh.DynamicConfigResource{Spec: &mesh_proto.DynamicConfig{}}
		if strings.HasSuffix(ruleName, consts.ConfiguratorRuleSuffix) {
			name = ruleName[:len(ruleName)-len(consts.ConfiguratorRuleSuffix)]
		} else {
			c.JSON(http.StatusBadRequest, model.NewErrorResp(fmt.Sprintf("ruleName must end with %s", consts.ConfiguratorRuleSuffix)))
			return
		}
		if err := service.DeleteConfigurator(rt, name, res); err != nil {
			c.JSON(http.StatusBadRequest, model.NewErrorResp(err.Error()))
			return
		}
		c.JSON(http.StatusOK, model.NewSuccessResp(""))
	}
}
