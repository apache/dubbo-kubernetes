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
	Code    int                   `json:"code"`
	Message string                `json:"message"`
	Data    ConfiguratorResp_Data `json:"data"`
}

type ConfiguratorResp_Data struct {
	ConfigVersion string                     `json:"configVersion"`
	Scope         string                     `json:"scope"`
	Key           string                     `json:"key"`
	Enabled       bool                       `json:"enabled"`
	Configs       []ConfiguratorResp_Configs `json:"configs"`
}

type ConfiguratorResp_Configs struct {
	Side       string                            `json:"side"`
	Enabled    bool                              `json:"enabled"`
	Match      ConfiguratorResp_Match            `json:"match"`
	Parameters ConfiguratorResp_ConfigParameters `json:"parameters"`
}

type ConfiguratorResp_Match struct {
	Param []ConfiguratorResp_Param `json:"param"`
}

type ConfiguratorResp_Param struct {
	Key   string                      `json:"key"`
	Value ConfiguratorResp_ParamValue `json:"value"`
}

type ConfiguratorResp_ParamValue struct {
	Exact string `json:"exact"`
}

type ConfiguratorResp_ConfigParameters struct {
	Timeout string `json:"timeout"`
}
