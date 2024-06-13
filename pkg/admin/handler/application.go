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
	"github.com/apache/dubbo-kubernetes/pkg/core/logger"
	"net/http"
	"strconv"
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
			ApplicationName string
			OperatorLog     bool
			isNotExist      = false
		)
		ApplicationName = c.Query("appName")
		OperatorLog, err := strconv.ParseBool(c.Query("operatorLog"))
		if err != nil {
			c.JSON(http.StatusBadRequest, model.NewErrorResp(err.Error()))
			return
		}
		res, err := getConfigurator(rt, ApplicationName)
		if err != nil {
			if core_store.IsResourceNotFound(err) {
				// for check app exist
				data, err := service.GetApplicationDetail(rt, &model.ApplicationDetailReq{AppName: ApplicationName})
				if err != nil || len(data) == 0 {
					c.JSON(http.StatusBadRequest, model.NewErrorResp(err.Error()))
					return
				}
				res = generatorDefaultConfigurator(ApplicationName, "Application", consts.DefaultConfiguratorVersion, true)
				isNotExist = true
			} else {
				c.JSON(http.StatusInternalServerError, model.NewErrorResp(err.Error()))
				return
			}
		}
		// append or remove
		if OperatorLog {
			// check is already exist
			alreadyExist := false
			res.Spec.Range(func(conf *mesh_proto.OverrideConfig) (isStop bool) {
				alreadyExist = isAppOperatorLogOpened(conf, ApplicationName)
				return alreadyExist
			})
			if alreadyExist {
				c.JSON(http.StatusOK, model.NewSuccessResp(nil))
				return
			}
			res.Spec.Configs = append(res.Spec.Configs, &mesh_proto.OverrideConfig{
				Side:          "provider",
				Parameters:    map[string]string{`accesslog`: `true`},
				Enabled:       true,
				Match:         &mesh_proto.ConditionMatch{Application: &mesh_proto.ListStringMatch{Oneof: []*mesh_proto.StringMatch{{Exact: ApplicationName}}}},
				XGenerateByCp: true,
			})
		} else {
			res.Spec.RangeConfigsToRemove(func(conf *mesh_proto.OverrideConfig) (IsRemove bool) {
				if conf == nil {
					return true
				}
				return isAppOperatorLogOpened(conf, ApplicationName)
			})
		}
		// restore
		if isNotExist {
			err = createConfigurator(rt, ApplicationName, res)
			if err != nil {
				c.JSON(http.StatusBadRequest, model.NewErrorResp(err.Error()))
				return
			}
		} else {
			err = updateConfigurator(rt, ApplicationName, res)
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
		ApplicationName := c.Query("appName")
		if ApplicationName == "" {
			c.JSON(http.StatusBadRequest, model.NewErrorResp("appName is required"))
			return
		}
		res, err := getConfigurator(rt, ApplicationName)
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
			if isExist = isAppOperatorLogOpened(conf, ApplicationName); isExist {
				return true
			}
			return false
		})
		c.JSON(http.StatusOK, model.NewSuccessResp(map[string]interface{}{"operatorLog": isExist}))
	}
}

func ApplicationConfigFlowWeightGET(rt core_runtime.Runtime) gin.HandlerFunc {
	return func(c *gin.Context) {
		var (
			ApplicationName string
			resp            = struct {
				FlowWeightSets []model.FlowWeightSet `json:"flowWeightSets"`
			}{}
		)
		ApplicationName = c.Query("appName")
		if ApplicationName == "" {
			c.JSON(http.StatusBadRequest, model.NewErrorResp("appName is required"))
			return
		}

		resp.FlowWeightSets = make([]model.FlowWeightSet, 0)

		res, err := getConfigurator(rt, ApplicationName)
		if err != nil {
			if core_store.IsResourceNotFound(err) {
				c.JSON(http.StatusOK, model.NewSuccessResp(resp))
				return
			}
			c.JSON(http.StatusBadRequest, model.NewErrorResp(err.Error()))
			return
		}

		w := 0
		res.Spec.Range(func(conf *mesh_proto.OverrideConfig) (isStop bool) {
			if isFlowWeight(conf) {
				w, err = strconv.Atoi(conf.Parameters[`weight`])
				if err != nil {
					logger.Error("parse weight failed", err)
					return true
				}
				resp.FlowWeightSets = append(resp.FlowWeightSets, model.FlowWeightSet{
					Weight: int32(w),
					Scope: model.RespParamMatch{
						Key:   &conf.Match.Param[0].Key,
						Value: model.StringMatchToModelStringMatch(conf.Match.Param[0].Value),
					},
				})
			}
			return false
		})
		if err != nil {
			c.JSON(http.StatusOK, model.NewErrorResp(err.Error()))
			return
		}
		c.JSON(http.StatusOK, model.NewSuccessResp(resp))
	}
}

func isFlowWeight(conf *mesh_proto.OverrideConfig) bool {
	if conf.Side != "provider" ||
		conf.Parameters == nil ||
		conf.Match == nil ||
		conf.Match.Param == nil ||
		len(conf.Match.Param) != 1 ||
		conf.Match.Param[0].Value == nil {
		return false
	} else if _, ok := conf.Parameters[`weight`]; !ok {
		return false
	}
	return true
}

func ApplicationConfigFlowWeightPUT(rt core_runtime.Runtime) gin.HandlerFunc {
	return func(c *gin.Context) {
		var (
			ApplicationName = ""
			body            = struct {
				FlowWeightSets []model.FlowWeightSet `json:"flowWeightSets"`
			}{}
		)
		ApplicationName = c.Query("appName")
		if ApplicationName == "" {
			c.JSON(http.StatusBadRequest, model.NewErrorResp("application name is required"))
		}
		if err := c.Bind(&body); err != nil {
			c.JSON(http.StatusBadRequest, model.NewErrorResp(err.Error()))
			return
		}

		isNotExist := false
		res, err := getConfigurator(rt, ApplicationName)
		if err != nil {
			if core_store.IsResourceNotFound(err) {
				// for check app exist
				data, err := service.GetApplicationDetail(rt, &model.ApplicationDetailReq{AppName: ApplicationName})
				if err != nil || len(data) == 0 {
					c.JSON(http.StatusBadRequest, model.NewErrorResp(err.Error()))
					return
				}
				res = generatorDefaultConfigurator(ApplicationName, "Application", consts.DefaultConfiguratorVersion, true)
				isNotExist = true
			} else {
				c.JSON(http.StatusInternalServerError, model.NewErrorResp(err.Error()))
				return
			}
		}

		res.Spec.RangeConfigsToRemove(func(conf *mesh_proto.OverrideConfig) (IsRemove bool) {
			return isFlowWeight(conf)
		})

		for _, set := range body.FlowWeightSets {
			// append to res
			res.Spec.Configs = append(res.Spec.Configs, &mesh_proto.OverrideConfig{
				Side:       "provider",
				Parameters: map[string]string{`weight`: string(set.Weight)},
				Match: &mesh_proto.ConditionMatch{
					Param: []*mesh_proto.ParamMatch{{
						Key:   *set.Scope.Key,
						Value: model.ModelStringMatchToStringMatch(set.Scope.Value),
					}},
				},
				XGenerateByCp: true,
			})
		}

		if isNotExist {
			err = createConfigurator(rt, ApplicationName, res)
			if err != nil {
				c.JSON(http.StatusBadRequest, model.NewErrorResp(err.Error()))
				return
			}
		} else {
			err = updateConfigurator(rt, ApplicationName, res)
			if err != nil {
				c.JSON(http.StatusBadRequest, model.NewErrorResp(err.Error()))
				return
			}
		}
		c.JSON(http.StatusOK, model.NewSuccessResp(nil))
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
