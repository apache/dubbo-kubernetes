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

type TagRuleSearchResp struct {
	Code    int64                     `json:"code"`
	Data    []TagRuleSearchResp_Datum `json:"data"`
	Message string                    `json:"message"`
}

type TagRuleSearchResp_Datum struct {
	CreateTime *string `json:"createTime,omitempty"`
	Enabled    *bool   `json:"enabled,omitempty"`
	RuleName   *string `json:"ruleName,omitempty"`
}

type TagRuleResp struct {
	Code    int          `json:"code"`
	Message string       `json:"message"`
	Data    *RespTagData `json:"data"`
}

type RespTagData struct {
	ConfigVersion string           `json:"configVersion"`
	Enabled       bool             `json:"enabled"`
	Key           string           `json:"key"`
	Runtime       bool             `json:"runtime"`
	Scope         string           `json:"scope"`
	Tags          []RespTagElement `json:"tags"`
}

type RespTagElement struct {
	Addresses []string     `json:"addresses,omitempty"`
	Match     []ParamMatch `json:"match,omitempty"`
	Name      string       `json:"name"`
}

func GenTagRouteResp(code int, message string, pb *mesh_proto.TagRoute) *TagRuleResp {
	if pb == nil {
		return &TagRuleResp{
			Code:    code,
			Message: message,
			Data:    &RespTagData{},
		}
	} else {
		return &TagRuleResp{
			Code:    code,
			Message: message,
			Data: &RespTagData{
				ConfigVersion: pb.ConfigVersion,
				Enabled:       pb.Enabled,
				Key:           pb.Key,
				Runtime:       pb.Runtime,
				Scope:         "application",
				Tags:          tagToRespTagElement(pb.Tags),
			},
		}
	}
}

func tagToRespTagElement(tags []*mesh_proto.Tag) []RespTagElement {
	res := make([]RespTagElement, 0, len(tags))
	if tags != nil {
		for _, tag := range tags {
			res = append(res, RespTagElement{
				Addresses: tag.Addresses,
				Match:     paramMatchToRespParamMatch(tag.Match),
				Name:      tag.Name,
			})
		}
	}
	return res
}
