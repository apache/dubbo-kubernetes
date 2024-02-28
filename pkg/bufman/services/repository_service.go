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

package services

import (
	"context"
	"errors"

	"github.com/apache/dubbo-kubernetes/pkg/bufman/e"
	registryv1alpha1 "github.com/apache/dubbo-kubernetes/pkg/bufman/gen/proto/go/registry/v1alpha1"
	"github.com/apache/dubbo-kubernetes/pkg/bufman/mapper"
	"github.com/apache/dubbo-kubernetes/pkg/bufman/model"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type RepositoryService interface {
	GetRepository(ctx context.Context, repositoryID string) (*model.Repository, e.ResponseError)
	GetRepositoryByUserNameAndRepositoryName(ctx context.Context, userName, repositoryName string) (*model.Repository, e.ResponseError)
	GetRepositoryCounts(ctx context.Context, repositoryID string) (*model.RepositoryCounts, e.ResponseError)
	ListRepositories(ctx context.Context, offset, limit int, reverse bool) (model.Repositories, e.ResponseError)
	ListUserRepositories(ctx context.Context, userID string, offset, limit int, reverse bool) (model.Repositories, e.ResponseError)
	ListRepositoriesUserCanAccess(ctx context.Context, userID string, offset, limit int, reverse bool) (model.Repositories, e.ResponseError)
	CreateRepositoryByUserNameAndRepositoryName(ctx context.Context, userID, userName, repositoryName string, visibility registryv1alpha1.Visibility) (*model.Repository, e.ResponseError)
	DeleteRepository(ctx context.Context, repositoryID string) e.ResponseError
	DeleteRepositoryByUserNameAndRepositoryName(ctx context.Context, userName, repositoryName string) e.ResponseError
	DeprecateRepositoryByName(ctx context.Context, ownerName, repositoryName, deprecateMsg string) (*model.Repository, e.ResponseError)
	UndeprecateRepositoryByName(ctx context.Context, ownerName, repositoryName string) (*model.Repository, e.ResponseError)
	UpdateRepositorySettingsByName(ctx context.Context, ownerName, repositoryName string, visibility registryv1alpha1.Visibility, description string) e.ResponseError
}

type RepositoryServiceImpl struct {
	repositoryMapper mapper.RepositoryMapper
	userMapper       mapper.UserMapper
	commitMapper     mapper.CommitMapper
	tagMapper        mapper.TagMapper
}

func NewRepositoryService() RepositoryService {
	return &RepositoryServiceImpl{
		repositoryMapper: &mapper.RepositoryMapperImpl{},
		userMapper:       &mapper.UserMapperImpl{},
		commitMapper:     &mapper.CommitMapperImpl{},
		tagMapper:        &mapper.TagMapperImpl{},
	}
}

func (repositoryService *RepositoryServiceImpl) GetRepository(ctx context.Context, repositoryID string) (*model.Repository, e.ResponseError) {
	// 查询
	repository, err := repositoryService.repositoryMapper.FindByRepositoryID(repositoryID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, e.NewNotFoundError(err)
		}
		return nil, e.NewInternalError(err)
	}

	return repository, nil
}

func (repositoryService *RepositoryServiceImpl) GetRepositoryByUserNameAndRepositoryName(ctx context.Context, userName, repositoryName string) (*model.Repository, e.ResponseError) {
	// 查询
	repository, err := repositoryService.repositoryMapper.FindByUserNameAndRepositoryName(userName, repositoryName)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, e.NewNotFoundError(err)
		}
		return nil, e.NewInternalError(err)
	}

	return repository, nil
}

func (repositoryService *RepositoryServiceImpl) GetRepositoryCounts(ctx context.Context, repositoryID string) (*model.RepositoryCounts, e.ResponseError) {
	// 忽略错误
	draftCounts, _ := repositoryService.commitMapper.GetDraftCountsByRepositoryID(repositoryID)
	tagCounts, _ := repositoryService.tagMapper.GetCountsByRepositoryID(repositoryID)
	repositoryCounts := &model.RepositoryCounts{
		TagsCount:   tagCounts,
		DraftsCount: draftCounts,
	}

	return repositoryCounts, nil
}

func (repositoryService *RepositoryServiceImpl) ListRepositories(ctx context.Context, offset, limit int, reverse bool) (model.Repositories, e.ResponseError) {
	repositories, err := repositoryService.repositoryMapper.FindPage(offset, limit, reverse)
	if err != nil {
		return nil, e.NewInternalError(err)
	}

	return repositories, nil
}

func (repositoryService *RepositoryServiceImpl) ListUserRepositories(ctx context.Context, userID string, offset, limit int, reverse bool) (model.Repositories, e.ResponseError) {
	repositories, err := repositoryService.repositoryMapper.FindPageByUserID(userID, offset, limit, reverse)
	if err != nil {
		return nil, e.NewInternalError(err)
	}

	return repositories, nil
}

func (repositoryService *RepositoryServiceImpl) ListRepositoriesUserCanAccess(ctx context.Context, userID string, offset, limit int, reverse bool) (model.Repositories, e.ResponseError) {
	repositories, err := repositoryService.repositoryMapper.FindAccessiblePageByUserID(userID, offset, limit, reverse)
	if err != nil {
		return nil, e.NewInternalError(err)
	}

	return repositories, nil
}

func (repositoryService *RepositoryServiceImpl) CreateRepositoryByUserNameAndRepositoryName(ctx context.Context, userID, userName, repositoryName string, visibility registryv1alpha1.Visibility) (*model.Repository, e.ResponseError) {
	// 查询用户
	user, err := repositoryService.userMapper.FindByUserName(userName)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, e.NewNotFoundError(err)
		}

		return nil, e.NewInternalError(err)
	}
	if user.UserID != userID {
		return nil, e.NewPermissionDeniedError(err)
	}

	// 创建repo
	repository := &model.Repository{
		UserID:         user.UserID,
		UserName:       user.UserName,
		RepositoryID:   uuid.NewString(),
		RepositoryName: repositoryName,
		Visibility:     uint8(visibility),
	}

	err = repositoryService.repositoryMapper.Create(repository)
	if err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return nil, e.NewAlreadyExistsError(err)
		}

		return nil, e.NewInternalError(err)
	}

	return repository, nil
}

func (repositoryService *RepositoryServiceImpl) DeleteRepository(ctx context.Context, repositoryID string) e.ResponseError {
	// 删除
	err := repositoryService.repositoryMapper.DeleteByRepositoryID(repositoryID)
	if err != nil {
		return e.NewInternalError(err)
	}

	return nil
}

func (repositoryService *RepositoryServiceImpl) DeleteRepositoryByUserNameAndRepositoryName(ctx context.Context, userName, repositoryName string) e.ResponseError {
	err := repositoryService.repositoryMapper.DeleteByUserNameAndRepositoryName(userName, repositoryName)
	if err != nil {
		return e.NewInternalError(err)
	}

	return nil
}

func (repositoryService *RepositoryServiceImpl) DeprecateRepositoryByName(ctx context.Context, ownerName, repositoryName, deprecateMsg string) (*model.Repository, e.ResponseError) {
	// 修改数据库
	updatedRepository := &model.Repository{
		Deprecated:     true,
		DeprecationMsg: deprecateMsg,
	}
	err := repositoryService.repositoryMapper.UpdateDeprecatedByUserNameAndRepositoryName(ownerName, repositoryName, updatedRepository)
	if err != nil {
		return nil, e.NewInternalError(err)
	}

	return updatedRepository, nil
}

func (repositoryService *RepositoryServiceImpl) UndeprecateRepositoryByName(ctx context.Context, ownerName, repositoryName string) (*model.Repository, e.ResponseError) {
	// 修改数据库
	updatedRepository := &model.Repository{
		Deprecated: false,
	}
	err := repositoryService.repositoryMapper.UpdateDeprecatedByUserNameAndRepositoryName(ownerName, repositoryName, updatedRepository)
	if err != nil {
		return nil, e.NewInternalError(err)
	}

	return updatedRepository, nil
}

func (repositoryService *RepositoryServiceImpl) UpdateRepositorySettingsByName(ctx context.Context, ownerName, repositoryName string, visibility registryv1alpha1.Visibility, description string) e.ResponseError {
	// 修改数据库
	updatedRepository := &model.Repository{
		Visibility:  uint8(visibility),
		Description: description,
	}
	err := repositoryService.repositoryMapper.UpdateByUserNameAndRepositoryName(ownerName, repositoryName, updatedRepository)
	if err != nil {
		return e.NewInternalError(err)
	}

	return nil
}
