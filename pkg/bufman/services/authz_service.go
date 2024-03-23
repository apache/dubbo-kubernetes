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
	"errors"
	"fmt"
)

import (
	"gorm.io/gorm"
)

import (
	"github.com/apache/dubbo-kubernetes/pkg/bufman/e"
	registryv1alpha1 "github.com/apache/dubbo-kubernetes/pkg/bufman/gen/proto/go/registry/v1alpha1"
	"github.com/apache/dubbo-kubernetes/pkg/bufman/mapper"
	"github.com/apache/dubbo-kubernetes/pkg/bufman/model"
)

// AuthorizationService 用户权限验证
type AuthorizationService interface {
	CheckRepositoryCanAccess(userID, ownerName, repositoryName string) (*model.Repository, e.ResponseError) // 检查user id用户是否可以访问repo
	CheckRepositoryCanAccessByID(userID, repositoryID string) (*model.Repository, e.ResponseError)
	CheckRepositoryCanEdit(userID, ownerName, repositoryName string) (*model.Repository, e.ResponseError) // 检查user是否可以修改repo
	CheckRepositoryCanEditByID(userID, repositoryID string) (*model.Repository, e.ResponseError)
	CheckRepositoryCanDelete(userID, ownerName, repositoryName string) (*model.Repository, e.ResponseError) // 检查用户是否可以删除repo
	CheckRepositoryCanDeleteByID(userID, repositoryID string) (*model.Repository, e.ResponseError)
}

func NewAuthorizationService() AuthorizationService {
	return &AuthorizationServiceImpl{
		repositoryMapper: &mapper.RepositoryMapperImpl{},
	}
}

type AuthorizationServiceImpl struct {
	repositoryMapper mapper.RepositoryMapper
}

func (authorizationService *AuthorizationServiceImpl) CheckRepositoryCanAccess(userID, ownerName, repositoryName string) (*model.Repository, e.ResponseError) {
	repository, err := authorizationService.repositoryMapper.FindByUserNameAndRepositoryName(ownerName, repositoryName)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, e.NewNotFoundError(fmt.Errorf("repository [name=%s/%s]", ownerName, repositoryName))
		}

		return nil, e.NewInternalError(err)
	}

	if registryv1alpha1.Visibility(repository.Visibility) != registryv1alpha1.Visibility_VISIBILITY_PUBLIC && repository.UserID != userID {
		return nil, e.NewPermissionDeniedError(fmt.Errorf("repository [name=%s/%s]", ownerName, repositoryName))
	}

	return repository, nil
}

func (authorizationService *AuthorizationServiceImpl) CheckRepositoryCanAccessByID(userID, repositoryID string) (*model.Repository, e.ResponseError) {
	repository, err := authorizationService.repositoryMapper.FindByRepositoryID(repositoryID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, e.NewNotFoundError(fmt.Errorf("repository [id=%s]", repositoryID))
		}

		return nil, e.NewInternalError(err)
	}

	if registryv1alpha1.Visibility(repository.Visibility) != registryv1alpha1.Visibility_VISIBILITY_PUBLIC && repository.UserID != userID {
		return nil, e.NewPermissionDeniedError(fmt.Errorf("repository [id=%s]", repositoryID))
	}

	return repository, nil
}

func (authorizationService *AuthorizationServiceImpl) CheckRepositoryCanEdit(userID, ownerName, repositoryName string) (*model.Repository, e.ResponseError) {
	repository, err := authorizationService.repositoryMapper.FindByUserNameAndRepositoryName(ownerName, repositoryName)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, e.NewNotFoundError(fmt.Errorf("repository [name=%s/%s]", ownerName, repositoryName))
		}

		return nil, e.NewInternalError(err)
	}

	// 只有所属用户才能修改
	if repository.UserID != userID {
		return nil, e.NewPermissionDeniedError(fmt.Errorf("repository [name=%s/%s]", ownerName, repositoryName))
	}

	return repository, nil
}

func (authorizationService *AuthorizationServiceImpl) CheckRepositoryCanEditByID(userID, repositoryID string) (*model.Repository, e.ResponseError) {
	repository, err := authorizationService.repositoryMapper.FindByRepositoryID(repositoryID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, e.NewNotFoundError(fmt.Errorf("repository [id=%s]", repositoryID))
		}

		return nil, e.NewInternalError(err)
	}

	// 只有所属用户才能修改
	if repository.UserID != userID {
		return nil, e.NewPermissionDeniedError(fmt.Errorf("repository [id=%s]", repositoryID))
	}

	return repository, nil
}

func (authorizationService *AuthorizationServiceImpl) CheckRepositoryCanDelete(userID, ownerName, repositoryName string) (*model.Repository, e.ResponseError) {
	repository, err := authorizationService.repositoryMapper.FindByUserNameAndRepositoryName(ownerName, repositoryName)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, e.NewNotFoundError(fmt.Errorf("repository [name=%s/%s]", ownerName, repositoryName))
		}

		return nil, e.NewInternalError(err)
	}

	// 只有所属用户才能修改
	if repository.UserID != userID {
		return nil, e.NewPermissionDeniedError(fmt.Errorf("repository [name=%s/%s]", ownerName, repositoryName))
	}

	return repository, nil
}

func (authorizationService *AuthorizationServiceImpl) CheckRepositoryCanDeleteByID(userID, repositoryID string) (*model.Repository, e.ResponseError) {
	repository, err := authorizationService.repositoryMapper.FindByRepositoryID(repositoryID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, e.NewNotFoundError(fmt.Errorf("repository [id=%s]", repositoryID))
		}

		return nil, e.NewInternalError(err)
	}

	// 只有所属用户才能修改
	if repository.UserID != userID {
		return nil, e.NewPermissionDeniedError(fmt.Errorf("repository [id=%s]", repositoryID))
	}

	return repository, nil
}
