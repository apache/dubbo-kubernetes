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

package bufbreaking

import (
	"context"

	"github.com/apache/dubbo-kubernetes/pkg/bufman/bufpkg/bufanalysis"
	"github.com/apache/dubbo-kubernetes/pkg/bufman/bufpkg/bufcheck/bufbreaking/bufbreakingconfig"
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

func newHandler(
	logger *zap.Logger,
) *handler {
	return &handler{
		logger: logger,
		// comment ignores are not allowed for breaking changes
		// so do not set the ignore prefix per the RunnerWithIgnorePrefix comments
		runner: internal.NewRunner(logger),
	}
}

func (h *handler) Check(
	ctx context.Context,
	config *bufbreakingconfig.Config,
	previousImage bufimage.Image,
	image bufimage.Image,
) ([]bufanalysis.FileAnnotation, error) {
	previousFiles, err := protosource.NewFilesUnstable(ctx, bufimageutil.NewInputFiles(previousImage.Files())...)
	if err != nil {
		return nil, err
	}
	files, err := protosource.NewFilesUnstable(ctx, bufimageutil.NewInputFiles(image.Files())...)
	if err != nil {
		return nil, err
	}
	internalConfig, err := internalConfigForConfig(config)
	if err != nil {
		return nil, err
	}
	return h.runner.Check(ctx, internalConfig, previousFiles, files)
}
