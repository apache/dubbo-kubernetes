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
	"github.com/google/uuid"

	"gorm.io/gorm"
)

import (
	"github.com/apache/dubbo-kubernetes/pkg/bufman/core/validity"
	"github.com/apache/dubbo-kubernetes/pkg/bufman/e"
	"github.com/apache/dubbo-kubernetes/pkg/bufman/mapper"
	"github.com/apache/dubbo-kubernetes/pkg/bufman/model"
)

type TagService interface {
	CreateRepositoryTag(ctx context.Context, repositoryID, TagName, commitName string) (*model.Tag, e.ResponseError)
	ListRepositoryTags(ctx context.Context, repositoryID string, offset, limit int, reverse bool) (model.Tags, e.ResponseError)
}

func NewTagService() TagService {
	return &TagServiceImpl{
		repositoryMapper: &mapper.RepositoryMapperImpl{},
		commitMapper:     &mapper.CommitMapperImpl{},
		tagMapper:        &mapper.TagMapperImpl{},
		validator:        validity.NewValidator(),
	}
}

type TagServiceImpl struct {
	repositoryMapper mapper.RepositoryMapper
	commitMapper     mapper.CommitMapper
	tagMapper        mapper.TagMapper
	validator        validity.Validator
}

func (tagService *TagServiceImpl) CreateRepositoryTag(ctx context.Context, repositoryID, TagName, commitName string) (*model.Tag, e.ResponseError) {
	// 查询commitName
	commit, err := tagService.commitMapper.FindByRepositoryIDAndCommitName(repositoryID, commitName)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, e.NewNotFoundError(err)
		}
	}

	tag := &model.Tag{
		UserID:       commit.UserID,
		UserName:     commit.UserName,
		RepositoryID: repositoryID,
		CommitID:     commit.CommitID,
		CommitName:   commitName,
		TagID:        uuid.NewString(),
		TagName:      TagName,
	}
	err = tagService.tagMapper.Create(tag)
	if err != nil {
		return nil, e.NewInternalError(err)
	}

	return tag, nil
}

func (tagService *TagServiceImpl) ListRepositoryTags(ctx context.Context, repositoryID string, offset, limit int, reverse bool) (model.Tags, e.ResponseError) {
	tags, err := tagService.tagMapper.FindPageByRepositoryID(repositoryID, limit, offset, reverse)
	if err != nil {
		return nil, e.NewInternalError(err)
	}

	return tags, nil
}
