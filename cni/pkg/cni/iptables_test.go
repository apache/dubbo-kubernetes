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

package cni

import (
	"context"
	"strings"
	"testing"
)

func TestIPTablesRuleManagerAddsGRPCInboundBoundaryRules(t *testing.T) {
	runner := &recordingRunner{}
	conf, err := ParseNetConf([]byte(`{"grpcInboundPort":15080}`))
	if err != nil {
		t.Fatalf("ParseNetConf() failed: %v", err)
	}
	manager := NewIPTablesRuleManagerWithRunner(conf, runner)

	if err := manager.AddPodRules(context.Background(), "10.244.0.12"); err != nil {
		t.Fatalf("AddPodRules() failed: %v", err)
	}

	joined := strings.Join(runner.commands, "\n")
	for _, want := range []string{
		"ipset create DUBBO-GRPC-INBOUND-PODS hash:ip -exist",
		"-N DUBBO-GRPC-INBOUND",
		"-I FORWARD 1 -j DUBBO-GRPC-INBOUND",
		"-I OUTPUT 1 -j DUBBO-GRPC-INBOUND",
		"-A DUBBO-GRPC-INBOUND -m set --match-set DUBBO-GRPC-INBOUND-PODS dst -p tcp --dport 15080 -j RETURN",
		"-A DUBBO-GRPC-INBOUND -m set --match-set DUBBO-GRPC-INBOUND-PODS dst -p tcp --dport 26021 -j RETURN",
		"-A DUBBO-GRPC-INBOUND -m set --match-set DUBBO-GRPC-INBOUND-PODS dst -p tcp -j REJECT",
		"ipset add DUBBO-GRPC-INBOUND-PODS 10.244.0.12 -exist",
	} {
		if !strings.Contains(joined, want) {
			t.Fatalf("commands missing %q:\n%s", want, joined)
		}
	}
}

type recordingRunner struct {
	commands []string
}

func (r *recordingRunner) Run(_ context.Context, name string, args ...string) ([]byte, error) {
	r.commands = append(r.commands, name+" "+strings.Join(args, " "))
	for _, arg := range args {
		if arg == "-C" || arg == "-D" {
			return []byte("not found"), errCommandFailed
		}
	}
	return nil, nil
}

var errCommandFailed = commandFailedError{}

type commandFailedError struct{}

func (commandFailedError) Error() string {
	return "command failed"
}
