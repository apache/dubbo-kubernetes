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

package bufpluginexec

import (
	"errors"
	"os"
	"testing"
)

import (
	"github.com/stretchr/testify/assert"
)

func TestValidateWASMFilePath(t *testing.T) {
	t.Parallel()
	wasmPath := t.TempDir() + "/test.wasm"
	notWasmPath := t.TempDir() + "/test.txt"
	assert.NoError(t, os.WriteFile(wasmPath, []byte("a"), 0o600))
	assert.NoError(t, os.WriteFile(notWasmPath, []byte("a"), 0o600))
	t.Run("pass for valid wasm", func(t *testing.T) {
		t.Parallel()
		_, err := validateWASMFilePath(wasmPath)
		assert.NoError(t, err)
	})
	t.Run("fail if not found", func(t *testing.T) {
		t.Parallel()
		_, err := validateWASMFilePath("notfound")
		assert.True(t, errors.Is(err, os.ErrNotExist))
	})
	t.Run("fail if invalid extension", func(t *testing.T) {
		t.Parallel()
		_, err := validateWASMFilePath(notWasmPath)
		assert.Error(t, err)
	})
}
