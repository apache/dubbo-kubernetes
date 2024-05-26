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

import mesh_proto "github.com/apache/dubbo-kubernetes/api/mesh/v1alpha1"

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
	Code    int                   `json:"code"`
	Message string                `json:"message"`
	Data    RespConditionRuleData `json:"data"`
}

type RespConditionRuleData struct {
	Conditions    []string `json:"conditions"`
	ConfigVersion string   `json:"configVersion"`
	Enabled       bool     `json:"enabled"`
	Key           string   `json:"key"`
	Runtime       bool     `json:"runtime"`
	Scope         string   `json:"scope"`
}

func GenConditionRuleToResp(code int, message string, pb *mesh_proto.ConditionRoute) *ConditionRuleResp {
	if pb == nil {
		return &ConditionRuleResp{
			Code:    code,
			Message: message,
			Data: RespConditionRuleData{
				Conditions: make([]string, 0),
			},
		}
	} else {
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
	}
}
