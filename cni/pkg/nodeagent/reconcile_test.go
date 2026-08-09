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
	"errors"
	"sync"
	"testing"
	"time"
)

type fakeStateLister struct {
	states []PodState
	err    error
}

func (f fakeStateLister) List() ([]PodState, error) { return f.states, f.err }

type recordingReconciler struct {
	mu    sync.Mutex
	calls [][]PodState
	err   error
}

func (r *recordingReconciler) Reconcile(_ context.Context, states []PodState) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.calls = append(r.calls, states)
	return r.err
}

func (r *recordingReconciler) callCount() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return len(r.calls)
}

func TestReconcileOnceReplaysPersistedState(t *testing.T) {
	lister := fakeStateLister{states: []PodState{
		{IP: "10.244.0.12", ExcludedPorts: []int{9090}},
		{IP: "10.244.0.13"},
	}}
	reconciler := &recordingReconciler{}

	if err := ReconcileOnce(context.Background(), lister, reconciler, nil); err != nil {
		t.Fatalf("ReconcileOnce() failed: %v", err)
	}
	if len(reconciler.calls) != 1 || len(reconciler.calls[0]) != 2 {
		t.Fatalf("reconciler calls = %v, want one call with two states", reconciler.calls)
	}
}

func TestReconcileOnceRequiresCollaborators(t *testing.T) {
	if err := ReconcileOnce(context.Background(), nil, &recordingReconciler{}, nil); err == nil {
		t.Fatal("ReconcileOnce() with no lister returned nil error")
	}
	if err := ReconcileOnce(context.Background(), fakeStateLister{}, nil, nil); err == nil {
		t.Fatal("ReconcileOnce() with no reconciler returned nil error")
	}
}

func TestReconcileOnceReportsListFailure(t *testing.T) {
	lister := fakeStateLister{err: errors.New("permission denied")}
	if err := ReconcileOnce(context.Background(), lister, &recordingReconciler{}, nil); err == nil {
		t.Fatal("ReconcileOnce() with a failing lister returned nil error")
	}
}

func TestReconcileLoopKeepsRunningAfterAFailure(t *testing.T) {
	lister := fakeStateLister{states: []PodState{{IP: "10.244.0.12"}}}
	reconciler := &recordingReconciler{err: errors.New("ipset missing")}
	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan error, 1)
	go func() { done <- ReconcileLoop(ctx, lister, reconciler, nil, time.Millisecond) }()

	deadline := time.After(2 * time.Second)
	for reconciler.callCount() < 3 {
		select {
		case <-deadline:
			cancel()
			t.Fatalf("reconcile ran %d times, want at least 3", reconciler.callCount())
		case <-time.After(time.Millisecond):
		}
	}
	cancel()

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("ReconcileLoop() = %v, want nil", err)
		}
	case <-time.After(time.Second):
		t.Fatal("ReconcileLoop() did not stop after cancel")
	}
}

type fakeClusterLister struct {
	states []PodState
	err    error
}

func (f fakeClusterLister) ManagedPodsOnNode(context.Context, string, string, string) ([]PodState, error) {
	return f.states, f.err
}

func TestReconcileOnceAddsPodsMissingFromLocalState(t *testing.T) {
	// The pod whose ADD could not read it is absent locally but present in the
	// cluster view; reconciliation is what installs its rules.
	local := fakeStateLister{states: []PodState{{IP: "10.244.0.12"}}}
	cluster := &ClusterSource{
		Lister: fakeClusterLister{states: []PodState{
			{IP: "10.244.0.12", ExcludedPorts: []int{9090}},
			{IP: "10.244.0.99"},
		}},
		NodeName:   "master",
		Label:      "inherent.dubbo.apache.org/managed",
		LabelValue: "true",
	}
	reconciler := &recordingReconciler{}

	if err := ReconcileOnce(context.Background(), local, reconciler, cluster); err != nil {
		t.Fatalf("ReconcileOnce() failed: %v", err)
	}
	got := reconciler.calls[0]
	if len(got) != 2 {
		t.Fatalf("reconciled %d states, want 2: %+v", len(got), got)
	}
	byIP := map[string][]int{}
	for _, state := range got {
		byIP[state.IP] = state.ExcludedPorts
	}
	if _, ok := byIP["10.244.0.99"]; !ok {
		t.Fatalf("pod missing from local state was not reconciled: %+v", got)
	}
	// The cluster view is authoritative for annotations.
	if ports := byIP["10.244.0.12"]; len(ports) != 1 || ports[0] != 9090 {
		t.Fatalf("excluded ports = %v, want [9090]", ports)
	}
}

func TestReconcileOnceFallsBackToLocalStateWhenListingFails(t *testing.T) {
	local := fakeStateLister{states: []PodState{{IP: "10.244.0.12"}}}
	cluster := &ClusterSource{Lister: fakeClusterLister{err: errors.New("connection refused")}, NodeName: "master", Label: "l", LabelValue: "true"}
	reconciler := &recordingReconciler{}

	if err := ReconcileOnce(context.Background(), local, reconciler, cluster); err != nil {
		t.Fatalf("ReconcileOnce() failed: %v", err)
	}
	if got := reconciler.calls[0]; len(got) != 1 || got[0].IP != "10.244.0.12" {
		t.Fatalf("reconciled %+v, want the local state only", got)
	}
}
