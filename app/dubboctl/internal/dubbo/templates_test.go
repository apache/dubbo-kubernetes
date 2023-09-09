//go:build !integration
// +build !integration

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

package dubbo_test

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/apache/dubbo-kubernetes/app/dubboctl/internal/dubbo"
	. "github.com/apache/dubbo-kubernetes/app/dubboctl/internal/testing"

	"github.com/google/go-cmp/cmp"
)

// TestTemplates_List ensures that all templates are listed taking into account
// both internal and extensible (prefixed) repositories.
func TestTemplates_List(t *testing.T) {
	// A client which specifies a location of exensible repositoreis on disk
	// will list all builtin plus exensible
	client := dubbo.New(dubbo.WithRepositoriesPath("testdata/repositories"))

	// list templates for the "go" runtime
	templates, err := client.Templates().List("go")
	if err != nil {
		t.Fatal(err)
	}

	// Note that this list will change as the customTemplateRepo
	// and builtin templates are shared.  THis could be mitigated
	// by creating a custom repository path for just this test, if
	// that becomes a hassle.
	expected := []string{
		"common",
		"customTemplateRepo/customTemplate",
	}

	if diff := cmp.Diff(expected, templates); diff != "" {
		t.Error("Unexpected templates (-want, +got):", diff)
	}
}

// TestTemplates_List_ExtendedNotFound ensures that an error is not returned
// when retrieving the list of templates for a runtime that does not exist
// in an extended repository, but does in the default.
func TestTemplates_List_ExtendedNotFound(t *testing.T) {
	client := dubbo.New(dubbo.WithRepositoriesPath("testdata/repositories"))

	// list templates for the "python" runtime -
	// not supplied by the extended repos
	templates, err := client.Templates().List("java")
	if err != nil {
		t.Fatal(err)
	}

	expected := []string{
		"common",
	}

	if diff := cmp.Diff(expected, templates); diff != "" {
		t.Error("Unexpected templates (-want, +got):", diff)
	}
}

// TestTemplates_Get ensures that a template's metadata object can
// be retrieved by full name (full name prefix optional for embedded).
func TestTemplates_Get(t *testing.T) {
	client := dubbo.New(dubbo.WithRepositoriesPath("testdata/repositories"))

	// Check embedded
	embedded, err := client.Templates().Get("go", "common")
	if err != nil {
		t.Fatal(err)
	}

	if embedded.Runtime() != "go" || embedded.Repository() != "default" || embedded.Name() != "common" {
		t.Logf("Expected template from embedded to have runtime 'go' repo 'default' name 'common', got '%v', '%v', '%v',",
			embedded.Runtime(), embedded.Repository(), embedded.Name())
	}

	// Check extended
	extended, err := client.Templates().Get("go", "customTemplateRepo/customTemplate")
	if err != nil {
		t.Fatal(err)
	}

	if embedded.Runtime() != "go" || embedded.Repository() != "default" || embedded.Name() != "common" {
		t.Logf("Expected template from extended repo to have runtime 'go' repo 'customTemplateRepo' name 'customTemplate', got '%v', '%v', '%v',",
			extended.Runtime(), extended.Repository(), extended.Name())
	}
}

// TestTemplates_Embedded ensures that embedded templates are copied on write.
func TestTemplates_Embedded(t *testing.T) {
	// create test directory
	root := "testdata/testTemplatesEmbedded"
	defer Using(t, root)()

	// Client whose internal (builtin default) templates will be used.
	client := dubbo.New()

	// write out a template
	_, err := client.Init(&dubbo.Dubbo{
		Root:     root,
		Runtime:  TestRuntime,
		Template: "common",
	})
	if err != nil {
		t.Fatal(err)
	}

	// Assert file exists as expected
	_, err = os.Stat(filepath.Join(root, "go.mod"))
	if err != nil {
		t.Fatal(err)
	}
}

// TestTemplates_Custom ensures that a template from a filesystem source
// (ie. custom provider on disk) can be specified as the source for a
// template.
func TestTemplates_Custom(t *testing.T) {
	// Create test directory
	root := "testdata/testTemplatesCustom"
	defer Using(t, root)()

	// Client which uses custom repositories
	// in form [provider]/[template], on disk the template is
	// at: testdata/repositories/[provider]/[runtime]/[template]
	client := dubbo.New(
		dubbo.WithRepositoriesPath("testdata/repositories"))

	// Create a function specifying a template from
	// the custom provider's directory in the on-disk template repo.
	_, err := client.Init(&dubbo.Dubbo{
		Root:     root,
		Runtime:  "customRuntime",
		Template: "customTemplateRepo/customTemplate",
	})
	if err != nil {
		t.Fatal(err)
	}

	// Assert file exists as expected
	_, err = os.Stat(filepath.Join(root, "custom.impl"))
	if err != nil {
		t.Fatal(err)
	}
}

// TestTemplates_Remote ensures that a Git template repository provided via URI
// can be specificed on creation of client, with subsequent calls to Create
// using this remote by default.
func TestTemplates_Remote(t *testing.T) {
	var err error

	root := "testdata/testTemplatesRemote"
	defer Using(t, root)()

	url := ServeRepo(RepositoriesTestRepo, t)

	// Create a client which explicitly specifies the Git repo at URL
	// rather than relying on the default internally builtin template repo
	client := dubbo.New(
		dubbo.WithRepository(url))

	// Create a default function, which should override builtin and use
	// that from the specified url (git repo)
	_, err = client.Init(&dubbo.Dubbo{
		Root:     root,
		Runtime:  "go",
		Template: "remote",
	})
	if err != nil {
		t.Fatal(err)
	}

	// Assert the sample file from the git repo was written
	_, err = os.Stat(filepath.Join(root, "remote-test"))
	if err != nil {
		t.Fatal(err)
	}
}

// TestTemplates_Default ensures that the expected default template
// is used when none specified.
func TestTemplates_Default(t *testing.T) {
	// create test directory
	root := "testdata/testTemplates_Default"
	defer Using(t, root)()

	client := dubbo.New()

	// The runtime is specified, and explicitly includes a
	// file for the default template of fn.DefaultTemplate
	_, err := client.Init(&dubbo.Dubbo{Root: root, Runtime: TestRuntime})
	if err != nil {
		t.Fatal(err)
	}

	// Assert file exists as expected
	_, err = os.Stat(filepath.Join(root, "go.mod"))
	if err != nil {
		t.Fatal(err)
	}
}

// TestTemplates_InvalidErrors ensures that specifying unrecgognized
// runtime/template errors
func TestTemplates_InvalidErrors(t *testing.T) {
	// create test directory
	root := "testdata/testTemplates_InvalidErrors"
	defer Using(t, root)()

	client := dubbo.New()

	// Error will be type-checked.
	var err error

	// Test for error writing an invalid runtime
	_, err = client.Init(&dubbo.Dubbo{
		Root:    root,
		Runtime: "invalid",
	})
	if !errors.Is(err, dubbo.ErrRuntimeNotFound) {
		t.Fatalf("Expected ErrRuntimeNotFound, got %v", err)
	}
	os.Remove(filepath.Join(root, ".gitignore"))

	// Test for error writing an invalid template
	_, err = client.Init(&dubbo.Dubbo{
		Root:     root,
		Runtime:  TestRuntime,
		Template: "invalid",
	})
	if !errors.Is(err, dubbo.ErrTemplateNotFound) {
		t.Fatalf("Expected ErrTemplateNotFound, got %v", err)
	}
}

// TestTemplates_ModeEmbedded ensures that templates written from the embedded
// templates retain their mode.
func TestTemplates_ModeEmbedded(t *testing.T) {
	if runtime.GOOS == "windows" {
		return
		// not applicable
	}

	// set up test directory
	root := "testdata/testTemplatesModeEmbedded"
	defer Using(t, root)()

	client := dubbo.New()

	// Write the embedded template that contains a file which
	// needs to be executable (only such is mvnw in quarkus)
	_, err := client.Init(&dubbo.Dubbo{
		Root:     root,
		Runtime:  "quarkus",
		Template: "http",
	})
	if err != nil {
		t.Fatal(err)
	}

	// Verify file mode was preserved
	file, err := os.Stat(filepath.Join(root, "mvnw"))
	if err != nil {
		t.Fatal(err)
	}
	if file.Mode() != os.FileMode(0o755) {
		t.Fatalf("The embedded executable's mode should be 0755 but was %v", file.Mode())
	}
}

// TestTemplates_ModeCustom ensures that templates written from custom templates
// retain their mode.
func TestTemplates_ModeCustom(t *testing.T) {
	if runtime.GOOS == "windows" {
		return // not applicable
	}

	// test directories
	root := "testdata/testTemplates_ModeCustom"
	defer Using(t, root)()

	client := dubbo.New(
		dubbo.WithRepositoriesPath("testdata/repositories"))

	// Write executable from custom repo
	_, err := client.Init(&dubbo.Dubbo{
		Root:     root,
		Runtime:  "test",
		Template: "customTemplateRepo/tplb",
	})
	if err != nil {
		t.Fatal(err)
	}

	// Verify custom file mode was preserved.
	file, err := os.Stat(filepath.Join(root, "executable.sh"))
	if err != nil {
		t.Fatal(err)
	}
	if file.Mode() != os.FileMode(0o755) {
		t.Fatalf("The custom executable file's mode should be 0755 but was %v", file.Mode())
	}
}

// TestTemplates_ModeRemote ensures that templates written from remote templates
// retain their mode.
func TestTemplates_ModeRemote(t *testing.T) {
	var err error

	if runtime.GOOS == "windows" {
		return // not applicable
	}

	// test directories
	root := "testdata/testTemplates_ModeRemote"
	defer Using(t, root)()

	url := ServeRepo(RepositoriesTestRepo, t)

	client := dubbo.New(
		dubbo.WithRepository(url))

	// Write executable from custom repo
	_, err = client.Init(&dubbo.Dubbo{
		Root:     root,
		Runtime:  "node",
		Template: "remote",
	})
	if err != nil {
		t.Fatal(err)
	}

	// Verify directory file mode was preserved
	file, err := os.Stat(filepath.Join(root, "test"))
	if err != nil {
		t.Fatal(err)
	}
	if file.Mode() != os.ModeDir|0o755 {
		t.Fatalf("The remote repositry directory mode should be 0755 but was %#o", file.Mode())
	}

	// Verify remote executible file mode was preserved.
	file, err = os.Stat(filepath.Join(root, "test", "executable.sh"))
	if err != nil {
		t.Fatal(err)
	}
	if file.Mode() != os.FileMode(0o755) {
		t.Fatalf("The remote executable's mode should be 0755 but was %v", file.Mode())
	}
}
