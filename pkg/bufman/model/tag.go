/*
 * Licensed to the Apache Software Foundation (ASF) under one or more
 * contributor license agreements.  See the NOTICE file distributed with
 * this work for additional information regarding copyright ownership.
 * The ASF licenses this file to You under the Apache License, Version 2.0
 * (the "License"); you may not use this file except in compliance with
 * the License.  You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package model

import (
	"time"
)

import (
	"google.golang.org/protobuf/types/known/timestamppb"
)

import (
	registryv1alpha1 "github.com/apache/dubbo-kubernetes/pkg/bufman/gen/proto/go/registry/v1alpha1"
)

type Tag struct {
	ID           int64  `gorm:"primaryKey;autoIncrement"`
	UserID       string `gorm:"type:varchar(64)"`
	UserName     string `gorm:"type:varchar(200);not null"`
	RepositoryID string `gorm:"type:varchar(64)"`
	// RepositoryName string    `gorm:"type:varchar(200)"`
	CommitID    string    `gorm:"type:varchar(64)"`
	CommitName  string    `gorm:"type:varchar(64)"`
	TagID       string    `gorm:"type:varchar(64);unique;not null"`
	CreatedTime time.Time `gorm:"autoCreateTime"`
	TagName     string    `gorm:"type:varchar(20)"`
}

func (tag *Tag) TableName() string {
	return "tags"
}

func (tag *Tag) ToProtoRepositoryTag() *registryv1alpha1.RepositoryTag {
	if tag == nil {
		return (&Tag{}).ToProtoRepositoryTag()
	}

	return &registryv1alpha1.RepositoryTag{
		Id:         tag.TagID,
		CreateTime: timestamppb.New(tag.CreatedTime),
		Name:       tag.TagName,
		CommitName: tag.CommitName,
		Author:     tag.UserName,
	}
}

type Tags []*Tag

func (tags *Tags) ToProtoRepositoryTags() []*registryv1alpha1.RepositoryTag {
	repositoryTags := make([]*registryv1alpha1.RepositoryTag, len(*tags))
	for i := 0; i < len(*tags); i++ {
		repositoryTags[i] = (*tags)[i].ToProtoRepositoryTag()
	}

	return repositoryTags
}
