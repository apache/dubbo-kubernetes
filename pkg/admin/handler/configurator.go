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
	"github.com/apache/dubbo-kubernetes/pkg/admin/model"
	"github.com/apache/dubbo-kubernetes/pkg/core/resources/apis/mesh"
	core_runtime "github.com/apache/dubbo-kubernetes/pkg/core/runtime"
	"github.com/gin-gonic/gin"
	"net/http"
)

func ConfiguratorSearch(rt core_runtime.Runtime) gin.HandlerFunc {
	return func(c *gin.Context) {
		res := &mesh.DynamicConfigResourceList{}
		err := rt.ResourceManager().List(rt.AppContext(), res)
		if err != nil {
			c.JSON(http.StatusBadRequest, model.NewErrorResp(err.Error()))
			return
		}
		resp := model.ConfiguratorSearchResp{
			Code:    200,
			Message: "success",
			Data:    make([]model.ConfiguratorSearchResp_Data, 0, len(res.Items)),
		}
		for _, item := range res.Items {
			resp.Data = append(resp.Data, model.ConfiguratorSearchResp_Data{
				RuleName:   item.Meta.GetName(),
				Scope:      item.Spec.GetScope(),
				CreateTime: item.Meta.GetCreationTime().String(),
				Enabled:    item.Spec.GetEnabled(),
			})
		}
		c.JSON(http.StatusOK, resp)
	}
}
