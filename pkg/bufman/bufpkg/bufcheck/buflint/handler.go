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

package buflint

import (
	"context"

	"github.com/apache/dubbo-kubernetes/pkg/bufman/bufpkg/bufanalysis"
	"github.com/apache/dubbo-kubernetes/pkg/bufman/bufpkg/bufcheck/buflint/buflintconfig"
	"github.com/apache/dubbo-kubernetes/pkg/bufman/bufpkg/bufcheck/buflint/internal/buflintcheck"
	"github.com/apache/dubbo-kubernetes/pkg/bufman/bufpkg/bufcheck/internal"
	"github.com/apache/dubbo-kubernetes/pkg/bufman/bufpkg/bufimage"
	"github.com/apache/dubbo-kubernetes/pkg/bufman/bufpkg/bufimage/bufimageutil"
	"github.com/apache/dubbo-kubernetes/pkg/bufman/pkg/protosource"
	"go.uber.org/zap"
)

type handler struct {
	logger *zap.Logger
	runner *internal.Runner
}

func newHandler(logger *zap.Logger) *handler {
	return &handler{
		logger: logger,
		// linting allows for comment ignores
		// note that comment ignores still need to be enabled within the config
		// for a given check, this just says that comment ignores are allowed
		// in the first place
		runner: internal.NewRunner(
			logger,
			internal.RunnerWithIgnorePrefix(buflintcheck.CommentIgnorePrefix),
		),
	}
}

func (h *handler) Check(
	ctx context.Context,
	config *buflintconfig.Config,
	image bufimage.Image,
) ([]bufanalysis.FileAnnotation, error) {
	files, err := protosource.NewFilesUnstable(ctx, bufimageutil.NewInputFiles(image.Files())...)
	if err != nil {
		return nil, err
	}
	internalConfig, err := internalConfigForConfig(config)
	if err != nil {
		return nil, err
	}
	return h.runner.Check(ctx, internalConfig, nil, files)
}
