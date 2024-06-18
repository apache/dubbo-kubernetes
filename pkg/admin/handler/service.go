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
	"net/http"
	"strconv"
)

import (
	"github.com/gin-gonic/gin"
)

import (
	mesh_proto "github.com/apache/dubbo-kubernetes/api/mesh/v1alpha1"
	"github.com/apache/dubbo-kubernetes/pkg/admin/model"
	"github.com/apache/dubbo-kubernetes/pkg/core/consts"
	"github.com/apache/dubbo-kubernetes/pkg/core/resources/apis/mesh"
	core_store "github.com/apache/dubbo-kubernetes/pkg/core/resources/store"
	core_runtime "github.com/apache/dubbo-kubernetes/pkg/core/runtime"
)

// SearchServices API Definition: https://app.apifox.com/project/3732499
// 资源详情-应用
func SearchServices(rt core_runtime.Runtime) gin.HandlerFunc {
	return func(c *gin.Context) {
		// req := &model.SearchInstanceReq{}

		c.JSON(http.StatusOK, model.NewSuccessResp(""))
	}
}

func ListServices(rt core_runtime.Runtime) gin.HandlerFunc {
	return func(c *gin.Context) {
		// req := &model.SearchInstanceReq{}

		c.JSON(http.StatusOK, model.NewSuccessResp(""))
	}
}

func GetServiceDetail(rt core_runtime.Runtime) gin.HandlerFunc {
	return func(c *gin.Context) {
		// req := &model.SearchInstanceReq{}

		c.JSON(http.StatusOK, model.NewSuccessResp(""))
	}
}

type baseService struct {
	Service string `json:"service"`
	Group   string `json:"group"`
	Version string `json:"version"`
}

func (b baseService) toInterface() string {
	return b.Service + consts.Colon + b.Group + consts.Colon + b.Version
}

func ServiceConfigTimeoutGET(rt core_runtime.Runtime) gin.HandlerFunc {
	return func(c *gin.Context) {
		param := baseService{}
		resp := struct {
			Timeout int32 `json:"timeout"`
		}{-1}
		if err := c.Bind(&param); err != nil {
			c.JSON(http.StatusBadRequest, model.NewErrorResp(err.Error()))
			return
		}
		res, err := getConfigurator(rt, param.toInterface())
		if err != nil {
			c.JSON(http.StatusBadRequest, model.NewErrorResp(err.Error()))
			return
		}

		res.Spec.RangeConfig(func(conf *mesh_proto.OverrideConfig) (isStop bool) {
			resp.Timeout, isStop = getServiceTimeout(conf)
			return isStop
		})

		c.JSON(http.StatusOK, model.NewSuccessResp(resp))
	}
}

func getServiceTimeout(conf *mesh_proto.OverrideConfig) (int32, bool) {
	if conf.Side == consts.SideProvider && conf.Parameters != nil && conf.Parameters[`timeout`] != "" {
		timeout, err := strconv.Atoi(conf.Parameters[`timeout`])
		if err == nil {
			return int32(timeout), true
		}
	}
	return -1, false
}

func ServiceConfigTimeoutPUT(rt core_runtime.Runtime) gin.HandlerFunc {
	return func(c *gin.Context) {
		param := struct {
			baseService
			Timeout int32 `json:"timeout"`
		}{}
		if err := c.Bind(&param); err != nil {
			c.JSON(http.StatusBadRequest, model.NewErrorResp(err.Error()))
			return
		}

		isExist := true
		res, err := getConfigurator(rt, param.toInterface())
		if err != nil {
			if !core_store.IsResourceNotFound(err) {
				c.JSON(http.StatusBadRequest, model.NewErrorResp(err.Error()))
				return
			} else if false {
				// TODO(YarBor): check service exist or not
			}
			res = generateDefaultConfigurator(param.toInterface(), consts.ScopeService, consts.ConfiguratorVersionV3, true)
			isExist = false
		} else {
			res.Spec.RangeConfig(func(conf *mesh_proto.OverrideConfig) (isStop bool) {
				_, ok := getServiceTimeout(conf)
				if ok {
					conf.Parameters[`timeout`] = strconv.Itoa(int(param.Timeout))
				}
				return ok
			})
		}

		if !isExist {
			res.Spec.Configs = append(res.Spec.Configs, &mesh_proto.OverrideConfig{
				Side:          consts.SideProvider,
				Parameters:    map[string]string{`timeout`: strconv.Itoa(int(param.Timeout))},
				XGenerateByCp: true,
			})
			err = createConfigurator(rt, param.toInterface(), res)
			if err != nil {
				c.JSON(http.StatusInternalServerError, model.NewErrorResp(err.Error()))
				return
			}
		} else {
			err = updateConfigurator(rt, param.toInterface(), res)
			if err != nil {
				c.JSON(http.StatusInternalServerError, model.NewErrorResp(err.Error()))
				return
			}
		}
		c.JSON(http.StatusOK, model.NewSuccessResp(nil))
	}
}

func ServiceConfigRetryGET(rt core_runtime.Runtime) gin.HandlerFunc {
	return func(c *gin.Context) {
		param := baseService{}
		resp := struct {
			RetryTimes int32 `json:"retryTimes"`
		}{-1}
		if err := c.Bind(&param); err != nil {
			c.JSON(http.StatusBadRequest, model.NewErrorResp(err.Error()))
			return
		}
		res, err := getConfigurator(rt, param.toInterface())
		if err != nil {
			c.JSON(http.StatusBadRequest, model.NewErrorResp(err.Error()))
			return
		}

		res.Spec.RangeConfig(func(conf *mesh_proto.OverrideConfig) (isStop bool) {
			resp.RetryTimes, isStop = getServiceRetryTimes(conf)
			return isStop
		})

		c.JSON(http.StatusOK, model.NewSuccessResp(resp))
	}
}

func getServiceRetryTimes(conf *mesh_proto.OverrideConfig) (int32, bool) {
	if conf.Side == consts.SideConsumer && conf.Parameters != nil && conf.Parameters[`retries`] != "" {
		retries, err := strconv.Atoi(conf.Parameters[`retries`])
		if err == nil {
			return int32(retries), true
		}
	}
	return -1, false
}

func ServiceConfigRetryPUT(rt core_runtime.Runtime) gin.HandlerFunc {
	return func(c *gin.Context) {
		param := struct {
			baseService
			RetryTimes int32 `json:"retryTimes"`
		}{}
		if err := c.Bind(&param); err != nil {
			c.JSON(http.StatusBadRequest, model.NewErrorResp(err.Error()))
			return
		}

		isExist := true
		res, err := getConfigurator(rt, param.toInterface())
		if err != nil {
			if !core_store.IsResourceNotFound(err) {
				c.JSON(http.StatusBadRequest, model.NewErrorResp(err.Error()))
				return
			} else if false {
				// TODO(YarBor): check service exist or not
			}
			res = generateDefaultConfigurator(param.toInterface(), consts.ScopeService, consts.ConfiguratorVersionV3, true)
			isExist = false
		} else {
			res.Spec.RangeConfig(func(conf *mesh_proto.OverrideConfig) (isStop bool) {
				_, ok := getServiceRetryTimes(conf)
				if ok {
					conf.Parameters[`retryTimes`] = strconv.Itoa(int(param.RetryTimes))
				}
				return ok
			})
		}

		if !isExist {
			res.Spec.Configs = append(res.Spec.Configs, &mesh_proto.OverrideConfig{
				Side:          consts.SideConsumer,
				Parameters:    map[string]string{`retries`: strconv.Itoa(int(param.RetryTimes))},
				XGenerateByCp: true,
			})
			err = createConfigurator(rt, param.toInterface(), res)
			if err != nil {
				c.JSON(http.StatusInternalServerError, model.NewErrorResp(err.Error()))
				return
			}
		} else {
			err = updateConfigurator(rt, param.toInterface(), res)
			if err != nil {
				c.JSON(http.StatusInternalServerError, model.NewErrorResp(err.Error()))
				return
			}
		}
		c.JSON(http.StatusOK, model.NewSuccessResp(nil))
	}
}

func ServiceConfigRegionPriorityGET(rt core_runtime.Runtime) gin.HandlerFunc {
	return func(c *gin.Context) {
		param := baseService{}
		resp := struct {
			RegionPriority bool `json:"regionPriority"`
			Ratio          int  `json:"ratio"`
		}{false, 0}
		if err := c.Bind(&param); err != nil {
			c.JSON(http.StatusBadRequest, model.NewErrorResp(err.Error()))
			return
		}

		rawRes, err := getConditionRule(rt, param.toInterface())
		if err != nil {
			if !core_store.IsResourceNotFound(err) {
				c.JSON(http.StatusBadRequest, model.NewErrorResp(err.Error()))
				return
			}
			c.JSON(http.StatusOK, model.NewSuccessResp(resp))
			return

		} else if rawRes.Spec.ToConditionRouteV3() != nil {
			c.JSON(http.StatusServiceUnavailable, model.NewErrorResp("this config only serve version v3.1, got v3.0 config "))
			return

		} else {
			res := rawRes.Spec.ToConditionRouteV3x1()
			if res.XGenerateByCp != nil {
				resp.RegionPriority, resp.Ratio = res.XGenerateByCp.RegionPrioritize, int(res.XGenerateByCp.RegionPrioritizeRate)
			}
			c.JSON(http.StatusOK, model.NewSuccessResp(resp))
			return
		}
	}
}

func ServiceConfigRegionPriorityPUT(rt core_runtime.Runtime) gin.HandlerFunc {
	return func(c *gin.Context) {
		param := struct {
			baseService
			Ratio          int32 `json:"ratio"`
			RegionPriority bool  `json:"regionPriority"`
		}{}
		if err := c.Bind(&param); err != nil {
			c.JSON(http.StatusBadRequest, model.NewErrorResp(err.Error()))
			return
		}

		isExist := true
		rawRes, err := getConditionRule(rt, param.toInterface())
		if err != nil {
			if !core_store.IsResourceNotFound(err) {
				c.JSON(http.StatusBadRequest, model.NewErrorResp(err.Error()))
				return
			} else if false {
				// TODO(YarBor): to check service exist or not
			} else {
				rawRes = new(mesh.ConditionRouteResource)
				rawRes.Spec = generateDefaultConditionV3x1(true, false, true, param.toInterface(), consts.ScopeService).ToConditionRoute()
				isExist = false
			}
			c.JSON(http.StatusOK, model.NewSuccessResp(nil))
			return

		} else if rawRes.Spec.ToConditionRouteV3() != nil {
			c.JSON(http.StatusServiceUnavailable, model.NewErrorResp("this config only serve version v3.1, got v3.0 config "))
			return
		}

		res := rawRes.Spec.ToConditionRouteV3x1()
		if res.XGenerateByCp == nil {
			res.XGenerateByCp = &mesh_proto.XAdminOption{
				DisabledIP:           make([]string, 0),
				RegionPrioritize:     false,
				RegionPrioritizeRate: 0,
			}
		}
		res.XGenerateByCp.RegionPrioritize, res.XGenerateByCp.RegionPrioritizeRate = param.RegionPriority, param.Ratio
		res.ReGenerateCondition()
		rawRes.Spec = res.ToConditionRoute()
		if isExist {
			err = updateConditionRule(rt, param.toInterface(), rawRes)
			if err != nil {
				c.JSON(http.StatusInternalServerError, model.NewErrorResp(err.Error()))
				return
			}
		} else {
			err = createConditionRule(rt, param.toInterface(), rawRes)
			if err != nil {
				c.JSON(http.StatusInternalServerError, model.NewErrorResp(err.Error()))
				return
			}
		}
		c.JSON(http.StatusOK, model.NewSuccessResp(nil))
		return
	}
}

func ServiceConfigArgumentRouteGET(rt core_runtime.Runtime) gin.HandlerFunc {
	return func(c *gin.Context) {
		param := struct {
			baseService
		}{}
		resp := model.ServiceArgumentRoute{Routes: make([]model.ServiceArgument, 0)}
		if err := c.Bind(&param); err != nil {
			c.JSON(http.StatusBadRequest, model.NewErrorResp(err.Error()))
			return
		}

		rawRes, err := getConditionRule(rt, param.toInterface())
		if err != nil {
			if !core_store.IsResourceNotFound(err) {
				c.JSON(http.StatusBadRequest, model.NewErrorResp(err.Error()))
				return
			}
			c.JSON(http.StatusOK, model.NewSuccessResp(resp))
			return

		} else if rawRes.Spec.ToConditionRouteV3() != nil {
			c.JSON(http.StatusServiceUnavailable, model.NewErrorResp("this config only serve version v3.1, got v3.0 config "))
			return

		} else {
			res := rawRes.Spec.ToConditionRouteV3x1()
			res.Conditions = res.ListUnGenConditions()                                      // 返回非生成的Condition
			res.RangeConditionsToRemove(func(r *mesh_proto.ConditionRule) (isRemove bool) { // 去除非方法匹配项
				_, ok := r.IsMatchMethod()
				return !ok
			})
			c.JSON(http.StatusOK, model.NewSuccessResp(model.ConditionV3x1ToServiceArgumentRoute(res.Conditions)))
			return
		}
	}
}

func ServiceConfigArgumentRoutePUT(rt core_runtime.Runtime) gin.HandlerFunc {
	return func(c *gin.Context) {
		param := struct {
			baseService
			model.ServiceArgumentRoute
		}{}
		if err := c.Bind(&param); err != nil {
			c.JSON(http.StatusBadRequest, model.NewErrorResp(err.Error()))
			return
		}

		isExist := true
		rawRes, err := getConditionRule(rt, param.toInterface())
		if err != nil {
			if !core_store.IsResourceNotFound(err) {
				c.JSON(http.StatusBadRequest, model.NewErrorResp(err.Error()))
				return
			} else if false {
				// TODO(check service is exist or not):
			}
			rawRes = new(mesh.ConditionRouteResource)
			rawRes.Spec = generateDefaultConditionV3x1(true, false, true, param.toInterface(), consts.ScopeService).ToConditionRoute()
			isExist = false
		}

		res := rawRes.Spec.ToConditionRouteV3x1()
		if res == nil {
			c.JSON(http.StatusServiceUnavailable, model.NewErrorResp("this config only serve version v3.1, got v3.0 config "))
			return
		}

		res.RangeConditionsToRemove(func(r *mesh_proto.ConditionRule) (isRemove bool) {
			_, ok := r.IsMatchMethod()
			return ok
		})
		res.Conditions = append(res.Conditions, param.ToConditionV3x1Condition()...)
		res.ReGenerateCondition()
		rawRes.Spec = res.ToConditionRoute()

		if isExist {
			err = updateConditionRule(rt, param.toInterface(), rawRes)
			if err != nil {
				c.JSON(http.StatusInternalServerError, model.NewErrorResp(err.Error()))
				return
			}
		} else {
			err = createConditionRule(rt, param.toInterface(), rawRes)
			if err != nil {
				c.JSON(http.StatusInternalServerError, model.NewErrorResp(err.Error()))
				return
			}
		}
		c.JSON(http.StatusOK, model.NewSuccessResp(nil))
	}
}
