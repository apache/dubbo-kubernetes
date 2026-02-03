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

package env

import (
	"path/filepath"
	"runtime"
)

var (
	// REPO_ROOT environment variable
	// nolint: revive, stylecheck
	REPO_ROOT Variable = "REPO_ROOT"

	DubboSrc = REPO_ROOT.ValueOrDefaultFunc(getDefaultDubboSrc)
)

var (
	_, b, _, _ = runtime.Caller(0)

	// Root folder of this project
	// This relies on the fact this file is 3 levels up from the root; if this changes, adjust the path below
	Root = filepath.Clean(filepath.Join(filepath.Dir(b), "../.."))
)

func getDefaultDubboSrc() string {
	return Root
}
