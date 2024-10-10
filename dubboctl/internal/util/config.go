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

package util

import (
	"fmt"
	"os"
	"path/filepath"
)

const (
	// Repositories is the default directory for repositories.
	Repositories = "repositories"

	// DefaultLanguage is intentionally undefined.
	DefaultLanguage = ""
)

// Dir is derived in the following order, from lowest
// to highest precedence.
//  1. The default path is the zero value, indicating "no config path available",
//     and users of this package should act accordingly.
//  2. ~/.config/dubbo if it exists (can be expanded: user has a home dir)
//  3. The value of $XDG_CONFIG_PATH/dubbo if the environment variable exists.
//
// The path is created if it does not already exist.
func Dir() (path string) {
	// Use home if available
	if home, err := os.UserHomeDir(); err == nil {
		path = filepath.Join(home, ".config", "dubbo")
	}

	// 'XDG_CONFIG_HOME/dubbo' takes precedence if defined
	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
		path = filepath.Join(xdg, "dubbo")
	}

	return
}

// RepositoriesPath returns the full path at which to look for repositories.
// Use DUBBO_REPOSITORIES_PATH to override default.
func RepositoriesPath() string {
	path := filepath.Join(Dir(), Repositories)
	if e := os.Getenv("DUBBO_REPOSITORIES_PATH"); e != "" {
		path = e
	}
	return path
}

// CreatePaths is a convenience function for creating the on-disk dubbo config
// structure.  All operations should be tolerant of nonexistent disk
// footprint where possible (for example listing repositories should not
// require an extant path, but _adding_ a repository does require that the dubbo
// config structure exist.
// Current structure is:
// ~/.config/dubbo
// ~/.config/dubbo/repositories
func CreatePaths() (err error) {
	if err = os.MkdirAll(Dir(), os.ModePerm); err != nil {
		return fmt.Errorf("error creating global config path: %v", err)
	}
	if err = os.MkdirAll(RepositoriesPath(), os.ModePerm); err != nil {
		return fmt.Errorf("error creating global config repositories path: %v", err)
	}
	return
}
