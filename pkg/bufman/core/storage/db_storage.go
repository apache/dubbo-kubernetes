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

package storage

import (
	"bytes"
	"context"
	"io"

	"github.com/apache/dubbo-kubernetes/pkg/bufman/dal"
	"github.com/apache/dubbo-kubernetes/pkg/bufman/model"
)

type DBStorageHelperImpl struct{}

func NewDBStorageHelper() *DBStorageHelperImpl {
	return &DBStorageHelperImpl{}
}

func (helper *DBStorageHelperImpl) StoreBlob(ctx context.Context, file *model.CommitFile) error {
	return dal.FileBlob.Create(&model.FileBlob{
		Digest:  file.Digest,
		Content: file.Content,
	})
}

func (helper *DBStorageHelperImpl) StoreManifest(ctx context.Context, manifest *model.CommitFile) error {
	return dal.FileBlob.Create(&model.FileBlob{
		Digest:  manifest.Digest,
		Content: manifest.Content,
	})
}

func (helper *DBStorageHelperImpl) StoreDocumentation(ctx context.Context, file *model.CommitFile) error {
	return dal.FileBlob.Create(&model.FileBlob{
		Digest:  file.Digest,
		Content: file.Content,
	})
}

func (helper *DBStorageHelperImpl) ReadBlobToReader(ctx context.Context, digest string) (io.Reader, error) {
	content, err := helper.ReadManifest(ctx, digest)
	if err != nil {
		return nil, err
	}

	return bytes.NewReader(content), nil
}

func (helper *DBStorageHelperImpl) ReadBlob(ctx context.Context, digest string) ([]byte, error) {
	blob, err := dal.FileBlob.Where(dal.FileBlob.Digest.Eq(digest)).First()
	if err != nil {
		return nil, err
	}

	return blob.Content, nil
}

func (helper *DBStorageHelperImpl) ReadManifestToReader(ctx context.Context, digest string) (io.Reader, error) {
	content, err := helper.ReadManifest(ctx, digest)
	if err != nil {
		return nil, err
	}

	return bytes.NewReader(content), nil
}

func (helper *DBStorageHelperImpl) ReadManifest(ctx context.Context, digest string) ([]byte, error) {
	blob, err := dal.FileBlob.Where(dal.FileBlob.Digest.Eq(digest)).First()
	if err != nil {
		return nil, err
	}

	return blob.Content, nil
}
