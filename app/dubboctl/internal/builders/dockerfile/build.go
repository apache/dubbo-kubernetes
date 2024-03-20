// Licensed to the Apache Software Foundation (ASF) under one or more
// contributor license agreements.  See the NOTICE file distributed with
// this work for additional information regarding copyright ownership.
// The ASF licenses this file to You under the Apache License, Version 2.0
// (the "License"); you may not use this file except in compliance with
// the License.  You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package dockerfile

import (
	"context"
	"os"
)

import (
	"github.com/containers/storage/pkg/archive"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/jsonmessage"

	"github.com/moby/term"
)

import (
	"github.com/apache/dubbo-kubernetes/app/dubboctl/internal/docker"
	"github.com/apache/dubbo-kubernetes/app/dubboctl/internal/dubbo"
)

type Builder struct{}

func (b Builder) Build(ctx context.Context, f *dubbo.Dubbo) error {
	cli, _, err := docker.NewClient(client.DefaultDockerHost)
	if err != nil {
		return err
	}
	buildOpts := types.ImageBuildOptions{
		Dockerfile: "Dockerfile",
		Tags:       []string{f.Image},
	}

	buildCtx, _ := archive.TarWithOptions(f.Root, &archive.TarOptions{})
	resp, err := cli.ImageBuild(ctx, buildCtx, buildOpts)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	termFd, isTerm := term.GetFdInfo(os.Stderr)
	err = jsonmessage.DisplayJSONMessagesStream(resp.Body, os.Stderr, termFd, isTerm, nil)
	if err != nil {
		return err
	}
	return nil
}

func NewBuilder() *Builder {
	return &Builder{}
}
