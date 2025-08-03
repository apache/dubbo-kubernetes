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

type indexedDependencyType uint8

type Key[O any] string

const (
	unknownIndexType indexedDependencyType = iota
	indexType        indexedDependencyType = iota
	getKeyType       indexedDependencyType = iota
)

type indexedDependency struct {
	id  collectionUID
	key string
	typ indexedDependencyType
}

type extractorKey struct {
	uid       collectionUID
	filterUID collectionUID
	typ       indexedDependencyType
}

type objectKeyExtractor = func(o any) []string

type multiIndex[I, O any] struct {
	outputs  map[Key[O]]O
	inputs   map[Key[I]]I
	mappings map[Key[I]]sets.Set[Key[O]]
}

type collectionIndex[I, O any] struct {
	extract func(o O) []string
	index   map[string]sets.Set[Key[O]]
	parent  *manyCollection[I, O]
}

type dependencyState[I any] struct {
	collectionDependencies       sets.Set[collectionUID]
	objectDependencies           map[Key[I]][]*dependency
	indexedDependencies          map[indexedDependency]sets.Set[Key[I]]
	indexedDependenciesExtractor map[extractorKey]objectKeyExtractor
}

func NewCollection[I, O any](c Collection[I], hf TransformationSingle[I, O], opts ...CollectionOption) Collection[O] {
	hm := func(ctx HandlerContext, i I) []O {
		res := hf(ctx, i)
		if res == nil {
			return nil
		}
		return []O{*res}
	}
	o := buildCollectionOptions(opts...)
	if o.name == "" {
		o.name = fmt.Sprintf("Collection[%v,%v]", ptr.TypeName[I](), ptr.TypeName[O]())
	}
	return newManyCollection(c, hm, o, nil)
}

func newManyCollection[I, O any](
	cc Collection[I],
	hf TransformationMulti[I, O],
	opts collectionOptions,
	onPrimaryInputEventHandler func([]Event[I]),
) Collection[O] {
	c := cc.(internalCollection[I])

	h := &manyCollection[I, O]{
		transformation: hf,
		collectionName: opts.name,
		id:             nextUID(),
		parent:         c,
		dependencyState: dependencyState[I]{
			collectionDependencies:       sets.New[collectionUID](),
			objectDependencies:           map[Key[I]][]*dependency{},
			indexedDependencies:          map[indexedDependency]sets.Set[Key[I]]{},
			indexedDependenciesExtractor: map[extractorKey]func(o any) []string{},
		},
		collectionState: multiIndex[I, O]{
			inputs:   map[Key[I]]I{},
			outputs:  map[Key[O]]O{},
			mappings: map[Key[I]]sets.Set[Key[O]]{},
		},
		indexes:                    make(map[string]collectionIndex[I, O]),
		eventHandlers:              newHandlerSet[O](),
		augmentation:               opts.augmentation,
		synced:                     make(chan struct{}),
		stop:                       opts.stop,
		onPrimaryInputEventHandler: onPrimaryInputEventHandler,
	}

	if opts.metadata != nil {
		h.metadata = opts.metadata
	}

	h.syncer = channelSyncer{
		name:   h.collectionName,
		synced: h.synced,
	}

	return h
}

func (h *manyCollection[I, O]) name() string {
	return h.collectionName
}

// nolint: unused // (not true, its to implement an interface)
func (h *manyCollection[I, O]) uid() collectionUID {
	return h.id
}

type internalCollection[T any] interface {
	Collection[T]
	name() string
	uid() collectionUID
	dump() CollectionDump
	augment(any) any
	index(name string, extract func(o T) []string) indexer[T]
}

type CollectionDump struct {
	Outputs         map[string]any       `json:"outputs,omitempty"`
	InputCollection string               `json:"inputCollection,omitempty"`
	Inputs          map[string]InputDump `json:"inputs,omitempty"`
	Synced          bool                 `json:"synced"`
}

type InputDump struct {
	Outputs      []string `json:"outputs,omitempty"`
	Dependencies []string `json:"dependencies,omitempty"`
}

type manyCollection[I, O any] struct {
	collectionName             string
	id                         collectionUID
	parent                     Collection[I]
	mu                         sync.RWMutex
	collectionState            multiIndex[I, O]
	dependencyState            dependencyState[I]
	indexes                    map[string]collectionIndex[I, O]
	eventHandlers              *handlerSet[O]
	transformation             TransformationMulti[I, O]
	augmentation               func(a any) any
	synced                     chan struct{}
	stop                       <-chan struct{}
	metadata                   Metadata
	onPrimaryInputEventHandler func(o []Event[I])
	syncer                     Syncer
}

type handlers[O any] struct {
	mu   sync.RWMutex
	h    []*singletonHandlerRegistration[O]
	init bool
}

type singletonHandlerRegistration[O any] struct {
	fn func(o []Event[O])
}

func (o *handlers[O]) Insert(f func(o []Event[O])) *singletonHandlerRegistration[O] {
	o.mu.Lock()
	defer o.mu.Unlock()
	reg := &singletonHandlerRegistration[O]{fn: f}
	o.h = append(o.h, reg)
	return reg
}

func (o *handlers[O]) Delete(toRemove *singletonHandlerRegistration[O]) {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.h = slices.FilterInPlace(o.h, func(s *singletonHandlerRegistration[O]) bool {
		return s != toRemove
	})
}

func (o *handlers[O]) Get() []func(o []Event[O]) {
	o.mu.RLock()
	defer o.mu.RUnlock()
	return slices.Map(o.h, func(e *singletonHandlerRegistration[O]) func(o []Event[O]) {
		return e.fn
	})
}

func (h *manyCollection[I, O]) WaitUntilSynced(stop <-chan struct{}) bool {
	return h.syncer.WaitUntilSynced(stop)
}

func (h *manyCollection[I, O]) GetKey(k string) (res *O) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	rf, f := h.collectionState.outputs[Key[O](k)]
	if f {
		return &rf
	}
	return nil
}

func (h *manyCollection[I, O]) List() (res []O) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return maps.Values(h.collectionState.outputs)
}

func (h *manyCollection[I, O]) Register(f func(o Event[O])) HandlerRegistration {
	return registerHandlerAsBatched(h, f)
}

func (h *manyCollection[I, O]) RegisterBatch(f func(o []Event[O]), runExistingState bool) HandlerRegistration {
	if !runExistingState {
		// If we don't to run the initial state this is simple, we just register the handler.
		h.mu.Lock()
		defer h.mu.Unlock()
		return h.eventHandlers.Insert(f, h, nil, h.stop)
	}
	// We need to run the initial state, but we don't want to get duplicate events.
	// We should get "ADD initialObject1, ADD initialObjectN, UPDATE someLaterUpdate" without mixing the initial ADDs
	// To do this we block any new event processing
	// Get initial state
	h.mu.RLock()
	defer h.mu.RUnlock()

	events := make([]Event[O], 0, len(h.collectionState.outputs))
	for _, o := range h.collectionState.outputs {
		events = append(events, Event[O]{
			New:   &o,
			Event: controllers.EventAdd,
		})
	}

	// Send out all the initial objects to the handler. We will then unlock the new events so it gets the future updates.
	return h.eventHandlers.Insert(f, h, events, h.stop)
}

func (h *manyCollection[I, O]) HasSynced() bool {
	return h.syncer.HasSynced()
}

func (h *manyCollection[I, O]) Metadata() Metadata {
	return h.metadata
}

func (h *manyCollection[I, O]) dump() CollectionDump {
	h.mu.RLock()
	defer h.mu.RUnlock()

	inputs := make(map[string]InputDump, len(h.collectionState.inputs))
	for k, v := range h.collectionState.mappings {
		output := make([]string, 0, len(v))
		for vv := range v {
			output = append(output, string(vv))
		}
		slices.Sort(output)
		inputs[string(k)] = InputDump{
			Outputs:      output,
			Dependencies: nil, // filled later
		}
	}
	for k, deps := range h.dependencyState.objectDependencies {
		depss := make([]string, 0, len(deps))
		for _, dep := range deps {
			depss = append(depss, dep.collectionName)
		}
		slices.Sort(depss)
		cur := inputs[string(k)]
		cur.Dependencies = depss
		inputs[string(k)] = cur
	}

	return CollectionDump{
		Outputs:         eraseMap(h.collectionState.outputs),
		Inputs:          inputs,
		InputCollection: h.parent.(internalCollection[I]).name(),
		Synced:          h.HasSynced(),
	}
}

func (h *manyCollection[I, O]) index(name string, extract func(o O) []string) indexer[O] {
	h.mu.Lock()
	defer h.mu.Unlock()
	if idx, ok := h.indexes[name]; ok {
		return idx
	}

	idx := collectionIndex[I, O]{
		extract: extract,
		index:   make(map[string]sets.Set[Key[O]]),
		parent:  h,
	}
	for k, v := range h.collectionState.outputs {
		idx.update(Event[O]{
			Old:   nil,
			New:   &v,
			Event: controllers.EventAdd,
		}, k)
	}
	h.indexes[name] = idx
	return idx
}

func (h *manyCollection[I, O]) augment(a any) any {
	if h.augmentation != nil {
		return h.augmentation(a)
	}
	return a
}

func eraseMap[T any](l map[Key[T]]T) map[string]any {
	nm := make(map[string]any, len(l))
	for k, v := range l {
		nm[string(k)] = v
	}
	return nm
}

func (c collectionIndex[I, O]) Lookup(key string) []O {
	c.parent.mu.RLock()
	defer c.parent.mu.RUnlock()
	keys := c.index[key]

	res := make([]O, 0, len(keys))
	for k := range keys {
		v, f := c.parent.collectionState.outputs[k]
		if !f {
			continue
		}
		res = append(res, v)
	}
	return res
}

func (c collectionIndex[I, O]) update(ev Event[O], oKey Key[O]) {
	if ev.Old != nil {
		c.delete(*ev.Old, oKey)
	}
	if ev.New != nil {
		newIndexKeys := c.extract(*ev.New)
		for _, newIndexKey := range newIndexKeys {
			sets.InsertOrNew(c.index, newIndexKey, oKey)
		}
	}
}

func (c collectionIndex[I, O]) delete(o O, oKey Key[O]) {
	oldIndexKeys := c.extract(o)
	for _, oldIndexKey := range oldIndexKeys {
		sets.DeleteCleanupLast(c.index, oldIndexKey, oKey)
	}
}
