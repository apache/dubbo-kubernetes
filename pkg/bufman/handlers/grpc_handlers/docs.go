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
)

type DocServiceHandler struct {
	registryv1alpha1.UnimplementedDocServiceServer

	docController *controllers.DocController
}

func NewDocServiceHandler() *DocServiceHandler {
	return &DocServiceHandler{
		docController: controllers.NewDocController(),
	}
}

func (handler *DocServiceHandler) GetSourceDirectoryInfo(ctx context.Context, req *registryv1alpha1.GetSourceDirectoryInfoRequest) (*registryv1alpha1.GetSourceDirectoryInfoResponse, error) {
	resp, err := handler.docController.GetSourceDirectoryInfo(ctx, req)
	if err != nil {
		return nil, err.Err()
	}

	return resp, nil
}

func (handler *DocServiceHandler) GetSourceFile(ctx context.Context, req *registryv1alpha1.GetSourceFileRequest) (*registryv1alpha1.GetSourceFileResponse, error) {
	resp, err := handler.docController.GetSourceFile(ctx, req)
	if err != nil {
		return nil, err.Err()
	}

	return resp, nil
}

func (handler *DocServiceHandler) GetModulePackages(ctx context.Context, req *registryv1alpha1.GetModulePackagesRequest) (*registryv1alpha1.GetModulePackagesResponse, error) {
	resp, err := handler.docController.GetModulePackages(ctx, req)
	if err != nil {
		return nil, err.Err()
	}

	return resp, nil
}

func (handler *DocServiceHandler) GetModuleDocumentation(ctx context.Context, req *registryv1alpha1.GetModuleDocumentationRequest) (*registryv1alpha1.GetModuleDocumentationResponse, error) {
	resp, err := handler.docController.GetModuleDocumentation(ctx, req)
	if err != nil {
		return nil, err.Err()
	}

	return resp, nil
}

func (handler *DocServiceHandler) GetPackageDocumentation(ctx context.Context, req *registryv1alpha1.GetPackageDocumentationRequest) (*registryv1alpha1.GetPackageDocumentationResponse, error) {
	resp, err := handler.docController.GetPackageDocumentation(ctx, req)
	if err != nil {
		return nil, err.Err()
	}

	return resp, nil
}
