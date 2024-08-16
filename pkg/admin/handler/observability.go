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
	"github.com/apache/dubbo-kubernetes/pkg/admin/constants"
	"github.com/apache/dubbo-kubernetes/pkg/admin/model"
	core_runtime "github.com/apache/dubbo-kubernetes/pkg/core/runtime"
)

type Dimension string

const (
	AppDimension      Dimension = constants.Application
	InstanceDimension Dimension = constants.Instance
	ServiceDimension  Dimension = constants.Service
)

func GetMetricDashBoard(rt core_runtime.Runtime, dim Dimension) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req model.DashboardReq
		var url string
		switch dim {
		case AppDimension:
			req = &model.AppDashboardReq{}
			url = rt.Config().Admin.MetricDashboards.Application.BaseURL + "?var-application="
		case InstanceDimension:
			req = &model.InstanceDashboardReq{}
			url = rt.Config().Admin.MetricDashboards.Instance.BaseURL + "?var-instance="
		case ServiceDimension:
			req = &model.ServiceDashboardReq{}
			url = rt.Config().Admin.MetricDashboards.Service.BaseURL + "?var-service="
		}
		if err := c.ShouldBindQuery(req); err != nil {
			c.JSON(http.StatusBadRequest, model.NewErrorResp(err.Error()))
			return
		}
		resp := model.DashboardResp{
			BaseURL: url + req.GetKeyVariable(),
		}

		c.JSON(http.StatusOK, model.NewSuccessResp(resp))
	}
}

func GetTraceDashBoard(rt core_runtime.Runtime, dim Dimension) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req model.DashboardReq
		var url string
		switch dim {
		case AppDimension:
			req = &model.AppDashboardReq{}
			url = rt.Config().Admin.TraceDashboards.Application.BaseURL + "?var-application="
		case InstanceDimension:
			req = &model.InstanceDashboardReq{}
			url = rt.Config().Admin.TraceDashboards.Instance.BaseURL + "?var-instance="
		case ServiceDimension:
			req = &model.ServiceDashboardReq{}
			url = rt.Config().Admin.TraceDashboards.Service.BaseURL + "?var-service="
		}
		if err := c.ShouldBindQuery(req); err != nil {
			c.JSON(http.StatusBadRequest, model.NewErrorResp(err.Error()))
			return
		}
		resp := model.DashboardResp{
			BaseURL: url + req.GetKeyVariable(),
		}
		c.JSON(http.StatusOK, model.NewSuccessResp(resp))
	}
}

func GetPrometheus(rt core_runtime.Runtime) gin.HandlerFunc {
	return func(c *gin.Context) {
		resp := rt.Config().Admin.Prometheus
		c.JSON(http.StatusOK, model.NewSuccessResp(resp))
	}
}
