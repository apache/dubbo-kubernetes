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

package search

import (
	"context"

	"github.com/apache/dubbo-kubernetes/pkg/bufman/dal"
	registryv1alpha1 "github.com/apache/dubbo-kubernetes/pkg/bufman/gen/proto/go/registry/v1alpha1"
	"github.com/apache/dubbo-kubernetes/pkg/bufman/model"
)

type DBSearcherImpl struct {
}

func NewDBSearcher() *DBSearcherImpl {
	return &DBSearcherImpl{}
}

func (searcher *DBSearcherImpl) SearchUsers(ctx context.Context, query string, offset, limit int, reverse bool) (model.Users, error) {
	order := dal.User.ID.Asc()
	if reverse {
		order = dal.User.ID.Desc()
	}
	return dal.User.Where(dal.User.UserName.Like("%" + query + "%")).Or(dal.User.Description.Like("%" + query + "query")).Offset(offset).Limit(limit).Order(order).Find()
}

func (searcher *DBSearcherImpl) SearchRepositories(ctx context.Context, query string, offset, limit int, reverse bool) (model.Repositories, error) {
	order := dal.Repository.ID.Asc()
	if reverse {
		order = dal.Repository.ID.Desc()
	}
	return dal.Repository.Where(dal.Repository.RepositoryName.Like("%" + query + "%")).Or(dal.Repository.Description.Like("%" + query + "query")).
		Offset(offset).Limit(limit).Order(order).Find()
}

func (searcher *DBSearcherImpl) SearchCommitsByContent(ctx context.Context, userID, query string, offset, limit int, reverse bool) (model.Commits, error) {
	// search file content
	blobs, err := dal.FileBlob.Where(dal.FileBlob.Content.Like("%" + query + "%")).Find()
	if err != nil {
		return nil, err
	}
	digestSet := make(map[string]struct{})
	for _, blob := range blobs {
		digestSet[blob.Digest] = struct{}{}
	}
	digests := make([]string, len(digestSet))
	for digest := range digestSet {
		digests = append(digests, digest)
	}

	// search commit file by digest
	commitFiles, err := dal.CommitFile.Where(dal.CommitFile.Digest.In(digests...)).Find()
	if err != nil {
		return nil, err
	}
	commitIDSet := make(map[string]struct{})
	for _, commitFile := range commitFiles {
		commitIDSet[commitFile.CommitID] = struct{}{}
	}
	commitIDs := make([]string, len(commitIDSet))
	for commitID := range commitIDSet {
		commitIDs = append(commitIDs, commitID)
	}

	// search repository by userID
	repositories, err := dal.Repository.Where(dal.Repository.Visibility.Eq(uint8(registryv1alpha1.Visibility_VISIBILITY_PUBLIC))).Or(dal.Repository.UserID.Eq(userID)).Find()
	if err != nil {
		return nil, err
	}
	repositoryIDSet := make(map[string]struct{})
	for _, repository := range repositories {
		repositoryIDSet[repository.RepositoryID] = struct{}{}
	}
	repositoryIDs := make([]string, 0, len(repositoryIDSet))
	for repositoryID := range repositoryIDSet {
		repositoryIDs = append(repositoryIDs, repositoryID)
	}

	// search commit by commitID
	order := dal.Commit.ID.Asc()
	if reverse {
		order = dal.Commit.ID.Desc()
	}

	return dal.Commit.Where(dal.Commit.RepositoryID.In(repositoryIDs...), dal.Commit.CommitID.In(commitIDs...), dal.Commit.DraftName.Eq("")).Offset(offset).Limit(limit).Order(order).Find()
}

func (searcher *DBSearcherImpl) SearchTag(ctx context.Context, repositoryID string, query string, offset, limit int, reverse bool) (model.Tags, error) {
	order := dal.Tag.ID.Asc()
	if reverse {
		order = dal.Tag.ID.Desc()
	}
	return dal.Tag.Where(dal.Repository.RepositoryID.Eq(repositoryID), dal.Tag.TagName.Like("%"+query+"%")).Offset(offset).Limit(limit).Order(order).Find()
}

func (searcher *DBSearcherImpl) SearchDraft(ctx context.Context, repositoryID string, query string, offset, limit int, reverse bool) (model.Commits, error) {
	order := dal.Commit.ID.Asc()
	if reverse {
		order = dal.Commit.ID.Desc()
	}
	return dal.Commit.Where(dal.Repository.RepositoryID.Eq(repositoryID), dal.Commit.DraftName.Like("%"+query+"%")).Offset(offset).Limit(limit).Order(order).Find()
}
