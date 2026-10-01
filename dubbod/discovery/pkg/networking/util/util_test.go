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

package util

import (
	"strconv"
	"testing"
)

func TestByteCount(t *testing.T) {
	cases := []struct {
		bytes int64
		want  string
	}{
		{-1, "-1B"},
		{-2_147_483_648, "-2147483648B"},
		{0, "0B"},
		{999, "999B"},
		{1_000, "1.0kB"},
		{10_149, "10.1kB"},
		{10_150, "10.2kB"},
		{10_151, "10.2kB"},
		{999_999, "1000.0kB"},
		{1_000_000, "1.0MB"},
		{999_999_999, "1000.0MB"},
		{1_000_000_000, "1.0GB"},
		{2_147_483_647, "2.1GB"},
		{999_999_999_999, "1000.0GB"},
		{1_000_000_000_000, "1.0TB"},
		{999_999_999_999_999, "1000.0TB"},
		{1_000_000_000_000_000, "1.0PB"},
		{999_999_999_999_999_999, "1000.0PB"},
		{1_000_000_000_000_000_000, "1.0EB"},
	}
	maxInt := int(^uint(0) >> 1)
	minInt := -maxInt - 1
	for _, tc := range cases {
		if tc.bytes > int64(maxInt) {
			continue
		}
		if got := ByteCount(int(tc.bytes)); got != tc.want {
			t.Errorf("ByteCount(%d) = %q, want %q", tc.bytes, got, tc.want)
		}
	}
	if got, want := ByteCount(minInt), strconv.Itoa(minInt)+"B"; got != want {
		t.Errorf("minimum int: got %q, want %q", got, want)
	}
	wantMax := "9.2EB"
	if strconv.IntSize == 32 {
		wantMax = "2.1GB"
	}
	if got := ByteCount(maxInt); got != wantMax {
		t.Errorf("maximum int: got %q, want %q", got, wantMax)
	}
}
