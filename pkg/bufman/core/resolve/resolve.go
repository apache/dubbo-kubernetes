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

package resolve

import (
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/apache/dubbo-kubernetes/pkg/bufman/bufpkg/bufconfig"
	"github.com/apache/dubbo-kubernetes/pkg/bufman/bufpkg/bufmodule/bufmoduleref"
	"github.com/apache/dubbo-kubernetes/pkg/bufman/config"
	"github.com/apache/dubbo-kubernetes/pkg/bufman/core/storage"
	"github.com/apache/dubbo-kubernetes/pkg/bufman/e"
	"github.com/apache/dubbo-kubernetes/pkg/bufman/mapper"
	"github.com/apache/dubbo-kubernetes/pkg/bufman/model"
	manifest2 "github.com/apache/dubbo-kubernetes/pkg/bufman/pkg/manifest"
	"gorm.io/gorm"
)

type Resolver interface {
	GetAllDependenciesFromBufConfig(ctx context.Context, bufConfig *bufconfig.Config) (model.Commits, e.ResponseError)                         // 获取全部依赖
	GetDirectDependenciesFromBufConfig(ctx context.Context, bufConfig *bufconfig.Config) (model.Commits, e.ResponseError)                      // 获取直接依赖
	GetAllDependenciesFromModuleRefs(ctx context.Context, moduleReferences []bufmoduleref.ModuleReference) (model.Commits, e.ResponseError)    // 获取全部依赖
	GetDirectDependenciesFromModuleRefs(ctx context.Context, moduleReferences []bufmoduleref.ModuleReference) (model.Commits, e.ResponseError) // 获取直接依赖
}

type ResolverImpl struct {
	repositoryMapper mapper.RepositoryMapper
	commitMapper     mapper.CommitMapper
	fileMapper       mapper.FileMapper
	storageHelper    storage.StorageHelper
}

func NewResolver() Resolver {
	return &ResolverImpl{
		repositoryMapper: &mapper.RepositoryMapperImpl{},
		commitMapper:     &mapper.CommitMapperImpl{},
		fileMapper:       &mapper.FileMapperImpl{},
		storageHelper:    storage.NewStorageHelper(),
	}
}

func (resolver *ResolverImpl) GetAllDependenciesFromBufConfig(ctx context.Context, bufConfig *bufconfig.Config) (model.Commits, e.ResponseError) {
	var dependentCommitSet map[string]*model.Commit
	var err e.ResponseError
	dependentCommitSet, err = resolver.doGetDependencies(ctx, dependentCommitSet, bufConfig.Build.DependencyModuleReferences, true)
	if err != nil {
		return nil, err
	}

	commits := make([]*model.Commit, 0, len(dependentCommitSet))
	for _, commit := range dependentCommitSet {
		commits = append(commits, commit)
	}

	return commits, nil
}

func (resolver *ResolverImpl) GetDirectDependenciesFromBufConfig(ctx context.Context, bufConfig *bufconfig.Config) (model.Commits, e.ResponseError) {
	var dependentCommitSet map[string]*model.Commit
	var err e.ResponseError
	dependentCommitSet, err = resolver.doGetDependencies(ctx, dependentCommitSet, bufConfig.Build.DependencyModuleReferences, false)
	if err != nil {
		return nil, err
	}

	commits := make([]*model.Commit, 0, len(dependentCommitSet))
	for _, commit := range dependentCommitSet {
		commits = append(commits, commit)
	}

	return commits, nil
}

func (resolver *ResolverImpl) GetAllDependenciesFromModuleRefs(ctx context.Context, moduleReferences []bufmoduleref.ModuleReference) (model.Commits, e.ResponseError) {
	var dependentCommitSet map[string]*model.Commit
	var err e.ResponseError
	dependentCommitSet, err = resolver.doGetDependencies(ctx, dependentCommitSet, moduleReferences, true)
	if err != nil {
		return nil, err
	}

	commits := make([]*model.Commit, 0, len(dependentCommitSet))
	for _, commit := range dependentCommitSet {
		commits = append(commits, commit)
	}

	return commits, nil
}

func (resolver *ResolverImpl) GetDirectDependenciesFromModuleRefs(ctx context.Context, moduleReferences []bufmoduleref.ModuleReference) (model.Commits, e.ResponseError) {
	var dependentCommitSet map[string]*model.Commit
	var err e.ResponseError
	dependentCommitSet, err = resolver.doGetDependencies(ctx, dependentCommitSet, moduleReferences, false)
	if err != nil {
		return nil, err
	}

	commits := make([]*model.Commit, 0, len(dependentCommitSet))
	for _, commit := range dependentCommitSet {
		commits = append(commits, commit)
	}

	return commits, nil
}

func (resolver *ResolverImpl) GetBufConfigFromCommitID(ctx context.Context, commitID string) (*bufconfig.Config, e.ResponseError) {
	// 查询manifest名称
	manifestModel, err := resolver.fileMapper.FindManifestByCommitID(commitID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, e.NewNotFoundError(fmt.Errorf("manifest(commit id = %s)", commitID))
		}
		return nil, e.NewInternalError(errors.New("GetDependenciesByCommitID"))
	}

	// 读取manifest
	reader, err := resolver.storageHelper.ReadManifestToReader(ctx, manifestModel.Digest)
	if err != nil {
		return nil, nil
	}
	fileManifest, err := manifest2.NewFromReader(reader)
	if err != nil {
		return nil, e.NewInternalError(errors.New("GetDependenciesByCommitID"))
	}

	// 根据文件manifest查找配置文件
	var configFileExist bool
	var configFileData []byte
	err = fileManifest.Range(func(path string, digest manifest2.Digest) error {
		// 如果遇到配置文件，就记录下来
		for _, configFilePath := range bufconfig.AllConfigFilePaths {
			if path == configFilePath {
				if configFileExist {
					return errors.New("two config files")
				}

				reader, err := resolver.storageHelper.ReadBlobToReader(ctx, digest.Hex())
				if err != nil {
					return err
				}
				configFileData, err = io.ReadAll(reader)
				if err != nil {
					return err
				}
				configFileExist = true
			}
		}

		return nil
	})
	if err != nil {
		return nil, e.NewInvalidArgumentError(err)
	}
	if !configFileExist {
		// 不存在配置文件
		return nil, e.NewInvalidArgumentError(errors.New("no config file"))
	}

	bufConfig, err := bufconfig.GetConfigForData(ctx, configFileData)
	if err != nil {
		return nil, e.NewInternalError(errors.New("GetDependenciesByCommitID"))
	}

	return bufConfig, nil
}

func (resolver *ResolverImpl) doGetDependencies(ctx context.Context, dependentCommitSet map[string]*model.Commit, dependencyReferences []bufmoduleref.ModuleReference, getAll bool) (map[string]*model.Commit, e.ResponseError) {
	if dependentCommitSet == nil {
		dependentCommitSet = map[string]*model.Commit{}
	}

	for i := 0; i < len(dependencyReferences); i++ {
		dependencyReference := dependencyReferences[i]
		if dependencyReference.Remote() == config.Properties.Server.ServerHost {
			// 查询repo
			repo, err := resolver.repositoryMapper.FindByUserNameAndRepositoryName(dependencyReference.Owner(), dependencyReference.Repository())
			if err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					return nil, e.NewNotFoundError(errors.New(dependencyReference.IdentityString()))
				}
				return nil, e.NewInternalError(fmt.Errorf("find repository(%s)", err.Error()))
			}

			// 查询当前reference，版本号是否一样
			commit, err := resolver.commitMapper.FindByRepositoryIDAndReference(repo.RepositoryID, dependencyReference.Reference())
			if err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					return nil, e.NewNotFoundError(fmt.Errorf("%s:%s", dependencyReference.IdentityString(), dependencyReference.Reference()))
				}
				return nil, e.NewInternalError(fmt.Errorf("find reference(%s)", err.Error()))
			}

			dependentCommit, ok := dependentCommitSet[dependencyReference.IdentityString()]
			if ok && dependentCommit.CommitName == dependencyReference.Reference() {
				continue
			}
			if ok {
				// 之前已经记录过 owner/repository 的commit
				if commit.CommitName != dependentCommit.CommitName {
					// 同一个仓库下的依赖版本号不同，返回错误
					return nil, e.NewInternalError(fmt.Errorf("two different version %s and %s for %s", dependentCommit.CommitName, dependencyReference.Reference(), dependencyReference.IdentityString()))
				}

				// 当前依赖已经记录，跳过
				continue
			} else {
				// 如果之前没有记录过，记录依赖commit
				dependentCommitSet[dependencyReference.IdentityString()] = commit

				if getAll {
					// 需要获取全部依赖，记录这个依赖下的依赖关系
					dependentBufConfig, configErr := resolver.GetBufConfigFromCommitID(ctx, commit.CommitID)
					if configErr != nil {
						return nil, configErr
					}

					var dependentErr e.ResponseError
					dependentCommitSet, dependentErr = resolver.doGetDependencies(ctx, dependentCommitSet, dependentBufConfig.Build.DependencyModuleReferences, true)
					if dependentErr != nil {
						return nil, dependentErr
					}
				}

			}
		}
	}

	// 通过
	return dependentCommitSet, nil
}
