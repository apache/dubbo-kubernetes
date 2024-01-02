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

package validity

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/apache/dubbo-kubernetes/pkg/bufman/bufpkg/bufconfig"
	"github.com/apache/dubbo-kubernetes/pkg/bufman/bufpkg/buflock"
	"github.com/apache/dubbo-kubernetes/pkg/bufman/bufpkg/bufmanifest"
	"github.com/apache/dubbo-kubernetes/pkg/bufman/bufpkg/bufmodule"
	"github.com/apache/dubbo-kubernetes/pkg/bufman/constant"
	"github.com/apache/dubbo-kubernetes/pkg/bufman/e"
	modulev1alpha1 "github.com/apache/dubbo-kubernetes/pkg/bufman/gen/proto/go/module/v1alpha1"
	manifest2 "github.com/apache/dubbo-kubernetes/pkg/bufman/pkg/manifest"
)

type Validator interface {
	CheckUserName(username string) e.ResponseError             // 检查用户名合法性
	CheckPassword(password string) e.ResponseError             // 检查密码合法性
	CheckRepositoryName(repositoryName string) e.ResponseError // 检查repo name合法性
	CheckTagName(tagName string) e.ResponseError               // 检查tag name合法性
	CheckDraftName(draftName string) e.ResponseError           // 检查draft name合法性
	CheckPageSize(pageSize uint32) e.ResponseError             // 检查page size合法性
	CheckQuery(query string) e.ResponseError
	SplitFullName(fullName string) (userName, repositoryName string, respErr e.ResponseError) // 分割full name

	// CheckManifestAndBlobs 检查上传的文件是否合法
	CheckManifestAndBlobs(ctx context.Context, protoManifest *modulev1alpha1.Blob, protoBlobs []*modulev1alpha1.Blob) (*manifest2.Manifest, *manifest2.BlobSet, e.ResponseError)
}

func NewValidator() Validator {
	return &ValidatorImpl{}
}

type ValidatorImpl struct{}

func (validator *ValidatorImpl) CheckUserName(username string) e.ResponseError {
	err := validator.doCheckByLengthAndPattern(username, constant.MinUserNameLength, constant.MaxUserNameLength, constant.UserNamePattern)
	if err != nil {
		return e.NewInvalidArgumentError(err)
	}

	return nil
}

func (validator *ValidatorImpl) CheckPassword(password string) e.ResponseError {
	err := validator.doCheckByLengthAndPattern(password, constant.MinPasswordLength, constant.MaxPasswordLength, constant.PasswordPattern)
	if err != nil {
		return e.NewInvalidArgumentError(err)
	}

	return nil
}

func (validator *ValidatorImpl) CheckRepositoryName(repositoryName string) e.ResponseError {
	err := validator.doCheckByLengthAndPattern(repositoryName, constant.MinRepositoryNameLength, constant.MaxRepositoryNameLength, constant.RepositoryNamePattern)
	if err != nil {
		return e.NewInvalidArgumentError(err)
	}

	return nil
}

func (validator *ValidatorImpl) CheckTagName(tagName string) e.ResponseError {
	err := validator.doCheckByLengthAndPattern(tagName, constant.MinTagLength, constant.MaxTagLength, constant.TagPattern)
	if err != nil {
		return e.NewInvalidArgumentError(err)
	}

	return nil
}

func (validator *ValidatorImpl) CheckDraftName(draftName string) e.ResponseError {
	if draftName == constant.DefaultBranch { // draft name不能为 main
		return e.NewInvalidArgumentError(fmt.Errorf("draft (can not be '%v')", constant.DefaultBranch))
	}

	err := validator.doCheckByLengthAndPattern(draftName, constant.MinDraftLength, constant.MaxDraftLength, constant.DraftPattern)
	if err != nil {
		return e.NewInvalidArgumentError(err)
	}

	return nil
}

func (validator *ValidatorImpl) CheckPageSize(pageSize uint32) e.ResponseError {
	if pageSize < constant.MinPageSize || pageSize > constant.MaxPageSize {
		return e.NewInvalidArgumentError(fmt.Errorf("page size: length is limited between %v and %v", constant.MinPageSize, constant.MaxPageSize))
	}

	return nil
}

func (validator *ValidatorImpl) CheckQuery(query string) e.ResponseError {
	err := validator.doCheckByLengthAndPattern(query, constant.MinQueryLength, constant.MaxQueryLength, constant.QueryPattern)
	if err != nil {
		return e.NewInvalidArgumentError(err)
	}

	return nil
}

func (validator *ValidatorImpl) SplitFullName(fullName string) (userName, repositoryName string, respErr e.ResponseError) {
	split := strings.SplitN(fullName, "/", 2)
	if len(split) != 2 {
		respErr = e.NewInvalidArgumentError(errors.New("full name"))
		return
	}

	ok := split[0] != "" && split[1] != ""
	if !ok {
		respErr = e.NewInvalidArgumentError(errors.New("full name"))
		return
	}

	userName, repositoryName = split[0], split[1]
	return userName, repositoryName, nil
}

func (validator *ValidatorImpl) doCheckByLengthAndPattern(str string, minLength, maxLength int, pattern string) error {
	// 长度检查
	if len(str) < minLength || len(str) > maxLength {
		return fmt.Errorf("length is limited between %v and %v", minLength, maxLength)
	}

	// 正则匹配
	match, _ := regexp.MatchString(pattern, str)
	if !match {
		return fmt.Errorf("pattern dont math %s", pattern)
	}

	return nil
}

func (validator *ValidatorImpl) CheckManifestAndBlobs(ctx context.Context, protoManifest *modulev1alpha1.Blob, protoBlobs []*modulev1alpha1.Blob) (*manifest2.Manifest, *manifest2.BlobSet, e.ResponseError) {
	// 读取文件清单
	fileManifest, err := bufmanifest.NewManifestFromProto(ctx, protoManifest)
	if err != nil {
		return nil, nil, e.NewInvalidArgumentError(err)
	}
	if fileManifest.Empty() {
		// 不允许上次空的commit
		return nil, nil, e.NewInvalidArgumentError(errors.New("no files"))
	}

	// 读取文件列表
	blobSet, err := bufmanifest.NewBlobSetFromProto(ctx, protoBlobs)
	if err != nil {
		return nil, nil, e.NewInvalidArgumentError(err)
	}

	// 检查文件清单和blobs
	externalPaths := []string{
		buflock.ExternalConfigFilePath,
		bufmodule.LicenseFilePath,
	}
	externalPaths = append(externalPaths, bufmodule.AllDocumentationPaths...)
	externalPaths = append(externalPaths, bufconfig.AllConfigFilePaths...)
	err = fileManifest.Range(func(path string, digest manifest2.Digest) error {
		_, ok := blobSet.BlobFor(digest.String())
		if !ok {
			// 文件清单中有的文件，在file blobs中没有
			return errors.New("check manifest and file blobs failed")
		}

		// 仅仅允许上传.proto、readme、license、配置文件
		if !strings.HasSuffix(path, ".proto") {
			unexpected := true
			for _, externalPath := range externalPaths {
				if path == externalPath {
					unexpected = false
					break
				}
			}

			if unexpected {
				return errors.New("only allow update .proto、readme、license、bufman config file")
			}
		}

		return nil
	})
	if err != nil {
		return nil, nil, e.NewInvalidArgumentError(err)
	}

	return fileManifest, blobSet, nil
}
