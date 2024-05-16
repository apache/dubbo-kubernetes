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
	"github.com/apache/dubbo-kubernetes/pkg/bufman/bufpkg/bufmanifest"
	"github.com/apache/dubbo-kubernetes/pkg/bufman/constant"
	"github.com/apache/dubbo-kubernetes/pkg/bufman/e"
	registryv1alpha1 "github.com/apache/dubbo-kubernetes/pkg/bufman/gen/proto/go/registry/v1alpha1"
	"github.com/apache/dubbo-kubernetes/pkg/bufman/services"
	"github.com/apache/dubbo-kubernetes/pkg/core/logger"
)

type DownloadServiceHandler struct {
	registryv1alpha1.UnimplementedDownloadServiceServer

	downloadService      services.DownloadService
	authorizationService services.AuthorizationService
}

func NewDownloadServiceHandler() *DownloadServiceHandler {
	return &DownloadServiceHandler{
		downloadService:      services.NewDownloadService(),
		authorizationService: services.NewAuthorizationService(),
	}
}

func (handler *DownloadServiceHandler) DownloadManifestAndBlobs(ctx context.Context, req *registryv1alpha1.DownloadManifestAndBlobsRequest) (*registryv1alpha1.DownloadManifestAndBlobsResponse, error) {
	// 检查用户权限
	userID, _ := ctx.Value(constant.UserIDKey).(string)
	repository, checkErr := handler.authorizationService.CheckRepositoryCanAccess(userID, req.GetOwner(), req.GetRepository())
	if checkErr != nil {
		logger.Sugar().Errorf("Error Check: %v\n", checkErr.Error())

		return nil, checkErr.Err()
	}

	// 获取对应文件内容、文件清单
	fileManifest, blobSet, err := handler.downloadService.DownloadManifestAndBlobs(ctx, repository.RepositoryID, req.GetReference())
	if err != nil {
		logger.Sugar().Errorf("Error download manifest and blobs: %v\n", err.Error())

		return nil, err.Err()
	}

	// 转为响应格式
	protoManifest, protoBlobs, toProtoErr := bufmanifest.ToProtoManifestAndBlobs(ctx, fileManifest, blobSet)
	if toProtoErr != nil {
		logger.Sugar().Errorf("Error transform to response proto manifest and blobs: %v\n", toProtoErr.Error())

		respErr := e.NewInternalError(toProtoErr)
		return nil, respErr.Err()
	}

	return &registryv1alpha1.DownloadManifestAndBlobsResponse{
		Manifest: protoManifest,
		Blobs:    protoBlobs,
	}, nil
}
