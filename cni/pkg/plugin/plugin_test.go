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

package plugin

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
)

type recordingSender struct {
	events []PodEvent
	err    error
}

func (r *recordingSender) Send(_ context.Context, _ string, event PodEvent) error {
	r.events = append(r.events, event)
	return r.err
}

func TestExecuteAddForwardsEventAndPassesThroughPrevResult(t *testing.T) {
	prev := `{"cniVersion":"1.0.0","ips":[{"address":"10.244.0.9/24"}]}`
	stdin := `{"cniVersion":"1.0.0","name":"kindnet","type":"dubbo-cni","socketPath":"/tmp/cni.sock","prevResult":` + prev + `}`
	env := Env{
		Command:     CmdAdd,
		ContainerID: "ctr-1",
		NetNS:       "/var/run/netns/pod-1",
		IfName:      "eth0",
		Args:        "IgnoreUnknown=1;K8S_POD_NAMESPACE=test-inject;K8S_POD_NAME=test-app;K8S_POD_UID=uid-1",
	}

	sender := &recordingSender{}
	var stdout bytes.Buffer
	if err := Execute(context.Background(), env, strings.NewReader(stdin), &stdout, sender); err != nil {
		t.Fatalf("Execute(ADD) error = %v", err)
	}
	if len(sender.events) != 1 {
		t.Fatalf("events = %d, want 1", len(sender.events))
	}
	ev := sender.events[0]
	if ev.PodName != "test-app" || ev.Namespace != "test-inject" || ev.PodIP != "10.244.0.9" || ev.NetNS != "/var/run/netns/pod-1" {
		t.Fatalf("unexpected PodEvent: %+v", ev)
	}
	if strings.TrimSpace(stdout.String()) != prev {
		t.Fatalf("stdout = %s, want prevResult %s", stdout.String(), prev)
	}
}

func TestExecuteAddSkipsSystemNamespacesAndHonorsFailOpen(t *testing.T) {
	stdin := `{"cniVersion":"1.0.0","name":"kindnet","type":"dubbo-cni","failOpen":true}`
	sender := &recordingSender{err: errors.New("socket unavailable")}

	// System namespace should not even invoke sender.
	var stdout bytes.Buffer
	err := Execute(context.Background(), Env{
		Command: CmdAdd,
		Args:    "K8S_POD_NAMESPACE=dubbo-system;K8S_POD_NAME=dubbod-1",
	}, strings.NewReader(stdin), &stdout, sender)
	if err != nil {
		t.Fatalf("Execute(dubbo-system) error = %v", err)
	}
	if len(sender.events) != 0 {
		t.Fatalf("expected 0 events for dubbo-system, got %d", len(sender.events))
	}

	// Regular namespace with failOpen=true ignores sender error and returns valid JSON.
	stdout.Reset()
	err = Execute(context.Background(), Env{
		Command: CmdAdd,
		Args:    "K8S_POD_NAMESPACE=app;K8S_POD_NAME=svc-1",
	}, strings.NewReader(stdin), &stdout, sender)
	if err != nil {
		t.Fatalf("Execute(failOpen) error = %v", err)
	}
	var out map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &out); err != nil || out["cniVersion"] != "1.0.0" {
		t.Fatalf("unexpected stdout JSON: %s (err=%v)", stdout.String(), err)
	}
}

func TestExecuteDelIsBestEffort(t *testing.T) {
	sender := &recordingSender{err: errors.New("agent down")}
	var stdout bytes.Buffer
	err := Execute(context.Background(), Env{
		Command: CmdDel,
		Args:    "K8S_POD_NAMESPACE=test-inject;K8S_POD_NAME=test-app",
	}, strings.NewReader(`{"cniVersion":"1.0.0","type":"dubbo-cni"}`), &stdout, sender)
	if err != nil {
		t.Fatalf("Execute(DEL) error = %v, want nil", err)
	}
	if len(sender.events) != 1 || sender.events[0].Command != CmdDel {
		t.Fatalf("expected DEL event, got %+v", sender.events)
	}
}
