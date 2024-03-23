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
)

import (
	"github.com/apache/dubbo-kubernetes/pkg/bufman/constant"
	"github.com/apache/dubbo-kubernetes/pkg/bufman/core/validity"
	"github.com/apache/dubbo-kubernetes/pkg/bufman/e"
	registryv1alpha1 "github.com/apache/dubbo-kubernetes/pkg/bufman/gen/proto/go/registry/v1alpha1"
	"github.com/apache/dubbo-kubernetes/pkg/bufman/services"
	"github.com/apache/dubbo-kubernetes/pkg/core/logger"
)

type DocController struct {
	docsService          services.DocsService
	authorizationService services.AuthorizationService
	validator            validity.Validator
}

func NewDocController() *DocController {
	return &DocController{
		docsService:          services.NewDocsService(),
		authorizationService: services.NewAuthorizationService(),
		validator:            validity.NewValidator(),
	}
}

func (controller *DocController) GetSourceDirectoryInfo(ctx context.Context, req *registryv1alpha1.GetSourceDirectoryInfoRequest) (*registryv1alpha1.GetSourceDirectoryInfoResponse, e.ResponseError) {
	userID, _ := ctx.Value(constant.UserIDKey).(string)

	// 检查用户权限
	repository, checkErr := controller.authorizationService.CheckRepositoryCanAccess(userID, req.GetOwner(), req.GetRepository())
	if checkErr != nil {
		logger.Sugar().Errorf("Error Check: %v\n", checkErr.Error())

		return nil, checkErr
	}

	// 获取目录结构信息
	directoryInfo, respErr := controller.docsService.GetSourceDirectoryInfo(ctx, repository.RepositoryID, req.GetReference())
	if respErr != nil {
		logger.Sugar().Errorf("Error get source dir info: %v\n", respErr.Error())

		return nil, respErr
	}

	resp := &registryv1alpha1.GetSourceDirectoryInfoResponse{
		Root: directoryInfo.ToProtoFileInfo(),
	}
	return resp, nil
}

func (controller *DocController) GetSourceFile(ctx context.Context, req *registryv1alpha1.GetSourceFileRequest) (*registryv1alpha1.GetSourceFileResponse, e.ResponseError) {
	userID, _ := ctx.Value(constant.UserIDKey).(string)

	// 检查用户权限
	repository, checkErr := controller.authorizationService.CheckRepositoryCanAccess(userID, req.GetOwner(), req.GetRepository())
	if checkErr != nil {
		logger.Sugar().Errorf("Error Check: %v\n", checkErr.Error())

		return nil, checkErr
	}

	// 获取源码内容
	content, respErr := controller.docsService.GetSourceFile(ctx, repository.RepositoryID, req.GetReference(), req.GetPath())
	if respErr != nil {
		logger.Sugar().Errorf("Error get source file: %v\n", respErr.Error())

		return nil, respErr
	}

	resp := &registryv1alpha1.GetSourceFileResponse{
		Content: content,
	}
	return resp, nil
}

func (controller *DocController) GetModulePackages(ctx context.Context, req *registryv1alpha1.GetModulePackagesRequest) (*registryv1alpha1.GetModulePackagesResponse, e.ResponseError) {
	userID, _ := ctx.Value(constant.UserIDKey).(string)

	// 检查用户权限
	repository, checkErr := controller.authorizationService.CheckRepositoryCanAccess(userID, req.GetOwner(), req.GetRepository())
	if checkErr != nil {
		logger.Sugar().Errorf("Error Check: %v\n", checkErr.Error())

		return nil, checkErr
	}

	modulePackages, respErr := controller.docsService.GetModulePackages(ctx, repository.RepositoryID, req.GetReference())
	if respErr != nil {
		logger.Sugar().Errorf("Error get module packages: %v\n", respErr.Error())

		return nil, respErr
	}

	resp := &registryv1alpha1.GetModulePackagesResponse{
		Name:           req.GetRepository(),
		ModulePackages: modulePackages,
	}
	return resp, nil
}

func (controller *DocController) GetModuleDocumentation(ctx context.Context, req *registryv1alpha1.GetModuleDocumentationRequest) (*registryv1alpha1.GetModuleDocumentationResponse, e.ResponseError) {
	userID, _ := ctx.Value(constant.UserIDKey).(string)

	// 检查用户权限
	repository, checkErr := controller.authorizationService.CheckRepositoryCanAccess(userID, req.GetOwner(), req.GetRepository())
	if checkErr != nil {
		logger.Sugar().Errorf("Error Check: %v\n", checkErr.Error())

		return nil, checkErr
	}

	moduleDocumentation, respErr := controller.docsService.GetModuleDocumentation(ctx, repository.RepositoryID, req.GetReference())
	if respErr != nil {
		logger.Sugar().Errorf("Error get module doc: %v\n", respErr.Error())

		return nil, respErr
	}

	resp := &registryv1alpha1.GetModuleDocumentationResponse{
		ModuleDocumentation: moduleDocumentation,
	}
	return resp, nil
}

func (controller *DocController) GetPackageDocumentation(ctx context.Context, req *registryv1alpha1.GetPackageDocumentationRequest) (*registryv1alpha1.GetPackageDocumentationResponse, e.ResponseError) {
	userID, _ := ctx.Value(constant.UserIDKey).(string)

	// 检查用户权限
	repository, checkErr := controller.authorizationService.CheckRepositoryCanAccess(userID, req.GetOwner(), req.GetRepository())
	if checkErr != nil {
		logger.Sugar().Errorf("Error Check: %v\n", checkErr.Error())

		return nil, checkErr
	}

	packageDocumentation, respErr := controller.docsService.GetPackageDocumentation(ctx, repository.RepositoryID, req.GetReference(), req.GetPackageName())
	if respErr != nil {
		logger.Sugar().Errorf("Error get package doc: %v\n", respErr.Error())

		return nil, respErr
	}

	resp := &registryv1alpha1.GetPackageDocumentationResponse{
		PackageDocumentation: packageDocumentation,
	}
	return resp, nil
}
