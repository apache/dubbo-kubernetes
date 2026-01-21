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

package controllers

import (
	"github.com/apache/dubbo-kubernetes/pkg/config"
	dubbolog "github.com/apache/dubbo-kubernetes/pkg/log"
	"go.uber.org/atomic"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/util/workqueue"
)

type ReconcilerFn func(key types.NamespacedName) error

type syncSignal struct{}

var defaultSyncSignal = syncSignal{}

type Queue struct {
	queue       workqueue.TypedRateLimitingInterface[any]
	initialSync *atomic.Bool
	name        string
	maxAttempts int
	workFn      func(key any) error
	closed      chan struct{}
	log         *dubbolog.Scope
}

func NewQueue(name string, options ...func(*Queue)) Queue {
	q := Queue{
		name:        name,
		closed:      make(chan struct{}),
		initialSync: atomic.NewBool(false),
	}
	for _, o := range options {
		o(&q)
	}
	if q.queue == nil {
		q.queue = workqueue.NewTypedRateLimitingQueueWithConfig[any](
			workqueue.DefaultTypedControllerRateLimiter[any](),
			workqueue.TypedRateLimitingQueueConfig[any]{
				Name: name,
			},
		)
	}
	return q
}

func (q Queue) Add(item any) {
	q.queue.Add(item)
}

func (q Queue) AddObject(obj Object) {
	q.queue.Add(config.NamespacedName(obj))
}

func (q Queue) HasSynced() bool {
	return q.initialSync.Load()
}

func (q Queue) ShutDownEarly() {
	q.queue.ShutDown()
}

func (q Queue) processNextItem() bool {
	// Wait until there is a new item in the working queue
	key, quit := q.queue.Get()
	if quit {
		// We are done, signal to exit the queue
		return false
	}

	// We got the sync signal. This is not a real event, so we exit early after signaling we are synced
	if key == defaultSyncSignal {
		q.initialSync.Store(true)
		return true
	}

	// 'Done marks item as done processing' - should be called at the end of all processing
	defer q.queue.Done(key)

	err := q.workFn(key)
	if err != nil {
		retryCount := q.queue.NumRequeues(key) + 1
		if retryCount < q.maxAttempts {
			q.queue.AddRateLimited(key)
			// Return early, so we do not call Forget(), allowing the rate limiting to backoff
			return true
		}
	}
	// 'Forget indicates that an item is finished being retried.' - should be called whenever we do not want to backoff on this key.
	q.queue.Forget(key)
	return true
}

func (q Queue) Run(stop <-chan struct{}) {
	defer q.queue.ShutDown()
	q.queue.Add(defaultSyncSignal)
	go func() {
		// Process updates until we return false, which indicates the queue is terminated
		for q.processNextItem() {
		}
		close(q.closed)
	}()
	select {
	case <-stop:
	case <-q.closed:
	}
}

func (q Queue) Closed() <-chan struct{} {
	return q.closed
}

func WithRateLimiter(r workqueue.TypedRateLimiter[any]) func(q *Queue) {
	return func(q *Queue) {
		q.queue = workqueue.NewTypedRateLimitingQueue[any](r)
	}
}

func WithMaxAttempts(n int) func(q *Queue) {
	return func(q *Queue) {
		q.maxAttempts = n
	}
}

func WithReconciler(f ReconcilerFn) func(q *Queue) {
	return func(q *Queue) {
		q.workFn = func(key any) error {
			return f(key.(types.NamespacedName))
		}
	}
}
