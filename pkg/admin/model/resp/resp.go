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

package resp

import (
	"github.com/apache/dubbo-kubernetes/pkg/admin/model/req"
	"github.com/vcraescu/go-paginator"
)

type BaseResp struct {
	Code    Code        `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data"`
}

type SuccessResp struct {
	BaseResp
}

type ErrorResp struct {
	BaseResp
}

type Code int

// TODO: refactor this
const (
	successCode            Code = 200
	InvalidParamCode       Code = 400
	DefaultServerErrorCode Code = 500
	CoreCacheErrorCode     Code = 501
)

func NewSuccessResp(data interface{}) *SuccessResp {
	return &SuccessResp{
		BaseResp: BaseResp{
			Code: successCode,
			Data: data,
		},
	}
}

func NewErrorResp(code Code, err error) *ErrorResp {
	return &ErrorResp{
		BaseResp: BaseResp{
			Code:    code,
			Message: err.Error(),
		},
	}
}

type PageResp struct {
	Content       any  `json:"content"`
	TotalPages    int  `json:"totalPages"`
	TotalElements int  `json:"totalElements"`
	Size          int  `json:"size"`
	First         bool `json:"first"`
	Last          bool `json:"last"`
	PageNumber    int  `json:"pageNumber"`
	Offset        int  `json:"offset"`
}

func NewSuccessPageResp(content any, p paginator.Paginator, pq *req.PageQuery) *BaseResp {
	return &BaseResp{
		Code: successCode,
		Data: &PageResp{
			Content:       content,
			TotalPages:    p.PageNums(),
			TotalElements: p.Nums(),
			Size:          pq.Size, // page size
			First:         pq.Page == 1,
			Last:          p.PageNums() == pq.Page,
			PageNumber:    pq.Page,
			Offset:        (pq.Page - 1) * pq.Size,
		},
	}
}
