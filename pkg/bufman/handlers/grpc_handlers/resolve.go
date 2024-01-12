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

	"github.com/apache/dubbo-kubernetes/pkg/bufman/bufpkg/bufmodule/bufmoduleref"
	"github.com/apache/dubbo-kubernetes/pkg/bufman/constant"
	"github.com/apache/dubbo-kubernetes/pkg/bufman/core/resolve"
	"github.com/apache/dubbo-kubernetes/pkg/bufman/e"
	registryv1alpha1 "github.com/apache/dubbo-kubernetes/pkg/bufman/gen/proto/go/registry/v1alpha1"
	"github.com/apache/dubbo-kubernetes/pkg/bufman/model"
	"github.com/apache/dubbo-kubernetes/pkg/bufman/services"
	"github.com/apache/dubbo-kubernetes/pkg/core/logger"
)

type ResolveServiceHandler struct {
	registryv1alpha1.UnimplementedResolveServiceServer

	resolver             resolve.Resolver
	authorizationService services.AuthorizationService
}

func NewResolveServiceHandler() *ResolveServiceHandler {
	return &ResolveServiceHandler{
		resolver:             resolve.NewResolver(),
		authorizationService: services.NewAuthorizationService(),
	}
}

func (handler *ResolveServiceHandler) GetModulePins(ctx context.Context, req *registryv1alpha1.GetModulePinsRequest) (*registryv1alpha1.GetModulePinsResponse, error) {
	userID, _ := ctx.Value(constant.UserIDKey).(string)

	// 首先检查用户权限，是否对repo有访问权限
	var checkErr e.ResponseError
	repositoryMap := map[string]*model.Repository{}
	for _, moduleReference := range req.GetModuleReferences() {
		fullName := moduleReference.GetOwner() + "/" + moduleReference.GetRepository()
		repo, ok := repositoryMap[fullName]
		if !ok {
			repo, checkErr = handler.authorizationService.CheckRepositoryCanAccess(userID, moduleReference.GetOwner(), moduleReference.GetRepository())
			if checkErr != nil {
				logger.Sugar().Errorf("Error check: %v\n", checkErr.Error())

				return nil, checkErr.Err()
			}
			repositoryMap[fullName] = repo
		}
		repositoryMap[fullName] = repo
	}

	moduleReferences, bufRefErr := bufmoduleref.NewModuleReferencesForProtos(req.GetModuleReferences()...)
	if bufRefErr != nil {
		logger.Sugar().Errorf("Error read mod ref from proto: %v\n", bufRefErr.Error())

		return nil, e.NewInternalError(bufRefErr).Err()
	}

	// 获取所有的依赖commits
	commits, err := handler.resolver.GetAllDependenciesFromModuleRefs(ctx, moduleReferences)
	if err != nil {
		logger.Sugar().Errorf("Error get all dependencies: %v\n", err.Error())

		return nil, err.Err()
	}

	retPins := commits.ToProtoModulePins()
	currentModulePins, curPinErr := bufmoduleref.NewModulePinsForProtos(req.GetCurrentModulePins()...)
	if curPinErr != nil {
		logger.Sugar().Errorf("Error read mod pins from proto: %v\n", curPinErr.Error())

		return nil, e.NewInternalError(curPinErr).Err()
	}
	// 处理CurrentModulePins
	for _, currentModulePin := range currentModulePins {
		for _, moduleRef := range moduleReferences {
			if currentModulePin.IdentityString() == moduleRef.IdentityString() {
				// 需要更新的依赖，已经加入到了返回的结果中
				continue
			}
		}

		ownerName := currentModulePin.Owner()
		repositoryName := currentModulePin.Repository()
		for _, commit := range commits {
			// 如果current module pin在reference的查询出的commits内，则有breaking的可能
			if commit.UserName == ownerName && commit.RepositoryName == repositoryName {
				commitName := currentModulePin.Commit()
				if commit.CommitName != commitName {
					// 版本号不一样，存在breaking
					respErr := e.NewInvalidArgumentError(fmt.Errorf("%s/%s (possible to cause breaking)", currentModulePin.Owner(), currentModulePin.Repository()))
					logger.Sugar().Errorf("Error has breaking: %v\n", respErr.Error())

					return nil, respErr.Err()
				}
			}
		}

		// 当前pin没有breaking的可能性，加入到返回结果中
		protoPin := bufmoduleref.NewProtoModulePinForModulePin(currentModulePin)
		retPins = append(retPins, protoPin)
	}

	return &registryv1alpha1.GetModulePinsResponse{
		ModulePins: retPins,
	}, nil
}
