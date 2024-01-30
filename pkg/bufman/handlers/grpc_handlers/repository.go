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

type RepositoryServiceHandler struct {
	registryv1alpha1.UnimplementedRepositoryServiceServer
	repositoryController *controllers.RepositoryController
}

func NewRepositoryServiceHandler() *RepositoryServiceHandler {
	return &RepositoryServiceHandler{
		repositoryController: controllers.NewRepositoryController(),
	}
}

func (handler *RepositoryServiceHandler) GetRepository(ctx context.Context, req *registryv1alpha1.GetRepositoryRequest) (*registryv1alpha1.GetRepositoryResponse, error) {
	resp, err := handler.repositoryController.GetRepository(ctx, req)
	if err != nil {
		return nil, err.Err()
	}

	return resp, nil
}

func (handler *RepositoryServiceHandler) GetRepositoryByFullName(ctx context.Context, req *registryv1alpha1.GetRepositoryByFullNameRequest) (*registryv1alpha1.GetRepositoryByFullNameResponse, error) {
	resp, err := handler.repositoryController.GetRepositoryByFullName(ctx, req)
	if err != nil {
		return nil, err.Err()
	}

	return resp, nil
}

func (handler *RepositoryServiceHandler) ListRepositories(ctx context.Context, req *registryv1alpha1.ListRepositoriesRequest) (*registryv1alpha1.ListRepositoriesResponse, error) {
	resp, err := handler.repositoryController.ListRepositories(ctx, req)
	if err != nil {
		return nil, err.Err()
	}

	return resp, nil
}

func (handler *RepositoryServiceHandler) ListUserRepositories(ctx context.Context, req *registryv1alpha1.ListUserRepositoriesRequest) (*registryv1alpha1.ListUserRepositoriesResponse, error) {
	resp, err := handler.repositoryController.ListUserRepositories(ctx, req)
	if err != nil {
		return nil, err.Err()
	}

	return resp, nil
}

func (handler *RepositoryServiceHandler) ListRepositoriesUserCanAccess(ctx context.Context, req *registryv1alpha1.ListRepositoriesUserCanAccessRequest) (*registryv1alpha1.ListRepositoriesUserCanAccessResponse, error) {
	resp, err := handler.repositoryController.ListRepositoriesUserCanAccess(ctx, req)
	if err != nil {
		return nil, err.Err()
	}

	return resp, nil
}

func (handler *RepositoryServiceHandler) CreateRepositoryByFullName(ctx context.Context, req *registryv1alpha1.CreateRepositoryByFullNameRequest) (*registryv1alpha1.CreateRepositoryByFullNameResponse, error) {
	resp, err := handler.repositoryController.CreateRepositoryByFullName(ctx, req)
	if err != nil {
		return nil, err.Err()
	}

	return resp, nil
}

func (handler *RepositoryServiceHandler) DeleteRepository(ctx context.Context, req *registryv1alpha1.DeleteRepositoryRequest) (*registryv1alpha1.DeleteRepositoryResponse, error) {
	resp, err := handler.repositoryController.DeleteRepository(ctx, req)
	if err != nil {
		return nil, err.Err()
	}

	return resp, nil
}

func (handler *RepositoryServiceHandler) DeleteRepositoryByFullName(ctx context.Context, req *registryv1alpha1.DeleteRepositoryByFullNameRequest) (*registryv1alpha1.DeleteRepositoryByFullNameResponse, error) {
	resp, err := handler.repositoryController.DeleteRepositoryByFullName(ctx, req)
	if err != nil {
		return nil, err.Err()
	}

	return resp, nil
}

func (handler *RepositoryServiceHandler) DeprecateRepositoryByName(ctx context.Context, req *registryv1alpha1.DeprecateRepositoryByNameRequest) (*registryv1alpha1.DeprecateRepositoryByNameResponse, error) {
	resp, err := handler.repositoryController.DeprecateRepositoryByName(ctx, req)
	if err != nil {
		return nil, err.Err()
	}

	return resp, nil
}

func (handler *RepositoryServiceHandler) UndeprecateRepositoryByName(ctx context.Context, req *registryv1alpha1.UndeprecateRepositoryByNameRequest) (*registryv1alpha1.UndeprecateRepositoryByNameResponse, error) {
	resp, err := handler.repositoryController.UndeprecateRepositoryByName(ctx, req)
	if err != nil {
		return nil, err.Err()
	}

	return resp, nil
}

func (handler *RepositoryServiceHandler) UpdateRepositorySettingsByName(ctx context.Context, req *registryv1alpha1.UpdateRepositorySettingsByNameRequest) (*registryv1alpha1.UpdateRepositorySettingsByNameResponse, error) {
	resp, err := handler.repositoryController.UpdateRepositorySettingsByName(ctx, req)
	if err != nil {
		return nil, err.Err()
	}

	return resp, nil
}

func (handler *RepositoryServiceHandler) GetRepositoriesByFullName(ctx context.Context, req *registryv1alpha1.GetRepositoriesByFullNameRequest) (*registryv1alpha1.GetRepositoriesByFullNameResponse, error) {
	resp, err := handler.repositoryController.GetRepositoriesByFullName(ctx, req)
	if err != nil {
		return nil, err.Err()
	}

	return resp, nil
}
