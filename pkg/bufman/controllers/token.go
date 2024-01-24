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

type TokenController struct {
	tokenService services.TokenService
	validator    validity.Validator
}

func NewTokenController() *TokenController {
	return &TokenController{
		tokenService: services.NewTokenService(),
		validator:    validity.NewValidator(),
	}
}

func (controller *TokenController) CreateToken(ctx context.Context, req *registryv1alpha1.CreateTokenRequest) (*registryv1alpha1.CreateTokenResponse, e.ResponseError) {
	token, err := controller.tokenService.CreateToken(ctx, req.GetUsername(), req.GetPassword(), req.GetExpireTime().AsTime(), req.GetNote())
	if err != nil {
		logger.Sugar().Errorf("Error create token: %v\n", err.Error())

		return nil, err
	}

	// success
	resp := &registryv1alpha1.CreateTokenResponse{
		Token: token.TokenName,
	}
	return resp, nil
}

func (controller *TokenController) GetToken(ctx context.Context, req *registryv1alpha1.GetTokenRequest) (*registryv1alpha1.GetTokenResponse, e.ResponseError) {
	userID, _ := ctx.Value(constant.UserIDKey).(string)

	// 查询token
	token, err := controller.tokenService.GetToken(ctx, userID, req.GetTokenId())
	if err != nil {
		logger.Sugar().Errorf("Error get token: %v\n", err.Error())

		return nil, err
	}

	resp := &registryv1alpha1.GetTokenResponse{
		Token: token.ToProtoToken(),
	}
	return resp, nil
}

func (controller *TokenController) ListTokens(ctx context.Context, req *registryv1alpha1.ListTokensRequest) (*registryv1alpha1.ListTokensResponse, e.ResponseError) {
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

	// 查询token
	tokens, listErr := controller.tokenService.ListTokens(ctx, userID, pageTokenChaim.PageOffset, int(req.GetPageSize()), req.GetReverse())
	if err != nil {
		logger.Sugar().Errorf("Error list tokens: %v\n", listErr.Error())

		return nil, listErr
	}

	// 生成下一页token
	nextPageToken, err := security.GenerateNextPageToken(pageTokenChaim.PageOffset, int(req.GetPageSize()), len(tokens))
	if err != nil {
		logger.Sugar().Errorf("Error generate next page token: %v\n", err.Error())

		respErr := e.NewInternalError(err)
		return nil, respErr
	}

	resp := &registryv1alpha1.ListTokensResponse{
		Tokens:        tokens.ToProtoTokens(),
		NextPageToken: nextPageToken,
	}
	return resp, nil
}

func (controller *TokenController) DeleteToken(ctx context.Context, req *registryv1alpha1.DeleteTokenRequest) (*registryv1alpha1.DeleteTokenResponse, e.ResponseError) {
	userID, _ := ctx.Value(constant.UserIDKey).(string)

	// 删除token
	err := controller.tokenService.DeleteToken(ctx, userID, req.GetTokenId())
	if err != nil {
		logger.Sugar().Errorf("Error delete token: %v\n", err.Error())

		return nil, err
	}

	resp := &registryv1alpha1.DeleteTokenResponse{}
	return resp, nil
}
