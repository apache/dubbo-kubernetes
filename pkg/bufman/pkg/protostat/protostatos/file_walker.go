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

package protostatos

import (
	"context"
	"io"
	"os"
	"path/filepath"
)

import (
	"go.uber.org/multierr"
)

type fileWalker struct {
	filenames []string
}

func newFileWalker(filenames []string) *fileWalker {
	return &fileWalker{
		filenames: filenames,
	}
}

func (f *fileWalker) Walk(ctx context.Context, fu func(io.Reader) error) error {
	for _, filename := range f.filenames {
		if filepath.Ext(filename) != ".proto" {
			continue
		}
		file, err := os.Open(filename)
		if err != nil {
			return err
		}
		if err := fu(file); err != nil {
			return multierr.Append(err, file.Close())
		}
		if err := file.Close(); err != nil {
			return err
		}
	}
	return nil
}
