package status

import (
	"strconv"
	"sync"

	schematypes "github.com/apache/dubbo-kubernetes/pkg/config/schema/kubetypes"
	"github.com/apache/dubbo-kubernetes/pkg/kube/controllers"
	"github.com/apache/dubbo-kubernetes/pkg/kube/krt"
	"github.com/apache/dubbo-kubernetes/pkg/log"
	"github.com/apache/dubbo-kubernetes/pkg/slices"
)

type Registration = func(statusWriter Queue) krt.HandlerRegistration

type StatusCollections struct {
	mu           sync.Mutex
	constructors []func(statusWriter Queue) krt.HandlerRegistration
	active       []krt.HandlerRegistration
	queue        Queue
}

func (s *StatusCollections) Register(sr Registration) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.constructors = append(s.constructors, sr)
}

func (s *StatusCollections) UnsetQueue() {
	// Now we are disabled
	s.queue = nil
	for _, act := range s.active {
		act.UnregisterHandler()
	}
	s.active = nil
}

func (s *StatusCollections) SetQueue(queue Queue) []krt.Syncer {
	s.mu.Lock()
	defer s.mu.Unlock()
	// Now we are enabled!
	s.queue = queue
	log.Infof("StatusCollections.SetQueue: registering %d status constructors", len(s.constructors))
	// Register all constructors
	s.active = slices.Map(s.constructors, func(reg Registration) krt.HandlerRegistration {
		return reg(queue)
	})
	log.Infof("StatusCollections.SetQueue: registered %d status handlers", len(s.active))
	return slices.Map(s.active, func(e krt.HandlerRegistration) krt.Syncer {
		return e
	})
}

func RegisterStatus[I controllers.Object, IS any](s *StatusCollections, statusCol krt.StatusCollection[I, IS], getStatus func(I) IS) {
	reg := func(statusWriter Queue) krt.HandlerRegistration {
		log.Infof("Registering status handler for status collection")
		if statusWriter == nil {
			log.Warnf("Status writer is nil when registering status handler")
		}
		h := statusCol.RegisterBatch(func(events []krt.Event[krt.ObjectWithStatus[I, IS]]) {
			log.Infof("Status update batch event received: %d events", len(events))
			for _, o := range events {
				l := o.Latest()
				liveStatus := getStatus(l.Obj)
				log.Infof("Status update event for %v (event=%v): liveStatus=%+v, desiredStatus=%+v", l.ResourceName(), o.Event, liveStatus, l.Status)
				if krt.Equal(liveStatus, l.Status) {
					// We want the same status we already have! No need for a write so skip this.
					// Note: the Equals() function on ObjectWithStatus does not compare these. It only compares "old live + desired" == "new live + desired".
					// So if either the live OR the desired status changes, this callback will trigger.
					// Here, we do smarter filtering and can just check whether we meet the desired state.
					log.Infof("suppress change for %v %v (statuses are equal)", l.ResourceName(), l.Obj.GetResourceVersion())
					continue
				}
				status := &l.Status
				if o.Event == controllers.EventDelete {
					// if the object is being deleted, we should not reset status
					var empty IS
					status = &empty
				}
				if statusWriter == nil {
					log.Warnf("Status writer is nil, cannot enqueue status update for %v", l.ResourceName())
					continue
				}
				enqueueStatus(statusWriter, l.Obj, status)
				log.Infof("Enqueued status update for %v %v: %+v", l.ResourceName(), l.Obj.GetResourceVersion(), status)
			}
		}, true) // runExistingState=true to process existing objects when handler is registered
		log.Infof("Status handler registered successfully")
		return h
	}
	s.Register(reg)
}

func enqueueStatus[T any](sw Queue, obj controllers.Object, ws T) {
	// TODO: this is a bit awkward since the status controller is reading from crdstore. I suppose it works -- it just means
	// we cannot remove Gateway API types from there.
	res := Resource{
		GroupVersionResource: schematypes.GvrFromObject(obj),
		Namespace:            obj.GetNamespace(),
		Name:                 obj.GetName(),
		Generation:           strconv.FormatInt(obj.GetGeneration(), 10),
	}
	sw.EnqueueStatusUpdateResource(ws, res)
}










