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

package server

import (
	"time"

	"github.com/google/uuid"
)

// Response 统一API响应格式
type Response struct {
	Message   string `json:"message"`        // 响应消息
	Data      any    `json:"data,omitempty"` // 响应数据
	RequestID string `json:"request_id"`     // 请求ID，用于追踪
	Timestamp int64  `json:"timestamp"`      // 响应时间戳
}

// NewResponse 创建响应
func NewResponse(message string, data any) *Response {
	return &Response{
		Message:   message,
		Data:      data,
		RequestID: generateRequestID(),
		Timestamp: time.Now().Unix(),
	}
}

// NewSuccessResponse 创建成功响应
func NewSuccessResponse(data any) *Response {
	return NewResponse("success", data)
}

// NewErrorResponse 创建错误响应
func NewErrorResponse(message string) *Response {
	return NewResponse(message, nil)
}

// ChatRequest 流式聊天请求
type ChatRequest struct {
	Message   string `json:"message" binding:"required"`   // 用户消息
	SessionID string `json:"sessionID" binding:"required"` // 会话ID
}

// generateRequestID 生成请求ID
func generateRequestID() string {
	return "req_" + uuid.New().String()
}
