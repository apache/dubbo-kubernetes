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

package cmd

import (
	"github.com/apache/dubbo-kubernetes/dubboctl/pkg/deploy"
	"testing"
)

// TestRepository_List ensures that the 'list' subcommand shows the client's
// set of repositories by name for builtin repositories, by explicitly
// setting the repositories' path to a new path which includes no others.
func TestRepository_List(t *testing.T) {
	_ = fromTempDirectory(t)

	cmd := deploy.NewRepositoryListCmd(deploy.NewClient)
	cmd.SetArgs([]string{}) // Do not use test command args

	// Execute the command, capturing the output sent to stdout
	stdout := piped(t)
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}

	// Assert the output matches expect (whitespace trimmed)
	expect := "default"
	output := stdout()
	if output != expect {
		t.Fatalf("expected:\n'%v'\ngot:\n'%v'\n", expect, output)
	}
}
