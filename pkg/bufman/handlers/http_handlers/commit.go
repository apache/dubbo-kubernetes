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
)

import (
	"github.com/gin-gonic/gin"
)

import (
	"github.com/apache/dubbo-kubernetes/pkg/bufman/controllers"
	registryv1alpha1 "github.com/apache/dubbo-kubernetes/pkg/bufman/gen/proto/go/registry/v1alpha1"
)

type commitGroup struct {
	commitController *controllers.CommitController
}

var CommitGroup = &commitGroup{
	commitController: controllers.NewCommitController(),
}

func (group *commitGroup) ListRepositoryCommitsByReference(c *gin.Context) {
	// 绑定参数
	req := &registryv1alpha1.ListRepositoryCommitsByReferenceRequest{}
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

	resp, err := group.commitController.ListRepositoryCommitsByReference(c, req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, NewHTTPResponse(err))
	}

	// 正常返回
	c.JSON(http.StatusOK, NewHTTPResponse(resp))
}

func (group *commitGroup) GetRepositoryCommitByReference(c *gin.Context) {
	// 绑定参数
	req := &registryv1alpha1.GetRepositoryCommitByReferenceRequest{}
	bindErr := c.ShouldBindUri(req)
	if bindErr != nil {
		c.JSON(http.StatusBadRequest, NewHTTPResponse(bindErr))
		return
	}

	resp, err := group.commitController.GetRepositoryCommitByReference(c, req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, NewHTTPResponse(err))
	}

	// 正常返回
	c.JSON(http.StatusOK, NewHTTPResponse(resp))
}

func (group *commitGroup) ListRepositoryDraftCommits(c *gin.Context) {
	// 绑定参数
	req := &registryv1alpha1.ListRepositoryDraftCommitsRequest{}
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

	resp, err := group.commitController.ListRepositoryDraftCommits(c, req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, NewHTTPResponse(err))
	}

	// 正常返回
	c.JSON(http.StatusOK, NewHTTPResponse(resp))
}

func (group *commitGroup) DeleteRepositoryDraftCommit(c *gin.Context) {
	// 绑定参数
	req := &registryv1alpha1.DeleteRepositoryDraftCommitRequest{}
	bindErr := c.ShouldBindUri(req)
	if bindErr != nil {
		c.JSON(http.StatusBadRequest, NewHTTPResponse(bindErr))
		return
	}

	resp, err := group.commitController.DeleteRepositoryDraftCommit(c, req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, NewHTTPResponse(err))
	}

	// 正常返回
	c.JSON(http.StatusOK, NewHTTPResponse(resp))
}
