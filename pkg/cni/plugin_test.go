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

func TestPluginAddSkipsWhenPodInfoIsUnavailable(t *testing.T) {
	conf := testConf(t)
	rules := &fakeRuleManager{}
	plugin := Plugin{
		PodInfoProvider: fakePodInfoProvider{err: errors.New("get pod app/nginx: Unauthorized")},
		RuleManager:     rules,
		StateStore:      NewFileStateStore(t.TempDir()),
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
	if len(rules.added) != 0 {
		t.Fatalf("added rules = %v, want none", rules.added)
	}
	if _, err := plugin.StateStore.Read("container-a"); !IsNotFound(err) {
		t.Fatalf("state read err = %v, want not found", err)
	}
}

func TestPluginAddSkipsWithoutPodInfoProvider(t *testing.T) {
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
		t.Fatalf("Run(ADD) failed: %v", err)
	}
	if len(rules.added) != 0 {
		t.Fatalf("added rules = %v, want none", rules.added)
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

type fakeRuleManager struct {
	added   []string
	deleted []string
}

func (f *fakeRuleManager) AddPodRules(_ context.Context, podIP string) error {
	f.added = append(f.added, podIP)
	return nil
}

func (f *fakeRuleManager) DeletePodRules(_ context.Context, podIP string) error {
	f.deleted = append(f.deleted, podIP)
	return nil
}
