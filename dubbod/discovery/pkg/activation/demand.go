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
	"time"
)

// Target is the Service whose requests are being held while it scales up.
type Target struct {
	Namespace string
	Name      string
}

// DemandSource reports how many requests are waiting for a target to get
// endpoints. It is the seam between the gateways that hold the requests and
// the KEDA scaler that reports them, so the scaler can be built and tested
// without depending on how demand arrives.
type DemandSource interface {
	// Pending returns the number of requests currently held for the target.
	Pending(Target) int64

	// Subscribe delivers the pending count whenever it changes. The channel is
	// closed when cancel is called. Implementations must not block on it: a
	// slow subscriber may miss intermediate values but must still converge on
	// the latest one, because a missed edge would leave a workload scaled to
	// zero with requests waiting on it.
	Subscribe(Target) (updates <-chan int64, cancel func())
}

// reporterTTL bounds how long one gateway's report is trusted. A gateway that
// dies mid-activation stops refreshing, and its demand has to expire on its
// own; otherwise the target stays scaled up forever with nothing waiting on it.
const reporterTTL = 30 * time.Second

// Registry aggregates pending counts reported by gateways.
//
// Reports are absolute counts per reporter, not deltas: a gateway that
// restarts, or whose report is lost, converges on the next report instead of
// leaving the total permanently skewed.
type Registry struct {
	mu sync.Mutex
	// Per target, the last count each reporter published and when.
	targets map[Target]map[string]report
	// Per target, the live subscribers.
	subscribers map[Target]map[int]chan int64
	nextID      int

	// now is swappable so expiry can be tested without sleeping.
	now func() time.Time
	ttl time.Duration
}

type report struct {
	pending  int64
	received time.Time
}

func NewRegistry() *Registry {
	return &Registry{
		targets:     map[Target]map[string]report{},
		subscribers: map[Target]map[int]chan int64{},
		now:         time.Now,
		ttl:         reporterTTL,
	}
}

// Report records the requests one gateway is currently holding for a target.
// It is called on every refresh, including with zero, which is how a gateway
// says it has drained.
func (r *Registry) Report(reporter string, target Target, pending int64) {
	if pending < 0 {
		pending = 0
	}
	r.mu.Lock()
	byReporter, ok := r.targets[target]
	if !ok {
		byReporter = map[string]report{}
		r.targets[target] = byReporter
	}
	byReporter[reporter] = report{pending: pending, received: r.now()}
	total := r.totalLocked(target)
	subscribers := r.snapshotSubscribersLocked(target)
	r.mu.Unlock()

	notify(subscribers, total)
}

// Forget drops a gateway's reports, for a clean shutdown that should not wait
// out the TTL.
func (r *Registry) Forget(reporter string) {
	type notification struct {
		subscribers []chan int64
		total       int64
	}

	r.mu.Lock()
	var notifications []notification
	for target, byReporter := range r.targets {
		if _, ok := byReporter[reporter]; !ok {
			continue
		}
		delete(byReporter, reporter)
		notifications = append(notifications, notification{
			subscribers: r.snapshotSubscribersLocked(target),
			total:       r.totalLocked(target),
		})
	}
	r.mu.Unlock()

	for _, item := range notifications {
		notify(item.subscribers, item.total)
	}
}

func (r *Registry) Pending(target Target) int64 {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.totalLocked(target)
}

// Reporters counts the gateways currently holding requests for a target, or
// standing by to. Zero means nothing would catch a request for it, which is
// what the policy controller reports rather than letting a policy look ready
// while no gateway can act on it.
func (r *Registry) Reporters(target Target) int {
	r.mu.Lock()
	defer r.mu.Unlock()
	// Expires stale reporters as a side effect, so a gateway that vanished does
	// not keep a policy looking healthy.
	r.totalLocked(target)
	return len(r.targets[target])
}

func (r *Registry) Subscribe(target Target) (<-chan int64, func()) {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Buffered by one so a notifier never blocks and a subscriber that is busy
	// still finds the latest value waiting for it.
	updates := make(chan int64, 1)
	id := r.nextID
	r.nextID++
	if r.subscribers[target] == nil {
		r.subscribers[target] = map[int]chan int64{}
	}
	r.subscribers[target][id] = updates

	cancel := func() {
		r.mu.Lock()
		defer r.mu.Unlock()
		if channels, ok := r.subscribers[target]; ok {
			if channel, ok := channels[id]; ok {
				delete(channels, id)
				close(channel)
			}
			if len(channels) == 0 {
				delete(r.subscribers, target)
			}
		}
	}
	return updates, cancel
}

// totalLocked sums the live reports for a target, dropping expired reporters as
// it goes so a vanished gateway cannot hold a workload up indefinitely.
func (r *Registry) totalLocked(target Target) int64 {
	byReporter, ok := r.targets[target]
	if !ok {
		return 0
	}
	cutoff := r.now().Add(-r.ttl)
	var total int64
	for reporter, entry := range byReporter {
		if entry.received.Before(cutoff) {
			delete(byReporter, reporter)
			continue
		}
		total += entry.pending
	}
	if len(byReporter) == 0 {
		delete(r.targets, target)
	}
	return total
}

func (r *Registry) snapshotSubscribersLocked(target Target) []chan int64 {
	channels := r.subscribers[target]
	if len(channels) == 0 {
		return nil
	}
	out := make([]chan int64, 0, len(channels))
	for _, channel := range channels {
		out = append(out, channel)
	}
	return out
}

// notify replaces any value a subscriber has not read yet. Only the latest
// count matters, and dropping a stale one keeps the notifier non-blocking.
func notify(subscribers []chan int64, total int64) {
	for _, channel := range subscribers {
		select {
		case <-channel:
		default:
		}
		select {
		case channel <- total:
		default:
		}
	}
}
