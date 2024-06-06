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

// API Definition: https://app.apifox.com/project/3732499
// 资源详情-应用

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

func GetApplicationDetail(rt core_runtime.Runtime) gin.HandlerFunc {
	return func(c *gin.Context) {
		req := &model.ApplicationDetailReq{}
		if err := c.ShouldBindQuery(req); err != nil {
			c.JSON(http.StatusBadRequest, model.NewErrorResp(err.Error()))
			return
		}
		resp, err := service.GetApplicationDetail(rt, req)
		if err != nil {
			c.JSON(http.StatusInternalServerError, model.NewErrorResp(err.Error()))
			return
		}
		c.JSON(http.StatusOK, model.NewSuccessResp(resp))
	}
}

func GetApplicationTabInstanceInfo(rt core_runtime.Runtime) gin.HandlerFunc {
	return func(c *gin.Context) {
		req := &model.ApplicationTabInstanceInfoReq{}
		if err := c.ShouldBindQuery(req); err != nil {
			c.JSON(http.StatusBadRequest, model.NewErrorResp(err.Error()))
			return
		}

		resp, err := service.GetApplicationTabInstanceInfo(rt, req)
		if err != nil {
			c.JSON(http.StatusInternalServerError, model.NewErrorResp(err.Error()))
			return
		}

		c.JSON(http.StatusOK, model.NewSuccessResp(resp))
	}
}

func GetApplicationServiceForm(rt core_runtime.Runtime) gin.HandlerFunc {
	return func(c *gin.Context) {
		req := &model.ApplicationServiceFormReq{}
		if err := c.ShouldBindQuery(req); err != nil {
			c.JSON(http.StatusBadRequest, model.NewErrorResp(err.Error()))
			return
		}
		resp, err := service.GetApplicationServiceFormInfo(rt, req)
		if err != nil {
			c.JSON(http.StatusBadRequest, model.NewErrorResp(err.Error()))
			return
		}
		c.JSON(http.StatusOK, model.NewSuccessResp(resp))
	}
}

func ApplicationSearch(rt core_runtime.Runtime) gin.HandlerFunc {
	return func(c *gin.Context) {
		resp, err := service.GetApplicationSearchInfo(rt)
		if err != nil {
			c.JSON(http.StatusBadRequest, model.NewErrorResp(err.Error()))
			return
		}
		c.JSON(http.StatusOK, model.NewSuccessResp(resp))
	}
}
