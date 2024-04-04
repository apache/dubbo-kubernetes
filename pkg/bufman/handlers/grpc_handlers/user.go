/*
 * Licensed to the Apache Software Foundation (ASF) under one or more
 * contributor license agreements.  See the NOTICE file distributed with
 * this work for additional information regarding copyright ownership.
 * The ASF licenses this file to You under the Apache License, Version 2.0
 * (the "License"); you may not use this file except in compliance with
 * the License.  You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package grpc_handlers

import (
	"context"
)

import (
	"github.com/apache/dubbo-kubernetes/pkg/bufman/controllers"
	registryv1alpha1 "github.com/apache/dubbo-kubernetes/pkg/bufman/gen/proto/go/registry/v1alpha1"
)

type UserServiceHandler struct {
	registryv1alpha1.UnimplementedUserServiceServer
	userController *controllers.UserController
}

func NewUserServiceHandler() *UserServiceHandler {
	return &UserServiceHandler{
		userController: controllers.NewUserController(),
	}
}

func (handler *UserServiceHandler) CreateUser(ctx context.Context, req *registryv1alpha1.CreateUserRequest) (*registryv1alpha1.CreateUserResponse, error) {
	resp, err := handler.userController.CreateUser(ctx, req)
	if err != nil {
		return nil, err.Err()
	}

	return resp, nil
}

func (handler *UserServiceHandler) GetUser(ctx context.Context, req *registryv1alpha1.GetUserRequest) (*registryv1alpha1.GetUserResponse, error) {
	resp, err := handler.userController.GetUser(ctx, req)
	if err != nil {
		return nil, err.Err()
	}

	return resp, nil
}

func (handler *UserServiceHandler) GetUserByUsername(ctx context.Context, req *registryv1alpha1.GetUserByUsernameRequest) (*registryv1alpha1.GetUserByUsernameResponse, error) {
	resp, err := handler.userController.GetUserByUsername(ctx, req)
	if err != nil {
		return nil, err.Err()
	}

	return resp, nil
}

func (handler *UserServiceHandler) ListUsers(ctx context.Context, req *registryv1alpha1.ListUsersRequest) (*registryv1alpha1.ListUsersResponse, error) {
	resp, err := handler.userController.ListUsers(ctx, req)
	if err != nil {
		return nil, err.Err()
	}

	return resp, nil
}
