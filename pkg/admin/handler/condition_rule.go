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
	"encoding/json"
	"fmt"
	"github.com/apache/dubbo-kubernetes/pkg/admin/service"
	"io"
	"net/http"
	"strings"
)

import (
	"github.com/gin-gonic/gin"

	"github.com/mitchellh/mapstructure"
)

import (
	mesh_proto "github.com/apache/dubbo-kubernetes/api/mesh/v1alpha1"
	"github.com/apache/dubbo-kubernetes/pkg/admin/model"
	"github.com/apache/dubbo-kubernetes/pkg/core/consts"
	"github.com/apache/dubbo-kubernetes/pkg/core/resources/apis/mesh"
	core_runtime "github.com/apache/dubbo-kubernetes/pkg/core/runtime"
)

func ConditionRuleSearch(rt core_runtime.Runtime) gin.HandlerFunc {
	return func(c *gin.Context) {
		req := model.NewSearchConditionRuleReq()
		if err := c.ShouldBindQuery(req); err != nil {
			c.JSON(http.StatusBadRequest, model.NewErrorResp(err.Error()))
			return
		}
		resp, err := service.SearchConditionRules(rt, req)
		if err != nil {
			c.JSON(http.StatusBadRequest, model.NewErrorResp(err.Error()))
			return
		}
		c.JSON(http.StatusOK, model.NewSuccessResp(resp))
	}
}

func GetConditionRuleWithRuleName(rt core_runtime.Runtime) gin.HandlerFunc {
	return func(c *gin.Context) {
		var name string
		ruleName := c.Param("ruleName")
		if strings.HasSuffix(ruleName, consts.ConditionRuleSuffix) {
			name = ruleName[:len(ruleName)-len(consts.ConditionRuleSuffix)]
		} else {
			c.JSON(http.StatusBadRequest, model.NewErrorResp(fmt.Sprintf("ruleName must end with %s", consts.ConditionRuleSuffix)))
			return
		}
		if res, err := service.GetConditionRule(rt, name); err != nil {
			c.JSON(http.StatusBadRequest, model.NewErrorResp(err.Error()))
			return
		} else {
			if v3x1 := res.Spec.ToConditionRouteV3x1(); v3x1 != nil {
				res.Spec = v3x1.ToConditionRoute()
			}
			c.JSON(http.StatusOK, model.GenConditionRuleToResp(res.Spec))
		}
	}
}

func bodyToMap(reader io.ReadCloser) (map[string]interface{}, error) {
	defer reader.Close()
	res := map[string]interface{}{}
	err := json.NewDecoder(reader).Decode(&res)
	return res, err
}

func mapToStructure(m map[string]interface{}, s interface{}) error {
	decoder, err := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
		Result:  s,
		TagName: "json",
	})
	if err != nil {
		return err
	}
	err = decoder.Decode(m)
	return err
}

func PutConditionRuleWithRuleName(rt core_runtime.Runtime) gin.HandlerFunc {
	return func(c *gin.Context) {
		var name string
		ruleName := c.Param("ruleName")
		if strings.HasSuffix(ruleName, consts.ConditionRuleSuffix) {
			name = ruleName[:len(ruleName)-len(consts.ConditionRuleSuffix)]
		} else {
			c.JSON(http.StatusBadRequest, model.NewErrorResp(fmt.Sprintf("ruleName must end with %s", consts.ConditionRuleSuffix)))
			return
		}
		_map, err := bodyToMap(c.Request.Body)
		if err != nil {
			c.JSON(http.StatusBadRequest, model.NewErrorResp(err.Error()))
			return
		}

		res := &mesh.ConditionRouteResource{}
		if version := _map[consts.ConfigVersionKey]; version == consts.ConfiguratorVersionV3 {
			v3 := new(mesh_proto.ConditionRouteV3)
			err = mapToStructure(_map, &v3)
			if err != nil {
				c.JSON(http.StatusInternalServerError, model.NewErrorResp(err.Error()))
				return
			}
			err = res.SetSpec(v3.ToConditionRoute())
			if err != nil {
				c.JSON(http.StatusInternalServerError, model.NewErrorResp(err.Error()))
				return
			}
		} else if version == consts.ConfiguratorVersionV3x1 {
			v3x1 := new(mesh_proto.ConditionRouteV3X1)
			err = mapToStructure(_map, &v3x1)
			if err != nil {
				c.JSON(http.StatusInternalServerError, model.NewErrorResp(err.Error()))
				return
			}

			err = res.SetSpec(v3x1.ToConditionRoute())
			if err != nil {
				c.JSON(http.StatusInternalServerError, model.NewErrorResp(err.Error()))
				return
			}
		} else {
			c.JSON(http.StatusBadRequest, model.NewErrorResp("invalid request body"))
			return
		}

		if err := service.UpdateConditionRule(rt, name, res); err != nil {
			c.JSON(http.StatusBadRequest, model.NewErrorResp(err.Error()))
			return
		} else {
			c.JSON(http.StatusOK, model.GenConditionRuleToResp(res.Spec))
		}
	}
}

func PostConditionRuleWithRuleName(rt core_runtime.Runtime) gin.HandlerFunc {
	return func(c *gin.Context) {
		var name string
		ruleName := c.Param("ruleName")
		if strings.HasSuffix(ruleName, consts.ConditionRuleSuffix) {
			name = ruleName[:len(ruleName)-len(consts.ConditionRuleSuffix)]
		} else {
			c.JSON(http.StatusBadRequest, model.NewErrorResp(fmt.Sprintf("ruleName must end with %s", consts.ConditionRuleSuffix)))
			return
		}
		_map, err := bodyToMap(c.Request.Body)
		if err != nil {
			c.JSON(http.StatusBadRequest, model.NewErrorResp(err.Error()))
			return
		}

		res := &mesh.ConditionRouteResource{}
		if version := _map[consts.ConfigVersionKey]; version == consts.ConfiguratorVersionV3 {
			v3 := new(mesh_proto.ConditionRouteV3)
			err = mapToStructure(_map, &v3)
			if err != nil {
				c.JSON(http.StatusInternalServerError, model.NewErrorResp(err.Error()))
				return
			}
			err = res.SetSpec(v3.ToConditionRoute())
			if err != nil {
				c.JSON(http.StatusInternalServerError, model.NewErrorResp(err.Error()))
				return
			}
		} else if version == consts.ConfiguratorVersionV3x1 {
			v3x1 := new(mesh_proto.ConditionRouteV3X1)
			err = mapToStructure(_map, &v3x1)
			if err != nil {
				c.JSON(http.StatusInternalServerError, model.NewErrorResp(err.Error()))
				return
			}
			err = res.SetSpec(v3x1.ToConditionRoute())
			if err != nil {
				c.JSON(http.StatusInternalServerError, model.NewErrorResp(err.Error()))
				return
			}
		} else {
			c.JSON(http.StatusBadRequest, model.NewErrorResp("invalid request body"))
			return
		}

		if err := service.CreateConditionRule(rt, name, res); err != nil {
			c.JSON(http.StatusBadRequest, model.NewErrorResp(err.Error()))
			return
		} else {
			c.JSON(http.StatusOK, model.GenConditionRuleToResp(res.Spec))
		}
	}
}

func DeleteConditionRuleWithRuleName(rt core_runtime.Runtime) gin.HandlerFunc {
	return func(c *gin.Context) {
		var name string
		ruleName := c.Param("ruleName")
		res := &mesh.ConditionRouteResource{Spec: &mesh_proto.ConditionRoute{}}
		if strings.HasSuffix(ruleName, consts.ConditionRuleSuffix) {
			name = ruleName[:len(ruleName)-len(consts.ConditionRuleSuffix)]
		} else {
			c.JSON(http.StatusBadRequest, model.NewErrorResp(fmt.Sprintf("ruleName must end with %s", consts.ConditionRuleSuffix)))
			return
		}
		if err := service.DeleteConditionRule(rt, name, res); err != nil {
			c.JSON(http.StatusBadRequest, model.NewErrorResp(err.Error()))
			return
		}
		c.JSON(http.StatusOK, model.NewSuccessResp(""))
	}
}
