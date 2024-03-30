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

package controllers

import (
	"context"
)

import (
	"github.com/apache/dubbo-kubernetes/pkg/bufman/constant"
	"github.com/apache/dubbo-kubernetes/pkg/bufman/core/security"
	"github.com/apache/dubbo-kubernetes/pkg/bufman/core/validity"
	"github.com/apache/dubbo-kubernetes/pkg/bufman/e"
	registryv1alpha1 "github.com/apache/dubbo-kubernetes/pkg/bufman/gen/proto/go/registry/v1alpha1"
	"github.com/apache/dubbo-kubernetes/pkg/bufman/services"
	"github.com/apache/dubbo-kubernetes/pkg/core/logger"
)

type TagController struct {
	tagService           services.TagService
	authorizationService services.AuthorizationService
	validator            validity.Validator
}

func NewTagController() *TagController {
	return &TagController{
		tagService:           services.NewTagService(),
		authorizationService: services.NewAuthorizationService(),
		validator:            validity.NewValidator(),
	}
}

func (controller *TagController) CreateRepositoryTag(ctx context.Context, req *registryv1alpha1.CreateRepositoryTagRequest) (*registryv1alpha1.CreateRepositoryTagResponse, e.ResponseError) {
	// 验证参数
	argErr := controller.validator.CheckTagName(req.GetName())
	if argErr != nil {
		logger.Sugar().Errorf("Error check: %v\n", argErr.Error())

		return nil, argErr
	}

	// 获取用户ID
	userID, _ := ctx.Value(constant.UserIDKey).(string)

	// 验证用户权限
	_, permissionErr := controller.authorizationService.CheckRepositoryCanEditByID(userID, req.GetRepositoryId())
	if permissionErr != nil {
		logger.Sugar().Errorf("Error check permission: %v", permissionErr.Error())

		return nil, permissionErr
	}

	tag, err := controller.tagService.CreateRepositoryTag(ctx, req.GetRepositoryId(), req.GetName(), req.GetCommitName())
	if err != nil {
		logger.Sugar().Errorf("Error create tag: %v", err.Error())

		return nil, err
	}

	resp := &registryv1alpha1.CreateRepositoryTagResponse{
		RepositoryTag: tag.ToProtoRepositoryTag(),
	}
	return resp, nil
}

func (controller *TagController) ListRepositoryTags(ctx context.Context, req *registryv1alpha1.ListRepositoryTagsRequest) (*registryv1alpha1.ListRepositoryTagsResponse, e.ResponseError) {
	// 验证参数
	argErr := controller.validator.CheckPageSize(req.GetPageSize())
	if argErr != nil {
		logger.Sugar().Errorf("Error check: %v\n", argErr.Error())

		return nil, argErr
	}

	// 解析page token
	pageTokenChaim, err := security.ParsePageToken(req.GetPageToken())
	if err != nil {
		logger.Sugar().Errorf("Error parse page token: %v\n", err.Error())

		respErr := e.NewInvalidArgumentError(err)
		return nil, respErr
	}

	// 尝试获取user ID
	userID, _ := ctx.Value(constant.UserIDKey).(string)

	// 验证用户权限
	_, permissionErr := controller.authorizationService.CheckRepositoryCanAccessByID(userID, req.GetRepositoryId())
	if permissionErr != nil {
		logger.Sugar().Errorf("Error check permission: %v", permissionErr.Error())

		return nil, permissionErr
	}

	tags, respErr := controller.tagService.ListRepositoryTags(ctx, req.GetRepositoryId(), pageTokenChaim.PageOffset, int(req.GetPageSize()), req.GetReverse())
	if respErr != nil {
		logger.Sugar().Errorf("Error list repo tags: %v", respErr.Error())

		return nil, respErr
	}

	// 生成下一页token
	nextPageToken, err := security.GenerateNextPageToken(pageTokenChaim.PageOffset, int(req.GetPageSize()), len(tags))
	if err != nil {
		logger.Sugar().Errorf("Error generate next page token: %v\n", err.Error())

		respErr := e.NewInternalError(err)
		return nil, respErr
	}

	resp := &registryv1alpha1.ListRepositoryTagsResponse{
		RepositoryTags: tags.ToProtoRepositoryTags(),
		NextPageToken:  nextPageToken,
	}
	return resp, nil
}
