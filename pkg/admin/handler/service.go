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

	"github.com/apache/dubbo-kubernetes/pkg/admin/model"
	"github.com/apache/dubbo-kubernetes/pkg/admin/service"
	core_runtime "github.com/apache/dubbo-kubernetes/pkg/core/runtime"
	"github.com/gin-gonic/gin"
)

// API Definition: https://app.apifox.com/project/3732499
// 资源详情-服务

// service search
func SearchServices(rt core_runtime.Runtime) gin.HandlerFunc {
	return func(c *gin.Context) {
		req := &model.ServiceSearchReq{}
		if err := c.ShouldBindQuery(req); err != nil {
			c.JSON(http.StatusBadRequest, model.NewErrorResp(err.Error()))
			return
		}

		resp, err := service.GetSearchServices(rt)
		if err != nil {
			c.JSON(http.StatusInternalServerError, model.NewErrorResp(err.Error()))
			return
		}

		c.JSON(http.StatusOK, model.NewSuccessResp(resp))
	}
}

// service distribution
func GetServiceTabDistribution(rt core_runtime.Runtime) gin.HandlerFunc {
	return func(c *gin.Context) {
		req := &model.ServiceTabDistributionReq{}
		if err := c.ShouldBindQuery(req); err != nil {
			c.JSON(http.StatusBadRequest, model.NewErrorResp(err.Error()))
			return
		}

		resp, err := service.GetServiceTabDistribution(rt, req)
		if err != nil {
			c.JSON(http.StatusInternalServerError, model.NewErrorResp(err.Error()))
			return
		}

		c.JSON(http.StatusOK, model.NewSuccessResp(resp))
	}
}

func ListServices(rt core_runtime.Runtime) gin.HandlerFunc {
	return func(c *gin.Context) {
		//req := &model.SearchInstanceReq{}

		c.JSON(http.StatusOK, model.NewSuccessResp(""))
	}
}

func GetServiceDetail(rt core_runtime.Runtime) gin.HandlerFunc {
	return func(c *gin.Context) {
		//req := &model.SearchInstanceReq{}

		c.JSON(http.StatusOK, model.NewSuccessResp(""))
	}
}

func GetServiceInterfaces(rt core_runtime.Runtime) gin.HandlerFunc {
	return func(c *gin.Context) {
		//req := &model.SearchInstanceReq{}

		c.JSON(http.StatusOK, model.NewSuccessResp(""))
	}
}
