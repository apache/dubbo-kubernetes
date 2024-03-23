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
	"time"
)

import (
	"github.com/apache/dubbo-kubernetes/pkg/bufman/dal"
	registryv1alpha1 "github.com/apache/dubbo-kubernetes/pkg/bufman/gen/proto/go/registry/v1alpha1"
	"github.com/apache/dubbo-kubernetes/pkg/bufman/model"
)

type RepositoryMapper interface {
	Create(repository *model.Repository) error
	FindByRepositoryID(repositoryID string) (*model.Repository, error)
	FindByUserNameAndRepositoryName(userName, RepositoryName string) (*model.Repository, error)
	FindPage(offset, limit int, reverse bool) (model.Repositories, error)
	FindPageByQuery(query string, offset, limit int, reverse bool) (model.Repositories, error)
	FindPageByUserID(userID string, offset, limit int, reverse bool) (model.Repositories, error)
	FindAccessiblePageByUserID(userID string, offset, limit int, reverse bool) (model.Repositories, error)
	DeleteByRepositoryID(repositoryID string) error
	DeleteByUserNameAndRepositoryName(userName, RepositoryName string) error
	UpdateByUserNameAndRepositoryName(userName, RepositoryName string, repository *model.Repository) error
	UpdateDeprecatedByUserNameAndRepositoryName(userName, RepositoryName string, repository *model.Repository) error
}

type RepositoryMapperImpl struct{}

func (r *RepositoryMapperImpl) Create(repository *model.Repository) error {
	return dal.Q.Transaction(func(tx *dal.Query) error {
		// 更新用户 update time
		_, err := tx.User.Where(tx.User.UserID.Eq(repository.UserID)).Update(tx.User.UpdateTime, time.Now())
		if err != nil {
			return err
		}

		// create
		return tx.Repository.Create(repository)
	})
}

func (r *RepositoryMapperImpl) FindByRepositoryID(repositoryID string) (*model.Repository, error) {
	return dal.Repository.Where(dal.Repository.RepositoryID.Eq(repositoryID)).First()
}

func (r *RepositoryMapperImpl) FindByUserNameAndRepositoryName(userName, RepositoryName string) (*model.Repository, error) {
	return dal.Repository.Where(dal.Repository.UserName.Eq(userName), dal.Repository.RepositoryName.Eq(RepositoryName)).First()
}

func (r *RepositoryMapperImpl) FindPage(offset, limit int, reverse bool) (model.Repositories, error) {
	stmt := dal.Repository.Offset(offset).Limit(limit)
	if reverse {
		stmt = stmt.Order(dal.Repository.ID.Desc())
	}

	return stmt.Find()
}

func (r *RepositoryMapperImpl) FindPageByQuery(query string, offset, limit int, reverse bool) (model.Repositories, error) {
	stmt := dal.Repository.Where(dal.Repository.RepositoryName.Like("%" + query + "%")).Or(dal.Repository.Description.Like("%" + query + "%")).Offset(offset).Limit(limit)
	if reverse {
		stmt = stmt.Order(dal.Repository.ID.Desc())
	}

	return stmt.Find()
}

func (r *RepositoryMapperImpl) FindPageByUserID(userID string, offset, limit int, reverse bool) (model.Repositories, error) {
	stmt := dal.Repository.Offset(offset).Where(dal.Repository.UserID.Eq(userID)).Limit(limit)
	if reverse {
		stmt = stmt.Order(dal.Repository.ID.Desc())
	}

	return stmt.Find()
}

func (r *RepositoryMapperImpl) FindAccessiblePageByUserID(userID string, offset, limit int, reverse bool) (model.Repositories, error) {
	stmt := dal.Repository.Offset(offset).Where(dal.Repository.Visibility.Eq(uint8(registryv1alpha1.Visibility_VISIBILITY_PUBLIC))).Or(dal.Repository.UserID.Eq(userID)).Limit(limit)
	if reverse {
		stmt = stmt.Order(dal.Repository.ID.Desc())
	}

	return stmt.Find()
}

func (r *RepositoryMapperImpl) DeleteByRepositoryID(repositoryID string) error {
	repository := &model.Repository{}
	return dal.Q.Transaction(func(tx *dal.Query) error {
		// 删除repo
		_, err := tx.Repository.Where(tx.Repository.RepositoryID.Eq(repositoryID)).Delete(repository)
		if err != nil {
			return err
		}

		// 删除commit
		_, err = tx.Commit.Where(tx.Commit.RepositoryID.Eq(repositoryID)).Delete()
		if err != nil {
			return err
		}

		// 删除tag
		_, err = tx.Tag.Where(tx.Tag.RepositoryID.Eq(repositoryID)).Delete()
		if err != nil {
			return err
		}

		return nil
	})
}

func (r *RepositoryMapperImpl) DeleteByUserNameAndRepositoryName(userName, RepositoryName string) error {
	repository := &model.Repository{}
	return dal.Q.Transaction(func(tx *dal.Query) error {
		// 删除repo
		_, err := tx.Repository.Where(tx.Repository.UserName.Eq(userName), tx.Repository.RepositoryName.Eq(RepositoryName)).Delete(repository)
		if err != nil {
			return err
		}

		// 删除commit
		_, err = tx.Commit.Where(tx.Commit.RepositoryID.Eq(repository.RepositoryID)).Delete()
		if err != nil {
			return err
		}

		// 删除tag
		_, err = tx.Tag.Where(tx.Tag.RepositoryID.Eq(repository.RepositoryID)).Delete()
		if err != nil {
			return err
		}

		return nil
	})
}

func (r *RepositoryMapperImpl) UpdateByUserNameAndRepositoryName(userName, RepositoryName string, repository *model.Repository) error {
	_, err := dal.Repository.Select(dal.Repository.Visibility, dal.Repository.Description).Where(dal.Repository.UserName.Eq(userName), dal.Repository.RepositoryName.Eq(RepositoryName)).Updates(repository)

	return err
}

func (r *RepositoryMapperImpl) UpdateDeprecatedByUserNameAndRepositoryName(userName, RepositoryName string, repository *model.Repository) error {
	_, err := dal.Repository.Select(dal.Repository.Deprecated, dal.Repository.DeprecationMsg).Where(dal.Repository.UserName.Eq(userName), dal.Repository.RepositoryName.Eq(RepositoryName)).Updates(repository)

	return err
}
