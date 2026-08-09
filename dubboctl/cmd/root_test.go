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

package cmd

import (
	"strings"
	"testing"
)

func TestRootCommandDoesNotExposeRemovedCommands(t *testing.T) {
	for _, name := range []string{"admin", "create", "dashboard", "image", "repo"} {
		t.Run(name, func(t *testing.T) {
			root := GetRootCmd([]string{name})
			if err := root.Execute(); err == nil || !strings.Contains(err.Error(), "unknown command") {
				t.Fatalf("%q command error = %v, want unknown command", name, err)
			}
		})
	}
}

func TestRootCommandHelpDoesNotListRemovedCommands(t *testing.T) {
	root := GetRootCmd([]string{"--help"})
	var output strings.Builder
	root.SetOut(&output)
	root.SetErr(&output)
	if err := root.Execute(); err != nil {
		t.Fatalf("root help failed: %v", err)
	}

	for _, name := range []string{"admin", "create", "dashboard", "image", "repo"} {
		if strings.Contains(output.String(), "\n  "+name+" ") {
			t.Fatalf("%q command found in root help:\n%s", name, output.String())
		}
	}
}
