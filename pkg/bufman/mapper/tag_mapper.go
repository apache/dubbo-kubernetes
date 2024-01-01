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
	"github.com/apache/dubbo-kubernetes/pkg/bufman/dal"
	"github.com/apache/dubbo-kubernetes/pkg/bufman/model"
)

type TagMapper interface {
	Create(tag *model.Tag) error
	GetCountsByRepositoryID(repositoryID string) (int64, error)
	FindPageByRepositoryID(repositoryID string, offset, limit int, reverse bool) (model.Tags, error)
	FindPageByRepositoryIDAndQuery(repositoryID, query string, offset, limit int, reverse bool) (model.Tags, error)
}

type TagMapperImpl struct{}

func (t *TagMapperImpl) Create(tag *model.Tag) error {
	return dal.Tag.Create(tag)
}

func (t *TagMapperImpl) GetCountsByRepositoryID(repositoryID string) (int64, error) {
	return dal.Tag.Where(dal.Tag.RepositoryID.Eq(repositoryID)).Count()
}

func (t *TagMapperImpl) FindPageByRepositoryID(repositoryID string, offset, limit int, reverse bool) (model.Tags, error) {
	stmt := dal.Tag.Where(dal.Tag.RepositoryID.Eq(repositoryID)).Offset(offset).Limit(limit)
	if reverse {
		stmt = stmt.Order(dal.Tag.ID.Desc())
	}

	return stmt.Find()
}

func (t *TagMapperImpl) FindPageByRepositoryIDAndQuery(repositoryID, query string, offset, limit int, reverse bool) (model.Tags, error) {
	stmt := dal.Tag.Where(dal.Tag.RepositoryID.Eq(repositoryID), dal.Tag.TagName.Like("%"+query+"%")).Offset(offset).Limit(limit)
	if reverse {
		stmt = stmt.Order(dal.Tag.ID.Desc())
	}

	return stmt.Find()
}
