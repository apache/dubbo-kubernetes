package controllers

import (
	"github.com/apache/dubbo-kubernetes/pkg/config"
	"go.uber.org/atomic"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/klog/v2"
)

type ReconcilerFn func(key types.NamespacedName) error

type Queue struct {
	queue       workqueue.TypedRateLimitingInterface[any]
	initialSync *atomic.Bool
	name        string
	maxAttempts int
	workFn      func(key any) error
	closed      chan struct{}
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
				Name:            name,
				MetricsProvider: nil,
				Clock:           nil,
				DelayingQueue:   nil,
			},
		)
	}
	klog.Infof("controller=%v", q.name)
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

func (q Queue) ShutDownEarly() {
	q.queue.ShutDown()
}

type syncSignal struct{}

var defaultSyncSignal = syncSignal{}

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
	klog.Infof("starting")
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
	klog.Info("stopped")
}

func (q Queue) Closed() <-chan struct{} {
	return q.closed
}
