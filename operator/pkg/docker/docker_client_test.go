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
	"github.com/apache/dubbo-kubernetes/operator/pkg/docker"
	"net"
	"net/http"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

import (
	"github.com/docker/docker/client"
)

// Test that we are creating client in accordance
// with the DOCKER_HOST environment variable
func TestNewClient(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("we can't do it on the windows")
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Minute*1)
	defer cancel()

	tmpDir := t.TempDir()
	sock := filepath.Join(tmpDir, "docker.sock")
	dockerHost := fmt.Sprintf("unix://%s", sock)

	startMockDaemonUnix(t, sock)

	t.Setenv("DOCKER_HOST", dockerHost)

	dockerClient, dockerHostInRemote, err := docker.NewClient(client.DefaultDockerHost)
	if err != nil {
		t.Error(err)
	}
	defer dockerClient.Close()

	if runtime.GOOS == "linux" && dockerHostInRemote != dockerHost {
		t.Errorf("unexpected dockerHostInRemote: expected %q, but got %q", dockerHost, dockerHostInRemote)
	}
	if runtime.GOOS == "darwin" && dockerHostInRemote != "" {
		t.Errorf("unexpected dockerHostInRemote: expected empty string, but got %q", dockerHostInRemote)
	}

	_, err = dockerClient.Ping(ctx)
	if err != nil {
		t.Error(err)
	}
}

func TestNewClient_DockerHost(t *testing.T) {
	tests := []struct {
		name                     string
		dockerHostEnvVar         string
		expectedRemoteDockerHost map[string]string
	}{
		{
			name:                     "tcp",
			dockerHostEnvVar:         "tcp://10.0.0.1:1234",
			expectedRemoteDockerHost: map[string]string{"darwin": "", "windows": "", "linux": ""},
		},
		{
			name:                     "unix",
			dockerHostEnvVar:         "unix:///some/path/docker.sock",
			expectedRemoteDockerHost: map[string]string{"darwin": "", "windows": "", "linux": "unix:///some/path/docker.sock"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.name == "unix" && runtime.GOOS == "windows" {
				t.Skip("Windows cannot handle Unix sockets")
			}

			t.Setenv("DOCKER_HOST", tt.dockerHostEnvVar)
			_, host, err := docker.NewClient(client.DefaultDockerHost)
			if err != nil {
				t.Fatal(err)
			}
			expectedRemoteDockerHost := tt.expectedRemoteDockerHost[runtime.GOOS]
			if host != expectedRemoteDockerHost {
				t.Errorf("expected docker host %q, but got %q", expectedRemoteDockerHost, host)
			}
		})
	}
}

func startMockDaemon(t *testing.T, listener net.Listener) {
	server := http.Server{}
	// mimics /_ping endpoint
	server.Handler = http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Add("Content-Type", "text/plain")
		writer.WriteHeader(200)
		_, _ = writer.Write([]byte("OK"))
	})

	serErrChan := make(chan error)
	go func() {
		serErrChan <- server.Serve(listener)
	}()
	t.Cleanup(func() {
		err := server.Shutdown(context.Background())
		if err != nil {
			t.Fatal(err)
		}
		err = <-serErrChan
		if err != nil && !strings.Contains(err.Error(), "Server closed") {
			t.Fatal(err)
		}
	})
}

func startMockDaemonUnix(t *testing.T, sock string) {
	l, err := net.Listen("unix", sock)
	if err != nil {
		t.Fatal(err)
	}
	startMockDaemon(t, l)
}
