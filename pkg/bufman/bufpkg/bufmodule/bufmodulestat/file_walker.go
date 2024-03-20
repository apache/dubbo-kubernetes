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

package bufmodulestat

import (
	"context"
	"io"
)

import (
	"go.uber.org/multierr"
)

import (
	"github.com/apache/dubbo-kubernetes/pkg/bufman/bufpkg/bufmodule"
)

type fileWalker struct {
	module bufmodule.Module
}

func newFileWalker(module bufmodule.Module) *fileWalker {
	return &fileWalker{
		module: module,
	}
}

func (f *fileWalker) Walk(ctx context.Context, fu func(io.Reader) error) error {
	fileInfos, err := f.module.TargetFileInfos(ctx)
	if err != nil {
		return err
	}
	for _, fileInfo := range fileInfos {
		moduleFile, err := f.module.GetModuleFile(ctx, fileInfo.Path())
		if err != nil {
			return err
		}
		if err := fu(moduleFile); err != nil {
			return multierr.Append(err, moduleFile.Close())
		}
		if err := moduleFile.Close(); err != nil {
			return err
		}
	}
	return nil
}
