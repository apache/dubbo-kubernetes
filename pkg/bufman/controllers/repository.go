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

type RepositoryController struct {
	repositoryService    services.RepositoryService
	authorizationService services.AuthorizationService
	validator            validity.Validator
}

func NewRepositoryController() *RepositoryController {
	return &RepositoryController{
		repositoryService:    services.NewRepositoryService(),
		authorizationService: services.NewAuthorizationService(),
		validator:            validity.NewValidator(),
	}
}

func (controller *RepositoryController) GetRepository(ctx context.Context, req *registryv1alpha1.GetRepositoryRequest) (*registryv1alpha1.GetRepositoryResponse, e.ResponseError) {
	// 查询
	repository, err := controller.repositoryService.GetRepository(ctx, req.GetId())
	if err != nil {
		logger.Sugar().Errorf("Error get repo: %v\n", err.Error())

		return nil, err
	}

	repositoryCounts, err := controller.repositoryService.GetRepositoryCounts(ctx, repository.RepositoryID)
	if err != nil {
		logger.Sugar().Errorf("Error get repo counts: %v\n", err.Error())

		return nil, err
	}

	// 查询成功
	resp := &registryv1alpha1.GetRepositoryResponse{
		Repository: repository.ToProtoRepository(),
		Counts:     repositoryCounts.ToProtoRepositoryCounts(),
	}
	return resp, nil
}

func (controller *RepositoryController) GetRepositoryByFullName(ctx context.Context, req *registryv1alpha1.GetRepositoryByFullNameRequest) (*registryv1alpha1.GetRepositoryByFullNameResponse, e.ResponseError) {
	// 验证参数
	userName, repositoryName, argErr := controller.validator.SplitFullName(req.GetFullName())
	if argErr != nil {
		logger.Sugar().Errorf("Error check: %v\n", argErr.Error())

		return nil, argErr
	}

	// 查询
	repository, err := controller.repositoryService.GetRepositoryByUserNameAndRepositoryName(ctx, userName, repositoryName)
	if err != nil {
		logger.Sugar().Errorf("Error get repo: %v\n", err.Error())

		return nil, err
	}

	repositoryCounts, err := controller.repositoryService.GetRepositoryCounts(ctx, repository.RepositoryID)
	if err != nil {
		logger.Sugar().Errorf("Error get repo counts: %v\n", err.Error())

		return nil, err
	}

	// 查询成功
	resp := &registryv1alpha1.GetRepositoryByFullNameResponse{
		Repository: repository.ToProtoRepository(),
		Counts:     repositoryCounts.ToProtoRepositoryCounts(),
	}
	return resp, nil
}

func (controller *RepositoryController) ListRepositories(ctx context.Context, req *registryv1alpha1.ListRepositoriesRequest) (*registryv1alpha1.ListRepositoriesResponse, e.ResponseError) {
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

	repositories, listErr := controller.repositoryService.ListRepositories(ctx, pageTokenChaim.PageOffset, int(req.GetPageSize()), req.Reverse)
	if listErr != nil {
		logger.Sugar().Errorf("Error list repos: %v\n", listErr.Error())

		return nil, listErr
	}

	// 生成下一页token
	nextPageToken, err := security.GenerateNextPageToken(pageTokenChaim.PageOffset, int(req.GetPageSize()), len(repositories))
	if err != nil {
		logger.Sugar().Errorf("Error generate next page token: %v\n", err.Error())

		respErr := e.NewInternalError(err)
		return nil, respErr
	}

	resp := &registryv1alpha1.ListRepositoriesResponse{
		Repositories:  repositories.ToProtoRepositories(),
		NextPageToken: nextPageToken,
	}
	return resp, nil
}

func (controller *RepositoryController) ListUserRepositories(ctx context.Context, req *registryv1alpha1.ListUserRepositoriesRequest) (*registryv1alpha1.ListUserRepositoriesResponse, e.ResponseError) {
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

	repositories, listErr := controller.repositoryService.ListUserRepositories(ctx, req.GetUserId(), pageTokenChaim.PageOffset, int(req.GetPageSize()), req.GetReverse())
	if err != nil {
		logger.Sugar().Errorf("Error list user repos: %v\n", listErr.Error())

		return nil, listErr
	}

	// 生成下一页token
	nextPageToken, err := security.GenerateNextPageToken(pageTokenChaim.PageOffset, int(req.GetPageSize()), len(repositories))
	if err != nil {
		logger.Sugar().Errorf("Error generate next page token: %v\n", err.Error())

		respErr := e.NewInternalError(err)
		return nil, respErr
	}

	resp := &registryv1alpha1.ListUserRepositoriesResponse{
		Repositories:  repositories.ToProtoRepositories(),
		NextPageToken: nextPageToken,
	}
	return resp, nil
}

func (controller *RepositoryController) ListRepositoriesUserCanAccess(ctx context.Context, req *registryv1alpha1.ListRepositoriesUserCanAccessRequest) (*registryv1alpha1.ListRepositoriesUserCanAccessResponse, e.ResponseError) {
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

	userID, _ := ctx.Value(constant.UserIDKey).(string)
	repositories, ListErr := controller.repositoryService.ListRepositoriesUserCanAccess(ctx, userID, pageTokenChaim.PageOffset, int(req.GetPageSize()), req.GetReverse())
	if err != nil {
		logger.Sugar().Errorf("Error list repos user can access: %v\n", ListErr.Error())

		return nil, ListErr
	}

	// 生成下一页token
	nextPageToken, err := security.GenerateNextPageToken(pageTokenChaim.PageOffset, int(req.GetPageSize()), len(repositories))
	if err != nil {
		logger.Sugar().Errorf("Error generate next page token: %v\n", err.Error())

		respErr := e.NewInternalError(err)
		return nil, respErr
	}

	resp := &registryv1alpha1.ListRepositoriesUserCanAccessResponse{
		Repositories:  repositories.ToProtoRepositories(),
		NextPageToken: nextPageToken,
	}
	return resp, nil
}

func (controller *RepositoryController) CreateRepositoryByFullName(ctx context.Context, req *registryv1alpha1.CreateRepositoryByFullNameRequest) (*registryv1alpha1.CreateRepositoryByFullNameResponse, e.ResponseError) {
	// 验证参数
	userName, repositoryName, argErr := controller.validator.SplitFullName(req.GetFullName())
	if argErr != nil {
		logger.Sugar().Errorf("Error check: %v", argErr.Error())

		return nil, argErr
	}
	argErr = controller.validator.CheckRepositoryName(repositoryName)
	if argErr != nil {
		logger.Sugar().Errorf("Error check: %v", argErr.Error())

		return nil, argErr
	}

	userID, _ := ctx.Value(constant.UserIDKey).(string)

	// 创建
	repository, err := controller.repositoryService.CreateRepositoryByUserNameAndRepositoryName(ctx, userID, userName, repositoryName, req.GetVisibility())
	if err != nil {
		logger.Sugar().Errorf("Error create repo: %v", err.Error())

		return nil, err
	}

	// 成功
	resp := &registryv1alpha1.CreateRepositoryByFullNameResponse{
		Repository: repository.ToProtoRepository(),
	}
	return resp, nil
}

func (controller *RepositoryController) DeleteRepository(ctx context.Context, req *registryv1alpha1.DeleteRepositoryRequest) (*registryv1alpha1.DeleteRepositoryResponse, e.ResponseError) {
	userID, _ := ctx.Value(constant.UserIDKey).(string)

	// 验证用户权限
	_, permissionErr := controller.authorizationService.CheckRepositoryCanDeleteByID(userID, req.GetId())
	if permissionErr != nil {
		logger.Sugar().Errorf("Error check permission: %v", permissionErr.Error())

		return nil, permissionErr
	}

	// 查询repository，检查是否可以删除
	err := controller.repositoryService.DeleteRepository(ctx, req.GetId())
	if err != nil {
		logger.Sugar().Errorf("Error delete repo: %v", err.Error())

		return nil, err
	}

	resp := &registryv1alpha1.DeleteRepositoryResponse{}
	return resp, nil
}

func (controller *RepositoryController) DeleteRepositoryByFullName(ctx context.Context, req *registryv1alpha1.DeleteRepositoryByFullNameRequest) (*registryv1alpha1.DeleteRepositoryByFullNameResponse, e.ResponseError) {
	// 验证参数
	userName, repositoryName, argErr := controller.validator.SplitFullName(req.GetFullName())
	if argErr != nil {
		logger.Sugar().Errorf("Error check: %v\n", argErr.Error())

		return nil, argErr
	}

	userID, _ := ctx.Value(constant.UserIDKey).(string)

	// 验证用户权限
	_, permissionErr := controller.authorizationService.CheckRepositoryCanDelete(userID, userName, repositoryName)
	if permissionErr != nil {
		logger.Sugar().Errorf("Error check permission: %v", permissionErr.Error())

		return nil, permissionErr
	}

	// 删除
	err := controller.repositoryService.DeleteRepositoryByUserNameAndRepositoryName(ctx, userName, repositoryName)
	if err != nil {
		logger.Sugar().Errorf("Error delete repo: %v", err.Error())

		return nil, err
	}

	resp := &registryv1alpha1.DeleteRepositoryByFullNameResponse{}
	return resp, nil
}

func (controller *RepositoryController) DeprecateRepositoryByName(ctx context.Context, req *registryv1alpha1.DeprecateRepositoryByNameRequest) (*registryv1alpha1.DeprecateRepositoryByNameResponse, e.ResponseError) {
	userID, _ := ctx.Value(constant.UserIDKey).(string)

	// 验证用户权限
	_, permissionErr := controller.authorizationService.CheckRepositoryCanEdit(userID, req.GetOwnerName(), req.GetRepositoryName())
	if permissionErr != nil {
		logger.Sugar().Errorf("Error check permission: %v", permissionErr.Error())

		return nil, permissionErr
	}

	// 修改数据库
	updatedRepository, err := controller.repositoryService.DeprecateRepositoryByName(ctx, req.GetOwnerName(), req.GetRepositoryName(), req.GetDeprecationMessage())
	if err != nil {
		logger.Sugar().Errorf("Error deprecate repo: %v", err.Error())

		return nil, err
	}

	resp := &registryv1alpha1.DeprecateRepositoryByNameResponse{
		Repository: updatedRepository.ToProtoRepository(),
	}
	return resp, nil
}

func (controller *RepositoryController) UndeprecateRepositoryByName(ctx context.Context, req *registryv1alpha1.UndeprecateRepositoryByNameRequest) (*registryv1alpha1.UndeprecateRepositoryByNameResponse, e.ResponseError) {
	userID, _ := ctx.Value(constant.UserIDKey).(string)

	// 验证用户权限
	_, permissionErr := controller.authorizationService.CheckRepositoryCanEdit(userID, req.GetOwnerName(), req.GetRepositoryName())
	if permissionErr != nil {
		logger.Sugar().Errorf("Error check permission: %v", permissionErr.Error())

		return nil, permissionErr
	}

	// 修改数据库
	updatedRepository, err := controller.repositoryService.UndeprecateRepositoryByName(ctx, req.GetOwnerName(), req.GetRepositoryName())
	if err != nil {
		logger.Sugar().Errorf("Error undeprecate repo: %v", err.Error())

		return nil, err
	}

	resp := &registryv1alpha1.UndeprecateRepositoryByNameResponse{
		Repository: updatedRepository.ToProtoRepository(),
	}
	return resp, nil
}

func (controller *RepositoryController) UpdateRepositorySettingsByName(ctx context.Context, req *registryv1alpha1.UpdateRepositorySettingsByNameRequest) (*registryv1alpha1.UpdateRepositorySettingsByNameResponse, e.ResponseError) {
	userID, _ := ctx.Value(constant.UserIDKey).(string)

	// 验证用户权限
	_, permissionErr := controller.authorizationService.CheckRepositoryCanEdit(userID, req.GetOwnerName(), req.GetRepositoryName())
	if permissionErr != nil {
		logger.Sugar().Errorf("Error check permission: %v", permissionErr.Error())

		return nil, permissionErr
	}

	// 修改数据库
	err := controller.repositoryService.UpdateRepositorySettingsByName(ctx, req.GetOwnerName(), req.GetRepositoryName(), req.GetVisibility(), req.GetDescription())
	if err != nil {
		logger.Sugar().Errorf("Error update repo settings: %v", err.Error())

		return nil, err
	}

	resp := &registryv1alpha1.UpdateRepositorySettingsByNameResponse{}
	return resp, nil
}

func (controller *RepositoryController) GetRepositoriesByFullName(ctx context.Context, req *registryv1alpha1.GetRepositoriesByFullNameRequest) (*registryv1alpha1.GetRepositoriesByFullNameResponse, e.ResponseError) {
	retRepos := make([]*registryv1alpha1.Repository, 0, len(req.FullNames))
	for _, fullName := range req.GetFullNames() {
		// 验证参数
		userName, repositoryName, argErr := controller.validator.SplitFullName(fullName)
		if argErr != nil {
			logger.Sugar().Errorf("Error check: %v\n", argErr.Error())

			return nil, argErr
		}

		// 查询
		repository, err := controller.repositoryService.GetRepositoryByUserNameAndRepositoryName(ctx, userName, repositoryName)
		if err != nil {
			logger.Sugar().Errorf("Error get repo: %v\n", err.Error())

			return nil, err
		}
		retRepos = append(retRepos, repository.ToProtoRepository())
	}

	resp := &registryv1alpha1.GetRepositoriesByFullNameResponse{
		Repositories: retRepos,
	}

	return resp, nil
}
