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
	"sort"

	"github.com/apache/dubbo-kubernetes/pkg/admin/errors"
	"github.com/apache/dubbo-kubernetes/pkg/admin/model/req"
	"github.com/apache/dubbo-kubernetes/pkg/admin/model/resp"
	servicesV2 "github.com/apache/dubbo-kubernetes/pkg/admin/services/v2"
	"github.com/gin-gonic/gin"
	"github.com/vcraescu/go-paginator"
	"github.com/vcraescu/go-paginator/adapter"
)

var resourceService = servicesV2.NewResourceService()

func SearchApplication(c *gin.Context) {
	// request params validation and binding
	namespace := c.DefaultQuery(req.Namespace, "")
	keywords := c.DefaultQuery(req.Keywords, "")
	pageQuery := &req.PageQuery{}
	if err := c.ShouldBindQuery(pageQuery); err != nil {
		_ = c.Error(errors.NewBizError(resp.InvalidParamCode, err))
		return
	}
	sortQuery := &req.SortQuery{}
	if err := c.ShouldBindQuery(sortQuery); err != nil {
		_ = c.Error(errors.NewBizError(resp.InvalidParamCode, err))
		return
	}

	// invoke service
	appOverviews, err := resourceService.SearchApplications(namespace, keywords)
	if err != nil {
		_ = c.Error(err)
		return
	}

	// sort
	switch sortQuery.SortType {
	case "appName":
		sort.Slice(appOverviews, func(i, j int) bool {
			if sortQuery.Order == req.Asc {
				return appOverviews[i].Name < appOverviews[j].Name
			} else if sortQuery.Order == req.Desc {
				return appOverviews[i].Name > appOverviews[j].Name
			}
			return false
		})
	case "instanceNum":
		sort.Slice(appOverviews, func(i, j int) bool {
			if sortQuery.Order == req.Asc {
				return appOverviews[i].InstanceCount < appOverviews[j].InstanceCount
			} else if sortQuery.Order == req.Desc {
				return appOverviews[i].InstanceCount > appOverviews[j].InstanceCount
			}
			return false
		})
	}

	// paging
	p := paginator.New(adapter.NewSliceAdapter(appOverviews), pageQuery.Size)
	p.SetPage(pageQuery.Page)
	var results []*resp.ApplicationOverview
	if err := p.Results(&results); err != nil {
		_ = c.Error(errors.NewBizError(resp.InvalidParamCode, err))
		return
	}

	// return response
	c.JSON(http.StatusOK, resp.NewSuccessPageResp(results, p, pageQuery))
}
