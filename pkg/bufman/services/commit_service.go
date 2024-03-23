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
)

import (
	"gorm.io/gorm"
)

import (
	"github.com/apache/dubbo-kubernetes/pkg/bufman/e"
	"github.com/apache/dubbo-kubernetes/pkg/bufman/mapper"
	"github.com/apache/dubbo-kubernetes/pkg/bufman/model"
)

type CommitService interface {
	ListRepositoryCommitsByReference(ctx context.Context, repositoryID, reference string, offset, limit int, reverse bool) (model.Commits, e.ResponseError)
	GetRepositoryCommitByReference(ctx context.Context, repositoryID, reference string) (*model.Commit, e.ResponseError)
	ListRepositoryDraftCommits(ctx context.Context, repositoryID string, offset, limit int, reverse bool) (model.Commits, e.ResponseError)
	DeleteRepositoryDraftCommit(ctx context.Context, repositoryID, draftName string) e.ResponseError
}

type CommitServiceImpl struct {
	repositoryMapper mapper.RepositoryMapper
	commitMapper     mapper.CommitMapper
}

func NewCommitService() CommitService {
	return &CommitServiceImpl{
		repositoryMapper: &mapper.RepositoryMapperImpl{},
		commitMapper:     &mapper.CommitMapperImpl{},
	}
}

func (commitService *CommitServiceImpl) ListRepositoryCommitsByReference(ctx context.Context, repositoryID, reference string, offset, limit int, reverse bool) (model.Commits, e.ResponseError) {
	// 查询commits
	commits, err := commitService.commitMapper.FindPageByRepositoryIDAndReference(repositoryID, reference, offset, limit, reverse)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, e.NewNotFoundError(err)
		}
		return nil, e.NewInternalError(err)
	}

	return commits, nil
}

func (commitService *CommitServiceImpl) GetRepositoryCommitByReference(ctx context.Context, repositoryID, reference string) (*model.Commit, e.ResponseError) {
	// 查询commit
	commit, err := commitService.commitMapper.FindByRepositoryIDAndReference(repositoryID, reference)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, e.NewNotFoundError(err)
		}

		return nil, e.NewInternalError(err)
	}

	return commit, nil
}

func (commitService *CommitServiceImpl) ListRepositoryDraftCommits(ctx context.Context, repositoryID string, offset, limit int, reverse bool) (model.Commits, e.ResponseError) {
	var commits model.Commits
	var err error
	// 查询draft
	commits, err = commitService.commitMapper.FindDraftPageByRepositoryID(repositoryID, offset, limit, reverse)
	if err != nil {
		return nil, e.NewInternalError(err)
	}

	return commits, nil
}

func (commitService *CommitServiceImpl) DeleteRepositoryDraftCommit(ctx context.Context, repositoryID, draftName string) e.ResponseError {
	// 删除
	err := commitService.commitMapper.DeleteByRepositoryIDAndDraftName(repositoryID, draftName)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return e.NewNotFoundError(err)
		}

		return e.NewInternalError(err)
	}

	return nil
}
