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
	"fmt"
	"io"

	"github.com/apache/dubbo-kubernetes/pkg/bufman/bufpkg/bufconfig"
	"github.com/apache/dubbo-kubernetes/pkg/bufman/constant"
	"github.com/apache/dubbo-kubernetes/pkg/bufman/core/parser"
	"github.com/apache/dubbo-kubernetes/pkg/bufman/core/resolve"
	"github.com/apache/dubbo-kubernetes/pkg/bufman/core/storage"
	"github.com/apache/dubbo-kubernetes/pkg/bufman/core/validity"
	"github.com/apache/dubbo-kubernetes/pkg/bufman/e"
	registryv1alpha1 "github.com/apache/dubbo-kubernetes/pkg/bufman/gen/proto/go/registry/v1alpha1"
	"github.com/apache/dubbo-kubernetes/pkg/bufman/model"
	manifest2 "github.com/apache/dubbo-kubernetes/pkg/bufman/pkg/manifest"
	"github.com/apache/dubbo-kubernetes/pkg/bufman/services"
	"github.com/apache/dubbo-kubernetes/pkg/core/logger"
)

type PushServiceHandler struct {
	registryv1alpha1.UnimplementedPushServiceServer

	pushService   services.PushService
	validator     validity.Validator
	resolver      resolve.Resolver
	storageHelper storage.StorageHelper
	protoParser   parser.ProtoParser
}

func NewPushServiceHandler() *PushServiceHandler {
	return &PushServiceHandler{
		pushService:   services.NewPushService(),
		validator:     validity.NewValidator(),
		resolver:      resolve.NewResolver(),
		storageHelper: storage.NewStorageHelper(),
		protoParser:   parser.NewProtoParser(),
	}
}

func (handler *PushServiceHandler) PushManifestAndBlobs(ctx context.Context, req *registryv1alpha1.PushManifestAndBlobsRequest) (*registryv1alpha1.PushManifestAndBlobsResponse, error) {
	// 验证参数

	// 检查tags名称合法性
	var argErr e.ResponseError
	for _, tag := range req.GetTags() {
		argErr = handler.validator.CheckTagName(tag)
		if argErr != nil {
			logger.Sugar().Errorf("Error check: %v\n", argErr.Err())

			return nil, argErr.Err()
		}
	}

	// 检查draft名称合法性
	if req.GetDraftName() != "" {
		argErr = handler.validator.CheckDraftName(req.GetDraftName())
		if argErr != nil {
			logger.Sugar().Errorf("Error check: %v\n", argErr.Err())

			return nil, argErr.Err()
		}
	}

	// draft和tag只能二选一
	if req.GetDraftName() != "" && len(req.GetTags()) > 0 {
		responseError := e.NewInvalidArgumentError(fmt.Errorf("draft and tags (only choose one)"))
		logger.Sugar().Errorf("Error draft and tag must choose one (not both): %v\n", responseError.Err())
		return nil, responseError.Err()
	}

	// 检查上传文件
	fileManifest, blobSet, checkErr := handler.validator.CheckManifestAndBlobs(ctx, req.GetManifest(), req.GetBlobs())
	if checkErr != nil {
		logger.Sugar().Errorf("Error check manifest and blobs: %v\n", checkErr.Err())

		return nil, checkErr.Err()
	}

	// 获取bufConfig
	bufConfigBlob, err := handler.storageHelper.GetBufManConfigFromBlob(ctx, fileManifest, blobSet)
	if err != nil {
		logger.Sugar().Errorf("Error get config: %v\n", err)

		configErr := e.NewInternalError(err)
		return nil, configErr.Err()
	}

	var dependentManifests []*manifest2.Manifest
	var dependentBlobSets []*manifest2.BlobSet
	if bufConfigBlob != nil {
		// 生成Config
		reader, err := bufConfigBlob.Open(ctx)
		if err != nil {
			logger.Sugar().Errorf("Error read config: %v\n", err)

			respErr := e.NewInternalError(err)
			return nil, respErr.Err()
		}
		defer reader.Close()
		configData, err := io.ReadAll(reader)
		if err != nil {
			logger.Sugar().Errorf("Error read config: %v\n", err)

			respErr := e.NewInternalError(err)
			return nil, respErr.Err()
		}
		bufConfig, err := bufconfig.GetConfigForData(ctx, configData)
		if err != nil {
			logger.Sugar().Errorf("Error read config: %v\n", err)

			// 无法解析配置文件
			respErr := e.NewInternalError(err)
			return nil, respErr.Err()
		}

		// 获取全部依赖commits
		dependentCommits, dependenceErr := handler.resolver.GetAllDependenciesFromBufConfig(ctx, bufConfig)
		if dependenceErr != nil {
			logger.Sugar().Errorf("Error get all dependencies: %v\n", dependenceErr)

			return nil, dependenceErr.Err()
		}

		// 读取依赖文件
		dependentManifests = make([]*manifest2.Manifest, 0, len(dependentCommits))
		dependentBlobSets = make([]*manifest2.BlobSet, 0, len(dependentCommits))
		for i := 0; i < len(dependentCommits); i++ {
			dependentCommit := dependentCommits[i]
			dependentManifest, dependentBlobSet, getErr := handler.pushService.GetManifestAndBlobSet(ctx, dependentCommit.RepositoryID, dependentCommit.CommitName)
			if getErr != nil {
				logger.Sugar().Errorf("Error get manifest and blob set: %v\n", getErr)

				return nil, getErr.Err()
			}

			dependentManifests = append(dependentManifests, dependentManifest)
			dependentBlobSets = append(dependentBlobSets, dependentBlobSet)
		}
	}

	// 编译检查
	compileErr := handler.protoParser.TryCompile(ctx, fileManifest, blobSet, dependentManifests, dependentBlobSets)
	if compileErr != nil {
		logger.Sugar().Errorf("Error try to compile proto: %v\n", compileErr.Error())

		return nil, compileErr.Err()
	}

	var commit *model.Commit
	var serviceErr e.ResponseError
	userID, _ := ctx.Value(constant.UserIDKey).(string)
	if req.DraftName != "" {
		commit, serviceErr = handler.pushService.PushManifestAndBlobsWithDraft(ctx, userID, req.GetOwner(), req.GetRepository(), fileManifest, blobSet, req.GetDraftName())
	} else if len(req.GetTags()) > 0 {
		commit, serviceErr = handler.pushService.PushManifestAndBlobsWithTags(ctx, userID, req.GetOwner(), req.GetRepository(), fileManifest, blobSet, req.GetTags())
	} else {
		commit, serviceErr = handler.pushService.PushManifestAndBlobs(ctx, userID, req.GetOwner(), req.GetRepository(), fileManifest, blobSet)
	}
	if serviceErr != nil {
		logger.Sugar().Errorf("Error push: %v\n", serviceErr.Error())

		return nil, serviceErr.Err()
	}

	return &registryv1alpha1.PushManifestAndBlobsResponse{
		LocalModulePin: commit.ToProtoLocalModulePin(),
	}, nil
}
