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
	"errors"
	"testing"

	"github.com/google/go-cmp/cmp"
	"gotest.tools/v3/assert"

	"github.com/apache/dubbo-kubernetes/app/dubboctl/internal/dubbo"
)

// TestTemplates_Default ensures that the default behavior is listing all
// templates for all language runtimes.
func TestTemplates_Default(t *testing.T) {
	_ = fromTempDirectory(t)

	buf := piped(t) // gather output
	cmd := getRootCmd([]string{"templates"})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}

	expected := `LANGUAGE   TEMPLATE
go         common
java       common`

	if d := cmp.Diff(expected, buf()); d != "" {
		t.Error("output missmatch (-want, +got):", d)
	}
}

// TestTemplates_JSON ensures that listing templates respects the --json
// output format.
func TestTemplates_JSON(t *testing.T) {
	_ = fromTempDirectory(t)

	buf := piped(t) // gather output
	cmd := getRootCmd([]string{"templates", "--json"})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}

	expected := `{
  "go": [
    "common"
  ],
  "java": [
    "common"
  ]
}`

	if d := cmp.Diff(expected, buf()); d != "" {
		t.Error("output missmatch (-want, +got):", d)
	}
}

// TestTemplates_ByLanguage ensures that the output is correctly filtered
// by language runtime when provided.
func TestTemplates_ByLanguage(t *testing.T) {
	_ = fromTempDirectory(t)

	cmd := getRootCmd([]string{"templates", "go"})

	// Test plain text
	buf := piped(t)
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}

	expected := `common`

	output := buf()
	if output != expected {
		t.Fatalf("expected plain text:\n'%v'\ngot:\n'%v'\n", expected, output)
	}

	// Test JSON output
	buf = piped(t)
	cmd.SetArgs([]string{"templates", "go", "--json"})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}

	expected = `[
 "common"
]`

	output = buf()
	if output != expected {
		t.Fatalf("expected JSON:\n'%v'\ngot:\n'%v'\n", expected, output)
	}
}

func TestTemplates_ErrTemplateRepoDoesNotExist(t *testing.T) {
	_ = fromTempDirectory(t)

	cmd := getRootCmd([]string{"templates", "--repository", "https://gitee.com/boson-project/repo-does-not-exist"})
	err := cmd.Execute()
	assert.Assert(t, err != nil)
	assert.Assert(t, errors.Is(err, dubbo.ErrTemplateRepoDoesNotExist))
}

func TestTemplates_WrongRepositoryUrl(t *testing.T) {
	_ = fromTempDirectory(t)

	cmd := getRootCmd([]string{"templates", "--repository", "wrong://github.com/boson-project/repo-does-not-exist"})
	err := cmd.Execute()
	assert.Assert(t, err != nil)
}
