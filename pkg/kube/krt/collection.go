package krt

import (
	"fmt"
	"github.com/apache/dubbo-kubernetes/pkg/kube/controllers"
	"github.com/apache/dubbo-kubernetes/pkg/maps"
	"github.com/apache/dubbo-kubernetes/pkg/ptr"
	"github.com/apache/dubbo-kubernetes/pkg/util/sets"
	"github.com/apache/dubbo-kubernetes/pkg/util/slices"
	"istio.io/istio/pkg/queue"
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

	h.queue = queue.NewWithSync(func() {
		close(h.synced)
		fmt.Printf("\n%v synced (uid %v)\n", h.name(), h.uid())
	}, h.collectionName)

	go h.runQueue()

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
	queue                      queue.Instance
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

func (h *manyCollection[I, O]) runQueue() {
	c := h.parent
	if !c.WaitUntilSynced(h.stop) {
		return
	}
	syncer := c.RegisterBatch(func(o []Event[I]) {
		if h.onPrimaryInputEventHandler != nil {
			h.onPrimaryInputEventHandler(o)
		}
		h.queue.Push(func() error {
			h.onPrimaryInputEvent(o)
			return nil
		})
	}, true)
	if !syncer.WaitUntilSynced(h.stop) {
		return
	}
	h.queue.Run(h.stop)
}

func (h *manyCollection[I, O]) onPrimaryInputEvent(items []Event[I]) {
	for idx, ev := range items {
		iKey := GetKey(ev.Latest())
		iObj := h.parent.GetKey(iKey)
		if iObj == nil {
			ev.Event = controllers.EventDelete
			if ev.Old == nil {
				// This was an add, now its a Delete. Make sure we don't have Old and New nil, which we claim to be illegal
				ev.Old = ev.New
			}
			ev.New = nil
		} else {
			ev.New = iObj
		}
		items[idx] = ev
	}
	h.handleChangedPrimaryInputEvents(items)
}

type collectionDependencyTracker[I, O any] struct {
	*manyCollection[I, O]
	d             []*dependency
	key           Key[I]
	discardUpdate bool
}

func (i *collectionDependencyTracker[I, O]) DiscardResult() {
	i.discardUpdate = true
}

const EnableAssertions = false

func (h *manyCollection[I, O]) handleChangedPrimaryInputEvents(items []Event[I]) {
	var events []Event[O]
	recomputedResults := make([]map[Key[O]]O, len(items))

	pendingDepStateUpdates := make(map[Key[I]]*collectionDependencyTracker[I, O], len(items))
	for idx, a := range items {
		if a.Event == controllers.EventDelete {
			// handled below, with full lock...
			continue
		}
		i := a.Latest()
		iKey := getTypedKey(i)

		ctx := &collectionDependencyTracker[I, O]{manyCollection: h, key: iKey}
		results := slices.GroupUnique(h.transformation(ctx, i), getTypedKey[O])
		recomputedResults[idx] = results
		// Store new dependency state, to insert in the next loop under the lock
		pendingDepStateUpdates[iKey] = ctx
	}

	// Now acquire the full lock.
	h.mu.Lock()
	defer h.mu.Unlock()
	for idx, a := range items {
		i := a.Latest()
		iKey := getTypedKey(i)
		if a.Event == controllers.EventDelete {
			for oKey := range h.collectionState.mappings[iKey] {
				oldRes, f := h.collectionState.outputs[oKey]
				if !f {
					continue
				}
				e := Event[O]{
					Event: controllers.EventDelete,
					Old:   &oldRes,
				}
				events = append(events, e)
				delete(h.collectionState.outputs, oKey)
				for _, index := range h.indexes {
					index.delete(oldRes, oKey)
				}
			}
			delete(h.collectionState.mappings, iKey)
			delete(h.collectionState.inputs, iKey)
			h.dependencyState.delete(iKey)
		} else {
			ctx := pendingDepStateUpdates[iKey]
			results := recomputedResults[idx]
			if ctx.discardUpdate {
				// Called when the collection explicitly calls DiscardResult() on the context.
				// This is typically used when we want to retain the last-correct state.
				_, alreadyHasAResult := h.collectionState.mappings[iKey]
				nowHasAResult := len(results) > 0
				if alreadyHasAResult || !nowHasAResult {
					continue
				}
			}
			h.dependencyState.update(iKey, ctx.d)
			newKeys := sets.New(maps.Keys(results)...)
			oldKeys := h.collectionState.mappings[iKey]
			h.collectionState.mappings[iKey] = newKeys
			h.collectionState.inputs[iKey] = i
			allKeys := newKeys.Copy().Merge(oldKeys)
			// We have now built up a set of I -> []O
			// and found the previous I -> []O mapping
			for key := range allKeys {
				// Find new O object
				newRes, newExists := results[key]
				// Find the old O object
				oldRes, oldExists := h.collectionState.outputs[key]
				e := Event[O]{}
				if newExists && oldExists {
					if Equal(newRes, oldRes) {
						// NOP change, skip
						continue
					}
					e.Event = controllers.EventUpdate
					e.New = &newRes
					e.Old = &oldRes
					h.collectionState.outputs[key] = newRes
				} else if newExists {
					e.Event = controllers.EventAdd
					e.New = &newRes
					h.collectionState.outputs[key] = newRes
				} else {
					if !oldExists && EnableAssertions {
						panic(fmt.Sprintf("!oldExists and !newExists, how did we get here? for output key %v input key %v", key, iKey))
					}
					e.Event = controllers.EventDelete
					e.Old = &oldRes
					delete(h.collectionState.outputs, key)
				}

				for _, index := range h.indexes {
					index.update(e, key)
				}

				events = append(events, e)
			}
		}
	}
	if EnableAssertions {
		h.assertIndexConsistency()
	}
	// Short circuit if we have nothing to do
	if len(events) == 0 {
		return
	}

	h.eventHandlers.Distribute(events, !h.HasSynced())
}

func (h *manyCollection[I, O]) assertIndexConsistency() {
	oToI := map[Key[O]]Key[I]{}
	for i, os := range h.collectionState.mappings {
		if _, f := h.collectionState.inputs[i]; !f {
			panic(fmt.Sprintf("for mapping key %v, no input found", i))
		}
		for o := range os {
			if ci, f := oToI[o]; f {
				panic(fmt.Sprintf("duplicate mapping %v: input %v and %v both map to it", o, ci, i))
			}
			oToI[o] = i
			if _, f := h.collectionState.outputs[o]; !f {
				panic(fmt.Sprintf("for mapping key %v->%v, no output found", i, o))
			}
		}
	}
}

func (i dependencyState[I]) update(key Key[I], deps []*dependency) {
	// Update the I -> Dependency mapping
	i.objectDependencies[key] = deps
	for _, d := range deps {
		if depKeys, typ, extractor, filterID, ok := d.filter.reverseIndexKey(); ok {
			for _, depKey := range depKeys {
				k := indexedDependency{
					id:  d.id,
					key: depKey,
					typ: typ,
				}
				if typ == unknownIndexType && extractor == nil {
					// no need to make keys in the reverse index
					// if no extractor specified
					continue
				}

				sets.InsertOrNew(i.indexedDependencies, k, key)
				kk := extractorKey{
					filterUID: filterID,
					uid:       d.id,
					typ:       typ,
				}
				i.indexedDependenciesExtractor[kk] = extractor
			}
		}
	}
}

func (i dependencyState[I]) delete(key Key[I]) {
	old, f := i.objectDependencies[key]
	if !f {
		return
	}
	delete(i.objectDependencies, key)
	for _, d := range old {
		if depKeys, typ, _, _, ok := d.filter.reverseIndexKey(); ok {
			for _, depKey := range depKeys {
				k := indexedDependency{
					id:  d.id,
					key: depKey,
					typ: typ,
				}
				sets.DeleteCleanupLast(i.indexedDependencies, k, key)
			}
		}
	}
}

func (i dependencyState[I]) changedInputKeys(sourceCollection collectionUID, events []Event[any]) sets.Set[Key[I]] {
	changedInputKeys := sets.Set[Key[I]]{}
	// Check old and new
	for _, ev := range events {
		// We have a possibly dependant object changed. For each input object, see if it depends on the object.
		// Naively, we can look through every item in this collection and check if it matches the filter. However, this is
		// inefficient, especially when the dependency changes frequently and the collection is large.
		// Where possible, we utilize the reverse-indexing to get the precise list of potentially changed objects.
		foundAny := false

		// find all the reverse indexes related to the sourceCollection
		// N here is usually going to be small (the number of FilterKey/FilterIndex)
		extractorKeys := []extractorKey{}
		for k := range i.indexedDependenciesExtractor {
			if k.typ != unknownIndexType && k.uid == sourceCollection {
				extractorKeys = append(extractorKeys, k)
			}
		}

		for _, ekey := range extractorKeys {
			if extractor, f := i.indexedDependenciesExtractor[ekey]; f {
				foundAny = true
				// We have a reverse index
				for _, item := range ev.Items() {
					// Find all the reverse index keys for this object. For each key we will find impacted input objects.
					keys := extractor(item)
					for _, key := range keys {
						for iKey := range i.indexedDependencies[indexedDependency{id: sourceCollection, key: key, typ: ekey.typ}] {
							if changedInputKeys.Contains(iKey) {
								// We may have already found this item, skip it
								continue
							}
							dependencies := i.objectDependencies[iKey]
							if changed := objectChanged(dependencies, sourceCollection, ev, true); changed {
								changedInputKeys.Insert(iKey)
							}
						}
					}
				}
			}
		}
		if !foundAny {
			for iKey, dependencies := range i.objectDependencies {
				if changed := objectChanged(dependencies, sourceCollection, ev, false); changed {
					changedInputKeys.Insert(iKey)
				}
			}
		}
	}
	return changedInputKeys
}

func objectChanged(dependencies []*dependency, sourceCollection collectionUID, ev Event[any], preFiltered bool) bool {
	for _, dep := range dependencies {
		id := dep.id
		if id != sourceCollection {
			continue
		}
		// For each input, we will check if it depends on this event.
		// We use Items() to check both the old and new object; we will recompute if either matched
		for _, item := range ev.Items() {
			match := dep.filter.Matches(item, preFiltered)
			if match {
				// Its a match! Return now. We don't need to check all dependencies, since we just need to find if any of them changed
				return true
			}
		}
	}
	return false
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
