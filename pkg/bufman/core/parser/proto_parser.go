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

package parser

import (
	"context"
	"strings"
)

import (
	"github.com/bufbuild/protocompile"
	"github.com/bufbuild/protocompile/linker"
)

import (
	"github.com/apache/dubbo-kubernetes/pkg/bufman/bufpkg/bufmodule"
	"github.com/apache/dubbo-kubernetes/pkg/bufman/bufpkg/bufmodule/bufmoduleprotocompile"
	"github.com/apache/dubbo-kubernetes/pkg/bufman/bufpkg/bufmodule/bufmoduleref"
	"github.com/apache/dubbo-kubernetes/pkg/bufman/e"
	registryv1alpha1 "github.com/apache/dubbo-kubernetes/pkg/bufman/gen/proto/go/registry/v1alpha1"
	manifest2 "github.com/apache/dubbo-kubernetes/pkg/bufman/pkg/manifest"
	"github.com/apache/dubbo-kubernetes/pkg/bufman/pkg/thread"
)

type ProtoParser interface {
	// TryCompile 尝试编译，查看是否能够编译成功
	TryCompile(ctx context.Context, fileManifest *manifest2.Manifest, blobSet *manifest2.BlobSet, dependentManifests []*manifest2.Manifest, dependentBlobSets []*manifest2.BlobSet) e.ResponseError
	// GetPackageDocumentation 获取package document
	GetPackageDocumentation(ctx context.Context, packageName string, moduleIdentity bufmoduleref.ModuleIdentity, commitName string, fileManifest *manifest2.Manifest, blobSet *manifest2.BlobSet, dependentIdentities []bufmoduleref.ModuleIdentity, dependentCommits []string, dependentManifests []*manifest2.Manifest, dependentBlobSets []*manifest2.BlobSet) (*registryv1alpha1.PackageDocumentation, e.ResponseError)
	// GetPackages 获取所有的package
	GetPackages(ctx context.Context, fileManifest *manifest2.Manifest, blobSet *manifest2.BlobSet, dependentManifests []*manifest2.Manifest, dependentBlobSets []*manifest2.BlobSet) ([]*registryv1alpha1.ModulePackage, e.ResponseError)
}

func NewProtoParser() ProtoParser {
	return &ProtoParserImpl{}
}

type ProtoParserImpl struct{}

func (protoParser *ProtoParserImpl) GetPackages(ctx context.Context, fileManifest *manifest2.Manifest, blobSet *manifest2.BlobSet, dependentManifests []*manifest2.Manifest, dependentBlobSets []*manifest2.BlobSet) ([]*registryv1alpha1.ModulePackage, e.ResponseError) {
	module, dependentModules, err := protoParser.getModules(ctx, fileManifest, blobSet, dependentManifests, dependentBlobSets)
	if err != nil {
		return nil, e.NewInternalError(err)
	}

	// 编译proto文件
	linkers, _, err := protoParser.compile(ctx, fileManifest, module, dependentModules)
	if err != nil {
		return nil, e.NewInternalError(err)
	}

	packagesSet := map[string]struct{}{}
	for _, link := range linkers {
		packagesSet[string(link.Package())] = struct{}{}
	}

	modulePackages := make([]*registryv1alpha1.ModulePackage, 0, len(packagesSet))
	for packageName := range packagesSet {
		modulePackages = append(modulePackages, &registryv1alpha1.ModulePackage{
			Name: packageName,
		})
	}

	return modulePackages, nil
}

func (protoParser *ProtoParserImpl) GetPackageDocumentation(ctx context.Context, packageName string, moduleIdentity bufmoduleref.ModuleIdentity, commitName string, fileManifest *manifest2.Manifest, blobSet *manifest2.BlobSet, dependentIdentities []bufmoduleref.ModuleIdentity, dependentCommits []string, dependentManifests []*manifest2.Manifest, dependentBlobSets []*manifest2.BlobSet) (*registryv1alpha1.PackageDocumentation, e.ResponseError) {
	module, dependentModules, err := protoParser.getModulesWithModuleIdentityAndCommit(ctx, moduleIdentity, commitName, fileManifest, blobSet, dependentIdentities, dependentCommits, dependentManifests, dependentBlobSets)
	if err != nil {
		return nil, e.NewInternalError(err)
	}

	// 编译proto文件
	linkers, parserAccessorHandler, err := protoParser.compile(ctx, fileManifest, module, dependentModules)
	if err != nil {
		return nil, e.NewInternalError(err)
	}

	// 生成package文档
	documentGenerator := NewDocumentGenerator(commitName, linkers, parserAccessorHandler)
	return documentGenerator.GenerateDocument(packageName), nil
}

func (protoParser *ProtoParserImpl) TryCompile(ctx context.Context, fileManifest *manifest2.Manifest, blobSet *manifest2.BlobSet, dependentManifests []*manifest2.Manifest, dependentBlobSets []*manifest2.BlobSet) e.ResponseError {
	module, dependentModules, err := protoParser.getModules(ctx, fileManifest, blobSet, dependentManifests, dependentBlobSets)
	if err != nil {
		return e.NewInternalError(err)
	}

	// 尝试编译，查看是否成功
	_, _, err = protoParser.compile(ctx, fileManifest, module, dependentModules)
	if err != nil {
		return e.NewInternalError(err)
	}

	return nil
}

func (protoParser *ProtoParserImpl) getModules(ctx context.Context, fileManifest *manifest2.Manifest, blobSet *manifest2.BlobSet, dependentManifests []*manifest2.Manifest, dependentBlobSets []*manifest2.BlobSet) (bufmodule.Module, []bufmodule.Module, error) {
	module, err := bufmodule.NewModuleForManifestAndBlobSet(ctx, fileManifest, blobSet)
	if err != nil {
		return nil, nil, err
	}
	dependentModules := make([]bufmodule.Module, 0, len(dependentManifests))
	for i := 0; i < len(dependentManifests); i++ {
		dependentModule, err := bufmodule.NewModuleForManifestAndBlobSet(ctx, dependentManifests[i], dependentBlobSets[i])
		if err != nil {
			return nil, nil, err
		}
		dependentModules = append(dependentModules, dependentModule)
	}

	return module, dependentModules, nil
}

func (protoParser *ProtoParserImpl) getModulesWithModuleIdentityAndCommit(ctx context.Context, moduleIdentity bufmoduleref.ModuleIdentity, commit string, fileManifest *manifest2.Manifest, blobSet *manifest2.BlobSet, dependentIdentities []bufmoduleref.ModuleIdentity, dependentCommits []string, dependentManifests []*manifest2.Manifest, dependentBlobSets []*manifest2.BlobSet) (bufmodule.Module, []bufmodule.Module, error) {
	module, err := bufmodule.NewModuleForManifestAndBlobSet(ctx, fileManifest, blobSet, bufmodule.ModuleWithModuleIdentityAndCommit(moduleIdentity, commit))
	if err != nil {
		return nil, nil, err
	}
	dependentModules := make([]bufmodule.Module, 0, len(dependentManifests))
	for i := 0; i < len(dependentManifests); i++ {
		dependentModule, err := bufmodule.NewModuleForManifestAndBlobSet(ctx, dependentManifests[i], dependentBlobSets[i], bufmodule.ModuleWithModuleIdentityAndCommit(dependentIdentities[i], dependentCommits[i]))
		if err != nil {
			return nil, nil, err
		}
		dependentModules = append(dependentModules, dependentModule)
	}

	return module, dependentModules, nil
}

func (protoParser *ProtoParserImpl) compile(ctx context.Context, fileManifest *manifest2.Manifest, module bufmodule.Module, dependentModules []bufmodule.Module) (linker.Files, bufmoduleprotocompile.ParserAccessorHandler, error) {
	moduleFileSet := bufmodule.NewModuleFileSet(module, dependentModules)
	parserAccessorHandler := bufmoduleprotocompile.NewParserAccessorHandler(ctx, moduleFileSet)
	compiler := protocompile.Compiler{
		MaxParallelism: thread.Parallelism(),
		SourceInfoMode: protocompile.SourceInfoStandard,
		Resolver:       &protocompile.SourceResolver{Accessor: parserAccessorHandler.Open},
	}

	// fileDescriptors are in the same order as paths per the documentation
	protoPaths := protoParser.getProtoPaths(fileManifest)
	linkers, err := compiler.Compile(ctx, protoPaths...)
	if err != nil {
		return nil, nil, err
	}

	return linkers, parserAccessorHandler, nil
}

func (protoParser *ProtoParserImpl) getProtoPaths(fileManifest *manifest2.Manifest) []string {
	var protoPaths []string
	_ = fileManifest.Range(func(path string, digest manifest2.Digest) error {
		if strings.HasSuffix(path, ".proto") {
			protoPaths = append(protoPaths, path)
		}

		return nil
	})

	return protoPaths
}
