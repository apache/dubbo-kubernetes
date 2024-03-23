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

package grpc_handlers

import (
	"context"
)

import (
	"github.com/apache/dubbo-kubernetes/pkg/bufman/controllers"
	registryv1alpha1 "github.com/apache/dubbo-kubernetes/pkg/bufman/gen/proto/go/registry/v1alpha1"
)

type TokenServiceHandler struct {
	registryv1alpha1.UnimplementedTokenServiceServer
	tokenController *controllers.TokenController
}

func NewTokenServiceHandler() *TokenServiceHandler {
	return &TokenServiceHandler{
		tokenController: controllers.NewTokenController(),
	}
}

func (handler *TokenServiceHandler) CreateToken(ctx context.Context, req *registryv1alpha1.CreateTokenRequest) (*registryv1alpha1.CreateTokenResponse, error) {
	resp, err := handler.tokenController.CreateToken(ctx, req)
	if err != nil {
		return nil, err.Err()
	}

	return resp, nil
}

func (handler *TokenServiceHandler) GetToken(ctx context.Context, req *registryv1alpha1.GetTokenRequest) (*registryv1alpha1.GetTokenResponse, error) {
	resp, err := handler.tokenController.GetToken(ctx, req)
	if err != nil {
		return nil, err.Err()
	}

	return resp, nil
}

func (handler *TokenServiceHandler) ListTokens(ctx context.Context, req *registryv1alpha1.ListTokensRequest) (*registryv1alpha1.ListTokensResponse, error) {
	resp, err := handler.tokenController.ListTokens(ctx, req)
	if err != nil {
		return nil, err.Err()
	}

	return resp, nil
}

func (handler *TokenServiceHandler) DeleteToken(ctx context.Context, req *registryv1alpha1.DeleteTokenRequest) (*registryv1alpha1.DeleteTokenResponse, error) {
	resp, err := handler.tokenController.DeleteToken(ctx, req)
	if err != nil {
		return nil, err.Err()
	}

	return resp, nil
}
