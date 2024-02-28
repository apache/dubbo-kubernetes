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

package bufmodulebuild

import (
	"context"

	"github.com/apache/dubbo-kubernetes/pkg/bufman/bufpkg/bufmodule"
	"github.com/apache/dubbo-kubernetes/pkg/bufman/bufpkg/bufmodule/internal"
	"github.com/apache/dubbo-kubernetes/pkg/bufman/pkg/normalpath"
	"github.com/apache/dubbo-kubernetes/pkg/bufman/pkg/storage"
	"github.com/apache/dubbo-kubernetes/pkg/bufman/pkg/storage/storageos"
	"go.uber.org/zap"
)

type moduleIncludeBuilder struct {
	logger            *zap.Logger
	storageosProvider storageos.Provider
}

func newModuleIncludeBuilder(
	logger *zap.Logger,
	storageosProvider storageos.Provider,
) *moduleIncludeBuilder {
	return &moduleIncludeBuilder{
		logger:            logger,
		storageosProvider: storageosProvider,
	}
}

func (b *moduleIncludeBuilder) BuildForIncludes(
	ctx context.Context,
	includeDirPaths []string,
	options ...BuildOption,
) (bufmodule.Module, error) {
	buildOptions := &buildOptions{}
	for _, option := range options {
		option(buildOptions)
	}
	return b.buildForIncludes(
		ctx,
		includeDirPaths,
		buildOptions.paths,
		buildOptions.pathsAllowNotExist,
	)
}

func (b *moduleIncludeBuilder) buildForIncludes(
	ctx context.Context,
	includeDirPaths []string,
	fileOrDirPaths *[]string,
	fileOrDirPathsAllowNotExist bool,
) (bufmodule.Module, error) {
	if len(includeDirPaths) == 0 {
		includeDirPaths = []string{"."}
	}
	absIncludeDirPaths, err := internal.NormalizeAndCheckPaths(
		includeDirPaths,
		"include directory",
		normalpath.Absolute,
		true,
	)
	if err != nil {
		return nil, err
	}
	var absFileOrDirPaths *[]string
	if fileOrDirPaths != nil {
		normalizedAndCheckedFileOrDirPaths, err := internal.NormalizeAndCheckPaths(
			*fileOrDirPaths,
			"input file",
			normalpath.Absolute,
			false,
		)
		if err != nil {
			return nil, err
		}
		if len(normalizedAndCheckedFileOrDirPaths) > 0 {
			absFileOrDirPaths = &normalizedAndCheckedFileOrDirPaths
		}
	}
	var rootBuckets []storage.ReadBucket
	for _, includeDirPath := range includeDirPaths {
		rootBucket, err := b.storageosProvider.NewReadWriteBucket(
			includeDirPath,
			storageos.ReadWriteBucketWithSymlinksIfSupported(),
		)
		if err != nil {
			return nil, err
		}
		// need to do match extension here
		// https://github.com/ProtobufMan/bufman-cli/issues/113
		rootBuckets = append(rootBuckets, storage.MapReadBucket(rootBucket, storage.MatchPathExt(".proto")))
	}
	module, err := bufmodule.NewModuleForBucket(ctx, storage.MultiReadBucket(rootBuckets...))
	if err != nil {
		return nil, err
	}
	return applyModulePaths(
		module,
		absIncludeDirPaths,
		absFileOrDirPaths,
		nil, // Excluding files isn't supported for the ModuleIncludeBuilder.
		fileOrDirPathsAllowNotExist,
		normalpath.Absolute,
	)
}
