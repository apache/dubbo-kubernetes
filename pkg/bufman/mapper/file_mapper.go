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

type FileMapper interface {
	FindCommitFilesExceptManifestByCommitID(commitID string) (model.CommitFiles, error)
	FindCommitManifestByCommitID(commitID string) (*model.CommitFile, error)
	FindCommitFileByCommitIDAndPath(commitID, path string) (*model.CommitFile, error)
}

type FileMapperImpl struct{}

func (f *FileMapperImpl) FindCommitFilesExceptManifestByCommitID(commitID string) (model.CommitFiles, error) {
	commit, err := dal.Commit.Where(dal.Commit.CommitID.Eq(commitID)).First()
	if err != nil {
		return nil, err
	}

	return dal.CommitFile.Where(dal.CommitFile.CommitID.Eq(commitID), dal.CommitFile.Digest.Neq(commit.ManifestDigest)).Find()
}

func (f *FileMapperImpl) FindCommitManifestByCommitID(commitID string) (*model.CommitFile, error) {
	commit, err := dal.Commit.Where(dal.Commit.CommitID.Eq(commitID)).First()
	if err != nil {
		return nil, err
	}

	return dal.CommitFile.Where(dal.CommitFile.CommitID.Eq(commitID), dal.CommitFile.Digest.Eq(commit.ManifestDigest)).First()
}

func (f *FileMapperImpl) FindCommitFileByCommitIDAndPath(commitID, path string) (*model.CommitFile, error) {
	return dal.CommitFile.Where(dal.CommitFile.CommitID.Eq(commitID), dal.CommitFile.FileName.Eq(path)).First()
}
