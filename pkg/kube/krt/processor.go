package krt

import (
	"github.com/apache/dubbo-kubernetes/pkg/util/slices"
	"sync"
	"sync/atomic"

	"github.com/apache/dubbo-kubernetes/pkg/util/sets"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/utils/buffer"
)

type handlerRegistration struct {
	Syncer
	remove func()
}

func (h handlerRegistration) UnregisterHandler() {
	h.remove()
}

// handlerSet tracks a set of handlers. Handlers can be added at any time.
type handlerSet[O any] struct {
	mu       sync.RWMutex
	handlers sets.Set[*processorListener[O]]
	wg       wait.Group
}

func newHandlerSet[O any]() *handlerSet[O] {
	return &handlerSet[O]{
		handlers: sets.New[*processorListener[O]](),
	}
}

func (o *handlerSet[O]) Insert(
	f func(o []Event[O]),
	parentSynced Syncer,
	initialEvents []Event[O],
	stopCh <-chan struct{},
) HandlerRegistration {
	o.mu.Lock()
	initialSynced := parentSynced.HasSynced()
	l := newProcessListener(f, parentSynced, stopCh)
	o.handlers.Insert(l)
	o.wg.Start(l.run)
	o.wg.Start(l.pop)
	var sendSynced bool
	if initialSynced {
		if len(initialEvents) == 0 {
			l.syncTracker.ParentSynced()
		} else {
			// Otherwise, queue up a 'synced' event after we process the initial state
			sendSynced = true
		}
	} else {
		o.wg.Start(func() {
			// If we didn't start synced, register a callback to mark ourselves synced once the parent is synced.
			if parentSynced.WaitUntilSynced(stopCh) {
				o.mu.RLock()
				defer o.mu.RUnlock()
				if !o.handlers.Contains(l) {
					return
				}

				select {
				case <-l.stop:
					return
				case l.addCh <- parentSyncedNotification{}:
				}
			}
		})
	}
	o.mu.Unlock()
	l.send(initialEvents, true)
	if sendSynced {
		l.addCh <- parentSyncedNotification{}
	}
	reg := handlerRegistration{
		Syncer: l.Synced(),
		remove: func() {
			o.remove(l)
		},
	}
	return reg
}

func (o *handlerSet[O]) remove(p *processorListener[O]) {
	o.mu.Lock()
	defer o.mu.Unlock()

	delete(o.handlers, p)
	close(p.addCh)
}

func (o *handlerSet[O]) Distribute(events []Event[O], initialSync bool) {
	o.mu.RLock()
	defer o.mu.RUnlock()
	for listener := range o.handlers {
		listener.send(slices.Clone(events), initialSync)
	}
}

func (o *handlerSet[O]) Synced() Syncer {
	o.mu.RLock()
	syncer := multiSyncer{syncers: make([]Syncer, 0, len(o.handlers))}
	for listener := range o.handlers {
		syncer.syncers = append(syncer.syncers, listener.Synced())
	}
	o.mu.RUnlock()
	return syncer
}

type processorListener[O any] struct {
	nextCh chan any
	addCh  chan any
	stop   <-chan struct{}

	handler func(o []Event[O])

	syncTracker *countingTracker

	pendingNotifications buffer.RingGrowing
}

func newProcessListener[O any](
	handler func(o []Event[O]),
	upstreamSyncer Syncer,
	stop <-chan struct{},
) *processorListener[O] {
	bufferSize := 1024
	ret := &processorListener[O]{
		nextCh:               make(chan any),
		addCh:                make(chan any),
		stop:                 stop,
		handler:              handler,
		syncTracker:          &countingTracker{upstreamSyncer: upstreamSyncer, synced: make(chan struct{})},
		pendingNotifications: *buffer.NewRingGrowing(bufferSize),
	}

	return ret
}

type eventSet[O any] struct {
	event           []Event[O]
	isInInitialList bool
}

func (p *processorListener[O]) send(event []Event[O], isInInitialList bool) {
	if isInInitialList {
		p.syncTracker.Start(len(event))
	}
	select {
	case <-p.stop:
		return
	case p.addCh <- eventSet[O]{event: event, isInInitialList: isInInitialList}:
	}
}

func (p *processorListener[O]) pop() {
	defer utilruntime.HandleCrash()
	defer close(p.nextCh)

	var nextCh chan<- any
	var notification any
	for {
		select {
		case <-p.stop:
			return
		case nextCh <- notification:
			// Notification dispatched
			var ok bool
			notification, ok = p.pendingNotifications.ReadOne()
			if !ok { // Nothing to pop
				nextCh = nil // Disable this select case
			}
		case notificationToAdd, ok := <-p.addCh:
			if !ok {
				return
			}
			if notification == nil { // No notification to pop (and pendingNotifications is empty)
				// Optimize the case - skip adding to pendingNotifications
				notification = notificationToAdd
				nextCh = p.nextCh
			} else { // There is already a notification waiting to be dispatched
				p.pendingNotifications.WriteOne(notificationToAdd)
			}
		}
	}
}

type parentSyncedNotification struct{}

func (p *processorListener[O]) run() {
	for {
		select {
		case <-p.stop:
			return
		case nextr, ok := <-p.nextCh:
			if !ok {
				return
			}
			if _, ok := nextr.(parentSyncedNotification); ok {
				p.syncTracker.ParentSynced()
				continue
			}
			next := nextr.(eventSet[O])
			if !next.isInInitialList {
				p.syncTracker.ParentSynced()
			}
			if len(next.event) > 0 {
				p.handler(next.event)
			}
			if next.isInInitialList {
				p.syncTracker.Finished(len(next.event))
			}
		}
	}
}

func (p *processorListener[O]) Synced() Syncer {
	return p.syncTracker.Synced()
}

type countingTracker struct {
	count int64

	// upstreamHasSyncedButEventsPending marks true if the parent has synced, but there are still events pending
	// This helps us known when we need to mark ourselves as 'synced' (which we do exactly once).
	upstreamHasSyncedButEventsPending bool
	upstreamSyncer                    Syncer
	synced                            chan struct{}
	hasSynced                         bool
}

func (t *countingTracker) Start(count int) {
	atomic.AddInt64(&t.count, int64(count))
}

func (t *countingTracker) ParentSynced() {
	if t.hasSynced {
		// Already synced, no change needed
		return
	}
	if atomic.LoadInt64(&t.count) == 0 {
		close(t.synced)
		t.hasSynced = true
	} else {
		t.upstreamHasSyncedButEventsPending = true
	}
}

func (t *countingTracker) Finished(count int) {
	result := atomic.AddInt64(&t.count, -int64(count))
	if result < 0 {
		panic("synctrack: negative counter; this logic error means HasSynced may return incorrect value")
	}
	if !t.hasSynced && t.upstreamHasSyncedButEventsPending && result == 0 && count != 0 {
		close(t.synced)
	}
}

func (t *countingTracker) Synced() Syncer {
	return multiSyncer{
		syncers: []Syncer{
			t.upstreamSyncer,
			channelSyncer{synced: t.synced, name: "tracker"},
		},
	}
}
