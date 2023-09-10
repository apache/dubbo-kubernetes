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
//go:build linux
// +build linux

package prompt

import (
	"io"
	"strings"
	"testing"
	"time"

	"github.com/apache/dubbo-kubernetes/app/dubboctl/internal/docker"
	"github.com/hinshun/vt10x"
)

const (
	enter = "\r"
)

func Test_NewPromptForCredentials(t *testing.T) {
	expectedCreds := docker.Credentials{
		Username: "testuser",
		Password: "testpwd",
	}

	console, _, err := vt10x.NewVT10XConsole()
	if err != nil {
		t.Fatal(err)
	}
	defer console.Close()

	go func() {
		_, _ = console.ExpectEOF()
	}()

	go func() {
		chars := expectedCreds.Username + enter + expectedCreds.Password + enter
		for _, ch := range chars {
			time.Sleep(time.Millisecond * 100)
			_, _ = console.Send(string(ch))
		}
	}()

	tests := []struct {
		name   string
		in     io.Reader
		out    io.Writer
		errOut io.Writer
	}{
		{
			name:   "with non-tty",
			in:     strings.NewReader(expectedCreds.Username + "\r\n" + expectedCreds.Password + "\r\n"),
			out:    io.Discard,
			errOut: io.Discard,
		},
		{
			name:   "with tty",
			in:     console.Tty(),
			out:    console.Tty(),
			errOut: console.Tty(),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			credPrompt := NewPromptForCredentials(tt.in, tt.out, tt.errOut)
			cred, err := credPrompt("example.com")
			if err != nil {
				t.Fatal(err)
			}
			if cred != expectedCreds {
				t.Errorf("bad credentials: %+v", cred)
			}
		})
	}
}
