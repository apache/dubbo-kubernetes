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

	registryv1alpha1 "github.com/apache/dubbo-kubernetes/pkg/bufman/gen/proto/go/registry/v1alpha1"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// User 用户表
type User struct {
	ID          int64     `gorm:"primaryKey;autoIncrement"`
	UserID      string    `gorm:"type:varchar(64);unique; not null"`
	UserName    string    `gorm:"type:varchar(200);unique;not null"`
	Password    string    `gorm:"type:varchar(64);not null"`
	CreatedTime time.Time `gorm:"autoCreateTime"`
	UpdateTime  time.Time `gorm:"autoUpdateTime"`
	Deactivated bool      // 无效
	Url         string
	Description string // 描述信息
	UserType    int32  `gorm:"default:1"`
}

func (user *User) TableName() string {
	return "users"
}

func (user *User) ToProtoUser() *registryv1alpha1.User {
	if user == nil {
		return (&User{}).ToProtoUser()
	}

	return &registryv1alpha1.User{
		Id:          user.UserID,
		CreateTime:  timestamppb.New(user.CreatedTime),
		UpdateTime:  timestamppb.New(user.UpdateTime),
		Username:    user.UserName,
		Deactivated: user.Deactivated,
		Description: user.Description,
		Url:         user.Url,
	}
}

func (user *User) ToProtoSearchResult() *registryv1alpha1.UserSearchResult {
	if user == nil {
		return (&User{}).ToProtoSearchResult()
	}

	return &registryv1alpha1.UserSearchResult{
		Id:          user.UserID,
		Username:    user.UserName,
		Deactivated: user.Deactivated,
	}
}

type Users []*User

func (users *Users) ToProtoUsers() []*registryv1alpha1.User {
	protoUsers := make([]*registryv1alpha1.User, 0, len(*users))
	for i := 0; i < len(*users); i++ {
		protoUsers = append(protoUsers, (*users)[i].ToProtoUser())
	}

	return protoUsers
}

func (users *Users) ToProtoSearchResults() []*registryv1alpha1.UserSearchResult {
	protoResults := make([]*registryv1alpha1.UserSearchResult, 0, len(*users))
	for i := 0; i < len(*users); i++ {
		protoResults = append(protoResults, (*users)[i].ToProtoSearchResult())
	}

	return protoResults
}
