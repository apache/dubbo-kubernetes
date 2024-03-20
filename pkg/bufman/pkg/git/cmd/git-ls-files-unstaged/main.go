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

// Package main implements a file lister for git that lists unstaged files.
//
// See the documentation for git.Lister for more details.
package main

import (
	"context"
	"strings"
)

import (
	"github.com/apache/dubbo-kubernetes/pkg/bufman/pkg/app"
	"github.com/apache/dubbo-kubernetes/pkg/bufman/pkg/command"
	"github.com/apache/dubbo-kubernetes/pkg/bufman/pkg/git"
)

func main() {
	app.Main(context.Background(), run)
}

func run(ctx context.Context, container app.Container) error {
	files, err := git.NewLister(command.NewRunner()).ListFilesAndUnstagedFiles(
		ctx,
		container,
		git.ListFilesAndUnstagedFilesOptions{},
	)
	if err != nil {
		return err
	}
	if len(files) > 0 {
		if _, err := container.Stdout().Write([]byte(strings.Join(files, "\n") + "\n")); err != nil {
			return err
		}
	}
	return nil
}
