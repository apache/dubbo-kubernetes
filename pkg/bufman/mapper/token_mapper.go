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
	"github.com/apache/dubbo-kubernetes/pkg/bufman/model"
)

type TokenMapper interface {
	Create(token *model.Token) error
	FindAvailableByTokenID(tokenID string) (*model.Token, error)
	FindAvailableByTokenName(tokenName string) (*model.Token, error)
	FindAvailablePageByUserID(userID string, offset int, limit int, reverse bool) (model.Tokens, error)
	DeleteByTokenID(tokenID string) error
}

type TokenMapperImpl struct{}

func (t *TokenMapperImpl) Create(token *model.Token) error {
	return dal.Token.Create(token)
}

func (t *TokenMapperImpl) FindAvailableByTokenID(tokenID string) (*model.Token, error) {
	return dal.Token.Where(dal.Token.TokenID.Eq(tokenID), dal.Token.ExpireTime.Gt(time.Now())).First()
}

func (t *TokenMapperImpl) FindAvailableByTokenName(tokenName string) (*model.Token, error) {
	return dal.Token.Where(dal.Token.TokenName.Eq(tokenName), dal.Token.ExpireTime.Gt(time.Now())).First()
}

func (t *TokenMapperImpl) FindAvailablePageByUserID(userID string, offset int, limit int, reverse bool) (model.Tokens, error) {
	stmt := dal.Token.Where(dal.Token.UserID.Eq(userID), dal.Token.ExpireTime.Gt(time.Now()))
	if reverse {
		stmt.Order(dal.Token.ID.Desc())
	}

	tokens, _, err := stmt.FindByPage(offset, limit)

	return tokens, err
}

func (t *TokenMapperImpl) DeleteByTokenID(tokenID string) error {
	token := &model.Token{}
	_, err := dal.Token.Where(dal.Token.TokenID.Eq(tokenID), dal.Token.ExpireTime.Gt(time.Now())).Delete(token)
	return err
}
