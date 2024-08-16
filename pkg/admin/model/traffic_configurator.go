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

type ConfiguratorSearchResp struct {
	Code    int                           `json:"code"`
	Message string                        `json:"message"`
	Data    []ConfiguratorSearchResp_Data `json:"data"`
}

type ConfiguratorSearchResp_Data struct {
	RuleName   string `json:"ruleName"`
	Scope      string `json:"scope"`
	CreateTime string `json:"createTime"`
	Enabled    bool   `json:"enabled"`
}

type ConfiguratorResp struct {
	Code    int              `json:"code"`
	Message string           `json:"message"`
	Data    RespConfigurator `json:"data"`
}

type RespConfigurator struct {
	Configs       []ConfigItem `json:"configs"`
	ConfigVersion string       `json:"configVersion"`
	Enabled       bool         `json:"enabled"`
	Key           string       `json:"key"`
	Scope         string       `json:"scope"`
}

type ConfigItem struct {
	Enabled    *bool             `json:"enabled,omitempty"`
	Match      *RespMatch        `json:"match,omitempty"`
	Parameters map[string]string `json:"parameters"`
	Side       string            `json:"side"`
}

type RespMatch struct {
	Address         *RespAddress         `json:"address,omitempty"`
	App             *RespListStringMatch `json:"app,omitempty"`
	Param           []ParamMatch         `json:"param,omitempty"`
	ProviderAddress *RespAddressMatch    `json:"providerAddress,omitempty"`
	Service         *RespListStringMatch `json:"service,omitempty"`
}

type RespAddress struct {
	Cird     *string `json:"cird,omitempty"`
	Exact    *string `json:"exact,omitempty"`
	Wildcard *string `json:"wildcard,omitempty"`
}

type RespListStringMatch struct {
	Oneof []StringMatch `json:"oneof,omitempty"`
}

type StringMatch struct {
	Empty    *string `json:"empty,omitempty"`
	Exact    *string `json:"exact,omitempty"`
	Noempty  *string `json:"noempty,omitempty"`
	Prefix   *string `json:"prefix,omitempty"`
	Regex    *string `json:"regex,omitempty"`
	Wildcard *string `json:"wildcard,omitempty"`
}

type ParamMatch struct {
	Key   *string      `json:"key,omitempty"`
	Value *StringMatch `json:"value,omitempty"`
}

type RespAddressMatch struct {
	Cird     *string `json:"cird,omitempty"`
	Exact    *string `json:"exact,omitempty"`
	Wildcard *string `json:"wildcard,omitempty"`
}

func GenDynamicConfigToResp(code int, message string, pb *mesh_proto.DynamicConfig) (res *ConfiguratorResp) {
	cfg := RespConfigurator{}
	if pb != nil {
		cfg.ConfigVersion = pb.ConfigVersion
		cfg.Key = pb.Key
		cfg.Scope = pb.Scope
		cfg.Enabled = pb.Enabled
		cfg.Configs = overrideConfigToRespConfigItem(pb.Configs)
	}
	res = &ConfiguratorResp{
		Code:    code,
		Message: message,
		Data:    cfg,
	}
	return
}

func overrideConfigToRespConfigItem(OverrideConfigs []*mesh_proto.OverrideConfig) []ConfigItem {
	res := make([]ConfigItem, 0, len(OverrideConfigs))
	if OverrideConfigs != nil {
		for _, config := range OverrideConfigs {
			resIt := ConfigItem{
				Enabled:    &config.Enabled,
				Match:      conditionMatchToRespMatch(config.Match),
				Parameters: config.Parameters,
				Side:       config.Side,
			}
			if resIt.Parameters == nil {
				resIt.Parameters = make(map[string]string)
			}
			res = append(res, resIt)
		}
	}
	return res
}

func conditionMatchToRespMatch(match *mesh_proto.ConditionMatch) *RespMatch {
	if match == nil {
		return nil
	}
	return &RespMatch{
		Address:         addressMatchToRespAddress(match.Address),
		App:             listStringMatchToRespListStringMatch(match.Application),
		Param:           paramMatchToRespParamMatch(match.Param),
		ProviderAddress: addressMatchToRespAddressMatch(match.ProviderAddress),
		Service:         listStringMatchToRespListStringMatch(match.Service),
	}
}

func addressMatchToRespAddress(address *mesh_proto.AddressMatch) *RespAddress {
	if address == nil {
		return nil
	}
	if address.Exact != "" {
		return &RespAddress{Exact: &address.Exact}
	}
	if address.Wildcard != "" {
		return &RespAddress{Wildcard: &address.Wildcard}
	}
	if address.Cird != "" {
		return &RespAddress{Cird: &address.Cird}
	} else {
		return &RespAddress{}
	}
}

func addressMatchToRespAddressMatch(address *mesh_proto.AddressMatch) *RespAddressMatch {
	if address == nil {
		return nil
	}
	if address.Exact != "" {
		return &RespAddressMatch{Exact: &address.Exact}
	}
	if address.Wildcard != "" {
		return &RespAddressMatch{Wildcard: &address.Wildcard}
	}
	if address.Cird != "" {
		return &RespAddressMatch{Cird: &address.Cird}
	} else {
		return &RespAddressMatch{}
	}
}

func paramMatchToRespParamMatch(param []*mesh_proto.ParamMatch) []ParamMatch {
	res := make([]ParamMatch, 0, len(param))
	if param != nil {
		for _, match := range param {
			res = append(res, ParamMatch{
				Key:   &match.Key,
				Value: StringMatchToModelStringMatch(match.Value),
			})
		}
	}
	return res
}

func StringMatchToModelStringMatch(stringMatch *mesh_proto.StringMatch_Dubbo) *StringMatch {
	if stringMatch == nil {
		return nil
	}
	if stringMatch.Exact != "" {
		return &StringMatch{Exact: &stringMatch.Exact}
	}
	if stringMatch.Prefix != "" {
		return &StringMatch{Prefix: &stringMatch.Prefix}
	}
	if stringMatch.Regex != "" {
		return &StringMatch{Regex: &stringMatch.Regex}
	}
	if stringMatch.Noempty != "" {
		return &StringMatch{Noempty: &stringMatch.Noempty}
	}
	if stringMatch.Empty != "" {
		return &StringMatch{Empty: &stringMatch.Empty}
	}
	if stringMatch.Wildcard != "" {
		return &StringMatch{Wildcard: &stringMatch.Wildcard}
	} else {
		return &StringMatch{}
	}
}

func ModelStringMatchToStringMatch(stringMatch *StringMatch) *mesh_proto.StringMatch_Dubbo {
	if stringMatch == nil {
		return nil
	}
	if stringMatch.Exact != nil {
		return &mesh_proto.StringMatch_Dubbo{Exact: *stringMatch.Exact}
	}
	if stringMatch.Prefix != nil {
		return &mesh_proto.StringMatch_Dubbo{Prefix: *stringMatch.Prefix}
	}
	if stringMatch.Regex != nil {
		return &mesh_proto.StringMatch_Dubbo{Regex: *stringMatch.Regex}
	}
	if stringMatch.Noempty != nil {
		return &mesh_proto.StringMatch_Dubbo{Noempty: *stringMatch.Noempty}
	}
	if stringMatch.Empty != nil {
		return &mesh_proto.StringMatch_Dubbo{Empty: *stringMatch.Empty}
	}
	if stringMatch.Wildcard != nil {
		return &mesh_proto.StringMatch_Dubbo{Wildcard: *stringMatch.Wildcard}
	} else {
		return &mesh_proto.StringMatch_Dubbo{}
	}
}

func listStringMatchToRespListStringMatch(listStringMatch *mesh_proto.ListStringMatch) *RespListStringMatch {
	if listStringMatch == nil {
		return nil
	}
	res := &RespListStringMatch{Oneof: make([]StringMatch, 0, len(listStringMatch.Oneof))}
	if listStringMatch.Oneof != nil {
		for _, match := range listStringMatch.Oneof {
			res.Oneof = append(res.Oneof, *StringMatchToModelStringMatch(match))
		}
	}
	return res
}
