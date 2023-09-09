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

import "testing"

// TestLanguages_Default ensures that the default behavior of listing
// all supported languages is to print a plain text list of all the builtin
// language runtimes.
func TestLanguages_Default(t *testing.T) {
	_ = fromTempDirectory(t)

	buf := piped(t) // gather output
	cmd := getRootCmd([]string{"languages"})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}

	expected := `go
java`
	output := buf()
	if output != expected {
		t.Fatalf("expected:\n'%v'\ngot:\n'%v'\n", expected, output)
	}
}

// TestLanguages_JSON ensures that listing languages in --json format returns
// builtin languages as a JSON array.
func TestLanguages_JSON(t *testing.T) {
	_ = fromTempDirectory(t)

	buf := piped(t) // gather output
	cmd := getRootCmd([]string{"languages", "--json"})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}

	expected := `[
  "go",
  "java"
]`
	output := buf()
	if output != expected {
		t.Fatalf("expected:\n%v\ngot:\n%v\n", expected, output)
	}
}
