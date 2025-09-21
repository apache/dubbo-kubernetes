package krt

import (
	"fmt"
	"github.com/apache/dubbo-kubernetes/pkg/kube/controllers"
	"github.com/apache/dubbo-kubernetes/pkg/maps"
	"github.com/apache/dubbo-kubernetes/pkg/ptr"
	"github.com/apache/dubbo-kubernetes/pkg/util/sets"
	"github.com/apache/dubbo-kubernetes/pkg/util/slices"
	"sync"
)

type StaticCollection[T any] struct {
	*staticList[T]
}

type staticList[T any] struct {
	mu             sync.RWMutex
	vals           map[string]T
	eventHandlers  *handlerSet[T]
	id             collectionUID
	stop           <-chan struct{}
	collectionName string
	syncer         Syncer
	metadata       Metadata
	indexes        map[string]staticListIndex[T]
}

func (s *staticList[T]) Register(f func(o Event[T])) HandlerRegistration {
	return registerHandlerAsBatched(s, f)
}

// nolint: unused // (not true, its to implement an interface)
func (s *staticList[T]) name() string {
	return s.collectionName
}

// nolint: unused // (not true, its to implement an interface)
func (s *staticList[T]) uid() collectionUID {
	return s.id
}

// nolint: unused // (not true, its to implement an interface)
func (s *staticList[T]) dump() CollectionDump {
	return CollectionDump{
		Outputs: eraseMap(slices.GroupUnique(s.List(), getTypedKey)),
		Synced:  s.HasSynced(),
	}
}

func (s *staticList[T]) index(name string, extract func(o T) []string) indexer[T] {
	s.mu.Lock()
	defer s.mu.Unlock()
	if idx, ok := s.indexes[name]; ok {
		return idx
	}

	idx := staticListIndex[T]{
		extract: extract,
		index:   make(map[string]sets.Set[string]),
		parent:  s,
	}

	for k, v := range s.vals {
		idx.update(Event[T]{
			Old:   nil,
			New:   &v,
			Event: controllers.EventAdd,
		}, k)
	}
	s.indexes[name] = idx

	return idx
}

// nolint: unused // (not true, its to implement an interface)
func (s *staticList[T]) augment(a any) any {
	return a
}

func (s *staticList[T]) HasSynced() bool {
	return s.syncer.HasSynced()
}

func (s *staticList[T]) Synced() Syncer {
	// We are always synced in the static collection since the initial state must be provided upfront
	return alwaysSynced{}
}

func (s *staticList[T]) GetKey(k string) *T {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if o, f := s.vals[k]; f {
		return &o
	}
	return nil
}

func (s *staticList[T]) Metadata() Metadata {
	return s.metadata
}

func (s *staticList[T]) WaitUntilSynced(stop <-chan struct{}) bool {
	return s.syncer.WaitUntilSynced(stop)
}

func (s *staticList[T]) RegisterBatch(f func(o []Event[T]), runExistingState bool) HandlerRegistration {
	s.mu.Lock()
	defer s.mu.Unlock()
	var objs []Event[T]
	if runExistingState {
		for _, v := range s.vals {
			objs = append(objs, Event[T]{
				New:   &v,
				Event: controllers.EventAdd,
			})
		}
	}
	return s.eventHandlers.Insert(f, s.Synced(), objs, s.stop)
}

func (s *staticList[T]) List() []T {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return maps.Values(s.vals)
}

func NewStaticCollection[T any](synced Syncer, vals []T, opts ...CollectionOption) StaticCollection[T] {
	o := buildCollectionOptions(opts...)
	if o.name == "" {
		o.name = fmt.Sprintf("Static[%v]", ptr.TypeName[T]())
	}

	res := make(map[string]T, len(vals))
	for _, v := range vals {
		res[GetKey(v)] = v
	}

	if synced == nil {
		synced = alwaysSynced{}
	}

	sl := &staticList[T]{
		eventHandlers:  newHandlerSet[T](),
		vals:           res,
		id:             nextUID(),
		stop:           o.stop,
		collectionName: o.name,
		syncer:         synced,
		indexes:        make(map[string]staticListIndex[T]),
	}

	if o.metadata != nil {
		sl.metadata = o.metadata
	}

	c := StaticCollection[T]{
		staticList: sl,
	}
	maybeRegisterCollectionForDebugging[T](c, o.debugger)
	return c
}

func (s StaticCollection[T]) Reset(newState []T) {
	s.mu.Lock()
	defer s.mu.Unlock()
	var updates []Event[T]
	nv := map[string]T{}
	for _, incoming := range newState {
		k := GetKey(incoming)
		nv[k] = incoming
		if old, f := s.vals[k]; f {
			if !Equal(old, incoming) {
				ev := Event[T]{
					Old:   &old,
					New:   &incoming,
					Event: controllers.EventUpdate,
				}
				for _, index := range s.indexes {
					index.update(ev, k)
				}
				updates = append(updates, ev)
			}
		} else {
			ev := Event[T]{
				New:   &incoming,
				Event: controllers.EventAdd,
			}
			for _, index := range s.indexes {
				index.update(ev, k)
			}
			updates = append(updates, ev)
		}
		delete(s.vals, k)
	}
	for k, remaining := range s.vals {
		for _, index := range s.indexes {
			index.delete(remaining, k)
		}
		updates = append(updates, Event[T]{
			Old:   &remaining,
			Event: controllers.EventDelete,
		})
	}
	s.vals = nv
	if len(updates) > 0 {
		s.eventHandlers.Distribute(updates, false)
	}
}

// nolint: unused // (not true)
type staticListIndex[T any] struct {
	extract func(o T) []string
	index   map[string]sets.Set[string]
	parent  *staticList[T]
}

func (s staticListIndex[T]) update(ev Event[T], oKey string) {
	if ev.Old != nil {
		s.delete(*ev.Old, oKey)
	}
	if ev.New != nil {
		newIndexKeys := s.extract(*ev.New)
		for _, newIndexKey := range newIndexKeys {
			sets.InsertOrNew(s.index, newIndexKey, oKey)
		}
	}
}

func (s staticListIndex[T]) delete(o T, oKey string) {
	oldIndexKeys := s.extract(o)
	for _, oldIndexKey := range oldIndexKeys {
		sets.DeleteCleanupLast(s.index, oldIndexKey, oKey)
	}
}

func (s staticListIndex[T]) Lookup(key string) []T {
	s.parent.mu.RLock()
	defer s.parent.mu.RUnlock()
	keys := s.index[key]

	res := make([]T, 0, len(keys))
	for k := range keys {
		v, f := s.parent.vals[k]
		if !f {
			continue
		}
		res = append(res, v)
	}
	return res
}
