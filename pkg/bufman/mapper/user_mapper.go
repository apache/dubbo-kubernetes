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

type UserMapper interface {
	Create(user *model.User) error
	FindByUserID(userID string) (*model.User, error)
	FindByUserName(userName string) (*model.User, error)
	FindPage(offset int, limit int, reverse bool) (model.Users, error)
	FindPageByQuery(query string, offset int, limit int, reverse bool) (model.Users, error)
}

type UserMapperImpl struct{}

func (u *UserMapperImpl) Create(user *model.User) error {
	return dal.User.Create(user)
}

func (u *UserMapperImpl) FindByUserID(userID string) (*model.User, error) {
	return dal.User.Where(dal.User.UserID.Eq(userID)).First()
}

func (u *UserMapperImpl) FindByUserName(userName string) (*model.User, error) {
	return dal.User.Where(dal.User.UserName.Eq(userName)).First()
}

func (u *UserMapperImpl) FindPage(offset int, limit int, reverse bool) (model.Users, error) {
	stmt := dal.User
	if reverse {
		stmt.Order(dal.User.ID.Desc())
	}

	users, _, err := stmt.FindByPage(offset, limit)
	return users, err
}

func (u *UserMapperImpl) FindPageByQuery(query string, offset int, limit int, reverse bool) (model.Users, error) {
	stmt := dal.User.Where(dal.User.UserName.Like("%" + query + "%"))
	if reverse {
		stmt.Order(dal.User.ID.Desc())
	}

	users, _, err := stmt.FindByPage(offset, limit)
	return users, err
}
