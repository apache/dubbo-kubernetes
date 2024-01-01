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
	FindAllBlobsByCommitID(commitID string) (model.FileBlobs, error)
	FindManifestByCommitID(commitID string) (*model.FileManifest, error)
	FindBlobByCommitIDAndPath(commitID, path string) (*model.FileBlob, error)
}

type FileMapperImpl struct{}

func (f *FileMapperImpl) FindAllBlobsByCommitID(commitID string) (model.FileBlobs, error) {
	return dal.FileBlob.Where(dal.FileBlob.CommitID.Eq(commitID)).Find()
}

func (f *FileMapperImpl) FindManifestByCommitID(commitID string) (*model.FileManifest, error) {
	return dal.FileManifest.Where(dal.FileManifest.CommitID.Eq(commitID)).First()
}

func (f *FileMapperImpl) FindBlobByCommitIDAndPath(commitID, path string) (*model.FileBlob, error) {
	return dal.FileBlob.Where(dal.FileBlob.CommitID.Eq(commitID), dal.FileBlob.FileName.Eq(path)).First()
}
