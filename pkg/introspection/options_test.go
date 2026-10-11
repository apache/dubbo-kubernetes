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

package introspection

import (
	"bytes"
	"net"
	"os"
	"strings"
	"sync"
	"testing"

	dubbolog "github.com/apache/dubbo-kubernetes/pkg/log"
	"github.com/spf13/cobra"
)

type synchronizedBuffer struct {
	mu     sync.Mutex
	buffer bytes.Buffer
}

func (b *synchronizedBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buffer.Write(p)
}

func (b *synchronizedBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buffer.String()
}

func legacyIntrospectionTokens() []string {
	return []string{"ctrl" + "z", "control" + "z"}
}

func TestDefaultOptionsListenAddress(t *testing.T) {
	got := DefaultOptions().ListenAddress()
	if got == "" {
		t.Fatal("DefaultOptions().ListenAddress() is empty")
	}
	host, port, err := net.SplitHostPort(got)
	if err != nil {
		t.Fatalf("ListenAddress %q is not host:port: %v", got, err)
	}
	if host != "localhost" {
		t.Fatalf("host = %q, want localhost", host)
	}
	if port != "9876" {
		t.Fatalf("port = %q, want 9876", port)
	}
}

func TestAttachCobraFlagsHasNoLegacyIntrospectionIdentifiers(t *testing.T) {
	if PortFlagName != "diag_port" || AddressFlagName != "diag_address" {
		t.Fatalf("diagnostic flag names = (%q, %q), want (diag_port, diag_address)", PortFlagName, AddressFlagName)
	}

	cmd := &cobra.Command{Use: "dubbo-discovery"}
	opts := DefaultOptions()
	opts.AttachCobraFlags(cmd)

	usages := cmd.PersistentFlags().FlagUsages()
	lower := strings.ToLower(usages)
	for _, banned := range legacyIntrospectionTokens() {
		if strings.Contains(lower, banned) {
			t.Fatalf("flag help still contains %q:\n%s", banned, usages)
		}
	}

	port := cmd.PersistentFlags().Lookup(PortFlagName)
	if port == nil {
		t.Fatalf("missing flag %q; flags=\n%s", PortFlagName, usages)
	}
	address := cmd.PersistentFlags().Lookup(AddressFlagName)
	if address == nil {
		t.Fatalf("missing flag %q; flags=\n%s", AddressFlagName, usages)
	}
	for _, banned := range legacyIntrospectionTokens() {
		if strings.Contains(strings.ToLower(port.Name+port.Usage+address.Name+address.Usage), banned) {
			t.Fatalf("flag metadata still uses banned identifier %q: port=%q usage=%q", banned, port.Name, port.Usage)
		}
	}
}

func TestRunLogsHaveNoLegacyIntrospectionIdentifiers(t *testing.T) {
	scope := dubbolog.FindScope("introspection")
	if scope == nil {
		t.Fatal("introspection log scope missing")
	}
	var out synchronizedBuffer
	scope.SetOutput(&out)
	defer scope.SetOutput(os.Stderr)

	opts := DefaultOptions()
	opts.Port = 0
	server, err := Run(opts)
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if server == nil || server.httpServer.Addr == "" {
		t.Fatal("Run() did not bind a listen address")
	}
	server.Close()

	logged := strings.ToLower(out.String())
	for _, banned := range legacyIntrospectionTokens() {
		if strings.Contains(logged, banned) {
			t.Fatalf("introspection logs still contain %q:\n%s", banned, out.String())
		}
	}
	if !strings.Contains(logged, "introspection available at") {
		t.Fatalf("missing start log, got:\n%s", out.String())
	}
}
