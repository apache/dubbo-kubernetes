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

package bufpluginexec

import (
	"context"
)

import (
	"go.uber.org/zap"

	"google.golang.org/protobuf/types/pluginpb"
)

import (
	"github.com/apache/dubbo-kubernetes/pkg/bufman/bufpkg/bufwasm"
	"github.com/apache/dubbo-kubernetes/pkg/bufman/pkg/app"
	"github.com/apache/dubbo-kubernetes/pkg/bufman/pkg/app/appproto"
	"github.com/apache/dubbo-kubernetes/pkg/bufman/pkg/command"
	"github.com/apache/dubbo-kubernetes/pkg/bufman/pkg/storage/storageos"
)

type generator struct {
	logger             *zap.Logger
	storageosProvider  storageos.Provider
	runner             command.Runner
	wasmPluginExecutor bufwasm.PluginExecutor
}

func newGenerator(
	logger *zap.Logger,
	storageosProvider storageos.Provider,
	runner command.Runner,
	wasmPluginExecutor bufwasm.PluginExecutor,
) *generator {
	return &generator{
		logger:             logger,
		storageosProvider:  storageosProvider,
		runner:             runner,
		wasmPluginExecutor: wasmPluginExecutor,
	}
}

func (g *generator) Generate(
	ctx context.Context,
	container app.EnvStderrContainer,
	pluginName string,
	requests []*pluginpb.CodeGeneratorRequest,
	options ...GenerateOption,
) (_ *pluginpb.CodeGeneratorResponse, retErr error) {
	generateOptions := newGenerateOptions()
	for _, option := range options {
		option(generateOptions)
	}
	handlerOptions := []HandlerOption{
		HandlerWithPluginPath(generateOptions.pluginPath...),
		HandlerWithProtocPath(generateOptions.protocPath),
	}
	if generateOptions.wasmEnabled {
		handlerOptions = append(
			handlerOptions,
			HandlerWithWASMEnabled(),
		)
	}
	handler, err := NewHandler(
		g.storageosProvider,
		g.runner,
		g.wasmPluginExecutor,
		pluginName,
		handlerOptions...,
	)
	if err != nil {
		return nil, err
	}
	return appproto.NewGenerator(
		g.logger,
		handler,
	).Generate(
		ctx,
		container,
		requests,
	)
}

type generateOptions struct {
	pluginPath  []string
	protocPath  string
	wasmEnabled bool
}

func newGenerateOptions() *generateOptions {
	return &generateOptions{}
}
