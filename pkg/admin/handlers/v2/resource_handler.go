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

package handlersV2

import (
	"net/http"

	"github.com/apache/dubbo-kubernetes/pkg/admin/model/resp"
	servicesV2 "github.com/apache/dubbo-kubernetes/pkg/admin/services/v2"
	"github.com/gin-gonic/gin"
)

var (
	resourceService = servicesV2.NewResourceService()
)

func AllApplications(c *gin.Context) {
	namespace := c.Query("namespace")
	appNames, err := resourceService.FindApplications(namespace)
	if err != nil {
		_ = c.Error(err)
		return
	}
	c.JSON(http.StatusOK, resp.NewSuccessResp(appNames))
}

func ApplicationDetail(c *gin.Context) {
	namespace := c.Query("namespace")
	appName := c.Query("appName")
	appDetail, err := resourceService.FindApplicationDetail(namespace, appName)
	if err != nil {
		_ = c.Error(err)
		return
	}
	c.JSON(http.StatusOK, resp.NewSuccessResp(appDetail))
}
