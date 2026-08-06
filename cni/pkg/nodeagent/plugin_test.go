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
	"encoding/json"
	"errors"
	"testing"

	"github.com/apache/dubbo-kubernetes/pkg/kube/inject"
)

func TestPluginAddInstallsRulesForManagedPod(t *testing.T) {
	conf := testConf(t)
	conf.StateDir = t.TempDir()
	rules := &fakeRuleManager{}
	plugin := Plugin{
		PodInfoProvider: fakePodInfoProvider{pod: PodInfo{Labels: map[string]string{
			inject.ProxylessManagedLabel: inject.ProxylessManagedLabelValue,
		}}},
		RuleManager: rules,
		StateStore:  NewFileStateStore(conf.StateDirectory()),
	}

	out, err := plugin.Run(context.Background(), Env{
		Command:     "ADD",
		ContainerID: "container-a",
		Args:        "K8S_POD_NAMESPACE=app;K8S_POD_NAME=nginx",
	}, conf)
	if err != nil {
		t.Fatalf("Run(ADD) failed: %v", err)
	}
	if len(out) == 0 {
		t.Fatal("Run(ADD) returned empty result")
	}
	if len(rules.added) != 1 || rules.added[0] != "10.244.0.12" {
		t.Fatalf("added rules = %v, want [10.244.0.12]", rules.added)
	}
	state, err := plugin.StateStore.Read("container-a")
	if err != nil {
		t.Fatalf("state read failed: %v", err)
	}
	if state.Namespace != "app" || state.Name != "nginx" || state.IP != "10.244.0.12" {
		t.Fatalf("state = %#v, want app/nginx/10.244.0.12", state)
	}
}

func TestPluginAddSkipsUnmanagedPod(t *testing.T) {
	conf := testConf(t)
	rules := &fakeRuleManager{}
	plugin := Plugin{
		PodInfoProvider: fakePodInfoProvider{pod: PodInfo{Labels: map[string]string{"app": "nginx"}}},
		RuleManager:     rules,
		StateStore:      NewFileStateStore(t.TempDir()),
	}

	if _, err := plugin.Run(context.Background(), Env{
		Command:     "ADD",
		ContainerID: "container-a",
		Args:        "K8S_POD_NAMESPACE=app;K8S_POD_NAME=nginx",
	}, conf); err != nil {
		t.Fatalf("Run(ADD) failed: %v", err)
	}
	if len(rules.added) != 0 {
		t.Fatalf("added rules = %v, want none", rules.added)
	}
}

func TestPluginAddAllowsUnreadablePodByDefault(t *testing.T) {
	conf := testConf(t)
	conf.PodLookupRetryMillis = 1
	rules := &fakeRuleManager{}
	provider := &countingPodInfoProvider{failures: -1, err: errors.New("get pod app/nginx: Unauthorized")}
	plugin := Plugin{
		PodInfoProvider: provider,
		RuleManager:     rules,
		StateStore:      NewFileStateStore(t.TempDir()),
	}

	// An unreadable pod may not be a mesh pod at all, so failing its ADD would
	// stop unrelated workloads from starting on this node. The reconcile loop
	// installs the rules later if it turns out to be mesh-managed.
	out, err := plugin.Run(context.Background(), Env{
		Command:     "ADD",
		ContainerID: "container-a",
		Args:        "K8S_POD_NAMESPACE=app;K8S_POD_NAME=nginx",
	}, conf)
	if err != nil {
		t.Fatalf("Run(ADD) with an unreadable pod failed: %v", err)
	}
	if len(out) == 0 {
		t.Fatal("Run(ADD) returned empty result")
	}
	if want := conf.PodLookupAttempts(); provider.calls != want {
		t.Fatalf("pod lookups = %d, want %d", provider.calls, want)
	}
	if len(rules.added) != 0 {
		t.Fatalf("added rules = %v, want none", rules.added)
	}
}

func TestPluginAddRetriesTransientPodInfoFailure(t *testing.T) {
	conf := testConf(t)
	conf.StateDir = t.TempDir()
	conf.PodLookupRetryMillis = 1
	rules := &fakeRuleManager{}
	provider := &countingPodInfoProvider{
		failures: 2,
		err:      errors.New("etcdserver: request timed out"),
		pod: PodInfo{Labels: map[string]string{
			inject.ProxylessManagedLabel: inject.ProxylessManagedLabelValue,
		}},
	}
	plugin := Plugin{
		PodInfoProvider: provider,
		RuleManager:     rules,
		StateStore:      NewFileStateStore(conf.StateDirectory()),
	}

	if _, err := plugin.Run(context.Background(), Env{
		Command:     "ADD",
		ContainerID: "container-a",
		Args:        "K8S_POD_NAMESPACE=app;K8S_POD_NAME=nginx",
	}, conf); err != nil {
		t.Fatalf("Run(ADD) failed: %v", err)
	}
	if provider.calls != 3 {
		t.Fatalf("pod lookups = %d, want 3", provider.calls)
	}
	if len(rules.added) != 1 || rules.added[0] != "10.244.0.12" {
		t.Fatalf("added rules = %v, want [10.244.0.12]", rules.added)
	}
}

func TestPluginAddFailClosedRejectsUnresolvedPod(t *testing.T) {
	conf := testConf(t)
	conf.PodLookupRetryMillis = 1
	conf.FailClosed = true
	rules := &fakeRuleManager{}
	plugin := Plugin{
		PodInfoProvider: &countingPodInfoProvider{failures: -1, err: errors.New("get pod app/nginx: Unauthorized")},
		RuleManager:     rules,
		StateStore:      NewFileStateStore(t.TempDir()),
	}

	if _, err := plugin.Run(context.Background(), Env{
		Command:     "ADD",
		ContainerID: "container-a",
		Args:        "K8S_POD_NAMESPACE=app;K8S_POD_NAME=nginx",
	}, conf); err == nil {
		t.Fatal("Run(ADD) with failClosed returned nil error")
	}
	if len(rules.added) != 0 {
		t.Fatalf("added rules = %v, want none", rules.added)
	}
	if _, err := plugin.StateStore.Read("container-a"); !IsNotFound(err) {
		t.Fatalf("state read err = %v, want not found", err)
	}
}

func TestPluginAddAllowsWithoutPodInfoProvider(t *testing.T) {
	conf := testConf(t)
	rules := &fakeRuleManager{}
	plugin := Plugin{
		RuleManager: rules,
		StateStore:  NewFileStateStore(t.TempDir()),
	}

	if _, err := plugin.Run(context.Background(), Env{
		Command:     "ADD",
		ContainerID: "container-a",
		Args:        "K8S_POD_NAMESPACE=app;K8S_POD_NAME=nginx",
	}, conf); err != nil {
		t.Fatalf("Run(ADD) without a pod info provider failed: %v", err)
	}
	if len(rules.added) != 0 {
		t.Fatalf("added rules = %v, want none", rules.added)
	}
}

func TestPluginAddPassesExcludedPortsThrough(t *testing.T) {
	conf := testConf(t)
	conf.StateDir = t.TempDir()
	rules := &fakeRuleManager{}
	plugin := Plugin{
		PodInfoProvider: fakePodInfoProvider{pod: PodInfo{
			Labels: map[string]string{
				inject.ProxylessManagedLabel: inject.ProxylessManagedLabelValue,
			},
			ExcludedPorts: []int{9090, 15020},
		}},
		RuleManager: rules,
		StateStore:  NewFileStateStore(conf.StateDirectory()),
	}

	if _, err := plugin.Run(context.Background(), Env{
		Command:     "ADD",
		ContainerID: "container-a",
		Args:        "K8S_POD_NAMESPACE=app;K8S_POD_NAME=nginx",
	}, conf); err != nil {
		t.Fatalf("Run(ADD) failed: %v", err)
	}
	if got := rules.addedPorts["10.244.0.12"]; len(got) != 2 || got[0] != 9090 || got[1] != 15020 {
		t.Fatalf("excluded ports = %v, want [9090 15020]", got)
	}
	state, err := plugin.StateStore.Read("container-a")
	if err != nil {
		t.Fatalf("state read failed: %v", err)
	}
	if len(state.ExcludedPorts) != 2 {
		t.Fatalf("state excluded ports = %v, want two entries", state.ExcludedPorts)
	}
}

func TestPluginDeleteCleansStoredRules(t *testing.T) {
	store := NewFileStateStore(t.TempDir())
	if err := store.Write(PodState{ContainerID: "container-a", Namespace: "app", Name: "nginx", IP: "10.244.0.12"}); err != nil {
		t.Fatalf("state write failed: %v", err)
	}
	rules := &fakeRuleManager{}
	plugin := Plugin{RuleManager: rules, StateStore: store}

	if _, err := plugin.Run(context.Background(), Env{Command: "DEL", ContainerID: "container-a"}, NetConf{}); err != nil {
		t.Fatalf("Run(DEL) failed: %v", err)
	}
	if len(rules.deleted) != 1 || rules.deleted[0] != "10.244.0.12" {
		t.Fatalf("deleted rules = %v, want [10.244.0.12]", rules.deleted)
	}
	if _, err := store.Read("container-a"); !IsNotFound(err) {
		t.Fatalf("state read err = %v, want not found", err)
	}
}

func testConf(t *testing.T) NetConf {
	t.Helper()
	prev, _ := json.Marshal(map[string]any{
		"cniVersion": "1.0.0",
		"ips": []map[string]string{
			{"address": "10.244.0.12/24"},
		},
	})
	conf, err := ParseNetConf([]byte(`{"cniVersion":"1.0.0"}`))
	if err != nil {
		t.Fatalf("ParseNetConf() failed: %v", err)
	}
	conf.PrevResult = prev
	return conf
}

type fakePodInfoProvider struct {
	pod PodInfo
	err error
}

func (f fakePodInfoProvider) PodInfo(context.Context, PodRef) (PodInfo, error) {
	return f.pod, f.err
}

// countingPodInfoProvider fails the first `failures` calls and then succeeds.
// A negative `failures` fails every call.
type countingPodInfoProvider struct {
	pod      PodInfo
	err      error
	failures int
	calls    int
}

func (f *countingPodInfoProvider) PodInfo(context.Context, PodRef) (PodInfo, error) {
	f.calls++
	if f.failures < 0 {
		return PodInfo{}, f.err
	}
	if f.failures > 0 {
		f.failures--
		return PodInfo{}, f.err
	}
	return f.pod, nil
}

type fakeRuleManager struct {
	added        []string
	deleted      []string
	addedPorts   map[string][]int
	deletedPorts map[string][]int
}

func (f *fakeRuleManager) AddPodRules(_ context.Context, podIP string, excludedPorts []int) error {
	f.added = append(f.added, podIP)
	if f.addedPorts == nil {
		f.addedPorts = map[string][]int{}
	}
	f.addedPorts[podIP] = excludedPorts
	return nil
}

func (f *fakeRuleManager) DeletePodRules(_ context.Context, podIP string, excludedPorts []int) error {
	f.deleted = append(f.deleted, podIP)
	if f.deletedPorts == nil {
		f.deletedPorts = map[string][]int{}
	}
	f.deletedPorts[podIP] = excludedPorts
	return nil
}
