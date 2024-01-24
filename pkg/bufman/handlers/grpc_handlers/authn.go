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

	"github.com/apache/dubbo-kubernetes/pkg/bufman/controllers"
	registryv1alpha1 "github.com/apache/dubbo-kubernetes/pkg/bufman/gen/proto/go/registry/v1alpha1"
	"github.com/apache/dubbo-kubernetes/pkg/core/logger"
)

type AuthnServiceHandler struct {
	registryv1alpha1.UnimplementedAuthnServiceServer
	authnController *controllers.AuthnController
}

func NewAuthnServiceHandler() *AuthnServiceHandler {
	return &AuthnServiceHandler{
		authnController: controllers.NewAuthnController(),
	}
}

func (handler *AuthnServiceHandler) GetCurrentUser(ctx context.Context, request *registryv1alpha1.GetCurrentUserRequest) (*registryv1alpha1.GetCurrentUserResponse, error) {
	resp, err := handler.authnController.GetCurrentUser(ctx, request)
	if err != nil {
		logger.Sugar().Errorf("Error Get User: %v\n", err.Error())
		return nil, err.Err()
	}

	return resp, nil
}
