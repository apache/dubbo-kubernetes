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
	"os"
	"path"
	"sync"

	"github.com/apache/dubbo-kubernetes/pkg/bufman/constant"
	"github.com/apache/dubbo-kubernetes/pkg/bufman/model"
)

// DiskStorageHelperImpl
// Deprecated
type DiskStorageHelperImpl struct {
	mu     sync.Mutex
	muDict map[string]*sync.RWMutex

	pluginMu     sync.Mutex
	pluginMuDict map[string]*sync.RWMutex
}

func (helper *DiskStorageHelperImpl) StoreBlob(ctx context.Context, blob *model.CommitFile) error {
	return helper.store(ctx, blob.Digest, []byte(blob.Content))
}

func (helper *DiskStorageHelperImpl) store(ctx context.Context, digest string, content []byte) error {
	helper.mu.Lock()
	defer helper.mu.Unlock()

	if _, ok := helper.muDict[digest]; !ok {
		helper.muDict[digest] = &sync.RWMutex{}
	}

	// 上写锁
	helper.muDict[digest].Lock()
	defer helper.muDict[digest].Unlock()

	// 打开文件
	filePath := helper.GetFilePath(digest)
	file, err := os.OpenFile(filePath, os.O_CREATE|os.O_WRONLY|os.O_EXCL, 0o666)
	if os.IsExist(err) {
		// 已经存在，直接返回
		return nil
	}
	if err != nil {
		return err
	}

	_, err = file.Write(content)
	if err != nil {
		return err
	}

	return nil
}

func (helper *DiskStorageHelperImpl) StoreManifest(ctx context.Context, manifest *model.CommitFile) error {
	return helper.store(ctx, manifest.Digest, []byte(manifest.Content))
}

func (helper *DiskStorageHelperImpl) StoreDocumentation(ctx context.Context, blob *model.CommitFile) error {
	return nil
}

func (helper *DiskStorageHelperImpl) ReadBlobToReader(ctx context.Context, digest string) (io.Reader, error) {
	content, err := helper.ReadBlob(ctx, digest)
	if err != nil {
		return nil, err
	}

	return bytes.NewReader(content), nil
}

func (helper *DiskStorageHelperImpl) ReadBlob(ctx context.Context, digest string) ([]byte, error) {
	return helper.read(ctx, digest)
}

func (helper *DiskStorageHelperImpl) ReadManifestToReader(ctx context.Context, digest string) (io.Reader, error) {
	content, err := helper.ReadManifest(ctx, digest)
	if err != nil {
		return nil, err
	}

	return bytes.NewReader(content), nil
}

func (helper *DiskStorageHelperImpl) ReadManifest(ctx context.Context, digest string) ([]byte, error) {
	return helper.read(ctx, digest)
}

func (helper *DiskStorageHelperImpl) read(ctx context.Context, digest string) ([]byte, error) {
	helper.mu.Lock()
	defer helper.mu.Unlock()

	if _, ok := helper.muDict[digest]; !ok {
		helper.muDict[digest] = &sync.RWMutex{}
	}

	// 上读锁
	helper.muDict[digest].RLock()
	defer helper.muDict[digest].RUnlock()

	// 读取文件
	filePath := helper.GetFilePath(digest)
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	return content, nil
}

func (helper *DiskStorageHelperImpl) GetFilePath(digest string) string {
	return path.Join(constant.FileSavaDir, digest)
}
