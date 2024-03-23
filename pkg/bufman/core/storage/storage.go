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
	"context"
	"errors"
	"io"
	"sync"
)

import (
	"github.com/apache/dubbo-kubernetes/pkg/bufman/bufpkg/bufconfig"
	"github.com/apache/dubbo-kubernetes/pkg/bufman/bufpkg/bufmodule"
	"github.com/apache/dubbo-kubernetes/pkg/bufman/model"
	manifest2 "github.com/apache/dubbo-kubernetes/pkg/bufman/pkg/manifest"
)

type BaseStorageHelper interface {
	StoreBlob(ctx context.Context, blob *model.CommitFile) error
	StoreManifest(ctx context.Context, manifest *model.CommitFile) error
	StoreDocumentation(ctx context.Context, blob *model.CommitFile) error
	ReadBlobToReader(ctx context.Context, digest string) (io.Reader, error) // 读取内容
	ReadBlob(ctx context.Context, digest string) ([]byte, error)
	ReadManifestToReader(ctx context.Context, digest string) (io.Reader, error)
	ReadManifest(ctx context.Context, digest string) ([]byte, error)
}

type StorageHelper interface {
	BaseStorageHelper
	ReadToManifestAndBlobSet(ctx context.Context, modelFileManifest *model.CommitFile, fileBlobs model.CommitFiles) (*manifest2.Manifest, *manifest2.BlobSet, error) // 读取为manifest和blob set
	GetDocumentAndLicenseFromBlob(ctx context.Context, fileManifest *manifest2.Manifest, blobSet *manifest2.BlobSet) (manifest2.Blob, manifest2.Blob, error)
	GetBufManConfigFromBlob(ctx context.Context, fileManifest *manifest2.Manifest, blobSet *manifest2.BlobSet) (manifest2.Blob, error)
	GetDocumentFromBlob(ctx context.Context, fileManifest *manifest2.Manifest, blobSet *manifest2.BlobSet) (manifest2.Blob, error)
	GetLicenseFromBlob(ctx context.Context, fileManifest *manifest2.Manifest, blobSet *manifest2.BlobSet) (manifest2.Blob, error)
}

type StorageHelperImpl struct {
	BaseStorageHelper
}

// 单例模式
var (
	storageHelperImpl *StorageHelperImpl
	once              sync.Once
)

func NewStorageHelper() StorageHelper {
	if storageHelperImpl == nil {
		// 对象初始化
		once.Do(func() {
			storageHelperImpl = &StorageHelperImpl{
				BaseStorageHelper: NewDBStorageHelper(),
			}
		})
	}

	return storageHelperImpl
}

func (helper *StorageHelperImpl) ReadToManifestAndBlobSet(ctx context.Context, modelFileManifest *model.CommitFile, fileBlobs model.CommitFiles) (*manifest2.Manifest, *manifest2.BlobSet, error) {
	// 读取文件清单
	reader, err := helper.ReadManifestToReader(ctx, modelFileManifest.Digest)
	if err != nil {
		return nil, nil, err
	}
	fileManifest, err := manifest2.NewFromReader(reader)
	if err != nil {
		return nil, nil, err
	}

	// 读取文件blobs
	blobs := make([]manifest2.Blob, 0, len(fileBlobs))
	for i := 0; i < len(fileBlobs); i++ {
		// 读取文件
		reader, err := helper.ReadBlobToReader(ctx, fileBlobs[i].Digest)
		if err != nil {
			return nil, nil, err
		}

		// 生成blob
		blob, err := manifest2.NewMemoryBlobFromReader(reader)
		if err != nil {
			return nil, nil, err
		}
		blobs = append(blobs, blob)
	}

	blobSet, err := manifest2.NewBlobSet(ctx, blobs)
	if err != nil {
		return nil, nil, err
	}

	return fileManifest, blobSet, nil
}

func (helper *StorageHelperImpl) GetDocumentAndLicenseFromBlob(ctx context.Context, fileManifest *manifest2.Manifest, blobSet *manifest2.BlobSet) (manifest2.Blob, manifest2.Blob, error) {
	var documentDataExists, licenseExists bool
	var documentBlob, licenseBlob manifest2.Blob

	externalPaths := []string{
		bufmodule.LicenseFilePath,
	}
	externalPaths = append(externalPaths, bufmodule.AllDocumentationPaths...)

	err := fileManifest.Range(func(path string, digest manifest2.Digest) error {
		blob, ok := blobSet.BlobFor(digest.String())
		if !ok {
			// 文件清单中有的文件，在file blobs中没有
			return errors.New("check manifest and file blobs failed")
		}

		// 如果遇到配置文件，就记录下来
		for _, externalPath := range externalPaths {
			if documentDataExists && licenseExists {
				break
			}

			if path == externalPath {
				if path == bufmodule.LicenseFilePath {
					// license文件
					licenseBlob = blob
					licenseExists = true
				} else {
					if documentDataExists {
						break
					}
					// document文件
					documentBlob = blob
					documentDataExists = true
				}
			}
		}

		return nil
	})
	if err != nil {
		return nil, nil, err
	}

	return documentBlob, licenseBlob, nil
}

func (helper *StorageHelperImpl) GetBufManConfigFromBlob(ctx context.Context, fileManifest *manifest2.Manifest, blobSet *manifest2.BlobSet) (manifest2.Blob, error) {
	var configFileExist bool
	var configFileBlob manifest2.Blob

	err := fileManifest.Range(func(path string, digest manifest2.Digest) error {
		blob, ok := blobSet.BlobFor(digest.String())
		if !ok {
			// 文件清单中有的文件，在file blobs中没有
			return errors.New("check manifest and file blobs failed")
		}

		// 如果遇到配置文件，就记录下来
		for _, configFilePath := range bufconfig.AllConfigFilePaths {
			if configFileExist {
				break
			}

			if path == configFilePath {
				configFileBlob = blob
				configFileExist = true
			}
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return configFileBlob, nil
}

func (helper *StorageHelperImpl) GetDocumentFromBlob(ctx context.Context, fileManifest *manifest2.Manifest, blobSet *manifest2.BlobSet) (manifest2.Blob, error) {
	var documentExist bool
	var documentBlob manifest2.Blob

	err := fileManifest.Range(func(path string, digest manifest2.Digest) error {
		blob, ok := blobSet.BlobFor(digest.String())
		if !ok {
			// 文件清单中有的文件，在file blobs中没有
			return errors.New("check manifest and file blobs failed")
		}

		// 如果遇到README文件，就记录下来
		for _, documentationPath := range bufmodule.AllDocumentationPaths {
			if documentExist {
				break
			}

			if path == documentationPath {
				documentBlob = blob
				documentExist = true
			}
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return documentBlob, nil
}

func (helper *StorageHelperImpl) GetLicenseFromBlob(ctx context.Context, fileManifest *manifest2.Manifest, blobSet *manifest2.BlobSet) (manifest2.Blob, error) {
	var licenseExist bool
	var licenseBlob manifest2.Blob

	err := fileManifest.Range(func(path string, digest manifest2.Digest) error {
		if licenseExist {
			return nil
		}

		blob, ok := blobSet.BlobFor(digest.String())
		if !ok {
			// 文件清单中有的文件，在file blobs中没有
			return errors.New("check manifest and file blobs failed")
		}

		// 如果遇到license，就记录下来
		if path == bufmodule.LicenseFilePath {
			licenseBlob = blob
			licenseExist = true
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return licenseBlob, nil
}
