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
// 资源详情-实例

import (
	mesh_proto "github.com/apache/dubbo-kubernetes/api/mesh/v1alpha1"
	"github.com/apache/dubbo-kubernetes/pkg/core/consts"
	"net/http"
	"strconv"
)

import (
	"github.com/gin-gonic/gin"
)

import (
	"github.com/apache/dubbo-kubernetes/pkg/admin/model"
	"github.com/apache/dubbo-kubernetes/pkg/admin/service"
	core_store "github.com/apache/dubbo-kubernetes/pkg/core/resources/store"
	core_runtime "github.com/apache/dubbo-kubernetes/pkg/core/runtime"
)

func GetInstanceDetail(rt core_runtime.Runtime) gin.HandlerFunc {
	return func(c *gin.Context) {
		req := &model.InstanceDetailReq{}
		if err := c.ShouldBindQuery(req); err != nil {
			c.JSON(http.StatusBadRequest, model.NewErrorResp(err.Error()))
			return
		}

		resp, err := service.GetInstanceDetail(rt, req)
		if err != nil {
			c.JSON(http.StatusInternalServerError, model.NewErrorResp(err.Error()))
			return
		}

		if len(resp) == 0 {
			c.JSON(http.StatusNotFound, model.NewErrorResp("instance not exist"))
		}
		c.JSON(http.StatusOK, model.NewSuccessResp(resp[0]))
	}
}

func SearchInstances(rt core_runtime.Runtime) gin.HandlerFunc {
	return func(c *gin.Context) {
		req := &model.SearchInstanceReq{}
		if err := c.ShouldBindQuery(req); err != nil {
			c.JSON(http.StatusBadRequest, model.NewErrorResp(err.Error()))
			return
		}

		instances, _, err := service.SearchInstances(rt, req)
		if err != nil {
			c.JSON(http.StatusInternalServerError, model.NewErrorResp(err.Error()))
			return
		}

		c.JSON(http.StatusOK, model.NewSuccessResp(instances))
	}
}

func InstanceConfigTrafficDisableGET(rt core_runtime.Runtime) gin.HandlerFunc {
	return func(c *gin.Context) {

	}
}

func InstanceConfigTrafficDisablePUT(rt core_runtime.Runtime) gin.HandlerFunc {
	return func(c *gin.Context) {

	}
}

func InstanceConfigOperatorLogGET(rt core_runtime.Runtime) gin.HandlerFunc {
	return func(c *gin.Context) {
		var (
			resp = struct {
				OperatorLog bool `json:"operatorLog"`
			}{false}
		)
		applicationName := c.Query(`appName`)
		if applicationName == "" {
			c.JSON(http.StatusBadRequest, model.NewErrorResp("application name is empty"))
		}
		instanceIP := c.Query(`instanceIP`)
		if instanceIP == "" {
			c.JSON(http.StatusBadRequest, model.NewErrorResp("instanceIP is empty"))
		}

		res, err := getConfigurator(rt, applicationName)
		if err != nil {
			if core_store.IsResourceNotFound(err) {
				c.JSON(http.StatusOK, model.NewSuccessResp(resp))
				return
			}
			c.JSON(http.StatusNotFound, model.NewErrorResp(err.Error()))
			return
		}

		if res.Spec.Enabled {
			res.Spec.Range(func(conf *mesh_proto.OverrideConfig) (isStop bool) {
				resp.OperatorLog = isInstanceOperatorLogOpen(conf, instanceIP)
				return resp.OperatorLog
			})
		}

		c.JSON(http.StatusOK, resp)
	}
}

func isInstanceOperatorLogOpen(conf *mesh_proto.OverrideConfig, IP string) bool {
	if conf != nil &&
		conf.Match != nil &&
		conf.Match.Address != nil &&
		conf.Match.Address.Wildcard == `"`+IP+`:*"` &&
		conf.Side == "provider" &&
		conf.Parameters != nil &&
		conf.Parameters[`accesslog`] == `true` {
		return true
	}
	return false
}

func InstanceConfigOperatorLogPUT(rt core_runtime.Runtime) gin.HandlerFunc {
	return func(c *gin.Context) {
		applicationName := c.Query(`appName`)
		if applicationName == "" {
			c.JSON(http.StatusBadRequest, model.NewErrorResp("application name is empty"))
		}
		instanceIP := c.Query(`instanceIP`)
		if instanceIP == "" {
			c.JSON(http.StatusBadRequest, model.NewErrorResp("instanceIP is empty"))
		}
		adminOperatorLog, err := strconv.ParseBool(c.Query(`OperatorLog`))
		if err != nil {
			c.JSON(http.StatusBadRequest, model.NewErrorResp(err.Error()))
		}

		res, err := getConfigurator(rt, applicationName)
		notExist := false
		if err != nil {
			if !core_store.IsResourceNotFound(err) {
				c.JSON(http.StatusNotFound, model.NewErrorResp(err.Error()))
				return
			}
			res = generateDefaultConfigurator(applicationName, `application`, consts.DefaultConfiguratorVersion, true)
			notExist = true
		}

		if adminOperatorLog {
			res.Spec.RangeConfigsToRemove(func(conf *mesh_proto.OverrideConfig) (IsRemove bool) {
				return isInstanceOperatorLogOpen(conf, instanceIP)
			})
		} else {
			var isExist bool
			res.Spec.Range(func(conf *mesh_proto.OverrideConfig) (isStop bool) {
				isExist = isInstanceOperatorLogOpen(conf, instanceIP)
				return isExist
			})
			if !isExist {
				res.Spec.Configs = append(res.Spec.Configs, &mesh_proto.OverrideConfig{
					Side:          "provider",
					Match:         &mesh_proto.ConditionMatch{Address: &mesh_proto.AddressMatch{Wildcard: `"` + instanceIP + `:*"`}},
					Parameters:    map[string]string{`accesslog`: `true`},
					XGenerateByCp: true,
				})
			}
		}

		if notExist {
			err = createConfigurator(rt, applicationName, res)
			if err != nil {
				c.JSON(http.StatusInternalServerError, model.NewErrorResp(err.Error()))
				return
			}
		} else {
			err = updateConfigurator(rt, applicationName, res)
			if err != nil {
				c.JSON(http.StatusInternalServerError, model.NewErrorResp(err.Error()))
				return
			}
		}

		c.JSON(http.StatusOK, model.NewSuccessResp(nil))
	}
}
