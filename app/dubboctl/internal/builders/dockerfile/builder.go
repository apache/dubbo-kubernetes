/*
 * Licensed to the Apache Software Foundation (ASF) under one or more
 * contributor license agreements.  See the NOTICE file distributed with
 * this work for additional information regarding copyright ownership.
 * The ASF licenses this file to You under the Apache License, Version 2.0
 * (the "License"); you may not use this file except in compliance with
 * the License.  You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package dockerfile

import (
	"context"
	"os"
	"os/exec"

	"github.com/apache/dubbo-kubernetes/app/dubboctl/internal/dubbo"
)

type DockerBuilder struct{}

func NewBuilder() *DockerBuilder {
	return &DockerBuilder{}
}

// TODO use docker client go
func (b *DockerBuilder) Build(ctx context.Context, f *dubbo.Dubbo) error {
	c := exec.CommandContext(ctx, "docker", "build", f.Root, "-t", f.Image)
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	return c.Run()
}
