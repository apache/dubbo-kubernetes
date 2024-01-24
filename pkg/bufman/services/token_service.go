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
	"time"

	"github.com/apache/dubbo-kubernetes/pkg/bufman/core/security"
	"github.com/apache/dubbo-kubernetes/pkg/bufman/e"
	"github.com/apache/dubbo-kubernetes/pkg/bufman/mapper"
	"github.com/apache/dubbo-kubernetes/pkg/bufman/model"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type TokenService interface {
	CreateToken(ctx context.Context, userName, password string, expireTime time.Time, note string) (*model.Token, e.ResponseError)
	GetToken(ctx context.Context, userID, tokenID string) (*model.Token, e.ResponseError)
	ListTokens(ctx context.Context, userID string, offset, limit int, reverse bool) (model.Tokens, e.ResponseError)
	DeleteToken(ctx context.Context, userID, tokenID string) e.ResponseError
}

type TokenServiceImpl struct {
	userMapper  mapper.UserMapper
	tokenMapper mapper.TokenMapper
}

func NewTokenService() TokenService {
	return &TokenServiceImpl{
		userMapper:  &mapper.UserMapperImpl{},
		tokenMapper: &mapper.TokenMapperImpl{},
	}
}

func (tokenService *TokenServiceImpl) CreateToken(ctx context.Context, userName, password string, expireTime time.Time, note string) (*model.Token, e.ResponseError) {
	user, err := tokenService.userMapper.FindByUserName(userName)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, e.NewPermissionDeniedError(err)
		}

		return nil, e.NewInternalError(err)
	}
	if security.EncryptPlainPassword(userName, password) != user.Password {
		// 密码不正确
		return nil, e.NewPermissionDeniedError(err)
	}

	token := &model.Token{
		ID:         0,
		UserID:     user.UserID,
		TokenID:    uuid.NewString(),
		TokenName:  security.GenerateToken(userName, note),
		ExpireTime: expireTime,
		Note:       note,
	}
	err = tokenService.tokenMapper.Create(token) // 创建token
	if err != nil {
		return nil, e.NewInternalError(err)
	}

	return token, nil
}

func (tokenService *TokenServiceImpl) GetToken(ctx context.Context, userID, tokenID string) (*model.Token, e.ResponseError) {
	token, err := tokenService.tokenMapper.FindAvailableByTokenID(tokenID)
	if err != nil {
		if err != nil && errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, e.NewNotFoundError(err)
		}

		return nil, e.NewInternalError(err)
	}
	if userID != token.UserID {
		// 不能查看其他人的token
		return nil, e.NewPermissionDeniedError(err)
	}

	return token, nil
}

func (tokenService *TokenServiceImpl) ListTokens(ctx context.Context, userID string, offset, limit int, reverse bool) (model.Tokens, e.ResponseError) {
	tokens, err := tokenService.tokenMapper.FindAvailablePageByUserID(userID, offset, limit, reverse)
	if err != nil {
		return nil, e.NewInternalError(err)
	}

	return tokens, nil
}

func (tokenService *TokenServiceImpl) DeleteToken(ctx context.Context, userID, tokenID string) e.ResponseError {
	// 查询token
	token, err := tokenService.tokenMapper.FindAvailableByTokenID(tokenID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return e.NewNotFoundError(err)
		}

		return e.NewInternalError(err)
	}
	if token.UserID != userID {
		// 不能删除其他人的token
		return e.NewPermissionDeniedError(err)
	}

	// 删除token
	err = tokenService.tokenMapper.DeleteByTokenID(tokenID)
	if err != nil {
		return e.NewInternalError(err)
	}

	return nil
}
