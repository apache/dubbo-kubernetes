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

package model

import (
	mesh_proto "github.com/apache/dubbo-kubernetes/api/mesh/v1alpha1"
)

type ConditionRuleSearchResp struct {
	Code    int64                          `json:"code"`
	Data    []ConditionRuleSearchResp_Data `json:"data"`
	Message string                         `json:"message"`
}

type ConditionRuleSearchResp_Data struct {
	CreateTime string `json:"createTime"`
	Enabled    bool   `json:"enabled"`
	RuleName   string `json:"ruleName"`
	Scope      string `json:"scope"`
}

type ConditionRuleResp struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data"`
}

type RespConditionRuleData struct {
	Conditions    []string `json:"conditions"`
	ConfigVersion string   `json:"configVersion"`
	Enabled       bool     `json:"enabled"`
	Key           string   `json:"key"`
	Runtime       bool     `json:"runtime"`
	Scope         string   `json:"scope"`
}

func GenConditionRuleToResp(code int, message string, data *mesh_proto.ConditionRoute) *ConditionRuleResp {
	if data == nil {
		return &ConditionRuleResp{
			Code:    code,
			Message: message,
			Data:    map[string]string{},
		}
	}
	if pb := data.ToConditionRouteV3(); pb != nil {
		return &ConditionRuleResp{
			Code:    code,
			Message: message,
			Data: RespConditionRuleData{
				Conditions:    pb.Conditions,
				ConfigVersion: pb.ConfigVersion,
				Enabled:       pb.Enabled,
				Key:           pb.Key,
				Runtime:       pb.Runtime,
				Scope:         pb.Scope,
			},
		}
	} else if pb := data.ToConditionRouteV3x1(); pb != nil {
		res := ConditionRuleV3X1{
			Conditions:    make([]Condition, 0, len(pb.Conditions)),
			ConfigVersion: "v3.1",
			Enabled:       pb.Enabled,
			Force:         pb.Force,
			Key:           pb.Key,
			Runtime:       pb.Runtime,
			Scope:         pb.Scope,
		}
		for _, condition := range pb.Conditions {
			ress := Condition{
				Disable:  condition.TrafficDisable,
				Force:    condition.Force,
				From:     Condition_From{Match: condition.From.Match},
				Priority: condition.Priority,
				Ratio:    condition.Priority,
				To:       make([]Condition_To, 0, len(condition.To)),
			}
			for _, to := range condition.To {
				ress.To = append(ress.To, Condition_To{
					Match:  to.Match,
					Weight: to.Weight,
				})
			}
			res.Conditions = append(res.Conditions, ress)
		}
		return &ConditionRuleResp{
			Code:    code,
			Message: message,
			Data:    res,
		}
	} else {
		return &ConditionRuleResp{
			Code:    code,
			Message: message,
			Data:    data,
		}
	}
}

type ConditionRuleV3X1 struct {
	Conditions    []Condition `json:"conditions"`
	ConfigVersion string      `json:"configVersion"`
	Enabled       bool        `json:"enabled"`
	Force         bool        `json:"force"`
	Key           string      `json:"key"`
	Runtime       bool        `json:"runtime"`
	Scope         string      `json:"scope"`
}

type Condition struct {
	Disable  bool           `json:"disable"`
	Force    bool           `json:"force"`
	From     Condition_From `json:"from"`
	Priority int32          `json:"priority"`
	Ratio    int32          `json:"ratio"`
	To       []Condition_To `json:"to"`
}

type Condition_From struct {
	Match string `json:"match"`
}

type Condition_To struct {
	Match  string `json:"match"`
	Weight int32  `json:"weight"`
}
