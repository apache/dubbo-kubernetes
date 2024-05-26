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
	"net/http"
)

import (
	"github.com/gin-gonic/gin"
)

import (
	"github.com/apache/dubbo-kubernetes/pkg/admin/model"
	"github.com/apache/dubbo-kubernetes/pkg/admin/service"
	core_runtime "github.com/apache/dubbo-kubernetes/pkg/core/runtime"
)

// Search API Definition: https://app.apifox.com/project/3732499
// 全局搜索
func Search(rt core_runtime.Runtime) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 参考 API 定义 request 参数
		req := &model.SearchInstanceReq{}
		if err := c.ShouldBindQuery(req); err != nil {
			c.JSON(http.StatusBadRequest, model.NewErrorResp(err.Error()))
			return
		}

		// 根据 request 分流调用，如服务未实现继续实现
		instances, _, _ := service.SearchInstances(rt, req)
		// applications, _, err := service.SearchApplications(rt, req)
		// services, _, err := service.SearchServices(rt, req)

		c.JSON(http.StatusOK, model.NewSuccessResp(instances))
	}
}
