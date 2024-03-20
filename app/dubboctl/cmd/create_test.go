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
)

import (
	"github.com/apache/dubbo-kubernetes/app/dubboctl/internal/util"
)

// TestCreate_Execute ensures that an invocation of create with minimal settings
// and valid input completes without error; degenerate case.
func TestCreate_Execute(t *testing.T) {
	_ = fromTempDirectory(t)

	cmd := getRootCmd([]string{"create", "--language", "go", "myfunc"})

	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
}

// TestCreate_NoRuntime ensures that an invocation of create must be
// done with a runtime.
func TestCreate_NoRuntime(t *testing.T) {
	_ = fromTempDirectory(t)

	cmd := getRootCmd([]string{"create", "myfunc"})

	err := cmd.Execute()
	var e ErrNoRuntime
	if !errors.As(err, &e) {
		t.Fatalf("Did not receive ErrNoRuntime. Got %v", err)
	}
}

// TestCreate_WithNoRuntime ensures that an invocation of create must be
// done with one of the valid runtimes only.
func TestCreate_WithInvalidRuntime(t *testing.T) {
	_ = fromTempDirectory(t)

	cmd := getRootCmd([]string{"create", "--language", "invalid", "myfunc"})

	err := cmd.Execute()
	var e ErrInvalidRuntime
	if !errors.As(err, &e) {
		t.Fatalf("Did not receive ErrInvalidRuntime. Got %v", err)
	}
}

// TestCreate_InvalidTemplate ensures that an invocation of create must be
// done with one of the valid templates only.
func TestCreate_InvalidTemplate(t *testing.T) {
	_ = fromTempDirectory(t)

	cmd := getRootCmd([]string{"create", "--language", "go", "--template", "invalid", "myfunc"})

	err := cmd.Execute()
	var e ErrInvalidTemplate
	if !errors.As(err, &e) {
		t.Fatalf("Did not receive ErrInvalidTemplate. Got %v", err)
	}
}

// TestCreate_ValidatesName ensures that the create command only accepts
// DNS-1123 labels for function name.
func TestCreate_ValidatesName(t *testing.T) {
	_ = fromTempDirectory(t)

	// Execute the command with a function name containing invalid characters and
	// confirm the expected error is returned
	cmd := getRootCmd([]string{"create", "invalid!"})
	err := cmd.Execute()
	var e util.ErrInvalidApplicationName
	if !errors.As(err, &e) {
		t.Fatalf("Did not receive ErrInvalidApplicationName. Got %v", err)
	}
}

// TestCreate_ConfigOptional ensures that the system can be used without
// any additional configuration being required.
func TestCreate_ConfigOptional(t *testing.T) {
	_ = fromTempDirectory(t)

	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	cmd := getRootCmd([]string{"create", "--language=go", "myfunc"})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}

	// Not failing is success.  Config files or settings beyond what are
	// automatically written to to the given config home are currently optional.
}
