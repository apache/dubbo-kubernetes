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
	"path/filepath"
	"testing"
)

func TestFileStateStoreList(t *testing.T) {
	store := NewFileStateStore(t.TempDir())
	if err := store.Write(PodState{ContainerID: "a", Namespace: "app", Name: "one", IP: "10.244.0.12", ExcludedPorts: []int{9090}}); err != nil {
		t.Fatalf("Write() failed: %v", err)
	}
	if err := store.Write(PodState{ContainerID: "b", Namespace: "app", Name: "two", IP: "10.244.0.13"}); err != nil {
		t.Fatalf("Write() failed: %v", err)
	}
	// Entries with no IP cannot be turned into rules and must be skipped.
	if err := store.Write(PodState{ContainerID: "c", Namespace: "app", Name: "three"}); err != nil {
		t.Fatalf("Write() failed: %v", err)
	}

	states, err := store.List()
	if err != nil {
		t.Fatalf("List() failed: %v", err)
	}
	if len(states) != 2 {
		t.Fatalf("List() returned %d states, want 2", len(states))
	}
	byIP := map[string][]int{}
	for _, state := range states {
		byIP[state.IP] = state.ExcludedPorts
	}
	if ports, ok := byIP["10.244.0.12"]; !ok || len(ports) != 1 || ports[0] != 9090 {
		t.Fatalf("state for 10.244.0.12 = %v, want [9090]", ports)
	}
	if _, ok := byIP["10.244.0.13"]; !ok {
		t.Fatalf("state for 10.244.0.13 is missing")
	}
}

func TestFileStateStoreListMissingDirectory(t *testing.T) {
	store := NewFileStateStore(filepath.Join(t.TempDir(), "absent"))
	states, err := store.List()
	if err != nil {
		t.Fatalf("List() on a missing directory failed: %v", err)
	}
	if len(states) != 0 {
		t.Fatalf("List() = %v, want empty", states)
	}
}
