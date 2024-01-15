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
	successCode      Code = 200
	DefaultErrorCode Code = 500
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
