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

	"github.com/apache/dubbo-kubernetes/pkg/bufman/core/storage"
	"github.com/apache/dubbo-kubernetes/pkg/bufman/e"
	"github.com/apache/dubbo-kubernetes/pkg/bufman/mapper"
	manifest2 "github.com/apache/dubbo-kubernetes/pkg/bufman/pkg/manifest"
	"gorm.io/gorm"
)

type DownloadService interface {
	DownloadManifestAndBlobs(ctx context.Context, registerID string, reference string) (*manifest2.Manifest, *manifest2.BlobSet, e.ResponseError)
}

type DownloadServiceImpl struct {
	commitMapper  mapper.CommitMapper
	fileMapper    mapper.FileMapper
	storageHelper storage.StorageHelper
}

func NewDownloadService() DownloadService {
	return &DownloadServiceImpl{
		commitMapper:  &mapper.CommitMapperImpl{},
		fileMapper:    &mapper.FileMapperImpl{},
		storageHelper: storage.NewStorageHelper(),
	}
}

func (downloadService *DownloadServiceImpl) DownloadManifestAndBlobs(ctx context.Context, registerID string, reference string) (*manifest2.Manifest, *manifest2.BlobSet, e.ResponseError) {
	// 查询reference对应的commit
	commit, err := downloadService.commitMapper.FindByRepositoryIDAndReference(registerID, reference)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil, e.NewNotFoundError(fmt.Errorf("reference %s", reference))
		}

		return nil, nil, e.NewInternalError(err)
	}

	// 查询文件清单
	modelFileManifest, err := downloadService.fileMapper.FindManifestByCommitID(commit.CommitID)
	if err != nil {
		if err != nil {
			return nil, nil, e.NewInternalError(err)
		}
	}

	// 接着查询blobs
	fileBlobs, err := downloadService.fileMapper.FindAllBlobsByCommitID(commit.CommitID)
	if err != nil {
		return nil, nil, e.NewInternalError(err)
	}

	// 读取
	fileManifest, blobSet, err := downloadService.storageHelper.ReadToManifestAndBlobSet(ctx, modelFileManifest, fileBlobs)
	if err != nil {
		return nil, nil, e.NewInternalError(err)
	}

	return fileManifest, blobSet, nil
}
