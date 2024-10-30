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

package docker_test

import (
	"context"
	"fmt"
	"testing"
	"time"
)

import (
	"github.com/Microsoft/go-winio"

	"github.com/docker/docker/client"
)

import (
	"github.com/apache/dubbo-kubernetes/dubboctl/operator/docker"
)

func TestNewClientWinPipe(t *testing.T) {
	const testNPipe = "test-npipe"

	startMockDaemonWinPipe(t, testNPipe)
	t.Setenv("DOCKER_HOST", fmt.Sprintf("npipe:////./pipe/%s", testNPipe))

	ctx, cancel := context.WithTimeout(context.Background(), time.Minute*1)
	defer cancel()

	dockerClient, dockerHostToMount, err := docker.NewClient(client.DefaultDockerHost)
	if err != nil {
		t.Error(err)
	}
	defer dockerClient.Close()

	if dockerHostToMount != "" {
		t.Error("dockerHostToMount should be empty for npipe")
	}

	_, err = dockerClient.Ping(ctx)
	if err != nil {
		t.Error(err)
	}
}

func startMockDaemonWinPipe(t *testing.T, pipeName string) {
	p, err := winio.ListenPipe(`\\.\pipe\`+pipeName, nil)
	if err != nil {
		t.Fatal(err)
	}
	startMockDaemon(t, p)
}
