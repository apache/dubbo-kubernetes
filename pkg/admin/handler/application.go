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

// API Definition: https://app.apifox.com/project/3732499
// 资源详情-应用

import (
	"net/http"
)

import (
	"github.com/gin-gonic/gin"
)

import (
	mesh_proto "github.com/apache/dubbo-kubernetes/api/mesh/v1alpha1"
	"github.com/apache/dubbo-kubernetes/pkg/admin/model"
	"github.com/apache/dubbo-kubernetes/pkg/admin/service"
	"github.com/apache/dubbo-kubernetes/pkg/core/consts"
	core_mesh "github.com/apache/dubbo-kubernetes/pkg/core/resources/apis/mesh"
	core_store "github.com/apache/dubbo-kubernetes/pkg/core/resources/store"
	core_runtime "github.com/apache/dubbo-kubernetes/pkg/core/runtime"
)

func GetApplicationDetail(rt core_runtime.Runtime) gin.HandlerFunc {
	return func(c *gin.Context) {
		req := &model.ApplicationDetailReq{}
		if err := c.ShouldBindQuery(req); err != nil {
			c.JSON(http.StatusBadRequest, model.NewErrorResp(err.Error()))
			return
		}
		resp, err := service.GetApplicationDetail(rt, req)
		if err != nil {
			c.JSON(http.StatusInternalServerError, model.NewErrorResp(err.Error()))
			return
		}
		c.JSON(http.StatusOK, model.NewSuccessResp(resp))
	}
}

func GetApplicationTabInstanceInfo(rt core_runtime.Runtime) gin.HandlerFunc {
	return func(c *gin.Context) {
		req := &model.ApplicationTabInstanceInfoReq{}
		if err := c.ShouldBindQuery(req); err != nil {
			c.JSON(http.StatusBadRequest, model.NewErrorResp(err.Error()))
			return
		}

		resp, err := service.GetApplicationTabInstanceInfo(rt, req)
		if err != nil {
			c.JSON(http.StatusInternalServerError, model.NewErrorResp(err.Error()))
			return
		}

		c.JSON(http.StatusOK, model.NewSuccessResp(resp))
	}
}

func GetApplicationServiceForm(rt core_runtime.Runtime) gin.HandlerFunc {
	return func(c *gin.Context) {
		req := &model.ApplicationServiceFormReq{}
		if err := c.ShouldBindQuery(req); err != nil {
			c.JSON(http.StatusBadRequest, model.NewErrorResp(err.Error()))
			return
		}
		resp, err := service.GetApplicationServiceFormInfo(rt, req)
		if err != nil {
			c.JSON(http.StatusBadRequest, model.NewErrorResp(err.Error()))
			return
		}
		c.JSON(http.StatusOK, model.NewSuccessResp(resp))
	}
}

func ApplicationSearch(rt core_runtime.Runtime) gin.HandlerFunc {
	return func(c *gin.Context) {
		resp, err := service.GetApplicationSearchInfo(rt)
		if err != nil {
			c.JSON(http.StatusBadRequest, model.NewErrorResp(err.Error()))
			return
		}
		c.JSON(http.StatusOK, model.NewSuccessResp(resp))
	}
}

func isAppOperatorLogOpened(conf *mesh_proto.OverrideConfig, appName string) bool {
	if conf.Side != "provider" ||
		!conf.Enabled ||
		conf.Parameters == nil ||
		conf.Match == nil ||
		conf.Match.Application == nil ||
		conf.Match.Application.Oneof == nil ||
		len(conf.Match.Application.Oneof) != 1 ||
		conf.Match.Application.Oneof[0].Exact != appName {
		return false
	} else if val, ok := conf.Parameters[`accesslog`]; !ok || val != `true` {
		return false
	}
	return true
}

func generatorDefaultConfigurator(name string, scope string, version string, enabled bool) *core_mesh.DynamicConfigResource {
	return &core_mesh.DynamicConfigResource{
		Meta: nil,
		Spec: &mesh_proto.DynamicConfig{
			Key:           name,
			Scope:         scope,
			ConfigVersion: version,
			Enabled:       enabled,
			Configs:       make([]*mesh_proto.OverrideConfig, 0),
		},
	}
}

func ApplicationConfigOperatorLogPut(rt core_runtime.Runtime) gin.HandlerFunc {
	return func(c *gin.Context) {
		var (
			param = struct {
				ApplicationName string `json:"applicationName"`
				OperatorLog     bool   `json:"operatorLog"`
			}{}
			isNotExist = false
		)
		err := c.BindQuery(&param)
		if err != nil {
			c.JSON(http.StatusBadRequest, model.NewErrorResp(err.Error()))
			return
		}
		res, err := getConfigurator(rt, param.ApplicationName)
		if err != nil {
			if core_store.IsResourceNotFound(err) {
				// for check app exist
				_, err = service.GetApplicationDetail(rt, &model.ApplicationDetailReq{AppName: param.ApplicationName})
				if err != nil {
					c.JSON(http.StatusBadRequest, model.NewErrorResp(err.Error()))
					return
				}
				res = generatorDefaultConfigurator(param.ApplicationName, "Application", consts.DefaultConfiguratorVersion, true)
				isNotExist = true
			} else {
				c.JSON(http.StatusInternalServerError, model.NewErrorResp(err.Error()))
				return
			}
		}
		// append or remove
		if param.OperatorLog {
			res.Spec.Configs = append(res.Spec.Configs, &mesh_proto.OverrideConfig{
				Side:          "provider",
				Parameters:    map[string]string{`accesslog`: `true`},
				Enabled:       true,
				Match:         &mesh_proto.ConditionMatch{Application: &mesh_proto.ListStringMatch{Oneof: []*mesh_proto.StringMatch{{Exact: param.ApplicationName}}}},
				XGenerateByCp: true,
			})
		} else {
			res.Spec.RangeConfigsToRemove(func(conf *mesh_proto.OverrideConfig) (IsRemove bool) {
				if conf == nil {
					return true
				}
				return isAppOperatorLogOpened(conf, param.ApplicationName)
			})
		}
		// rewrite
		if isNotExist {
			err = createConfigurator(rt, param.ApplicationName, res)
			if err != nil {
				c.JSON(http.StatusBadRequest, model.NewErrorResp(err.Error()))
				return
			}
		} else {
			err = updateConfigurator(rt, param.ApplicationName, res)
			if err != nil {
				c.JSON(http.StatusBadRequest, model.NewErrorResp(err.Error()))
				return
			}
		}
		c.JSON(http.StatusOK, model.NewSuccessResp(nil))
	}
}

func ApplicationConfigOperatorLogGet(rt core_runtime.Runtime) gin.HandlerFunc {
	return func(c *gin.Context) {
		var (
			param = struct {
				ApplicationName string `json:"applicationName"`
			}{}
		)
		err := c.BindQuery(&param)
		if err != nil {
			c.JSON(http.StatusBadRequest, model.NewErrorResp(err.Error()))
			return
		}
		res, err := getConfigurator(rt, param.ApplicationName)
		if err != nil {
			if core_store.IsResourceNotFound(err) {
				c.JSON(http.StatusOK, model.NewSuccessResp(map[string]interface{}{"operatorLog": false}))
				return
			}
			c.JSON(http.StatusBadRequest, model.NewErrorResp(err.Error()))
			return
		}
		isExist := false
		res.Spec.Range(func(conf *mesh_proto.OverrideConfig) (isStop bool) {
			if isExist = isAppOperatorLogOpened(conf, param.ApplicationName); isExist {
				return true
			}
			return false
		})
		c.JSON(http.StatusOK, model.NewSuccessResp(map[string]interface{}{"operatorLog": isExist}))
	}
}

func ApplicationConfigFlowWeightGET(rt core_runtime.Runtime) gin.HandlerFunc {
	return func(c *gin.Context) {

	}
}

func ApplicationConfigFlowWeightPUT(rt core_runtime.Runtime) gin.HandlerFunc {
	return func(c *gin.Context) {

	}
}

func ApplicationConfigGrayGET(rt core_runtime.Runtime) gin.HandlerFunc {
	return func(c *gin.Context) {

	}
}

func ApplicationConfigGrayPUT(rt core_runtime.Runtime) gin.HandlerFunc {
	return func(c *gin.Context) {

	}
}
