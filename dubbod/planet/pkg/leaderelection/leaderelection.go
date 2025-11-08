/*
 * Licensed to the Apache Software Foundation (ASF) under one or more
 * contributor license agreements.  See the NOTICE file distributed with
 * this work for additional information regarding copyright ownership.
 * The ASF licenses this file to You under the Apache License, Version 2.0
 * (the "License"); you may not use this file except in compliance with
 * the License.  You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package leaderelection

import (
	"context"
	"fmt"
	"github.com/apache/dubbo-kubernetes/pkg/log"
	"os"
	"sync"
	"time"

	"github.com/apache/dubbo-kubernetes/pkg/kube"
	"github.com/apache/dubbo-kubernetes/dubbod/planet/pkg/features"
	"github.com/apache/dubbo-kubernetes/dubbod/planet/pkg/leaderelection/k8sleaderelection"
	"github.com/apache/dubbo-kubernetes/dubbod/planet/pkg/leaderelection/k8sleaderelection/k8sresourcelock"
	"go.uber.org/atomic"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

const (
	NamespaceController          = "dubbo-namespace-controller-election"
	ClusterTrustBundleController = "dubbo-clustertrustbundle-controller-election"
)

type LeaderElection struct {
	namespace string
	name      string
	runFns    []func(stop <-chan struct{})
	client    kubernetes.Interface
	ttl       time.Duration

	// enabled sets whether leader election is enabled. Setting enabled=false
	// before calling Run() bypasses leader election and assumes that we are
	// always leader, avoiding unnecessary lease updates on single-node
	// clusters.
	enabled bool

	// Criteria to determine leader priority.
	revision     string
	perRevision  bool
	remote       bool
	useLeaseLock bool
	cycle        *atomic.Int32
	electionID   string
	le           *k8sleaderelection.LeaderElector
	mu           sync.RWMutex

	// leaderCtx is used to cancel all leader goroutines when leader is lost
	leaderCtx    context.Context
	leaderCancel context.CancelFunc
	leaderMu     sync.Mutex
}

func NewLeaderElectionMulticluster(namespace, name, electionID, revision string, remote bool, client kube.Client) *LeaderElection {
	return newLeaderElection(namespace, name, electionID, revision, false, remote, false, client)
}

func newLeaderElection(namespace, name, electionID, revision string, perRevision bool, remote bool, leaseLock bool, client kube.Client) *LeaderElection {
	if revision == "" {
		revision = "default"
	}
	if name == "" {
		hn, _ := os.Hostname()
		name = fmt.Sprintf("unknown-%s", hn)
	}

	if perRevision && revision != "" {
		electionID += "-" + revision
	}

	return &LeaderElection{
		namespace:    namespace,
		name:         name,
		client:       client.Kube(),
		electionID:   electionID,
		revision:     revision,
		perRevision:  perRevision,
		useLeaseLock: leaseLock,
		enabled:      features.EnableLeaderElection,
		remote:       remote,
		// Default to a 30s ttl. Overridable for tests
		ttl:   time.Second * 30,
		cycle: atomic.NewInt32(0),
		mu:    sync.RWMutex{},
	}
}

func (l *LeaderElection) AddRunFunction(f func(stop <-chan struct{})) *LeaderElection {
	l.runFns = append(l.runFns, f)
	return l
}

// SetEnabled sets whether leader election is enabled. This can be used to override
// the global EnableLeaderElection setting for specific cases (e.g., single-node deployments).
func (l *LeaderElection) SetEnabled(enabled bool) *LeaderElection {
	l.enabled = enabled
	return l
}

func (l *LeaderElection) create() (*k8sleaderelection.LeaderElector, error) {
	callbacks := k8sleaderelection.LeaderCallbacks{
		OnStartedLeading: func(ctx context.Context) {
			l.leaderMu.Lock()
			// Cancel any existing leader context to stop old goroutines
			if l.leaderCancel != nil {
				l.leaderCancel()
			}
			// Create new context for this leadership period
			l.leaderCtx, l.leaderCancel = context.WithCancel(ctx)
			leaderCtx := l.leaderCtx
			l.leaderMu.Unlock()

			log.Infof("leader election lock obtained: %v", l.electionID)
			for _, f := range l.runFns {
				go f(leaderCtx.Done())
			}
		},
		OnStoppedLeading: func() {
			l.leaderMu.Lock()
			// Cancel leader context to stop all leader goroutines
			if l.leaderCancel != nil {
				l.leaderCancel()
				l.leaderCancel = nil
				l.leaderCtx = nil
			}
			l.leaderMu.Unlock()
			log.Infof("leader election lock lost: %v", l.electionID)
		},
	}

	key := l.revision
	// TODO remote
	var lock k8sresourcelock.Interface = &k8sresourcelock.ConfigMapLock{
		ConfigMapMeta: metav1.ObjectMeta{Namespace: l.namespace, Name: l.electionID},
		Client:        l.client.CoreV1(),
		LockConfig: k8sresourcelock.ResourceLockConfig{
			Identity: l.name,
			Key:      key,
		},
	}
	if l.perRevision || l.useLeaseLock {
		lock = &k8sresourcelock.LeaseLock{
			LeaseMeta: metav1.ObjectMeta{Namespace: l.namespace, Name: l.electionID},
			Client:    l.client.CoordinationV1(),
			// Note: Key is NOT used. This is not implemented in the library for Lease nor needed, since this is already per-revision.
			// See below, where we disable KeyComparison
			LockConfig: k8sresourcelock.ResourceLockConfig{
				Identity: l.name,
			},
		}
	}

	config := k8sleaderelection.LeaderElectionConfig{
		Lock:            lock,
		LeaseDuration:   l.ttl,
		RenewDeadline:   l.ttl / 2,
		RetryPeriod:     l.ttl / 4,
		Callbacks:       callbacks,
		ReleaseOnCancel: true,
	}

	return k8sleaderelection.NewLeaderElector(config)
}

func (l *LeaderElection) Run(stop <-chan struct{}) {
	if !l.enabled {
		// Silently bypass leader election for single-node deployments or when disabled
		// No need to log this as it's expected behavior
		log.Infof("bypassing leader election: %v", l.electionID)
		for _, f := range l.runFns {
			go f(stop)
		}
		<-stop
		return
	}

	for {
		// Check if we should stop before starting a new cycle
		select {
		case <-stop:
			// Ensure all leader goroutines are stopped before exiting
			l.leaderMu.Lock()
			if l.leaderCancel != nil {
				l.leaderCancel()
				l.leaderCancel = nil
				l.leaderCtx = nil
			}
			l.leaderMu.Unlock()
			return
		default:
		}

		// Ensure any previous leader goroutines are stopped before starting new cycle
		l.leaderMu.Lock()
		if l.leaderCancel != nil {
			l.leaderCancel()
			l.leaderCancel = nil
			l.leaderCtx = nil
		}
		l.leaderMu.Unlock()

		le, err := l.create()
		if err != nil {
			// This should never happen; errors are only from invalid input and the input is not user modifiable
			panic("LeaderElection creation failed: " + err.Error())
		}
		l.mu.Lock()
		l.le = le
		l.cycle.Inc()
		l.mu.Unlock()

		ctx, cancel := context.WithCancel(context.Background())
		stopTriggered := make(chan struct{})
		go func() {
			select {
			case <-stop:
				close(stopTriggered)
				cancel()
			case <-ctx.Done():
			}
		}()

		le.Run(ctx)

		// OnStoppedLeading is called via defer when le.Run() returns
		// Ensure all leader goroutines are stopped
		l.leaderMu.Lock()
		if l.leaderCancel != nil {
			l.leaderCancel()
			l.leaderCancel = nil
			l.leaderCtx = nil
		}
		l.leaderMu.Unlock()
		cancel()

		// Check if stop was triggered while le.Run() was executing
		select {
		case <-stopTriggered:
			// We were told to stop explicitly. Exit now
			return
		default:
			// Otherwise, we may have lost our lock. This can happen when the default revision changes and steals
			// the lock from us.
			log.Infof("Leader election cycle %v lost. Trying again", l.cycle.Load())
		}
	}
}
