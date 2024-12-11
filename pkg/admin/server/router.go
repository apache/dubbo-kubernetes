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
	"github.com/gin-gonic/gin"
)

import (
	"github.com/apache/dubbo-kubernetes/pkg/admin/handler"
	core_runtime "github.com/apache/dubbo-kubernetes/pkg/core/runtime"
)

func initRouter(r *gin.Engine, rt core_runtime.Runtime) {
	grafanaRouter := r.Group("/grafana")
	{
		grafanaRouter.Any("/*any", handler.Grafana(rt))
	}

	router := r.Group("/api/v1")
	{
		prometheus := router.Group("/promQL")
		prometheus.GET("/query", handler.PromQL(rt))
	}

	{
		instance := router.Group("/instance")
		instance.GET("/search", handler.SearchInstances(rt))
		instance.GET("/detail", handler.GetInstanceDetail(rt))
		{
			instanceConfig := instance.Group("/config")
			instanceConfig.GET("/trafficDisable", handler.InstanceConfigTrafficDisableGET(rt))
			instanceConfig.PUT("/trafficDisable", handler.InstanceConfigTrafficDisablePUT(rt))

			instanceConfig.GET("/operatorLog", handler.InstanceConfigOperatorLogGET(rt))
			instanceConfig.PUT("/operatorLog", handler.InstanceConfigOperatorLogPUT(rt))
		}
		instance.GET("/metric-dashboard", handler.GetMetricDashBoard(rt, handler.InstanceDimension))
		instance.GET("/instance-dashboard", handler.GetTraceDashBoard(rt, handler.InstanceDimension))
		instance.GET("/metrics-list", handler.GetMetricsList(rt))
	}

	{
		application := router.Group("/application")
		application.GET("/detail", handler.GetApplicationDetail(rt))
		application.GET("/instance/info", handler.GetApplicationTabInstanceInfo(rt))
		application.GET("/service/form", handler.GetApplicationServiceForm(rt))
		application.GET("/search", handler.ApplicationSearch(rt))
		{
			applicationConfig := application.Group("/config")
			applicationConfig.PUT("/operatorLog", handler.ApplicationConfigOperatorLogPut(rt))
			applicationConfig.GET("/operatorLog", handler.ApplicationConfigOperatorLogGet(rt))

			applicationConfig.GET("/flowWeight", handler.ApplicationConfigFlowWeightGET(rt))
			applicationConfig.PUT("/flowWeight", handler.ApplicationConfigFlowWeightPUT(rt))

			applicationConfig.GET("/gray", handler.ApplicationConfigGrayGET(rt))
			applicationConfig.PUT("/gray", handler.ApplicationConfigGrayPUT(rt))
		}
		application.GET("/metric-dashboard", handler.GetMetricDashBoard(rt, handler.AppDimension))
		application.GET("/trace-dashboard", handler.GetTraceDashBoard(rt, handler.AppDimension))
	}

	{
		service := router.Group("/service")
		{
			serviceConfig := service.Group("/config")
			serviceConfig.GET("/timeout", handler.ServiceConfigTimeoutGET(rt))
			serviceConfig.PUT("/timeout", handler.ServiceConfigTimeoutPUT(rt))

			serviceConfig.GET("/regionPriority", handler.ServiceConfigRegionPriorityGET(rt))
			serviceConfig.PUT("/regionPriority", handler.ServiceConfigRegionPriorityPUT(rt))

			serviceConfig.GET("/retry", handler.ServiceConfigRetryGET(rt))
			serviceConfig.PUT("/retry", handler.ServiceConfigRetryPUT(rt))

			serviceConfig.GET("/argumentRoute", handler.ServiceConfigArgumentRouteGET(rt))
			serviceConfig.PUT("/argumentRoute", handler.ServiceConfigArgumentRoutePUT(rt))
		}
		service.GET("/metric-dashboard", handler.GetMetricDashBoard(rt, handler.ServiceDimension))
		service.GET("/trace-dashboard", handler.GetTraceDashBoard(rt, handler.ServiceDimension))
	}

	{
		service := router.Group("/service")
		service.GET("/distribution", handler.GetServiceTabDistribution(rt))
		service.GET("/search", handler.SearchServices(rt))
		service.GET("/detail", handler.GetServiceDetail(rt))
		service.GET("/interfaces", handler.GetServiceInterfaces(rt))
	}

	{
		configuration := router.Group("/configurator")
		configuration.GET("/search", handler.ConfiguratorSearch(rt))
		configuration.GET("/:ruleName", handler.GetConfiguratorWithRuleName(rt))
		configuration.PUT("/:ruleName", handler.PutConfiguratorWithRuleName(rt))
		configuration.POST("/:ruleName", handler.PostConfiguratorWithRuleName(rt))
		configuration.DELETE("/:ruleName", handler.DeleteConfiguratorWithRuleName(rt))
	}

	{
		conditionRule := router.Group("/condition-rule")
		conditionRule.GET("/search", handler.ConditionRuleSearch(rt))
		conditionRule.GET("/:ruleName", handler.GetConditionRuleWithRuleName(rt))
		conditionRule.PUT("/:ruleName", handler.PutConditionRuleWithRuleName(rt))
		conditionRule.POST("/:ruleName", handler.PostConditionRuleWithRuleName(rt))
		conditionRule.DELETE("/:ruleName", handler.DeleteConditionRuleWithRuleName(rt))
	}

	{
		tagRule := router.Group("/tag-rule")
		tagRule.GET("/search", handler.TagRuleSearch(rt))
		tagRule.GET("/:ruleName", handler.GetTagRuleWithRuleName(rt))
		tagRule.PUT("/:ruleName", handler.PutTagRuleWithRuleName(rt))
		tagRule.POST("/:ruleName", handler.PostTagRuleWithRuleName(rt))
		tagRule.DELETE("/:ruleName", handler.DeleteTagRuleWithRuleName(rt))
	}

	router.GET("/prometheus", handler.GetPrometheus(rt))
	router.GET("/search", handler.BannerGlobalSearch(rt))
	router.GET("/overview", handler.ClusterOverview(rt))
	router.GET("/metadata", handler.AdminMetadata(rt))
}
