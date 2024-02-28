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

	"github.com/apache/dubbo-kubernetes/pkg/bufman/constant"
	"github.com/apache/dubbo-kubernetes/pkg/bufman/core/security"
	"github.com/apache/dubbo-kubernetes/pkg/bufman/core/validity"
	"github.com/apache/dubbo-kubernetes/pkg/bufman/e"
	registryv1alpha1 "github.com/apache/dubbo-kubernetes/pkg/bufman/gen/proto/go/registry/v1alpha1"
	"github.com/apache/dubbo-kubernetes/pkg/bufman/services"
	"github.com/apache/dubbo-kubernetes/pkg/core/logger"
)

type CommitController struct {
	commitService        services.CommitService
	authorizationService services.AuthorizationService
	validator            validity.Validator
}

func NewCommitController() *CommitController {
	return &CommitController{
		commitService:        services.NewCommitService(),
		authorizationService: services.NewAuthorizationService(),
		validator:            validity.NewValidator(),
	}
}

func (controller *CommitController) ListRepositoryCommitsByReference(ctx context.Context, req *registryv1alpha1.ListRepositoryCommitsByReferenceRequest) (*registryv1alpha1.ListRepositoryCommitsByReferenceResponse, e.ResponseError) {
	// 验证参数
	argErr := controller.validator.CheckPageSize(req.GetPageSize())
	if argErr != nil {
		logger.Sugar().Errorf("Error Check Args: %v\n", argErr.Error())
		return nil, argErr
	}

	// 尝试获取user ID
	userID, _ := ctx.Value(constant.UserIDKey).(string)

	// 验证用户权限
	repository, permissionErr := controller.authorizationService.CheckRepositoryCanAccess(userID, req.GetRepositoryOwner(), req.GetRepositoryName())
	if permissionErr != nil {
		logger.Sugar().Errorf("Error Check Permission: %v\n", permissionErr.Error())
		return nil, permissionErr
	}

	// 解析page token
	pageTokenChaim, err := security.ParsePageToken(req.GetPageToken())
	if err != nil {
		logger.Sugar().Errorf("Error Parse Page Token: %v\n", err.Error())

		respErr := e.NewInvalidArgumentError(err)
		return nil, respErr
	}

	// 查询
	commits, respErr := controller.commitService.ListRepositoryCommitsByReference(ctx, repository.RepositoryID, req.GetReference(), pageTokenChaim.PageOffset, int(req.GetPageSize()), req.GetReverse())
	if respErr != nil {
		logger.Sugar().Errorf("Error list repository commits: %v\n", respErr.Error())
		return nil, respErr
	}

	// 生成下一页token
	nextPageToken, err := security.GenerateNextPageToken(pageTokenChaim.PageOffset, int(req.GetPageSize()), len(commits))
	if err != nil {
		logger.Sugar().Errorf("Error generate next page token: %v\n", err.Error())

		respErr := e.NewInternalError(err)
		return nil, respErr
	}

	resp := &registryv1alpha1.ListRepositoryCommitsByReferenceResponse{
		RepositoryCommits: commits.ToProtoRepositoryCommits(),
		NextPageToken:     nextPageToken,
	}
	return resp, nil
}

func (controller *CommitController) GetRepositoryCommitByReference(ctx context.Context, req *registryv1alpha1.GetRepositoryCommitByReferenceRequest) (*registryv1alpha1.GetRepositoryCommitByReferenceResponse, e.ResponseError) {
	// 尝试获取user ID
	userID, _ := ctx.Value(constant.UserIDKey).(string)

	// 验证用户权限
	repository, permissionErr := controller.authorizationService.CheckRepositoryCanAccess(userID, req.GetRepositoryOwner(), req.GetRepositoryName())
	if permissionErr != nil {
		logger.Sugar().Errorf("Error Check Permission: %v\n", permissionErr.Error())
		return nil, permissionErr
	}

	// 查询
	commit, respErr := controller.commitService.GetRepositoryCommitByReference(ctx, repository.RepositoryID, req.GetReference())
	if respErr != nil {
		logger.Sugar().Errorf("Error list repository commits: %v\n", respErr.Error())
		return nil, respErr
	}

	resp := &registryv1alpha1.GetRepositoryCommitByReferenceResponse{
		RepositoryCommit: commit.ToProtoRepositoryCommit(),
	}
	return resp, nil
}

func (controller *CommitController) ListRepositoryDraftCommits(ctx context.Context, req *registryv1alpha1.ListRepositoryDraftCommitsRequest) (*registryv1alpha1.ListRepositoryDraftCommitsResponse, e.ResponseError) {
	// 验证参数
	argErr := controller.validator.CheckPageSize(req.GetPageSize())
	if argErr != nil {
		logger.Sugar().Errorf("Error Check Args: %v\n", argErr.Error())
		return nil, argErr
	}

	// 尝试获取user ID
	userID, _ := ctx.Value(constant.UserIDKey).(string)

	// 验证用户权限
	repository, permissionErr := controller.authorizationService.CheckRepositoryCanAccess(userID, req.GetRepositoryOwner(), req.GetRepositoryName())
	if permissionErr != nil {
		logger.Sugar().Errorf("Error Check Permission: %v\n", permissionErr.Error())
		return nil, permissionErr
	}

	// 解析page token
	pageTokenChaim, err := security.ParsePageToken(req.GetPageToken())
	if err != nil {
		logger.Sugar().Errorf("Error Parse Page Token: %v\n", err.Error())

		respErr := e.NewInvalidArgumentError(err)
		return nil, respErr
	}

	// 查询
	commits, respErr := controller.commitService.ListRepositoryDraftCommits(ctx, repository.RepositoryID, pageTokenChaim.PageOffset, int(req.GetPageSize()), req.GetReverse())
	if respErr != nil {
		logger.Sugar().Errorf("Error list repository draft commits: %v\n", respErr.Error())
		return nil, respErr
	}

	// 生成下一页token
	nextPageToken, err := security.GenerateNextPageToken(pageTokenChaim.PageOffset, int(req.GetPageSize()), len(commits))
	if err != nil {
		logger.Sugar().Errorf("Error generate next page token: %v\n", err.Error())

		respErr := e.NewInternalError(err)
		return nil, respErr
	}

	resp := &registryv1alpha1.ListRepositoryDraftCommitsResponse{
		RepositoryCommits: commits.ToProtoRepositoryCommits(),
		NextPageToken:     nextPageToken,
	}
	return resp, nil
}

func (controller *CommitController) DeleteRepositoryDraftCommit(ctx context.Context, req *registryv1alpha1.DeleteRepositoryDraftCommitRequest) (*registryv1alpha1.DeleteRepositoryDraftCommitResponse, e.ResponseError) {
	// 获取user ID
	userID, _ := ctx.Value(constant.UserIDKey).(string)

	// 验证用户权限
	repository, permissionErr := controller.authorizationService.CheckRepositoryCanEdit(userID, req.GetRepositoryOwner(), req.GetRepositoryName())
	if permissionErr != nil {
		logger.Sugar().Errorf("Error Check Permission: %v\n", permissionErr.Error())
		return nil, permissionErr
	}

	// 删除
	err := controller.commitService.DeleteRepositoryDraftCommit(ctx, repository.RepositoryID, req.GetDraftName())
	if err != nil {
		logger.Sugar().Errorf("Error delete repository draft commits: %v\n", err.Error())
		return nil, err
	}

	resp := &registryv1alpha1.DeleteRepositoryDraftCommitResponse{}
	return resp, nil
}
