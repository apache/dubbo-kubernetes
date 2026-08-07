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

package activation

import (
	"sync"
	"testing"
	"time"
)

var orders = Target{Namespace: "app", Name: "orders"}

func TestRegistrySumsReportersAndTreatsReportsAsAbsolute(t *testing.T) {
	registry := NewRegistry()

	registry.Report("gateway-a", orders, 3)
	registry.Report("gateway-b", orders, 2)
	if got := registry.Pending(orders); got != 5 {
		t.Fatalf("pending = %d, want 5", got)
	}

	// Reports are absolute, not deltas: a gateway restating its count must
	// replace its own contribution rather than add to it.
	registry.Report("gateway-a", orders, 1)
	if got := registry.Pending(orders); got != 3 {
		t.Fatalf("pending after restatement = %d, want 3", got)
	}

	registry.Report("gateway-a", orders, 0)
	registry.Report("gateway-b", orders, 0)
	if got := registry.Pending(orders); got != 0 {
		t.Fatalf("pending after drain = %d, want 0", got)
	}
}

func TestRegistryIsolatesTargets(t *testing.T) {
	registry := NewRegistry()
	reviews := Target{Namespace: "app", Name: "reviews"}
	otherNamespace := Target{Namespace: "staging", Name: "orders"}

	registry.Report("gateway-a", orders, 4)
	if got := registry.Pending(reviews); got != 0 {
		t.Fatalf("unrelated service pending = %d, want 0", got)
	}
	// Same Service name in another namespace is a different workload.
	if got := registry.Pending(otherNamespace); got != 0 {
		t.Fatalf("same name in another namespace pending = %d, want 0", got)
	}
}

// A gateway that dies mid-activation stops refreshing. Its demand has to age
// out, or the target stays scaled up forever with nothing waiting on it.
func TestRegistryExpiresStaleReporters(t *testing.T) {
	registry := NewRegistry()
	now := time.Unix(0, 0)
	registry.now = func() time.Time { return now }

	registry.Report("gateway-a", orders, 5)
	registry.Report("gateway-b", orders, 1)

	now = now.Add(reporterTTL / 2)
	registry.Report("gateway-b", orders, 1)

	// gateway-a has not refreshed for longer than the TTL; gateway-b has.
	now = now.Add(reporterTTL/2 + time.Second)
	if got := registry.Pending(orders); got != 1 {
		t.Fatalf("pending after gateway-a expired = %d, want 1", got)
	}

	now = now.Add(reporterTTL + time.Second)
	if got := registry.Pending(orders); got != 0 {
		t.Fatalf("pending after all reporters expired = %d, want 0", got)
	}
}

func TestRegistryForgetDropsAReporterImmediately(t *testing.T) {
	registry := NewRegistry()
	registry.Report("gateway-a", orders, 3)
	registry.Report("gateway-b", orders, 2)

	registry.Forget("gateway-a")
	if got := registry.Pending(orders); got != 2 {
		t.Fatalf("pending after forget = %d, want 2", got)
	}
}

func TestRegistryReportersCountsLiveGateways(t *testing.T) {
	registry := NewRegistry()
	if got := registry.Reporters(orders); got != 0 {
		t.Fatalf("reporters with no gateway = %d, want 0", got)
	}

	registry.Report("gateway-a", orders, 0)
	registry.Report("gateway-b", orders, 0)
	// A gateway holding zero requests is still standing by to catch one.
	if got := registry.Reporters(orders); got != 2 {
		t.Fatalf("reporters = %d, want 2", got)
	}

	registry.Forget("gateway-b")
	if got := registry.Reporters(orders); got != 1 {
		t.Fatalf("reporters after forget = %d, want 1", got)
	}
}

func TestRegistryNotifiesSubscribers(t *testing.T) {
	registry := NewRegistry()
	updates, cancel := registry.Subscribe(orders)
	defer cancel()

	registry.Report("gateway-a", orders, 2)
	if got := receive(t, updates); got != 2 {
		t.Fatalf("update = %d, want 2", got)
	}

	registry.Report("gateway-a", orders, 0)
	if got := receive(t, updates); got != 0 {
		t.Fatalf("drain update = %d, want 0", got)
	}
}

// Only the newest count matters. A subscriber that was busy must not be handed
// a stale value that says zero while requests are waiting.
func TestRegistrySubscriberSeesLatestCountAfterMissingUpdates(t *testing.T) {
	registry := NewRegistry()
	updates, cancel := registry.Subscribe(orders)
	defer cancel()

	registry.Report("gateway-a", orders, 1)
	registry.Report("gateway-a", orders, 7)
	registry.Report("gateway-a", orders, 4)

	if got := receive(t, updates); got != 4 {
		t.Fatalf("update = %d, want the latest count 4", got)
	}
}

func TestRegistryCancelStopsDelivery(t *testing.T) {
	registry := NewRegistry()
	updates, cancel := registry.Subscribe(orders)

	cancel()
	if _, open := <-updates; open {
		t.Fatal("channel still open after cancel")
	}

	// Reporting after cancel must not panic on the closed channel.
	registry.Report("gateway-a", orders, 1)
}

// Reporting must never block on a subscriber, or one stuck gateway stream
// would stall every other gateway's reports.
func TestRegistryReportDoesNotBlockOnIdleSubscriber(t *testing.T) {
	registry := NewRegistry()
	_, cancel := registry.Subscribe(orders)
	defer cancel()

	done := make(chan struct{})
	go func() {
		defer close(done)
		for i := 0; i < 100; i++ {
			registry.Report("gateway-a", orders, int64(i))
		}
	}()

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("Report blocked on a subscriber that never read")
	}
}

func TestRegistryConcurrentReportsAndReads(t *testing.T) {
	registry := NewRegistry()
	var wait sync.WaitGroup
	for reporter := 0; reporter < 8; reporter++ {
		wait.Add(2)
		name := "gateway-" + string(rune('a'+reporter))
		go func() {
			defer wait.Done()
			for i := 0; i < 200; i++ {
				registry.Report(name, orders, int64(i%5))
			}
		}()
		go func() {
			defer wait.Done()
			for i := 0; i < 200; i++ {
				registry.Pending(orders)
			}
		}()
	}
	wait.Wait()
}

func receive(t *testing.T, updates <-chan int64) int64 {
	t.Helper()
	select {
	case value := <-updates:
		return value
	case <-time.After(2 * time.Second):
		t.Fatal("no update delivered")
		return 0
	}
}

func TestReportSnapshotReplacesTheWholeReporterView(t *testing.T) {
	registry := NewRegistry()
	reviews := Target{Namespace: "app", Name: "reviews"}

	registry.ReportSnapshot("gateway-a", map[Target]int64{orders: 3, reviews: 2})
	if got := registry.Pending(orders); got != 3 {
		t.Fatalf("orders pending = %d, want 3", got)
	}
	if got := registry.Pending(reviews); got != 2 {
		t.Fatalf("reviews pending = %d, want 2", got)
	}

	// A target absent from the new snapshot has drained; there is no separate
	// clear message, so its absence is what has to clear it.
	registry.ReportSnapshot("gateway-a", map[Target]int64{orders: 1})
	if got := registry.Pending(orders); got != 1 {
		t.Fatalf("orders pending after replacement = %d, want 1", got)
	}
	if got := registry.Pending(reviews); got != 0 {
		t.Fatalf("reviews pending after omission = %d, want 0", got)
	}

	// An empty snapshot means the gateway drained everything.
	registry.ReportSnapshot("gateway-a", nil)
	if got := registry.Pending(orders); got != 0 {
		t.Fatalf("orders pending after empty snapshot = %d, want 0", got)
	}
}

// Snapshots replace only the reporting gateway's own view; a broadcast from
// one gateway must never clear another's demand.
func TestReportSnapshotLeavesOtherReportersAlone(t *testing.T) {
	registry := NewRegistry()

	registry.ReportSnapshot("gateway-a", map[Target]int64{orders: 2})
	registry.ReportSnapshot("gateway-b", map[Target]int64{orders: 3})
	if got := registry.Pending(orders); got != 5 {
		t.Fatalf("pending = %d, want 5", got)
	}

	registry.ReportSnapshot("gateway-a", nil)
	if got := registry.Pending(orders); got != 3 {
		t.Fatalf("pending after gateway-a drained = %d, want 3", got)
	}
}

func TestReportSnapshotNotifiesSubscribers(t *testing.T) {
	registry := NewRegistry()
	updates, cancel := registry.Subscribe(orders)
	defer cancel()

	registry.ReportSnapshot("gateway-a", map[Target]int64{orders: 4})
	if got := receive(t, updates); got != 4 {
		t.Fatalf("update = %d, want 4", got)
	}

	// Dropping the target must wake the subscriber too, or a KEDA stream would
	// never learn the workload can scale back down.
	registry.ReportSnapshot("gateway-a", nil)
	if got := receive(t, updates); got != 0 {
		t.Fatalf("drain update = %d, want 0", got)
	}
}
