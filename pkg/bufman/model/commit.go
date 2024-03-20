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
	"github.com/apache/dubbo-kubernetes/pkg/bufman/config"
	"github.com/apache/dubbo-kubernetes/pkg/bufman/constant"
	modulev1alpha1 "github.com/apache/dubbo-kubernetes/pkg/bufman/gen/proto/go/module/v1alpha1"
	registryv1alpha1 "github.com/apache/dubbo-kubernetes/pkg/bufman/gen/proto/go/registry/v1alpha1"
	"github.com/apache/dubbo-kubernetes/pkg/bufman/pkg/manifest"
)

type Commit struct {
	ID                 int64     `gorm:"primaryKey;autoIncrement"`
	UserID             string    `gorm:"type:varchar(64);"`
	UserName           string    `gorm:"type:varchar(200);not null"`
	RepositoryID       string    `gorm:"type:varchar(64)"`
	RepositoryName     string    `gorm:"type:varchar(200)"`
	CommitID           string    `gorm:"type:varchar(64);unique;not null"`
	CommitName         string    `gorm:"type:varchar(64);unique"`
	DraftName          string    `gorm:"type:varchar(20)"`
	CreatedTime        time.Time `gorm:"autoCreateTime"`
	ManifestDigest     string    `gorm:"type:string;"`
	BufManConfigDigest string    `gorm:"not null"` // bufman配置文件digest
	DocumentDigest     string    // README文档digest
	LicenseDigest      string    // README文档digest

	SequenceID int64

	// 清单文件
	CommitManifest *CommitFile `gorm:"-"`
	// 其他所有文件
	CommitFiles CommitFiles `gorm:"-"`
	// 关联的tag
	Tags Tags `gorm:"-"`
}

func (commit *Commit) TableName() string {
	return "commits"
}

func (commit *Commit) ToProtoLocalModulePin() *registryv1alpha1.LocalModulePin {
	if commit == nil {
		return (&Commit{}).ToProtoLocalModulePin()
	}

	modulePin := &registryv1alpha1.LocalModulePin{
		Owner:          commit.UserName,
		Repository:     commit.RepositoryName,
		Commit:         commit.CommitName,
		CreateTime:     timestamppb.New(commit.CreatedTime),
		ManifestDigest: string(manifest.DigestTypeShake256) + ":" + commit.ManifestDigest,
	}

	if commit.DraftName == "" {
		modulePin.Branch = constant.DefaultBranch
	}

	if commit.DraftName != "" {
		modulePin.DraftName = commit.DraftName
	}

	return modulePin
}

func (commit *Commit) ToProtoModulePin() *modulev1alpha1.ModulePin {
	if commit == nil {
		return (&Commit{}).ToProtoModulePin()
	}

	modulePin := &modulev1alpha1.ModulePin{
		Remote:         config.Properties.Server.ServerHost,
		Owner:          commit.UserName,
		Repository:     commit.RepositoryName,
		Commit:         commit.CommitName,
		CreateTime:     timestamppb.New(commit.CreatedTime),
		ManifestDigest: string(manifest.DigestTypeShake256) + ":" + commit.ManifestDigest,
	}

	return modulePin
}

func (commit *Commit) ToProtoRepositoryCommit() *registryv1alpha1.RepositoryCommit {
	if commit == nil {
		return (&Commit{}).ToProtoRepositoryCommit()
	}

	repositoryCommit := &registryv1alpha1.RepositoryCommit{
		Id:               commit.CommitID,
		CreateTime:       timestamppb.New(commit.CreatedTime),
		Name:             commit.CommitName,
		CommitSequenceId: commit.SequenceID,
		Author:           commit.UserName,
		ManifestDigest:   string(manifest.DigestTypeShake256) + ":" + commit.ManifestDigest,
	}

	if commit.DraftName != "" {
		repositoryCommit.DraftName = commit.DraftName
	}

	if len(commit.Tags) > 0 {
		repositoryCommit.Tags = commit.Tags.ToProtoRepositoryTags()
	}

	return repositoryCommit
}

func (commit *Commit) ToProtoSearchResult() *registryv1alpha1.CommitSearchResult {
	if commit == nil {
		return (&Commit{}).ToProtoSearchResult()
	}

	return &registryv1alpha1.CommitSearchResult{
		Id:             commit.CommitID,
		Name:           commit.CommitName,
		Owner:          commit.UserName,
		RepositoryName: commit.RepositoryName,
	}
}

type Commits []*Commit

func (commits *Commits) ToProtoRepositoryCommits() []*registryv1alpha1.RepositoryCommit {
	repositoryCommits := make([]*registryv1alpha1.RepositoryCommit, len(*commits))
	for i := 0; i < len(*commits); i++ {
		repositoryCommits[i] = (*commits)[i].ToProtoRepositoryCommit()
	}

	return repositoryCommits
}

func (commits *Commits) ToProtoModulePins() []*modulev1alpha1.ModulePin {
	modulePins := make([]*modulev1alpha1.ModulePin, len(*commits))
	for i := 0; i < len(*commits); i++ {
		modulePins[i] = (*commits)[i].ToProtoModulePin()
	}

	return modulePins
}

func (commits *Commits) ToProtoSearchResults() []*registryv1alpha1.CommitSearchResult {
	commitSearchResults := make([]*registryv1alpha1.CommitSearchResult, len(*commits))
	for i := 0; i < len(*commits); i++ {
		commitSearchResults[i] = (*commits)[i].ToProtoSearchResult()
	}

	return commitSearchResults
}
