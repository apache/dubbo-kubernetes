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

package iptables

import (
	"context"
	"strings"
	"testing"
)

func TestBuildRestoreScriptIncludesRedirectionAndExclusions(t *testing.T) {
	cfg := DefaultConfig()
	cfg.NodeIP = "172.18.0.2"
	cfg.ExcludeOutboundCIDRs = []string{"169.254.169.254/32"}

	script := BuildRestoreScript(cfg)
	for _, want := range []string{
		"*nat",
		":DUBBO_INBOUND - [0:0]",
		":DUBBO_OUTPUT - [0:0]",
		"-A DUBBO_REDIRECT -p tcp -j DNAT --to-destination 172.18.0.2:15001",
		"-A DUBBO_IN_REDIRECT -p tcp -j DNAT --to-destination 172.18.0.2:15006",
		"-A DUBBO_INBOUND -p tcp --dport 15008 -j RETURN",
		"-A DUBBO_OUTPUT -o lo -j RETURN",
		"-A DUBBO_OUTPUT -m owner --uid-owner 2509 -j RETURN",
		"-A DUBBO_OUTPUT -d 169.254.169.254/32 -j RETURN",
		"-A DUBBO_OUTPUT -p tcp --dport 26012 -j RETURN",
		"COMMIT",
	} {
		if !strings.Contains(script, want) {
			t.Fatalf("iptables script missing %q:\n%s", want, script)
		}
	}
}

func TestNetNSRedirectorInvokesNsenter(t *testing.T) {
	var calledName string
	var calledArgs []string
	var calledStdin string

	r := &NetNSRedirector{
		Runner: func(_ context.Context, stdin, name string, args ...string) ([]byte, error) {
			calledStdin = stdin
			calledName = name
			calledArgs = args
			return nil, nil
		},
	}

	if err := r.Apply(context.Background(), "/var/run/netns/test-app", DefaultConfig()); err != nil {
		t.Fatalf("Apply() error = %v", err)
	}
	if calledName != "nsenter" || len(calledArgs) < 3 || calledArgs[0] != "--net=/var/run/netns/test-app" {
		t.Fatalf("unexpected command: %s %v", calledName, calledArgs)
	}
	if !strings.Contains(calledStdin, "DUBBO_REDIRECT") {
		t.Fatalf("unexpected stdin script: %s", calledStdin)
	}

	if err := r.Cleanup(context.Background(), "/var/run/netns/test-app"); err != nil {
		t.Fatalf("Cleanup() error = %v", err)
	}
	if !strings.Contains(calledStdin, "-X DUBBO_REDIRECT") {
		t.Fatalf("unexpected cleanup script: %s", calledStdin)
	}
}
