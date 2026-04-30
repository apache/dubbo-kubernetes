//
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

package app

import (
	"bytes"
	"os"
	"strings"
	"testing"

	"github.com/apache/dubbo-kubernetes/pkg/log"
)

func TestRootCommandRegistersRenamedCommands(t *testing.T) {
	root := NewRootCommand()
	commands := map[string]bool{}
	for _, command := range root.Commands() {
		commands[command.Name()] = true
	}

	for _, name := range []string{"startup", "xclient"} {
		if !commands[name] {
			t.Fatalf("expected command %q to be registered; commands=%v", name, commands)
		}
	}
	for _, name := range []string{"discovery", "xds-client"} {
		if commands[name] {
			t.Fatalf("old command %q is still registered; commands=%v", name, commands)
		}
	}
}

func TestCommandsSetLogScopes(t *testing.T) {
	defer func() {
		log.SetDefaultScope("log")
		for _, name := range []string{"log", startupLogScope, waitLogScope, xclientLogScope} {
			if scope := log.FindScope(name); scope != nil {
				scope.SetOutput(os.Stderr)
			}
		}
	}()

	root := NewRootCommand()
	for _, tt := range []struct {
		command string
		scope   string
	}{
		{command: "startup", scope: "setup"},
		{command: "wait", scope: "wait"},
		{command: "xclient", scope: "xclient"},
	} {
		t.Run(tt.command, func(t *testing.T) {
			var out bytes.Buffer
			command, _, err := root.Find([]string{tt.command})
			if err != nil {
				t.Fatal(err)
			}
			if command.PreRunE == nil {
				t.Fatalf("%s command has no PreRunE", tt.command)
			}
			if err := command.PreRunE(command, nil); err != nil {
				t.Fatal(err)
			}

			scope := log.FindScope(tt.scope)
			if scope == nil {
				t.Fatalf("scope %q was not registered", tt.scope)
			}
			scope.SetOutput(&out)
			log.Info("scope check")

			got := out.String()
			if !strings.Contains(got, tt.scope) {
				t.Fatalf("log output %q does not contain scope %q", got, tt.scope)
			}
			if strings.Contains(got, " default ") {
				t.Fatalf("log output still uses default scope: %q", got)
			}
		})
	}
}
