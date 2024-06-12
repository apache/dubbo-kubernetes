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
	core_runtime "github.com/apache/dubbo-kubernetes/pkg/core/runtime"
)

// SearchServices API Definition: https://app.apifox.com/project/3732499
// 资源详情-应用
func SearchServices(rt core_runtime.Runtime) gin.HandlerFunc {
	return func(c *gin.Context) {
		// req := &model.SearchInstanceReq{}

		c.JSON(http.StatusOK, model.NewSuccessResp(""))
	}
}

func ListServices(rt core_runtime.Runtime) gin.HandlerFunc {
	return func(c *gin.Context) {
		// req := &model.SearchInstanceReq{}

		c.JSON(http.StatusOK, model.NewSuccessResp(""))
	}
}

func GetServiceDetail(rt core_runtime.Runtime) gin.HandlerFunc {
	return func(c *gin.Context) {
		// req := &model.SearchInstanceReq{}

		c.JSON(http.StatusOK, model.NewSuccessResp(""))
	}
}

func ServiceConfigTimeoutPUT(rt core_runtime.Runtime) gin.HandlerFunc {
	return func(c *gin.Context) {

	}
}

func ServiceConfigRetryPUT(rt core_runtime.Runtime) gin.HandlerFunc {
	return func(c *gin.Context) {

	}
}

func ServiceConfigRegionPriorityPUT(rt core_runtime.Runtime) gin.HandlerFunc {
	return func(c *gin.Context) {

	}
}

func ServiceConfigArgumentRoutePUT(rt core_runtime.Runtime) gin.HandlerFunc {
	return func(c *gin.Context) {

	}
}

func ServiceConfigArgumentRouteGET(rt core_runtime.Runtime) gin.HandlerFunc {
	return func(c *gin.Context) {

	}
}

func ServiceConfigArgumentRouteDELETE(rt core_runtime.Runtime) gin.HandlerFunc {
	return func(c *gin.Context) {

	}
}

func ServiceConfigArgumentRoutePOST(rt core_runtime.Runtime) gin.HandlerFunc {
	return func(c *gin.Context) {

	}
}
