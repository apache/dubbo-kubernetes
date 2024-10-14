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

package util

import (
	"fmt"
	"strings"
	"testing"
)

// TestValidateFunctionName tests that only correct function names are accepted
func TestValidateFunctionName(t *testing.T) {
	cases := []struct {
		In    string
		Valid bool
	}{
		{"", false},
		{"*", false},
		{"-", false},
		{"example", true},
		{"example-com", true},
		{"example.com", false},
		{"-example-com", false},
		{"example-com-", false},
		{"Example", false},
		{"EXAMPLE", false},
		{"42", false},
	}

	for _, c := range cases {
		err := ValidateApplicationName(c.In)
		if err != nil && c.Valid {
			t.Fatalf("Unexpected error: %v, for '%v'", err, c.In)
		}
		if err == nil && !c.Valid {
			t.Fatalf("Expected error for invalid entry: %v", c.In)
		}
	}
}

func TestValidateFunctionNameErrMsg(t *testing.T) {
	invalidFnName := "EXAMPLE"
	errMsgPrefix := fmt.Sprintf("Function name '%v'", invalidFnName)

	err := ValidateApplicationName(invalidFnName)
	if err != nil {
		if !strings.HasPrefix(err.Error(), errMsgPrefix) {
			t.Fatalf("Unexpected error message: %v, the message should start with '%v' string", err.Error(), errMsgPrefix)
		}
	} else {
		t.Fatalf("Expected error for invalid entry: %v", invalidFnName)
	}
}
