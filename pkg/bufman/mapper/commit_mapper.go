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

package mapper

import (
	"errors"

	"github.com/apache/dubbo-kubernetes/pkg/bufman/constant"
	"github.com/apache/dubbo-kubernetes/pkg/bufman/dal"
	"github.com/apache/dubbo-kubernetes/pkg/bufman/model"
	"gorm.io/gorm"
)

type CommitMapper interface {
	Create(commit *model.Commit) error
	GetDraftCountsByRepositoryID(repositoryID string) (int64, error)
	FindLastByRepositoryID(repositoryID string) (*model.Commit, error)
	FindByRepositoryIDAndCommitName(repositoryID string, commitName string) (*model.Commit, error)
	FindByRepositoryIDAndTagName(repositoryID string, tagName string) (*model.Commit, error)
	FindByRepositoryIDAndDraftName(repositoryID string, draftName string) (*model.Commit, error)
	FindByRepositoryIDAndReference(repositoryID string, reference string) (*model.Commit, error)
	FindPageByRepositoryID(repositoryID string, offset, limit int, reverse bool) (model.Commits, error)
	FindPageByRepositoryIDAndDraftName(repositoryID, draftName string, offset, limit int, reverse bool) (model.Commits, error)
	FindPageByRepositoryIDAndTagName(repositoryID string, tagName string, offset, limit int, reverse bool) (model.Commits, error)
	FindPageByRepositoryIDAndCommitName(repositoryID string, commitName string, offset, limit int, reverse bool) (model.Commits, error)
	FindPageByRepositoryIDAndReference(repositoryID string, reference string, offset, limit int, reverse bool) (model.Commits, error)
	FindDraftPageByRepositoryID(repositoryID string, offset, limit int, reverse bool) (model.Commits, error)
	FindDraftPageByRepositoryIDAndQuery(repositoryID, query string, offset, limit int, reverse bool) (model.Commits, error)
	DeleteByRepositoryIDAndDraftName(repositoryID string, draftName string) error
}

type CommitMapperImpl struct{}

var (
	ErrTagAndDraftDuplicated = errors.New("tag and draft duplicated")
	ErrLastCommitDuplicated  = errors.New("same commit compared to las commit")
)

func (c *CommitMapperImpl) Create(commit *model.Commit) error {
	return dal.Q.Transaction(func(tx *dal.Query) error {
		// 检查与上次提交的是否相同
		lastCommit, err := tx.Commit.Where(tx.Commit.RepositoryID.Eq(commit.RepositoryID)).Last()
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		if lastCommit != nil && lastCommit.ManifestDigest == commit.ManifestDigest {
			return ErrLastCommitDuplicated
		}

		// 检查tag和draft是否冲突
		if len(commit.Tags) > 0 {
			tagNames := make([]string, len(commit.Tags))
			for i := 0; i < len(commit.Tags); i++ {
				tagNames[i] = commit.Tags[i].TagName
			}
			_, err = tx.Commit.Where(tx.Commit.RepositoryID.Eq(commit.RepositoryID), tx.Commit.DraftName.In(tagNames...)).First()
			if err == nil || !errors.Is(err, gorm.ErrRecordNotFound) {
				// 冲突
				return ErrTagAndDraftDuplicated
			}
		}

		// 存储
		err = tx.Commit.Create(commit)
		if err != nil {
			return err
		}

		// 查询sequence id
		sequenceID, err := c.FindSequenceID(commit)
		if err != nil {
			return err
		}
		commit.SequenceID = sequenceID

		return tx.Commit.Save(commit)
	})
}

func (c *CommitMapperImpl) GetDraftCountsByRepositoryID(repositoryID string) (int64, error) {
	return dal.Commit.Where(dal.Commit.CommitID.Eq(repositoryID), dal.Commit.DraftName.Neq("")).Count()
}

func (c *CommitMapperImpl) FindLastByRepositoryID(repositoryID string) (*model.Commit, error) {
	return dal.Commit.Where(dal.Commit.RepositoryID.Eq(repositoryID), dal.Commit.DraftName.Eq("")).Last()
}

func (c *CommitMapperImpl) FindByRepositoryIDAndCommitName(repositoryID string, commitName string) (*model.Commit, error) {
	return dal.Commit.Where(dal.Commit.RepositoryID.Eq(repositoryID), dal.Commit.CommitName.Eq(commitName)).First()
}

func (c *CommitMapperImpl) FindByRepositoryIDAndTagName(repositoryID string, tagName string) (*model.Commit, error) {
	tag := &model.Tag{}
	commit := &model.Commit{}
	err := dal.Q.Transaction(func(tx *dal.Query) error {
		// 查询tag
		var err error
		tag, err = tx.Tag.Where(tx.Tag.RepositoryID.Eq(repositoryID), tx.Tag.TagName.Eq(tagName)).Last()
		if err != nil {
			return err
		}

		commit, err = tx.Commit.Where(tx.Commit.CommitID.Eq(tag.CommitID)).First()
		if err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return commit, nil
}

func (c *CommitMapperImpl) FindByRepositoryIDAndDraftName(repositoryID string, draftName string) (*model.Commit, error) {
	return dal.Commit.Where(dal.Commit.RepositoryID.Eq(repositoryID), dal.Commit.DraftName.Eq(draftName)).Last()
}

func (c *CommitMapperImpl) FindByRepositoryIDAndReference(repositoryID string, reference string) (*model.Commit, error) {
	var commit *model.Commit
	var err error
	if reference == "" || reference == constant.DefaultBranch {
		commit, err = c.FindLastByRepositoryID(repositoryID)
	} else if len(reference) == constant.CommitLength {
		// 查询commit
		commit, err = c.FindByRepositoryIDAndCommitName(repositoryID, reference)
		if err != nil {
			return nil, err
		}
	}

	if commit != nil && err == nil {
		return commit, nil
	}

	// 查询tag
	commit, err = c.FindByRepositoryIDAndTagName(repositoryID, reference)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	if err != nil {
		// 查询draft
		commit, err = c.FindByRepositoryIDAndDraftName(repositoryID, reference)
		if err != nil {
			return nil, err
		}
	}

	return commit, nil
}

func (c *CommitMapperImpl) FindByRepositoryNameAndReference(repositoryID string, reference string) (*model.Commit, error) {
	if reference == "" || reference == constant.DefaultBranch {
		commit, err := c.FindLastByRepositoryID(repositoryID)
		if err != nil {
			return nil, err
		}

		return commit, nil
	} else if len(reference) == constant.CommitLength {
		// 查询commit
		commit, err := c.FindByRepositoryIDAndCommitName(repositoryID, reference)
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, err
		}

		if commit != nil {
			return commit, nil
		}
	}

	// 查询tag
	commit, err := c.FindByRepositoryIDAndTagName(repositoryID, reference)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	if err != nil {
		// 查询draft
		commit, err = c.FindByRepositoryIDAndDraftName(repositoryID, reference)
		if err != nil {
			return nil, err
		}
	}

	return commit, nil
}

func (c *CommitMapperImpl) FindPageByRepositoryID(repositoryID string, offset, limit int, reverse bool) (model.Commits, error) {
	stmt := dal.Commit.Where(dal.Commit.RepositoryID.Eq(repositoryID), dal.Commit.DraftName.Eq("")).Offset(offset).Limit(limit)
	if reverse {
		stmt = stmt.Order(dal.Commit.SequenceID.Desc())
	} else {
		stmt = stmt.Order(dal.Commit.SequenceID)
	}

	return stmt.Find()
}

func (c *CommitMapperImpl) FindPageByRepositoryIDAndDraftName(repositoryID, draftName string, offset, limit int, reverse bool) (model.Commits, error) {
	stmt := dal.Commit.Where(dal.Commit.RepositoryID.Eq(repositoryID), dal.Commit.DraftName.Eq(draftName)).Offset(offset).Limit(limit)
	if reverse {
		stmt = stmt.Order(dal.Commit.ID.Desc())
	}

	return stmt.Find()
}

func (c *CommitMapperImpl) FindPageByRepositoryIDAndCommitName(repositoryID string, commitName string, offset, limit int, reverse bool) (model.Commits, error) {
	var commits model.Commits
	err := dal.Q.Transaction(func(tx *dal.Query) error {
		// 查询commit name对应的sequence id
		commit, err := tx.Commit.Where(tx.Commit.RepositoryID.Eq(repositoryID), tx.Commit.CommitName.Eq(commitName)).Last()
		if err != nil {
			return err
		}

		// 查询commits
		stmt := tx.Commit.Where(tx.Commit.RepositoryID.Eq(repositoryID), tx.Commit.DraftName.Eq(""), tx.Commit.SequenceID.Lte(commit.SequenceID)).Offset(offset).Limit(limit)
		if reverse {
			stmt.Order(tx.Commit.SequenceID.Desc())
		}
		commits, err = stmt.Find()
		if err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return commits, nil
}

func (c *CommitMapperImpl) FindPageByRepositoryIDAndTagName(repositoryID string, tagName string, offset, limit int, reverse bool) (model.Commits, error) {
	var commits model.Commits
	err := dal.Q.Transaction(func(tx *dal.Query) error {
		// 查询tag
		tag, err := tx.Tag.Where(tx.Tag.RepositoryID.Eq(repositoryID), tx.Tag.TagName.Eq(tagName)).Last()
		if err != nil {
			return err
		}

		// 查询commits
		commits, err = c.FindPageByRepositoryIDAndCommitName(repositoryID, tag.CommitName, offset, limit, reverse)
		if err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return commits, nil
}

func (c *CommitMapperImpl) FindPageByRepositoryIDAndReference(repositoryID string, reference string, offset, limit int, reverse bool) (model.Commits, error) {
	var commits model.Commits
	var err error
	if reference == "" || reference == constant.DefaultBranch {
		commits, err = c.FindPageByRepositoryID(repositoryID, offset, limit, reverse)
	} else if len(reference) == constant.CommitLength {
		commits, err = c.FindPageByRepositoryIDAndCommitName(repositoryID, reference, offset, limit, reverse)
	} else {
		commits, err = c.FindPageByRepositoryIDAndTagName(repositoryID, reference, offset, limit, reverse)
	}

	if err != nil {
		return nil, err
	}

	if len(commits) == 0 {
		// 查询drafts
		commits, err = c.FindPageByRepositoryIDAndDraftName(repositoryID, reference, offset, limit, reverse)
		if err != nil {
			return nil, err
		}
	}

	return commits, nil
}

func (c *CommitMapperImpl) FindDraftPageByRepositoryID(repositoryID string, offset, limit int, reverse bool) (model.Commits, error) {
	stmt := dal.Commit.Where(dal.Commit.RepositoryID.Eq(repositoryID), dal.Commit.DraftName.Neq("")).Offset(offset).Limit(limit)
	if reverse {
		stmt = stmt.Order(dal.Commit.ID.Desc())
	}

	return stmt.Find()
}

func (c *CommitMapperImpl) FindDraftPageByRepositoryIDAndQuery(repositoryID, query string, offset, limit int, reverse bool) (model.Commits, error) {
	stmt := dal.Commit.Where(dal.Commit.RepositoryID.Eq(repositoryID), dal.Commit.DraftName.Neq(""), dal.Commit.DraftName.Like("%"+query+"%")).Offset(offset).Limit(limit)
	if reverse {
		stmt = stmt.Order(dal.Commit.ID.Desc())
	}

	return stmt.Find()
}

func (c *CommitMapperImpl) DeleteByRepositoryIDAndDraftName(repositoryID string, draftName string) error {
	_, err := dal.Commit.Where(dal.Commit.RepositoryID.Eq(repositoryID), dal.Commit.DraftName.Eq(draftName), dal.Commit.DraftName.Neq("")).Delete()
	return err
}

func (c *CommitMapperImpl) FindSequenceID(commit *model.Commit) (int64, error) {
	var sequenceID int64
	var err error
	if commit.DraftName == "" {
		sequenceID, err = dal.Commit.Where(dal.Commit.DraftName.Eq(""), dal.Commit.ID.Lte(commit.ID), dal.Commit.CreatedTime.Lte(commit.CreatedTime)).Count()
	} else {
		// draft 没有sequence id
		return 0, nil
	}
	if err != nil {
		return 0, err
	}

	return sequenceID, nil
}
