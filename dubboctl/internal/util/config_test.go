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

package util_test

import (
	"os"
	"path/filepath"
	"testing"
)

import (
	. "github.com/apache/dubbo-kubernetes/dubboctl/internal/testing"
	config "github.com/apache/dubbo-kubernetes/dubboctl/internal/util"
)

// TestPath ensures that the Path accessor returns
// XDG_CONFIG_HOME/.config/dubbo
func TestPath(t *testing.T) {
	home := t.TempDir()                  // root of all configs
	path := filepath.Join(home, "dubbo") // our config

	t.Setenv("XDG_CONFIG_HOME", home)

	if config.Dir() != path {
		t.Fatalf("expected config path '%v', got '%v'", path, config.Dir())
	}
}

// TestCreatePaths ensures that the paths are created when requested.
func TestCreatePaths(t *testing.T) {
	home, cleanup := Mktemp(t)
	t.Cleanup(cleanup)

	t.Setenv("XDG_CONFIG_HOME", home)

	if err := config.CreatePaths(); err != nil {
		t.Fatal(err)
	}

	if _, err := os.Stat(config.Dir()); err != nil {
		if os.IsNotExist(err) {
			t.Fatalf("config path '%v' not created", config.Dir())
		}
		t.Fatal(err)
	}

	if _, err := os.Stat(filepath.Join(config.Dir(), "repositories")); err != nil {
		if os.IsNotExist(err) {
			t.Fatalf("config path '%v' not created", config.Dir())
		}
		t.Fatal(err)
	}

	// Trying to create when repositories path is invalid should error
	_ = os.WriteFile("./invalidRepositoriesPath.txt", []byte{}, os.ModePerm)
	t.Setenv("DUBBO_REPOSITORIES_PATH", "./invalidRepositoriesPath.txt")
	if err := config.CreatePaths(); err == nil {
		t.Fatal("did not receive error when creating paths with an invalid FUNC_REPOSITORIES_PATH")
	}

	// Trying to Create config path should bubble errors, for example when HOME is
	// set to a nonexistent path.
	_ = os.WriteFile("./invalidConfigHome.txt", []byte{}, os.ModePerm)
	t.Setenv("XDG_CONFIG_HOME", "./invalidConfigHome.txt")
	if err := config.CreatePaths(); err == nil {
		t.Fatal("did not receive error when creating paths in an invalid home")
	}
}

// TestRepositoriesPath returns the path expected
// (XDG_CONFIG_HOME/dubbo/repositories by default)
func TestRepositoriesPath(t *testing.T) {
	home, cleanup := Mktemp(t)
	t.Cleanup(cleanup)
	t.Setenv("XDG_CONFIG_HOME", home)

	expected := filepath.Join(home, "dubbo", config.Repositories)
	if config.RepositoriesPath() != expected {
		t.Fatalf("unexpected reposiories path: %v", config.RepositoriesPath())
	}
}
