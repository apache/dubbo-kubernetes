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

package leaderelection

import (
	"fmt"
	"strconv"
	"testing"
	"time"

	"github.com/apache/dubbo-kubernetes/pkg/kube"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"
	clienttesting "k8s.io/client-go/testing"
)

type electionTestClient struct {
	kube.Client
	client kubernetes.Interface
}

func (c electionTestClient) Kube() kubernetes.Interface { return c.client }

func TestNamespaceElectionReplicasAndHandoff(t *testing.T) {
	client := fake.NewSimpleClientset()
	// The fake tracker does not implement API-server resource-version conflicts.
	client.PrependReactor("update", "configmaps", func(action clienttesting.Action) (bool, runtime.Object, error) {
		candidate := action.(clienttesting.UpdateAction).GetObject().(*corev1.ConfigMap)
		stored, err := client.Tracker().Get(action.GetResource(), candidate.Namespace, candidate.Name)
		if err != nil {
			return true, nil, err
		}
		current := stored.(*corev1.ConfigMap)
		if candidate.ResourceVersion != current.ResourceVersion {
			return true, nil, apierrors.NewConflict(action.GetResource().GroupResource(), candidate.Name, fmt.Errorf("resource version changed"))
		}
		revision, _ := strconv.Atoi(current.ResourceVersion)
		updated := candidate.DeepCopy()
		updated.ResourceVersion = strconv.Itoa(revision + 1)
		if err := client.Tracker().Update(action.GetResource(), updated, updated.Namespace); err != nil {
			return true, nil, err
		}
		return true, updated, nil
	})
	started := make(chan int, 4)
	stops := []chan struct{}{make(chan struct{}), make(chan struct{})}
	finished := []chan struct{}{make(chan struct{}), make(chan struct{})}
	closed := []bool{false, false}
	for id := 0; id < 2; id++ {
		election := NewLeaderElection("dubbo-system", fmt.Sprintf("dubbod-%d", id), NamespaceController, "default", electionTestClient{client: client}).SetEnabled(true)
		election.ttl = 2 * time.Second
		election.AddRunFunction(func(stop <-chan struct{}) { started <- id; <-stop })
		go func() { election.Run(stops[id]); close(finished[id]) }()
	}
	t.Cleanup(func() {
		for id := range stops {
			if !closed[id] {
				close(stops[id])
			}
		}
		for _, done := range finished {
			select {
			case <-done:
			case <-time.After(5 * time.Second):
				t.Error("election did not stop")
			}
		}
	})
	var leader int
	select {
	case leader = <-started:
	case <-time.After(5 * time.Second):
		t.Fatal("no namespace leader")
	}
	select {
	case other := <-started:
		t.Fatalf("two active namespace leaders: %d and %d", leader, other)
	case <-time.After(700 * time.Millisecond):
	}
	close(stops[leader])
	closed[leader] = true
	select {
	case <-finished[leader]:
	case <-time.After(5 * time.Second):
		t.Fatal("old leader did not stop")
	}
	select {
	case next := <-started:
		if next == leader {
			t.Fatal("stopped replica regained leadership")
		}
	case <-time.After(5 * time.Second):
		t.Fatal("standby did not take over namespace reconciliation")
	}
}
