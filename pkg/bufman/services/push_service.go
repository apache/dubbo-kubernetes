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
	"time"

	"github.com/apache/dubbo-kubernetes/pkg/bufman/core/security"
	"github.com/apache/dubbo-kubernetes/pkg/bufman/core/storage"
	"github.com/apache/dubbo-kubernetes/pkg/bufman/e"
	"github.com/apache/dubbo-kubernetes/pkg/bufman/mapper"
	"github.com/apache/dubbo-kubernetes/pkg/bufman/model"
	manifest2 "github.com/apache/dubbo-kubernetes/pkg/bufman/pkg/manifest"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type PushService interface {
	PushManifestAndBlobs(ctx context.Context, userID, ownerName, repositoryName string, fileManifest *manifest2.Manifest, fileBlobs *manifest2.BlobSet) (*model.Commit, e.ResponseError)
	PushManifestAndBlobsWithTags(ctx context.Context, userID, ownerName, repositoryName string, fileManifest *manifest2.Manifest, fileBlobs *manifest2.BlobSet, tagNames []string) (*model.Commit, e.ResponseError)
	PushManifestAndBlobsWithDraft(ctx context.Context, userID, ownerName, repositoryName string, fileManifest *manifest2.Manifest, fileBlobs *manifest2.BlobSet, draftName string) (*model.Commit, e.ResponseError)
	GetManifestAndBlobSet(ctx context.Context, repositoryID string, reference string) (*manifest2.Manifest, *manifest2.BlobSet, e.ResponseError)
}

type PushServiceImpl struct {
	userMapper       mapper.UserMapper
	repositoryMapper mapper.RepositoryMapper
	fileMapper       mapper.FileMapper
	commitMapper     mapper.CommitMapper
	storageHelper    storage.StorageHelper
}

func NewPushService() PushService {
	return &PushServiceImpl{
		userMapper:       &mapper.UserMapperImpl{},
		repositoryMapper: &mapper.RepositoryMapperImpl{},
		commitMapper:     &mapper.CommitMapperImpl{},
		fileMapper:       &mapper.FileMapperImpl{},
		storageHelper:    storage.NewStorageHelper(),
	}
}

func (pushService *PushServiceImpl) GetManifestAndBlobSet(ctx context.Context, repositoryID string, reference string) (*manifest2.Manifest, *manifest2.BlobSet, e.ResponseError) {
	// 查询reference对应的commit
	commit, err := pushService.commitMapper.FindByRepositoryIDAndReference(repositoryID, reference)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil, e.NewNotFoundError(fmt.Errorf("repository %s", repositoryID))
		}

		return nil, nil, e.NewInternalError(err)
	}

	// 查询文件清单
	modelFileManifest, err := pushService.fileMapper.FindManifestByCommitID(commit.CommitID)
	if err != nil {
		if err != nil {
			return nil, nil, e.NewInternalError(err)
		}
	}

	// 接着查询blobs
	fileBlobs, err := pushService.fileMapper.FindAllBlobsByCommitID(commit.CommitID)
	if err != nil {
		return nil, nil, e.NewInternalError(err)
	}

	// 读取
	fileManifest, blobSet, err := pushService.storageHelper.ReadToManifestAndBlobSet(ctx, modelFileManifest, fileBlobs)
	if err != nil {
		return nil, nil, e.NewInternalError(err)
	}

	return fileManifest, blobSet, nil
}

func (pushService *PushServiceImpl) PushManifestAndBlobs(ctx context.Context, userID, ownerName, repositoryName string, fileManifest *manifest2.Manifest, fileBlobs *manifest2.BlobSet) (*model.Commit, e.ResponseError) {
	commit, err := pushService.toCommit(ctx, userID, ownerName, repositoryName, fileManifest, fileBlobs)
	if err != nil {
		return nil, err
	}

	// 写入文件
	err = pushService.saveFileManifestAndBlobs(ctx, commit)
	if err != nil {
		return nil, err
	}

	// 写入数据库
	createErr := pushService.commitMapper.Create(commit)
	if createErr != nil {
		if errors.Is(createErr, gorm.ErrDuplicatedKey) {
			return nil, e.NewInternalError(createErr)
		}
		if errors.Is(createErr, mapper.ErrLastCommitDuplicated) {
			return nil, e.NewAlreadyExistsError(createErr)
		}

		return nil, e.NewInternalError(createErr)
	}

	return commit, nil
}

func (pushService *PushServiceImpl) PushManifestAndBlobsWithTags(ctx context.Context, userID, ownerName, repositoryName string, fileManifest *manifest2.Manifest, fileBlobs *manifest2.BlobSet, tagNames []string) (*model.Commit, e.ResponseError) {
	commit, err := pushService.toCommit(ctx, userID, ownerName, repositoryName, fileManifest, fileBlobs)
	if err != nil {
		return nil, err
	}

	// 生成tags
	var tags []*model.Tag
	for i := 0; i < len(tagNames); i++ {
		tags = append(tags, &model.Tag{
			UserID:       commit.UserID,
			RepositoryID: commit.RepositoryID,
			CommitID:     commit.CommitID,
			TagID:        uuid.NewString(),
			TagName:      tagNames[i],
		})
	}
	commit.Tags = tags

	// 写入文件
	err = pushService.saveFileManifestAndBlobs(ctx, commit)
	if err != nil {
		return nil, err
	}

	createErr := pushService.commitMapper.Create(commit)
	if createErr != nil {
		if errors.Is(createErr, mapper.ErrTagAndDraftDuplicated) || errors.Is(createErr, gorm.ErrDuplicatedKey) {
			return nil, e.NewInternalError(createErr)
		}
		if errors.Is(createErr, mapper.ErrLastCommitDuplicated) {
			return nil, e.NewAlreadyExistsError(createErr)
		}

		return nil, e.NewInternalError(createErr)
	}

	return commit, nil
}

func (pushService *PushServiceImpl) PushManifestAndBlobsWithDraft(ctx context.Context, userID, ownerName, repositoryName string, fileManifest *manifest2.Manifest, fileBlobs *manifest2.BlobSet, draftName string) (*model.Commit, e.ResponseError) {
	commit, err := pushService.toCommit(ctx, userID, ownerName, repositoryName, fileManifest, fileBlobs)
	if err != nil {
		return nil, err
	}
	commit.DraftName = draftName

	// 写入文件
	err = pushService.saveFileManifestAndBlobs(ctx, commit)
	if err != nil {
		return nil, err
	}

	createErr := pushService.commitMapper.Create(commit)
	if createErr != nil {
		if errors.Is(createErr, mapper.ErrTagAndDraftDuplicated) {
			return nil, e.NewInternalError(createErr)
		}
		if errors.Is(createErr, mapper.ErrLastCommitDuplicated) {
			return nil, e.NewAlreadyExistsError(createErr)
		}

		return nil, e.NewInternalError(createErr)
	}

	return commit, nil
}

func (pushService *PushServiceImpl) toCommit(ctx context.Context, userID, ownerName, repositoryName string, fileManifest *manifest2.Manifest, fileBlobs *manifest2.BlobSet) (*model.Commit, e.ResponseError) {
	// 获取user
	user, err := pushService.userMapper.FindByUserID(userID)
	if err != nil || user.UserName != ownerName {
		return nil, e.NewPermissionDeniedError(err)
	}

	// 获取repo
	repository, err := pushService.repositoryMapper.FindByUserNameAndRepositoryName(ownerName, repositoryName)
	if err != nil {
		return nil, e.NewNotFoundError(err)
	}

	commitID := uuid.NewString()
	commitName := security.GenerateCommitName(user.UserName, repositoryName)
	createTime := time.Now()

	// 生成file blobs
	modelBlobs := make([]*model.FileBlob, 0, len(fileManifest.Paths()))
	err = fileManifest.Range(func(path string, digest manifest2.Digest) error {
		// 读取文件内容
		blob, ok := fileBlobs.BlobFor(digest.String())
		if !ok {
			return e.NewInvalidArgumentError(fmt.Errorf("blob is not valid"))
		}

		readCloser, err := blob.Open(ctx)
		if err != nil {
			return e.NewInternalError(err)
		}

		content, err := io.ReadAll(readCloser)
		if err != nil {
			return e.NewInternalError(err)
		}

		modelBlobs = append(modelBlobs, &model.FileBlob{
			Digest:         digest.Hex(),
			CommitID:       commitID,
			FileName:       path,
			Content:        string(content),
			UserID:         user.UserID,
			UserName:       user.UserName,
			RepositoryID:   repository.RepositoryID,
			RepositoryName: repository.RepositoryName,
			CommitName:     commitName,
			CreatedTime:    createTime,
		})
		return nil
	})
	if err != nil {
		return nil, e.NewInternalError(err)
	}

	// 生成manifest
	fileManifestBlob, err := fileManifest.Blob()
	if err != nil {
		return nil, e.NewInternalError(err)
	}
	readCloser, err := fileManifestBlob.Open(ctx)
	if err != nil {
		return nil, e.NewInternalError(err)
	}
	content, err := io.ReadAll(readCloser)
	if err != nil {
		return nil, e.NewInternalError(err)
	}
	modelFileManifest := &model.FileManifest{
		ID:             0,
		Digest:         fileManifestBlob.Digest().Hex(),
		CommitID:       commitID,
		Content:        string(content),
		UserID:         user.UserID,
		UserName:       user.UserName,
		RepositoryID:   repository.RepositoryID,
		RepositoryName: repository.RepositoryName,
		CommitName:     commitName,
		CreatedTime:    createTime,
	}

	// 获取bufman config blob
	configBlob, err := pushService.storageHelper.GetBufManConfigFromBlob(ctx, fileManifest, fileBlobs)
	if err != nil {
		return nil, e.NewInternalError(err)
	}
	// 获取README LICENSE
	documentBlob, licenseBlob, err := pushService.storageHelper.GetDocumentAndLicenseFromBlob(ctx, fileManifest, fileBlobs)
	if err != nil {
		return nil, e.NewInternalError(err)
	}

	commit := &model.Commit{
		UserID:         user.UserID,
		UserName:       user.UserName,
		RepositoryID:   repository.RepositoryID,
		RepositoryName: repositoryName,
		CommitID:       commitID,
		CommitName:     commitName,
		CreatedTime:    createTime,
		ManifestDigest: fileManifestBlob.Digest().Hex(),
		SequenceID:     0,
		FileManifest:   modelFileManifest,
		FileBlobs:      modelBlobs,
	}
	if configBlob != nil {
		commit.BufManConfigDigest = configBlob.Digest().Hex()
	}
	if documentBlob != nil {
		commit.DocumentDigest = documentBlob.Digest().Hex()
	}
	if licenseBlob != nil {
		commit.LicenseDigest = licenseBlob.Digest().Hex()
	}

	return commit, nil
}

func (pushService *PushServiceImpl) saveFileManifestAndBlobs(ctx context.Context, commit *model.Commit) e.ResponseError {
	// 保存file blobs
	for i := 0; i < len(commit.FileBlobs); i++ {
		fileBlob := commit.FileBlobs[i]

		// 如果是README文件
		if fileBlob.Digest == commit.DocumentDigest {
			err := pushService.storageHelper.StoreDocumentation(ctx, fileBlob)
			if err != nil {
				return e.NewInternalError(err)
			}

		}

		// 普通文件
		err := pushService.storageHelper.StoreBlob(ctx, fileBlob)
		if err != nil {
			return e.NewInternalError(err)
		}
	}

	// 保存file manifest
	err := pushService.storageHelper.StoreManifest(ctx, commit.FileManifest)
	if err != nil {
		return e.NewInternalError(err)
	}

	return nil
}
