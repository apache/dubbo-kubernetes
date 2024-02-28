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

	"github.com/apache/dubbo-kubernetes/pkg/bufman/core/security"
	"github.com/apache/dubbo-kubernetes/pkg/bufman/core/validity"
	"github.com/apache/dubbo-kubernetes/pkg/bufman/e"
	registryv1alpha1 "github.com/apache/dubbo-kubernetes/pkg/bufman/gen/proto/go/registry/v1alpha1"
	"github.com/apache/dubbo-kubernetes/pkg/bufman/services"
	"github.com/apache/dubbo-kubernetes/pkg/core/logger"
)

type UserController struct {
	userService services.UserService
	validator   validity.Validator
}

func NewUserController() *UserController {
	return &UserController{
		userService: services.NewUserService(),
		validator:   validity.NewValidator(),
	}
}

func (controller *UserController) CreateUser(ctx context.Context, req *registryv1alpha1.CreateUserRequest) (*registryv1alpha1.CreateUserResponse, e.ResponseError) {
	// 验证参数
	argErr := controller.validator.CheckUserName(req.GetUsername())
	if argErr != nil {
		logger.Sugar().Errorf("Error check: %v\n", argErr.Error())

		return nil, argErr
	}
	argErr = controller.validator.CheckPassword(req.GetPassword())
	if argErr != nil {
		logger.Sugar().Errorf("Error check: %v\n", argErr.Error())

		return nil, argErr
	}

	user, err := controller.userService.CreateUser(ctx, req.GetUsername(), req.GetPassword()) // 创建用户
	if err != nil {
		logger.Sugar().Errorf("Error create user: %v\n", err.Error())

		return nil, err
	}

	// success
	resp := &registryv1alpha1.CreateUserResponse{
		User: user.ToProtoUser(),
	}
	return resp, nil
}

func (controller *UserController) GetUser(ctx context.Context, req *registryv1alpha1.GetUserRequest) (*registryv1alpha1.GetUserResponse, e.ResponseError) {
	user, err := controller.userService.GetUser(ctx, req.GetId()) // 创建用户
	if err != nil {
		logger.Sugar().Errorf("Error get user: %v\n", err.Error())

		return nil, err
	}

	resp := &registryv1alpha1.GetUserResponse{
		User: user.ToProtoUser(),
	}
	return resp, nil
}

func (controller *UserController) GetUserByUsername(ctx context.Context, req *registryv1alpha1.GetUserByUsernameRequest) (*registryv1alpha1.GetUserByUsernameResponse, e.ResponseError) {
	user, err := controller.userService.GetUserByUsername(ctx, req.GetUsername()) // 创建用户
	if err != nil {
		logger.Sugar().Errorf("Error get user: %v\n", err.Error())

		return nil, err
	}

	resp := &registryv1alpha1.GetUserByUsernameResponse{
		User: user.ToProtoUser(),
	}
	return resp, nil
}

func (controller *UserController) ListUsers(ctx context.Context, req *registryv1alpha1.ListUsersRequest) (*registryv1alpha1.ListUsersResponse, e.ResponseError) {
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

	users, ListErr := controller.userService.ListUsers(ctx, pageTokenChaim.PageOffset, int(req.GetPageSize()), req.GetReverse()) // 创建用户
	if err != nil {
		logger.Sugar().Errorf("Error list users: %v\n", ListErr.Error())

		return nil, ListErr
	}

	// 生成下一页token
	nextPageToken, err := security.GenerateNextPageToken(pageTokenChaim.PageOffset, int(req.GetPageSize()), len(users))
	if err != nil {
		logger.Sugar().Errorf("Error generate next page token: %v\n", err.Error())

		respErr := e.NewInternalError(err)
		return nil, respErr
	}

	resp := &registryv1alpha1.ListUsersResponse{
		Users:         users.ToProtoUsers(),
		NextPageToken: nextPageToken,
	}

	return resp, nil
}
