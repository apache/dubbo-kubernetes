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

package services

import (
	"context"
	"errors"
	"fmt"
	"io"
)

import (
	"gorm.io/gorm"
)

import (
	"github.com/apache/dubbo-kubernetes/pkg/bufman/bufpkg/bufconfig"
	"github.com/apache/dubbo-kubernetes/pkg/bufman/bufpkg/bufmodule/bufmoduleref"
	"github.com/apache/dubbo-kubernetes/pkg/bufman/config"
	"github.com/apache/dubbo-kubernetes/pkg/bufman/core/parser"
	"github.com/apache/dubbo-kubernetes/pkg/bufman/core/resolve"
	"github.com/apache/dubbo-kubernetes/pkg/bufman/core/storage"
	"github.com/apache/dubbo-kubernetes/pkg/bufman/e"
	registryv1alpha1 "github.com/apache/dubbo-kubernetes/pkg/bufman/gen/proto/go/registry/v1alpha1"
	"github.com/apache/dubbo-kubernetes/pkg/bufman/mapper"
	"github.com/apache/dubbo-kubernetes/pkg/bufman/model"
	manifest2 "github.com/apache/dubbo-kubernetes/pkg/bufman/pkg/manifest"
)

type DocsService interface {
	GetSourceDirectoryInfo(ctx context.Context, repositoryID, reference string) (model.CommitFiles, e.ResponseError)
	GetSourceFile(ctx context.Context, repositoryID, reference, path string) ([]byte, e.ResponseError)
	GetModulePackages(ctx context.Context, repositoryID, reference string) ([]*registryv1alpha1.ModulePackage, e.ResponseError)
	GetModuleDocumentation(ctx context.Context, repositoryID, reference string) (*registryv1alpha1.ModuleDocumentation, e.ResponseError)
	GetPackageDocumentation(ctx context.Context, repositoryID, reference, packageName string) (*registryv1alpha1.PackageDocumentation, e.ResponseError)
}

type DocsServiceImpl struct {
	commitMapper  mapper.CommitMapper
	fileMapper    mapper.FileMapper
	storageHelper storage.StorageHelper
	protoParser   parser.ProtoParser
	resolver      resolve.Resolver
}

func NewDocsService() DocsService {
	return &DocsServiceImpl{
		commitMapper:  &mapper.CommitMapperImpl{},
		fileMapper:    &mapper.FileMapperImpl{},
		storageHelper: storage.NewStorageHelper(),
		protoParser:   parser.NewProtoParser(),
		resolver:      resolve.NewResolver(),
	}
}

func (docsService *DocsServiceImpl) GetSourceDirectoryInfo(ctx context.Context, repositoryID, reference string) (model.CommitFiles, e.ResponseError) {
	// 根据reference查询commit
	commit, err := docsService.commitMapper.FindByRepositoryIDAndReference(repositoryID, reference)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, e.NewNotFoundError(err)
		}

		return nil, e.NewInternalError(err)
	}

	// 查询所有文件
	fileBlobs, err := docsService.fileMapper.FindCommitFilesExceptManifestByCommitID(commit.CommitID)
	if err != nil {
		return nil, e.NewInternalError(err)
	}

	return fileBlobs, nil
}

func (docsService *DocsServiceImpl) GetSourceFile(ctx context.Context, repositoryID, reference, path string) ([]byte, e.ResponseError) {
	// 根据reference查询commit
	commit, err := docsService.commitMapper.FindByRepositoryIDAndReference(repositoryID, reference)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, e.NewNotFoundError(err)
		}

		return nil, e.NewInternalError(err)
	}

	// 查询file
	fileBlob, err := docsService.fileMapper.FindCommitFileByCommitIDAndPath(commit.CommitID, path)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, e.NewNotFoundError(err)
		}

		return nil, e.NewInternalError(err)
	}

	// 读取文件
	content, err := docsService.storageHelper.ReadBlob(ctx, fileBlob.Digest)
	if err != nil {
		return nil, e.NewInternalError(err)
	}

	return content, nil
}

func (docsService *DocsServiceImpl) GetModulePackages(ctx context.Context, repositoryID, reference string) ([]*registryv1alpha1.ModulePackage, e.ResponseError) {
	// 读取commit文件
	fileManifest, blobSet, err := docsService.getManifestAndBlobSet(ctx, repositoryID, reference)
	if err != nil {
		return nil, err
	}

	// 读取依赖
	dependentManifests, dependentBlobSets, err := docsService.getDependentManifestsAndBlobSets(ctx, fileManifest, blobSet)
	if err != nil {
		return nil, err
	}

	// 获取所有的packages
	packages, err := docsService.protoParser.GetPackages(ctx, fileManifest, blobSet, dependentManifests, dependentBlobSets)
	if err != nil {
		return nil, err
	}

	return packages, nil
}

func (docsService *DocsServiceImpl) GetModuleDocumentation(ctx context.Context, repositoryID, reference string) (*registryv1alpha1.ModuleDocumentation, e.ResponseError) {
	// 读取commit文件
	fileManifest, blobSet, err := docsService.getManifestAndBlobSet(ctx, repositoryID, reference)
	if err != nil {
		return nil, err
	}

	documentBlob, licenseBlob, readErr := docsService.storageHelper.GetDocumentAndLicenseFromBlob(ctx, fileManifest, blobSet)
	if readErr != nil {
		return nil, e.NewInternalError(readErr)
	}

	// 读取document
	var documentation string
	if documentBlob != nil {
		documentData, readErr := docsService.storageHelper.ReadBlob(ctx, documentBlob.Digest().Hex())
		if readErr != nil {
			return nil, e.NewInternalError(readErr)
		}
		documentation = string(documentData)
	}

	// 读取license
	var license string
	if licenseBlob != nil {
		licenceData, readErr := docsService.storageHelper.ReadBlob(ctx, documentBlob.Digest().Hex())
		if readErr != nil {
			return nil, e.NewInternalError(readErr)
		}
		license = string(licenceData)
	}

	// 获取documentation path
	paths, _ := fileManifest.PathsFor(documentBlob.Digest().String())
	documentPath := paths[0]

	return &registryv1alpha1.ModuleDocumentation{
		Documentation:     documentation,
		License:           license,
		DocumentationPath: documentPath,
	}, nil
}

func (docsService *DocsServiceImpl) GetPackageDocumentation(ctx context.Context, repositoryID, reference, packageName string) (*registryv1alpha1.PackageDocumentation, e.ResponseError) {
	// 查询reference对应的commit
	commit, err := docsService.commitMapper.FindByRepositoryIDAndReference(repositoryID, reference)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, e.NewNotFoundError(fmt.Errorf("repository %s", repositoryID))
		}

		return nil, e.NewInternalError(err)
	}

	// 获取文件清单
	fileManifest, blobSet, err := docsService.getManifestAndBlobSetByCommitID(ctx, commit.CommitID)
	if err != nil {
		return nil, e.NewInternalError(err)
	}

	identity, err := bufmoduleref.NewModuleIdentity(config.Properties.Server.ServerHost, commit.UserName, commit.RepositoryName)
	if err != nil {
		return nil, e.NewInternalError(err)
	}
	commitName := commit.CommitName

	// 获取bufConfig
	bufConfigBlob, configErr := docsService.storageHelper.GetBufManConfigFromBlob(ctx, fileManifest, blobSet)
	if configErr != nil {
		return nil, e.NewInternalError(configErr)
	}

	var dependentManifests []*manifest2.Manifest
	var dependentBlobSets []*manifest2.BlobSet
	var dependentIdentities []bufmoduleref.ModuleIdentity
	var dependentCommitNames []string
	if bufConfigBlob != nil {
		// 生成Config
		reader, configErr := bufConfigBlob.Open(ctx)
		if configErr != nil {
			return nil, e.NewInternalError(configErr)
		}
		defer reader.Close()
		configData, configErr := io.ReadAll(reader)
		if configErr != nil {
			return nil, e.NewInternalError(configErr)
		}
		bufConfig, configErr := bufconfig.GetConfigForData(ctx, configData)
		if configErr != nil {
			// 无法解析配置文件
			return nil, e.NewInternalError(configErr)
		}

		// 获取全部依赖commits
		dependentCommits, dependenceErr := docsService.resolver.GetAllDependenciesFromBufConfig(ctx, bufConfig)
		if dependenceErr != nil {
			return nil, e.NewInternalError(dependenceErr)
		}

		// 读取依赖文件
		dependentManifests = make([]*manifest2.Manifest, 0, len(dependentCommits))
		dependentBlobSets = make([]*manifest2.BlobSet, 0, len(dependentCommits))
		dependentIdentities = make([]bufmoduleref.ModuleIdentity, 0, len(dependentCommits))
		dependentCommitNames = make([]string, 0, len(dependentCommits))
		for i := 0; i < len(dependentCommits); i++ {
			dependentCommit := dependentCommits[i]
			dependentManifest, dependentBlobSet, getErr := docsService.getManifestAndBlobSetByCommitID(ctx, dependentCommit.CommitID)
			if getErr != nil {
				return nil, getErr
			}

			dependentIdentity, err := bufmoduleref.NewModuleIdentity(config.Properties.Server.ServerHost, dependentCommit.UserName, dependentCommit.RepositoryName)
			if err != nil {
				return nil, e.NewInternalError(err)
			}
			dependentIdentities = append(dependentIdentities, dependentIdentity)
			dependentCommitNames = append(dependentCommitNames, dependentCommit.CommitName)
			dependentManifests = append(dependentManifests, dependentManifest)
			dependentBlobSets = append(dependentBlobSets, dependentBlobSet)
		}
	}

	// 根据proto文件生成文档
	packageDocument, documentErr := docsService.protoParser.GetPackageDocumentation(ctx, packageName, identity, commitName, fileManifest, blobSet, dependentIdentities, dependentCommitNames, dependentManifests, dependentBlobSets)
	if err != nil {
		return nil, documentErr
	}

	return packageDocument, nil
}

// getDependentManifestsAndBlobSets 获取依赖的manifests和blob sets
func (docsService *DocsServiceImpl) getDependentManifestsAndBlobSets(ctx context.Context, fileManifest *manifest2.Manifest, blobSet *manifest2.BlobSet) ([]*manifest2.Manifest, []*manifest2.BlobSet, e.ResponseError) {
	// 获取bufConfig
	bufConfigBlob, configErr := docsService.storageHelper.GetBufManConfigFromBlob(ctx, fileManifest, blobSet)
	if configErr != nil {
		return nil, nil, e.NewInternalError(configErr)
	}

	var dependentManifests []*manifest2.Manifest
	var dependentBlobSets []*manifest2.BlobSet
	if bufConfigBlob != nil {
		// 生成Config
		reader, configErr := bufConfigBlob.Open(ctx)
		if configErr != nil {
			return nil, nil, e.NewInternalError(configErr)
		}
		defer reader.Close()
		configData, configErr := io.ReadAll(reader)
		if configErr != nil {
			return nil, nil, e.NewInternalError(configErr)
		}
		bufConfig, configErr := bufconfig.GetConfigForData(ctx, configData)
		if configErr != nil {
			// 无法解析配置文件
			return nil, nil, e.NewInternalError(configErr)
		}

		// 获取全部依赖commits
		dependentCommits, dependenceErr := docsService.resolver.GetAllDependenciesFromBufConfig(ctx, bufConfig)
		if dependenceErr != nil {
			return nil, nil, e.NewInternalError(dependenceErr)
		}

		// 读取依赖文件
		dependentManifests = make([]*manifest2.Manifest, 0, len(dependentCommits))
		dependentBlobSets = make([]*manifest2.BlobSet, 0, len(dependentCommits))
		for i := 0; i < len(dependentCommits); i++ {
			dependentCommit := dependentCommits[i]
			dependentManifest, dependentBlobSet, getErr := docsService.getManifestAndBlobSetByCommitID(ctx, dependentCommit.CommitID)
			if getErr != nil {
				return nil, nil, getErr
			}

			dependentManifests = append(dependentManifests, dependentManifest)
			dependentBlobSets = append(dependentBlobSets, dependentBlobSet)
		}
	}

	return dependentManifests, dependentBlobSets, nil
}

func (docsService *DocsServiceImpl) getManifestAndBlobSet(ctx context.Context, repositoryID string, reference string) (*manifest2.Manifest, *manifest2.BlobSet, e.ResponseError) {
	// 查询reference对应的commit
	commit, err := docsService.commitMapper.FindByRepositoryIDAndReference(repositoryID, reference)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil, e.NewNotFoundError(fmt.Errorf("repository %s", repositoryID))
		}

		return nil, nil, e.NewInternalError(err)
	}

	return docsService.getManifestAndBlobSetByCommitID(ctx, commit.CommitID)
}

func (docsService *DocsServiceImpl) getManifestAndBlobSetByCommitID(ctx context.Context, commitID string) (*manifest2.Manifest, *manifest2.BlobSet, e.ResponseError) {
	// 查询文件清单
	modelFileManifest, err := docsService.fileMapper.FindCommitManifestByCommitID(commitID)
	if err != nil {
		if err != nil {
			return nil, nil, e.NewInternalError(err)
		}
	}

	// 接着查询blobs
	fileBlobs, err := docsService.fileMapper.FindCommitFilesExceptManifestByCommitID(commitID)
	if err != nil {
		return nil, nil, e.NewInternalError(err)
	}

	// 读取
	fileManifest, blobSet, err := docsService.storageHelper.ReadToManifestAndBlobSet(ctx, modelFileManifest, fileBlobs)
	if err != nil {
		return nil, nil, e.NewInternalError(err)
	}

	return fileManifest, blobSet, nil
}
