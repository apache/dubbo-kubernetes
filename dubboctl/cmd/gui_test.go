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

package cmd

import (
	"strings"
	"testing"
)

func TestBuildGUIURLs(t *testing.T) {
	tests := []struct {
		name        string
		listen      string
		basePath    string
		wantConsole string
		wantHealth  string
	}{
		{
			name:        "loopback and root base path",
			listen:      "127.0.0.1:8080",
			basePath:    "/",
			wantConsole: "http://127.0.0.1:8080/",
			wantHealth:  "http://127.0.0.1:8080/healthz",
		},
		{
			name:        "wildcard listen address",
			listen:      ":8080",
			basePath:    "/console",
			wantConsole: "http://127.0.0.1:8080/console/",
			wantHealth:  "http://127.0.0.1:8080/console/healthz",
		},
		{
			name:        "base path without leading slash",
			listen:      "0.0.0.0:9090",
			basePath:    "dashboard",
			wantConsole: "http://127.0.0.1:9090/dashboard/",
			wantHealth:  "http://127.0.0.1:9090/dashboard/healthz",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			gotConsole, gotHealth, err := buildGUIURLs(test.listen, test.basePath)
			if err != nil {
				t.Fatalf("buildGUIURLs() returned error: %v", err)
			}
			if gotConsole != test.wantConsole {
				t.Fatalf("buildGUIURLs() console URL = %q, want %q", gotConsole, test.wantConsole)
			}
			if gotHealth != test.wantHealth {
				t.Fatalf("buildGUIURLs() health URL = %q, want %q", gotHealth, test.wantHealth)
			}
		})
	}
}

func TestBuildGUIURLsRejectsEmptyListen(t *testing.T) {
	if _, _, err := buildGUIURLs("  ", "/"); err == nil {
		t.Fatal("buildGUIURLs() with empty listen address returned no error")
	}
}

func TestConsoleArgs(t *testing.T) {
	tests := []struct {
		name string
		args guiArgs
		want string
	}{
		{
			name: "defaults fall back to kubeconfig discovery",
			args: guiArgs{listen: "127.0.0.1:8080", basePath: "/"},
			want: "--listen 127.0.0.1:8080 --base-path / --discover-kubeconfig",
		},
		{
			name: "explicit discovery suppresses the fallback",
			args: guiArgs{
				listen:    "127.0.0.1:8080",
				basePath:  "/",
				contexts:  []string{"east", "west"},
				endpoints: []string{"local=http://127.0.0.1:26080"},
			},
			want: "--listen 127.0.0.1:8080 --base-path / --context east --context west --endpoint local=http://127.0.0.1:26080",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := strings.Join(test.args.consoleArgs(nil), " ")
			if got != test.want {
				t.Fatalf("consoleArgs() = %q, want %q", got, test.want)
			}
		})
	}
}

func TestConsoleArgsAppendsExtraArgs(t *testing.T) {
	args := guiArgs{listen: "127.0.0.1:8080", basePath: "/"}
	got := args.consoleArgs([]string{"--upstream-timeout", "10s"})
	want := "--listen 127.0.0.1:8080 --base-path / --discover-kubeconfig --upstream-timeout 10s"
	if strings.Join(got, " ") != want {
		t.Fatalf("consoleArgs() = %q, want %q", strings.Join(got, " "), want)
	}
}
