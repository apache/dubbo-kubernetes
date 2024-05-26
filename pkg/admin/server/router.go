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
	router := r.Group("/api/v1")
	{
		instance := router.Group("/instance")
		instance.GET("/search", handler.SearchInstances(rt))
		instance.GET("/detail", handler.GetInstanceDetail(rt))
	}

	{
		application := router.Group("/application")
		application.GET("/detail", handler.GetApplicationDetail(rt))
		application.GET("/instance/info", handler.GetApplicationTabInstanceInfo(rt))
		application.GET("/service/form", handler.GetApplicationServiceForm(rt))
		application.GET("/search", handler.ApplicationSearch(rt))
	}

	{
		dev := router.Group("/dev")
		dev.GET("/instances", handler.GetInstances(rt))
		dev.GET("/metas", handler.GetMetas(rt))
		dev.GET("/mappings", handler.GetMappings(rt))
	}
}
