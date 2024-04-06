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
)

import (
	"github.com/google/uuid"

	"gorm.io/gorm"
)

import (
	"github.com/apache/dubbo-kubernetes/pkg/bufman/core/security"
	"github.com/apache/dubbo-kubernetes/pkg/bufman/e"
	"github.com/apache/dubbo-kubernetes/pkg/bufman/mapper"
	"github.com/apache/dubbo-kubernetes/pkg/bufman/model"
)

type UserService interface {
	CreateUser(ctx context.Context, userName, password string) (*model.User, e.ResponseError)
	GetUser(ctx context.Context, userID string) (*model.User, e.ResponseError)
	GetUserByUsername(ctx context.Context, userName string) (*model.User, e.ResponseError)
	ListUsers(ctx context.Context, offset int, limit int, reverse bool) (model.Users, e.ResponseError)
}

type UserServiceImpl struct {
	userMapper mapper.UserMapper
}

func NewUserService() UserService {
	return &UserServiceImpl{
		userMapper: &mapper.UserMapperImpl{},
	}
}

func (userService *UserServiceImpl) CreateUser(ctx context.Context, userName, password string) (*model.User, e.ResponseError) {
	user := &model.User{
		UserID:   uuid.NewString(),
		UserName: userName,
		Password: security.EncryptPlainPassword(userName, password), // 加密明文密码
	}

	err := userService.userMapper.Create(user) // 创建用户
	if err != nil {
		// 用户重复
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return nil, e.NewAlreadyExistsError(err)
		}

		return nil, e.NewInternalError(err)
	}

	return user, nil
}

func (userService *UserServiceImpl) GetUser(ctx context.Context, userID string) (*model.User, e.ResponseError) {
	user, err := userService.userMapper.FindByUserID(userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, e.NewNotFoundError(err)
		}

		return nil, e.NewInternalError(err)
	}

	return user, nil
}

func (userService *UserServiceImpl) GetUserByUsername(ctx context.Context, userName string) (*model.User, e.ResponseError) {
	user, err := userService.userMapper.FindByUserName(userName)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, e.NewNotFoundError(err)
		}

		return nil, e.NewInternalError(err)
	}

	return user, nil
}

func (userService *UserServiceImpl) ListUsers(ctx context.Context, offset int, limit int, reverse bool) (model.Users, e.ResponseError) {
	users, err := userService.userMapper.FindPage(offset, limit, reverse)
	if err != nil {
		return nil, e.NewInternalError(err)
	}

	return users, nil
}
