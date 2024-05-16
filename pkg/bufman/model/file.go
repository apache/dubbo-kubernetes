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
	"path/filepath"
	"strings"
	"time"
)

import (
	registryv1alpha1 "github.com/apache/dubbo-kubernetes/pkg/bufman/gen/proto/go/registry/v1alpha1"
)

// CommitFile 保存一个commit中的文件名和摘要
type CommitFile struct {
	ID             int64  `gorm:"primaryKey;autoIncrement"`
	Digest         string // 文件哈希
	CommitID       string `gorm:"type:varchar(64);index"`
	FileName       string
	Content        []byte    `gorm:"-"`
	UserID         string    `gorm:"-"`
	UserName       string    `gorm:"-"`
	RepositoryID   string    `gorm:"-"`
	RepositoryName string    `gorm:"-"`
	CommitName     string    `gorm:"-"`
	CreatedTime    time.Time `gorm:"-"`
}

// FileBlob 以哈希作为区分，记录文件内容
type FileBlob struct {
	ID      int64  `gorm:"primaryKey;autoIncrement"`
	Digest  string // 文件哈希
	Content []byte `gorm:"type:blob"`
}

type CommitFiles []*CommitFile

func (commitFiles *CommitFiles) ToProtoFileInfo() *registryv1alpha1.FileInfo {
	if commitFiles == nil {
		return nil
	}

	root := &registryv1alpha1.FileInfo{
		Path:  ".",
		IsDir: true,
	}

	for i := 0; i < len(*commitFiles); i++ {
		commitFile := (*commitFiles)[i]
		if commitFile == nil {
			commitFile = &CommitFile{}
		}
		doToProtoFileInfo(root, commitFile.FileName)
	}

	return root
}

func doToProtoFileInfo(root *registryv1alpha1.FileInfo, filePath string) {
	// 分割file path
	filePath = filepath.ToSlash(filePath)
	pathLists := strings.Split(filePath, "/")
	// level指向当前层数
	level := root

	// 按照路径去查找
	for i := 0; i < len(pathLists); i++ {
		path := strings.Join(pathLists[:i+1], "/")
		exists := false
		var next *registryv1alpha1.FileInfo
		if level.Children != nil {
			// 在child中寻找path
			for _, child := range level.Children {
				if child.Path == path {
					exists = true
					next = child
					break
				}
			}
		}

		// path不存在
		if !exists {
			child := &registryv1alpha1.FileInfo{
				Path:  path,
				IsDir: i < len(pathLists)-1,
			}
			level.Children = append(level.Children, child)
			next = child
		}

		level = next // 指向下一层
	}
}

//// FileIdentity 唯一文件表，根据哈希值记录文件实际存储地址
//type FileIdentity struct {
//	ID       int64  `gorm:"primaryKey;autoIncrement"`
//	Digest   string `gorm:"unique"` // 文件哈希
//	Location string // 文件实际存储地址
//	FileName string // 文件实际存储名字
//}
