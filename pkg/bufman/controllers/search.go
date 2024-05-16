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

	"github.com/apache/dubbo-kubernetes/pkg/bufman/core/lru"
	"github.com/apache/dubbo-kubernetes/pkg/bufman/model"

	"github.com/apache/dubbo-kubernetes/pkg/bufman/constant"
	"github.com/apache/dubbo-kubernetes/pkg/bufman/core/search"
	"github.com/apache/dubbo-kubernetes/pkg/bufman/core/security"
	"github.com/apache/dubbo-kubernetes/pkg/bufman/core/validity"
	"github.com/apache/dubbo-kubernetes/pkg/bufman/e"
	registryv1alpha1 "github.com/apache/dubbo-kubernetes/pkg/bufman/gen/proto/go/registry/v1alpha1"
	"github.com/apache/dubbo-kubernetes/pkg/bufman/services"
	"github.com/apache/dubbo-kubernetes/pkg/core/logger"
)

type SearchController struct {
	searcher             search.Searcher
	validator            validity.Validator
	authorizationService services.AuthorizationService
}

func NewSearchController() *SearchController {
	return &SearchController{
		validator:            validity.NewValidator(),
		searcher:             search.NewSearcher(),
		authorizationService: services.NewAuthorizationService(),
	}
}

func (controller *SearchController) SearchUser(ctx context.Context, req *registryv1alpha1.SearchUserRequest) (*registryv1alpha1.SearchUserResponse, e.ResponseError) {
	// 验证参数
	argErr := controller.validator.CheckPageSize(req.GetPageSize())
	if argErr != nil {
		logger.Errorf("Error check: %v\n", argErr.Error())

		return nil, argErr
	}
	argErr = controller.validator.CheckQuery(req.GetQuery())
	if argErr != nil {
		logger.Errorf("Error check: %v\n", argErr.Error())

		return nil, argErr
	}

	// 解析page token
	pageTokenChaim, err := security.ParsePageToken(req.GetPageToken())
	if err != nil {
		logger.Errorf("Error parse page token: %v\n", err.Error())

		respErr := e.NewInvalidArgumentError(err)
		return nil, respErr
	}

	// 查询结果
	users, searchErr := controller.searcher.SearchUsers(ctx, req.GetQuery(), pageTokenChaim.PageOffset, int(req.GetPageSize()), req.GetReverse())
	if searchErr != nil {
		logger.Errorf("Error search user: %v\n", searchErr.Error())

		return nil, e.NewInternalError(searchErr)
	}

	// 生成下一页token
	nextPageToken, err := security.GenerateNextPageToken(pageTokenChaim.PageOffset, int(req.GetPageSize()), len(users))
	if err != nil {
		logger.Errorf("Error generate next page token: %v\n", err.Error())

		return nil, e.NewInternalError(err)
	}

	resp := &registryv1alpha1.SearchUserResponse{
		Users:         users.ToProtoSearchResults(),
		NextPageToken: nextPageToken,
	}

	return resp, nil
}

func (controller *SearchController) SearchRepository(ctx context.Context, req *registryv1alpha1.SearchRepositoryRequest) (*registryv1alpha1.SearchRepositoryResponse, e.ResponseError) {
	// 验证参数
	argErr := controller.validator.CheckPageSize(req.GetPageSize())
	if argErr != nil {
		logger.Errorf("Error check: %v\n", argErr.Error())

		return nil, argErr
	}
	argErr = controller.validator.CheckQuery(req.GetQuery())
	if argErr != nil {
		logger.Errorf("Error check: %v\n", argErr.Error())

		return nil, argErr
	}

	// 解析page token
	pageTokenChaim, err := security.ParsePageToken(req.GetPageToken())
	if err != nil {
		logger.Errorf("Error parse page token: %v\n", err.Error())

		respErr := e.NewInvalidArgumentError(err)
		return nil, respErr
	}

	// 查询结果
	repositories, searchErr := controller.searcher.SearchRepositories(ctx, req.GetQuery(), pageTokenChaim.PageOffset, int(req.GetPageSize()), req.GetReverse())
	if searchErr != nil {
		logger.Errorf("Error search repo: %v\n", searchErr.Error())

		return nil, e.NewInternalError(searchErr)
	}

	// 生成下一页token
	nextPageToken, err := security.GenerateNextPageToken(pageTokenChaim.PageOffset, int(req.GetPageSize()), len(repositories))
	if err != nil {
		logger.Errorf("Error generate next page token: %v\n", err.Error())

		respErr := e.NewInternalError(err)
		return nil, respErr
	}

	resp := &registryv1alpha1.SearchRepositoryResponse{
		Repositories:  repositories.ToProtoSearchResults(),
		NextPageToken: nextPageToken,
	}

	return resp, nil
}

func (controller *SearchController) SearchLastCommitByContent(ctx context.Context, req *registryv1alpha1.SearchLastCommitByContentRequest) (*registryv1alpha1.SearchLastCommitByContentResponse, e.ResponseError) {
	userID, _ := ctx.Value(constant.UserIDKey).(string)

	// 验证参数
	argErr := controller.validator.CheckPageSize(req.GetPageSize())
	if argErr != nil {
		logger.Errorf("Error check: %v\n", argErr.Error())

		return nil, argErr
	}
	argErr = controller.validator.CheckQuery(req.GetQuery())
	if argErr != nil {
		logger.Errorf("Error check: %v\n", argErr.Error())

		return nil, argErr
	}

	// 解析page token
	pageTokenChaim, err := security.ParsePageToken(req.GetPageToken())
	if err != nil {
		logger.Errorf("Error parse page token: %v\n", err.Error())

		return nil, e.NewInvalidArgumentError(err)
	}

	// 查询结果
	commits, searchErr := controller.searcher.SearchCommitsByContent(ctx, userID, req.GetQuery(), pageTokenChaim.PageOffset, int(req.GetPageSize()), req.GetReverse())
	if searchErr != nil {
		logger.Errorf("Error search commit by content: %v\n", searchErr.Error())

		return nil, e.NewInternalError(searchErr)
	}

	lruQueue := lru.NewLru(len(commits))
	for i := 0; i < len(commits); i++ {
		commit := commits[i]

		identity := commit.RepositoryID
		if v, ok := lruQueue.Get(identity); !ok || (ok && v.(*model.Commit).CreatedTime.Before(commit.CreatedTime)) {
			// 同一个repo下，只记录最晚的匹配commit
			_ = lruQueue.Add(identity, commits)
		}
	}

	// 转为commit
	// 在LRU队列上，越靠近后方，在查询结果位置越靠前，所以倒序遍历
	retCommits := make([]*model.Commit, 0, lruQueue.Len())
	_ = lruQueue.RangeValue(true, func(key, value interface{}) error {
		commit, ok := value.(*model.Commit)
		if !ok {
			return nil
		}

		retCommits = append(retCommits, commit)
		return nil
	})

	// 生成下一页token
	nextPageToken, err := security.GenerateNextPageToken(pageTokenChaim.PageOffset, int(req.GetPageSize()), len(commits))
	if err != nil {
		logger.Errorf("Error generate next page token: %v\n", err.Error())

		respErr := e.NewInternalError(err)
		return nil, respErr
	}

	m := model.Commits(retCommits)
	resp := &registryv1alpha1.SearchLastCommitByContentResponse{
		Commits:       m.ToProtoSearchResults(),
		NextPageToken: nextPageToken,
	}

	return resp, nil
}

func (controller *SearchController) SearchTag(ctx context.Context, req *registryv1alpha1.SearchTagRequest) (*registryv1alpha1.SearchTagResponse, e.ResponseError) {
	userID, _ := ctx.Value(constant.UserIDKey).(string)

	// 验证参数
	argErr := controller.validator.CheckPageSize(req.GetPageSize())
	if argErr != nil {
		logger.Errorf("Error check: %v\n", argErr.Error())

		return nil, argErr
	}
	argErr = controller.validator.CheckQuery(req.GetQuery())
	if argErr != nil {
		logger.Errorf("Error check: %v\n", argErr.Error())

		return nil, argErr
	}

	// 查询权限
	repository, checkErr := controller.authorizationService.CheckRepositoryCanAccess(userID, req.GetRepositoryOwner(), req.GetRepositoryName())
	if checkErr != nil {
		logger.Errorf("Error check: %v\n", argErr.Error())

		return nil, checkErr
	}

	// 解析page token
	pageTokenChaim, err := security.ParsePageToken(req.GetPageToken())
	if err != nil {
		logger.Errorf("Error parse page token: %v\n", err.Error())

		return nil, e.NewInvalidArgumentError(err)
	}

	// 查询结果
	tags, searchErr := controller.searcher.SearchTag(ctx, repository.RepositoryID, req.GetQuery(), pageTokenChaim.PageOffset, int(req.GetPageSize()), req.GetReverse())
	if searchErr != nil {
		logger.Errorf("Error search tag: %v\n", searchErr.Error())

		return nil, e.NewInternalError(searchErr)
	}

	// 生成下一页token
	nextPageToken, err := security.GenerateNextPageToken(pageTokenChaim.PageOffset, int(req.GetPageSize()), len(tags))
	if err != nil {
		logger.Errorf("Error generate next page token: %v\n", err.Error())

		respErr := e.NewInternalError(err)
		return nil, respErr
	}

	resp := &registryv1alpha1.SearchTagResponse{
		RepositoryTags: tags.ToProtoRepositoryTags(),
		NextPageToken:  nextPageToken,
	}

	return resp, nil
}

func (controller *SearchController) SearchDraft(ctx context.Context, req *registryv1alpha1.SearchDraftRequest) (*registryv1alpha1.SearchDraftResponse, e.ResponseError) {
	userID, _ := ctx.Value(constant.UserIDKey).(string)

	// 验证参数
	argErr := controller.validator.CheckPageSize(req.GetPageSize())
	if argErr != nil {
		logger.Errorf("Error check: %v\n", argErr.Error())

		return nil, argErr
	}
	argErr = controller.validator.CheckQuery(req.GetQuery())
	if argErr != nil {
		logger.Errorf("Error check: %v\n", argErr.Error())

		return nil, argErr
	}

	// 查询权限
	repository, checkErr := controller.authorizationService.CheckRepositoryCanAccess(userID, req.GetRepositoryOwner(), req.GetRepositoryName())
	if checkErr != nil {
		logger.Errorf("Error check: %v\n", argErr.Error())

		return nil, checkErr
	}

	// 解析page token
	pageTokenChaim, err := security.ParsePageToken(req.GetPageToken())
	if err != nil {
		logger.Errorf("Error parse page token: %v\n", err.Error())

		return nil, e.NewInvalidArgumentError(err)
	}

	// 查询结果
	commits, searchErr := controller.searcher.SearchDraft(ctx, repository.RepositoryID, req.GetQuery(), pageTokenChaim.PageOffset, int(req.GetPageSize()), req.GetReverse())
	if searchErr != nil {
		logger.Errorf("Error search draft: %v\n", searchErr.Error())

		return nil, e.NewInternalError(searchErr)
	}

	// 生成下一页token
	nextPageToken, err := security.GenerateNextPageToken(pageTokenChaim.PageOffset, int(req.GetPageSize()), len(commits))
	if err != nil {
		logger.Errorf("Error generate next page token: %v\n", err.Error())

		respErr := e.NewInternalError(err)
		return nil, respErr
	}

	resp := &registryv1alpha1.SearchDraftResponse{
		RepositoryCommits: commits.ToProtoRepositoryCommits(),
		NextPageToken:     nextPageToken,
	}

	return resp, nil
}
