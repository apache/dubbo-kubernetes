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
	"io"
	"os"
	"strings"
	"testing"

	. "github.com/apache/dubbo-kubernetes/app/dubboctl/internal/testing"
	"github.com/ory/viper"
)

// fromTempDirectory is a test helper which endeavors to create
// an environment clean of developer's settings for use during CLI testing.
func fromTempDirectory(t *testing.T) string {
	t.Helper()
	ClearEnvs(t)

	// By default unit tests presume no config exists unless provided in testdata.
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	// creates and CDs to a temp directory
	d, done := Mktemp(t)

	// Return to original directory and resets viper.
	t.Cleanup(func() { done(); viper.Reset() })
	return d
}

// pipe the output of stdout to a buffer whose value is returned
// from the returned function.  Call pipe() to start piping output
// to the buffer, call the returned function to access the data in
// the buffer.
func piped(t *testing.T) func() string {
	t.Helper()
	var (
		o = os.Stdout
		c = make(chan error, 1)
		b strings.Builder
	)

	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}

	os.Stdout = w

	go func() {
		_, err := io.Copy(&b, r)
		r.Close()
		c <- err
	}()

	return func() string {
		os.Stdout = o
		w.Close()
		err := <-c
		if err != nil {
			t.Fatal(err)
		}
		return strings.TrimSpace(b.String())
	}
}
