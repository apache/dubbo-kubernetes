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
	Code    int64                `json:"code"`
	Message string               `json:"message"`
	Data    *mesh_proto.TagRoute `json:"data"`
}
