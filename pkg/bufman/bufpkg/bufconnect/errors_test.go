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

package bufconnect

import (
	"errors"
	"fmt"
	"testing"
)

import (
	"github.com/stretchr/testify/assert"
)

func TestAuthErrorUnwrap(t *testing.T) {
	t.Parallel()
	cause := errors.New("underlying cause")
	err := &AuthError{cause: cause}
	assert.Equal(t, cause, err.Unwrap())
}

func TestAuthErrorError(t *testing.T) {
	t.Parallel()
	cause := errors.New("underlying cause")
	err := &AuthError{cause: cause}
	assert.Equal(t, "underlying cause", err.Error())
}

func TestAuthErrorTokenEnvKey(t *testing.T) {
	t.Parallel()
	cause := errors.New("underlying cause")
	err := &AuthError{cause: cause, tokenEnvKey: "abcd"}
	assert.Equal(t, "abcd", err.TokenEnvKey())
}

func TestAsAuthError(t *testing.T) {
	t.Parallel()
	cause := errors.New("underlying cause")
	authErr := &AuthError{cause: cause}
	err := fmt.Errorf("wrapped error: %w", authErr)

	unwrapped, ok := AsAuthError(err)
	assert.True(t, ok)
	assert.Equal(t, authErr, unwrapped)

	unwrapped, ok = AsAuthError(cause)
	assert.False(t, ok)
	assert.Nil(t, unwrapped)
}
