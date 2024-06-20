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
	"github.com/apache/dubbo-kubernetes/pkg/core/logger"
	"github.com/apache/dubbo-kubernetes/pkg/core/resources/apis/mesh"
	res_model "github.com/apache/dubbo-kubernetes/pkg/core/resources/model"
	"github.com/apache/dubbo-kubernetes/pkg/core/resources/store"
	core_runtime "github.com/apache/dubbo-kubernetes/pkg/core/runtime"
)

func ConditionRuleSearch(rt core_runtime.Runtime) gin.HandlerFunc {
	return func(c *gin.Context) {
		resList := &mesh.ConditionRouteResourceList{
			Items: make([]*mesh.ConditionRouteResource, 0),
		}
		if err := rt.ResourceManager().List(rt.AppContext(), resList); err != nil {
			c.JSON(http.StatusBadRequest, model.NewErrorResp(err.Error()))
			return
		}
		resp := model.ConditionRuleSearchResp{
			Code:    http.StatusOK,
			Message: "success",
			Data:    make([]model.ConditionRuleSearchResp_Data, 0, len(resList.Items)),
		}
		for _, item := range resList.Items {
			if v3 := item.Spec.ToConditionRouteV3(); v3 != nil {
				resp.Data = append(resp.Data, model.ConditionRuleSearchResp_Data{
					RuleName:   item.Meta.GetName(),
					Scope:      v3.GetScope(),
					CreateTime: item.Meta.GetCreationTime().String(),
					Enabled:    v3.GetEnabled(),
				})
			} else if v3x1 := item.Spec.ToConditionRouteV3x1(); v3x1 != nil {
				resp.Data = append(resp.Data, model.ConditionRuleSearchResp_Data{
					RuleName:   item.Meta.GetName(),
					Scope:      v3x1.GetScope(),
					CreateTime: item.Meta.GetCreationTime().String(),
					Enabled:    v3x1.GetEnabled(),
				})
			} else {
				panic("invalid condition route item")
			}
		}
		c.JSON(http.StatusOK, resp)
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
		if res, err := getConditionRule(rt, name); err != nil {
			c.JSON(http.StatusBadRequest, model.NewErrorResp(err.Error()))
			return
		} else {
			if v3x1 := res.Spec.ToConditionRouteV3x1(); v3x1 != nil {
				v3x1.Conditions = v3x1.ListUnGenConditions()
				res.Spec = v3x1.ToConditionRoute()
			}
			c.JSON(http.StatusOK, model.GenConditionRuleToResp(http.StatusOK, "success", res.Spec))
		}
	}
}

func getConditionRule(rt core_runtime.Runtime, name string) (*mesh.ConditionRouteResource, error) {
	res := &mesh.ConditionRouteResource{Spec: &mesh_proto.ConditionRoute{}}
	if err := rt.ResourceManager().Get(rt.AppContext(), res,
		// here `name` may be service-name or app-name, set *ByApplication(`name`) is ok.
		store.GetByApplication(name), store.GetByKey(name+consts.ConditionRuleSuffix, res_model.DefaultMesh)); err != nil {
		logger.Warnf("get %s condition failed with error: %s", name, err.Error())
		return nil, err
	}
	return res, nil
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
			res, err = getConditionRule(rt, name)
			if err != nil {
				if !store.IsResourceNotFound(err) {
					c.JSON(http.StatusInternalServerError, model.NewErrorResp(err.Error()))
					return
				}
			} else if _v3x1 := res.Spec.ToConditionRouteV3x1(); _v3x1 != nil {
				v3x1.XGenerateByCp = _v3x1.XGenerateByCp
				v3x1.ReGenerateCondition()
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

		if err := updateConditionRule(rt, name, res); err != nil {
			c.JSON(http.StatusBadRequest, model.NewErrorResp(err.Error()))
			return
		} else {
			c.JSON(http.StatusOK, model.GenConditionRuleToResp(http.StatusOK, "success", nil))
		}
	}
}

func updateConditionRule(rt core_runtime.Runtime, name string, res *mesh.ConditionRouteResource) error {
	if err := rt.ResourceManager().Update(rt.AppContext(), res,
		// here `name` may be service-name or app-name, set *ByApplication(`name`) is ok.
		store.UpdateByApplication(name), store.UpdateByKey(name+consts.ConditionRuleSuffix, res_model.DefaultMesh)); err != nil {
		logger.Warnf("update %s condition failed with error: %s", name, err.Error())
		return err
	}
	return nil
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

		if err := createConditionRule(rt, name, res); err != nil {
			c.JSON(http.StatusBadRequest, model.NewErrorResp(err.Error()))
			return
		} else {
			c.JSON(http.StatusCreated, model.GenConditionRuleToResp(http.StatusCreated, "success", nil))
		}
	}
}

func createConditionRule(rt core_runtime.Runtime, name string, res *mesh.ConditionRouteResource) error {
	if err := rt.ResourceManager().Create(rt.AppContext(), res,
		// here `name` may be service-name or app-name, set *ByApplication(`name`) is ok.
		store.CreateByApplication(name), store.CreateByKey(name+consts.ConditionRuleSuffix, res_model.DefaultMesh)); err != nil {
		logger.Warnf("create %s condition failed with error: %s", name, err.Error())
		return err
	}
	return nil
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
		if err := deleteConditionRule(rt, name, res); err != nil {
			c.JSON(http.StatusBadRequest, model.NewErrorResp(err.Error()))
			return
		}
		c.JSON(http.StatusNoContent, model.GenConditionRuleToResp(http.StatusNoContent, "success", nil))
	}
}

func deleteConditionRule(rt core_runtime.Runtime, name string, res *mesh.ConditionRouteResource) error {
	if err := rt.ResourceManager().Delete(rt.AppContext(), res,
		// here `name` may be service-name or app-name, set *ByApplication(`name`) is ok.
		store.DeleteByApplication(name), store.DeleteByKey(name+consts.ConditionRuleSuffix, res_model.DefaultMesh)); err != nil {
		logger.Warnf("delete %s condition failed with error: %s", name, err.Error())
		return err
	}
	return nil
}
