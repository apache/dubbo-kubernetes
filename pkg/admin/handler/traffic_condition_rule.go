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
	"context"
	mesh_proto "github.com/apache/dubbo-kubernetes/api/mesh/v1alpha1"
	"github.com/apache/dubbo-kubernetes/pkg/admin/model"
	"github.com/apache/dubbo-kubernetes/pkg/core/resources/apis/mesh"
	res_model "github.com/apache/dubbo-kubernetes/pkg/core/resources/model"
	"github.com/apache/dubbo-kubernetes/pkg/core/resources/store"
	core_runtime "github.com/apache/dubbo-kubernetes/pkg/core/runtime"
	"github.com/gin-gonic/gin"
	"net/http"
)

type ConditionRuleSearchWithPath struct{}

func ConditionRuleSearch(rt core_runtime.Runtime) gin.HandlerFunc {
	return func(c *gin.Context) {
		resList := &mesh.ConditionRouteResourceList{}
		if err := rt.ResourceManager().List(rt.AppContext(), resList); err != nil {
			c.JSON(http.StatusBadRequest, model.NewErrorResp(err.Error()))
			return
		}
		resp := model.ConditionRuleSearchResp{
			Code:    200,
			Message: "success",
			Data:    make([]model.ConditionRuleSearchResp_Data, 0, len(resList.Items)),
		}
		for _, item := range resList.Items {
			resp.Data = append(resp.Data, model.ConditionRuleSearchResp_Data{
				RuleName:   item.Meta.GetName(),
				Scope:      item.Spec.GetScope(),
				CreateTime: item.Meta.GetCreationTime().String(),
				Enabled:    item.Spec.GetEnabled(),
			})
		}
		c.JSON(http.StatusOK, resp)
	}
}

func GetConditionRuleWithRuleName(rt core_runtime.Runtime) gin.HandlerFunc {
	return func(c *gin.Context) {
		ruleName := c.Param("ruleName")
		res := &mesh.ConditionRouteResource{}
		ctx := context.WithValue(rt.AppContext(), ConditionRuleSearchWithPath{}, ruleName)
		if err := rt.ResourceManager().Get(ctx, res, store.GetByKey(res_model.DefaultMesh, res_model.DefaultMesh)); err != nil {
			c.JSON(http.StatusBadRequest, model.NewErrorResp(err.Error()))
			return
		}
		c.JSON(http.StatusOK, model.ConditionRuleResp{
			Code:    200,
			Message: "success",
			Data:    res.Spec,
		})
	}
}

func PutConditionRuleWithRuleName(rt core_runtime.Runtime) gin.HandlerFunc {
	return func(c *gin.Context) {
		ruleName := c.Param("ruleName")
		res := &mesh.ConditionRouteResource{
			Meta: nil,
			Spec: &mesh_proto.ConditionRoute{},
		}
		err := c.Bind(res.Spec)
		if err != nil {
			c.JSON(http.StatusBadRequest, model.NewErrorResp(err.Error()))
			return
		}
		ctx := context.WithValue(rt.AppContext(), ConditionRuleSearchWithPath{}, ruleName)
		if err = rt.ResourceManager().Update(ctx, res, store.UpdateByKey(res_model.DefaultMesh, res_model.DefaultMesh)); err != nil {
			c.JSON(http.StatusBadRequest, model.NewErrorResp(err.Error()))
			return
		} else {
			c.JSON(http.StatusOK, model.ConditionRuleResp{
				Code:    200,
				Message: "success",
				Data:    &mesh_proto.ConditionRoute{},
			})
		}
	}
}

func PostConditionRuleWithRuleName(rt core_runtime.Runtime) gin.HandlerFunc {
	return func(c *gin.Context) {
		ruleName := c.Param("ruleName")
		res := &mesh.ConditionRouteResource{
			Meta: nil,
			Spec: &mesh_proto.ConditionRoute{},
		}

		err := c.Bind(res.Spec)
		if err != nil {
			c.JSON(http.StatusBadRequest, model.NewErrorResp(err.Error()))
			return
		}
		ctx := context.WithValue(rt.AppContext(), ConditionRuleSearchWithPath{}, ruleName)
		if err = rt.ResourceManager().Create(ctx, res, store.CreateByKey(res_model.DefaultMesh, res_model.DefaultMesh)); err != nil {
			c.JSON(http.StatusBadRequest, model.NewErrorResp(err.Error()))
			return
		} else {
			c.JSON(http.StatusCreated, model.ConditionRuleResp{
				Code:    200,
				Message: "success",
				Data:    &mesh_proto.ConditionRoute{},
			})
		}
	}
}

func DeleteConditionRuleWithRuleName(rt core_runtime.Runtime) gin.HandlerFunc {
	return func(c *gin.Context) {
		ruleName := c.Param("ruleName")
		res := &mesh.ConditionRouteResource{}
		ctx := context.WithValue(rt.AppContext(), ConditionRuleSearchWithPath{}, ruleName)
		if err := rt.ResourceManager().Delete(ctx, res, store.DeleteByKey(res_model.DefaultMesh, res_model.DefaultMesh)); err != nil {
			c.JSON(http.StatusBadRequest, model.NewErrorResp(err.Error()))
			return
		}
		c.JSON(http.StatusNoContent, model.ConditionRuleResp{
			Code:    1000,
			Message: "success",
			Data:    &mesh_proto.ConditionRoute{},
		})
	}
}
