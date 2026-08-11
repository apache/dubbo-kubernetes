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

package nodeagent

import (
	"context"
	"os/exec"
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

	if err := manager.AddPodRules(context.Background(), "10.244.0.12", []int{9090}); err != nil {
		t.Fatalf("AddPodRules() failed: %v", err)
	}

	joined := strings.Join(runner.commands, "\n")
	for _, want := range []string{
		"ipset create DUBBO-GRPC-INBOUND-PODS hash:ip -exist",
		"ipset create DUBBO-GRPC-INBOUND-EXCLUDE hash:ip,port -exist",
		"-N DUBBO-GRPC-INBOUND",
		"-I FORWARD 1 -j DUBBO-GRPC-INBOUND",
		"-I OUTPUT 1 -j DUBBO-GRPC-INBOUND",
		"-A DUBBO-GRPC-INBOUND -m set --match-set DUBBO-GRPC-INBOUND-EXCLUDE dst,dst -p tcp -j RETURN",
		"-A DUBBO-GRPC-INBOUND -m set --match-set DUBBO-GRPC-INBOUND-PODS dst -p tcp --dport 15080 -j RETURN",
		"-A DUBBO-GRPC-INBOUND -m set --match-set DUBBO-GRPC-INBOUND-PODS dst -p tcp --dport 26021 -j RETURN",
		"-A DUBBO-GRPC-INBOUND -m set --match-set DUBBO-GRPC-INBOUND-PODS dst -p tcp --dport 15020 -j RETURN",
		"-A DUBBO-GRPC-INBOUND -m set --match-set DUBBO-GRPC-INBOUND-PODS dst -p tcp -j REJECT",
		"ipset add DUBBO-GRPC-INBOUND-PODS 10.244.0.12 -exist",
		"ipset add DUBBO-GRPC-INBOUND-EXCLUDE 10.244.0.12,tcp:9090 -exist",
	} {
		if !strings.Contains(joined, want) {
			t.Fatalf("commands missing %q:\n%s", want, joined)
		}
	}

	// The exclusion must be evaluated before the catch-all REJECT.
	excludeAt := strings.Index(joined, "--match-set DUBBO-GRPC-INBOUND-EXCLUDE dst,dst -p tcp -j RETURN")
	rejectAt := strings.Index(joined, "DUBBO-GRPC-INBOUND-PODS dst -p tcp -j REJECT")
	if excludeAt < 0 || rejectAt < 0 || excludeAt > rejectAt {
		t.Fatalf("exclusion rule is not appended before the REJECT rule:\n%s", joined)
	}
}

func TestIPTablesRuleManagerReconcileRebuildsFence(t *testing.T) {
	runner := &recordingRunner{}
	conf, err := ParseNetConf([]byte(`{"grpcInboundPort":15080}`))
	if err != nil {
		t.Fatalf("ParseNetConf() failed: %v", err)
	}
	manager := NewIPTablesRuleManagerWithRunner(conf, runner)

	err = manager.Reconcile(context.Background(), []PodState{
		{IP: "10.244.0.12", ExcludedPorts: []int{9090}},
		{IP: "10.244.0.13"},
		{IP: "not-an-ip"},
	})
	if err != nil {
		t.Fatalf("Reconcile() failed: %v", err)
	}

	joined := strings.Join(runner.commands, "\n")
	for _, want := range []string{
		"ipset flush DUBBO-GRPC-INBOUND-PODS",
		"ipset flush DUBBO-GRPC-INBOUND-EXCLUDE",
		"ipset add DUBBO-GRPC-INBOUND-PODS 10.244.0.12 -exist",
		"ipset add DUBBO-GRPC-INBOUND-EXCLUDE 10.244.0.12,tcp:9090 -exist",
		"ipset add DUBBO-GRPC-INBOUND-PODS 10.244.0.13 -exist",
	} {
		if !strings.Contains(joined, want) {
			t.Fatalf("commands missing %q:\n%s", want, joined)
		}
	}
	if strings.Contains(joined, "not-an-ip") {
		t.Fatalf("malformed state was not skipped:\n%s", joined)
	}
}

func TestIPTablesRuleManagerFallsBackToDirectRulesWithoutIPSet(t *testing.T) {
	runner := &ipsetMissingRunner{}
	conf, err := ParseNetConf([]byte(`{"grpcInboundPort":15080}`))
	if err != nil {
		t.Fatalf("ParseNetConf() failed: %v", err)
	}
	manager := NewIPTablesRuleManagerWithRunner(conf, runner)

	if err := manager.AddPodRules(context.Background(), "10.244.0.12", []int{9090}); err != nil {
		t.Fatalf("AddPodRules() failed: %v", err)
	}

	joined := strings.Join(runner.commands, "\n")
	for _, want := range []string{
		"ipset create DUBBO-GRPC-INBOUND-PODS hash:ip -exist",
		"-A DUBBO-GRPC-INBOUND -d 10.244.0.12 -p tcp --dport 9090 -j RETURN",
		"-A DUBBO-GRPC-INBOUND -d 10.244.0.12 -p tcp --dport 15080 -j RETURN",
		"-A DUBBO-GRPC-INBOUND -d 10.244.0.12 -p tcp --dport 26021 -j RETURN",
		"-A DUBBO-GRPC-INBOUND -d 10.244.0.12 -p tcp --dport 15020 -j RETURN",
		"-A DUBBO-GRPC-INBOUND -d 10.244.0.12 -p tcp -j REJECT",
	} {
		if !strings.Contains(joined, want) {
			t.Fatalf("commands missing %q:\n%s", want, joined)
		}
	}
	if strings.Contains(joined, "--match-set") {
		t.Fatalf("ipset rule leaked into direct fallback:\n%s", joined)
	}
}

func TestIPTablesRuleManagerReconcilesDirectRulesWithoutIPSet(t *testing.T) {
	runner := &ipsetMissingRunner{}
	conf, err := ParseNetConf([]byte(`{"grpcInboundPort":15080}`))
	if err != nil {
		t.Fatalf("ParseNetConf() failed: %v", err)
	}
	manager := NewIPTablesRuleManagerWithRunner(conf, runner)

	if err := manager.Reconcile(context.Background(), []PodState{{
		IP:            "10.244.0.12",
		ExcludedPorts: []int{9090},
	}}); err != nil {
		t.Fatalf("Reconcile() failed: %v", err)
	}

	joined := strings.Join(runner.commands, "\n")
	for _, want := range []string{
		"-F DUBBO-GRPC-INBOUND",
		"-A DUBBO-GRPC-INBOUND -d 10.244.0.12 -p tcp --dport 9090 -j RETURN",
		"-A DUBBO-GRPC-INBOUND -d 10.244.0.12 -p tcp -j REJECT",
	} {
		if !strings.Contains(joined, want) {
			t.Fatalf("commands missing %q:\n%s", want, joined)
		}
	}
}

func TestIPTablesRuleManagerDeletesDirectRulesWithoutIPSet(t *testing.T) {
	runner := &ipsetMissingRunner{}
	conf, err := ParseNetConf([]byte(`{"grpcInboundPort":15080}`))
	if err != nil {
		t.Fatalf("ParseNetConf() failed: %v", err)
	}
	manager := NewIPTablesRuleManagerWithRunner(conf, runner)

	if err := manager.DeletePodRules(context.Background(), "10.244.0.12", []int{9090}); err != nil {
		t.Fatalf("DeletePodRules() failed: %v", err)
	}

	joined := strings.Join(runner.commands, "\n")
	for _, want := range []string{
		"-D DUBBO-GRPC-INBOUND -d 10.244.0.12 -p tcp --dport 9090 -j RETURN",
		"-D DUBBO-GRPC-INBOUND -d 10.244.0.12 -p tcp --dport 15080 -j RETURN",
		"-D DUBBO-GRPC-INBOUND -d 10.244.0.12 -p tcp -j REJECT",
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

type ipsetMissingRunner struct {
	recordingRunner
}

func (r *ipsetMissingRunner) Run(ctx context.Context, name string, args ...string) ([]byte, error) {
	if name == "ipset" {
		r.commands = append(r.commands, name+" "+strings.Join(args, " "))
		return nil, &exec.Error{Name: name, Err: exec.ErrNotFound}
	}
	return r.recordingRunner.Run(ctx, name, args...)
}

var errCommandFailed = commandFailedError{}

type commandFailedError struct{}

func (commandFailedError) Error() string {
	return "command failed"
}
