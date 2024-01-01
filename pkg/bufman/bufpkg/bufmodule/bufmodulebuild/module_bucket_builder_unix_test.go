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

//go:build aix || darwin || dragonfly || freebsd || (js && wasm) || linux || netbsd || openbsd || solaris
// +build aix darwin dragonfly freebsd js,wasm linux netbsd openbsd solaris

package bufmodulebuild

import (
	"testing"

	"github.com/apache/dubbo-kubernetes/pkg/bufman/bufpkg/bufmodule/bufmoduletesting"
)

func TestLicenseSymlink(t *testing.T) {
	t.Parallel()
	testLicenseBucket(
		t,
		"testdata/6",
		"Test Module License", // expecting the same license with testdata/5 as it symlink to the license there
		bufmoduletesting.NewFileInfo(t, "proto/1.proto", "testdata/6/proto/1.proto", false, nil, ""),
		bufmoduletesting.NewFileInfo(t, "proto/a/2.proto", "testdata/6/proto/a/2.proto", false, nil, ""),
	)
}
