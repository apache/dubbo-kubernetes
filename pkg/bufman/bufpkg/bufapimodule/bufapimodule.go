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

package bufapimodule

import (
	"go.uber.org/zap"
)

import (
	"github.com/apache/dubbo-kubernetes/pkg/bufman/bufpkg/bufmodule"
	"github.com/apache/dubbo-kubernetes/pkg/bufman/gen/proto/connect/registry/v1alpha1/registryv1alpha1connect"
	"github.com/apache/dubbo-kubernetes/pkg/bufman/pkg/connectclient"
)

type (
	DownloadServiceClientFactory         func(address string) registryv1alpha1connect.DownloadServiceClient
	RepositoryCommitServiceClientFactory func(address string) registryv1alpha1connect.RepositoryCommitServiceClient
)

func NewDownloadServiceClientFactory(clientConfig *connectclient.Config) DownloadServiceClientFactory {
	return func(address string) registryv1alpha1connect.DownloadServiceClient {
		return connectclient.Make(clientConfig, address, registryv1alpha1connect.NewDownloadServiceClient)
	}
}

func NewRepositoryCommitServiceClientFactory(clientConfig *connectclient.Config) RepositoryCommitServiceClientFactory {
	return func(address string) registryv1alpha1connect.RepositoryCommitServiceClient {
		return connectclient.Make(clientConfig, address, registryv1alpha1connect.NewRepositoryCommitServiceClient)
	}
}

// NewModuleReader returns a new ModuleReader backed by the download service.
func NewModuleReader(
	downloadClientFactory DownloadServiceClientFactory,
	opts ...ModuleReaderOption,
) bufmodule.ModuleReader {
	return newModuleReader(
		downloadClientFactory,
		opts...,
	)
}

// ModuleReaderOption allows configuration of a module reader.
type ModuleReaderOption func(reader *moduleReader)

// NewModuleResolver returns a new ModuleResolver backed by the resolve service.
func NewModuleResolver(
	logger *zap.Logger,
	repositoryCommitClientFactory RepositoryCommitServiceClientFactory,
) bufmodule.ModuleResolver {
	return newModuleResolver(logger, repositoryCommitClientFactory)
}
