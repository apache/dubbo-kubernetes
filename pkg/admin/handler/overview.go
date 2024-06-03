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
	"github.com/apache/dubbo-kubernetes/pkg/core/resources/apis/mesh"
	core_runtime "github.com/apache/dubbo-kubernetes/pkg/core/runtime"
)

func GetInstances(rt core_runtime.Runtime) gin.HandlerFunc {
	return func(c *gin.Context) {
		manager := rt.ResourceManager()
		dataplaneList := &mesh.DataplaneResourceList{}
		if err := manager.List(rt.AppContext(), dataplaneList); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": err.Error(),
			})
			return
		}
		c.JSON(http.StatusOK, model.NewSuccessResp(dataplaneList))
	}
}

func GetMetas(rt core_runtime.Runtime) gin.HandlerFunc {
	return func(c *gin.Context) {
		manager := rt.ResourceManager()
		metadataList := &mesh.MetaDataResourceList{}
		if err := manager.List(rt.AppContext(), metadataList); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": err.Error(),
			})
			return
		}
		c.JSON(http.StatusOK, model.NewSuccessResp(metadataList))
	}
}

func GetMappings(rt core_runtime.Runtime) gin.HandlerFunc {
	return func(c *gin.Context) {
		manager := rt.ResourceManager()
		mappingList := &mesh.MappingResourceList{}
		if err := manager.List(rt.AppContext(), mappingList); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": err.Error(),
			})
			return
		}
		c.JSON(http.StatusOK, model.NewSuccessResp(mappingList))
	}
}
