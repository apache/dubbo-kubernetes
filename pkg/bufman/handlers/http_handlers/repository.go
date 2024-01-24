// Licensed to the Apache Software Foundation (ASF) under one or more
// contributor license agreements.  See the NOTICE file distributed with
// this work for additional information regarding copyright ownership.
// The ASF licenses this file to You under the Apache License, Version 2.0
// (the "License"); you may not use this file except in compliance with
// the License.  You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package http_handlers

import (
	"net/http"

	"github.com/apache/dubbo-kubernetes/pkg/bufman/controllers"
	registryv1alpha1 "github.com/apache/dubbo-kubernetes/pkg/bufman/gen/proto/go/registry/v1alpha1"
	"github.com/gin-gonic/gin"
)

type repositoryGroup struct {
	repositoryController *controllers.RepositoryController
}

var RepositoryGroup = &repositoryGroup{
	repositoryController: controllers.NewRepositoryController(),
}

func (group *repositoryGroup) CreateRepositoryByFullName(c *gin.Context) {
	// 绑定参数
	req := &registryv1alpha1.CreateRepositoryByFullNameRequest{}
	bindErr := c.ShouldBindJSON(req)
	if bindErr != nil {
		c.JSON(http.StatusBadRequest, NewHTTPResponse(bindErr))
		return
	}

	resp, err := group.repositoryController.CreateRepositoryByFullName(c, req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, NewHTTPResponse(err))
	}

	// 正常返回
	c.JSON(http.StatusOK, NewHTTPResponse(resp))
}

func (group *repositoryGroup) GetRepository(c *gin.Context) {
	// 绑定参数
	req := &registryv1alpha1.GetRepositoryRequest{}
	bindErr := c.ShouldBindUri(req)
	if bindErr != nil {
		c.JSON(http.StatusBadRequest, NewHTTPResponse(bindErr))
		return
	}

	resp, err := group.repositoryController.GetRepository(c, req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, NewHTTPResponse(err))
	}

	// 正常返回
	c.JSON(http.StatusOK, NewHTTPResponse(resp))
}

func (group *repositoryGroup) GetRepositoryByFullName(c *gin.Context) {
	// 绑定参数
	req := &registryv1alpha1.GetRepositoryByFullNameRequest{}
	repositoryName := c.Param("repository_name")
	repositoryOwner := c.Param("repository_owner")
	fullName := repositoryOwner + "/" + repositoryName
	req.FullName = fullName

	resp, err := group.repositoryController.GetRepositoryByFullName(c, req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, NewHTTPResponse(err))
	}

	// 正常返回
	c.JSON(http.StatusOK, NewHTTPResponse(resp))
}

func (group *repositoryGroup) ListRepositories(c *gin.Context) {
	// 绑定参数
	req := &registryv1alpha1.ListRepositoriesRequest{}
	bindErr := c.ShouldBindJSON(req)
	if bindErr != nil {
		c.JSON(http.StatusBadRequest, NewHTTPResponse(bindErr))
		return
	}

	resp, err := group.repositoryController.ListRepositories(c, req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, NewHTTPResponse(err))
	}

	// 正常返回
	c.JSON(http.StatusOK, NewHTTPResponse(resp))
}

func (group *repositoryGroup) DeleteRepository(c *gin.Context) {
	// 绑定参数
	req := &registryv1alpha1.DeleteRepositoryRequest{}
	bindErr := c.ShouldBindJSON(req)
	if bindErr != nil {
		c.JSON(http.StatusBadRequest, NewHTTPResponse(bindErr))
		return
	}

	resp, err := group.repositoryController.DeleteRepository(c, req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, NewHTTPResponse(err))
	}

	// 正常返回
	c.JSON(http.StatusOK, NewHTTPResponse(resp))
}

func (group *repositoryGroup) ListUserRepositories(c *gin.Context) {
	// 绑定参数
	req := &registryv1alpha1.ListUserRepositoriesRequest{}
	bindErr := c.ShouldBindUri(req)
	if bindErr != nil {
		c.JSON(http.StatusBadRequest, NewHTTPResponse(bindErr))
		return
	}
	bindErr = c.ShouldBindJSON(req)
	if bindErr != nil {
		c.JSON(http.StatusBadRequest, NewHTTPResponse(bindErr))
		return
	}

	resp, err := group.repositoryController.ListUserRepositories(c, req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, NewHTTPResponse(err))
	}

	// 正常返回
	c.JSON(http.StatusOK, NewHTTPResponse(resp))
}

func (group *repositoryGroup) ListRepositoriesUserCanAccess(c *gin.Context) {
	// 绑定参数
	req := &registryv1alpha1.ListRepositoriesUserCanAccessRequest{}
	bindErr := c.ShouldBindJSON(req)
	if bindErr != nil {
		c.JSON(http.StatusBadRequest, NewHTTPResponse(bindErr))
		return
	}

	resp, err := group.repositoryController.ListRepositoriesUserCanAccess(c, req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, NewHTTPResponse(err))
	}

	// 正常返回
	c.JSON(http.StatusOK, NewHTTPResponse(resp))
}

func (group *repositoryGroup) DeprecateRepositoryByName(c *gin.Context) {
	// 绑定参数
	req := &registryv1alpha1.DeprecateRepositoryByNameRequest{}
	bindErr := c.ShouldBindJSON(req)
	if bindErr != nil {
		c.JSON(http.StatusBadRequest, NewHTTPResponse(bindErr))
		return
	}

	resp, err := group.repositoryController.DeprecateRepositoryByName(c, req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, NewHTTPResponse(err))
	}

	// 正常返回
	c.JSON(http.StatusOK, NewHTTPResponse(resp))
}

func (group *repositoryGroup) UndeprecateRepositoryByName(c *gin.Context) {
	// 绑定参数
	req := &registryv1alpha1.UndeprecateRepositoryByNameRequest{}
	bindErr := c.ShouldBindJSON(req)
	if bindErr != nil {
		c.JSON(http.StatusBadRequest, NewHTTPResponse(bindErr))
		return
	}

	resp, err := group.repositoryController.UndeprecateRepositoryByName(c, req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, NewHTTPResponse(err))
	}

	// 正常返回
	c.JSON(http.StatusOK, NewHTTPResponse(resp))
}

func (group *repositoryGroup) UpdateRepositorySettingsByName(c *gin.Context) {
	// 绑定参数
	req := &registryv1alpha1.UpdateRepositorySettingsByNameRequest{}
	bindErr := c.ShouldBindJSON(req)
	if bindErr != nil {
		c.JSON(http.StatusBadRequest, NewHTTPResponse(bindErr))
		return
	}

	resp, err := group.repositoryController.UpdateRepositorySettingsByName(c, req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, NewHTTPResponse(err))
	}

	// 正常返回
	c.JSON(http.StatusOK, NewHTTPResponse(resp))
}
