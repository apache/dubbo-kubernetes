package krt

import (
	"fmt"
	"github.com/apache/dubbo-kubernetes/pkg/ptr"
	"github.com/apache/dubbo-kubernetes/pkg/util/sets"
	"k8s.io/apimachinery/pkg/util/wait"
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

type handlerSet[O any] struct {
	mu sync.RWMutex
	wg wait.Group
}

func newHandlerSet[O any]() *handlerSet[O] {
	return &handlerSet[O]{}
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

	return nil
}

type internalCollection[T any] interface {
	Collection[T]

	// Name is a human facing name for this collection.
	// Note this may not be universally unique
	name() string
	// Uid is an internal unique ID for this collection. MUST be globally unique
	uid() collectionUID

	dump() CollectionDump

	// Augment mutates an object for use in various function calls. See WithObjectAugmentation
	augment(any) any

	// Create a new index into the collection
	index(name string, extract func(o T) []string) indexer[T]
}

type CollectionDump struct {
	// Map of output key -> output
	Outputs map[string]any `json:"outputs,omitempty"`
	// Name of the input collection
	InputCollection string `json:"inputCollection,omitempty"`
	// Map of input key -> info
	Inputs map[string]InputDump `json:"inputs,omitempty"`
	// Synced returns whether the collection is synced or not
	Synced bool `json:"synced"`
}

type InputDump struct {
	Outputs      []string `json:"outputs,omitempty"`
	Dependencies []string `json:"dependencies,omitempty"`
}

type manyCollection[I, O any] struct {
	// collectionName provides the collectionName for this collection.
	collectionName string
	id             collectionUID
	// parent is the input collection we are building off of.
	parent          Collection[I]
	mu              sync.RWMutex
	collectionState multiIndex[I, O]
	dependencyState dependencyState[I]
	// internal indexes
	indexes map[string]collectionIndex[I, O]

	// eventHandlers is a list of event handlers registered for the collection. On any changes, each will be notified.
	eventHandlers *handlerSet[O]

	transformation TransformationMulti[I, O]

	// augmentation allows transforming an object into another for usage throughout the library. See WithObjectAugmentation.
	augmentation func(a any) any
	synced       chan struct{}
	stop         <-chan struct{}
	metadata     Metadata

	// onPrimaryInputEventHandler is a specialized internal handler that runs synchronously when a primary input changes
	onPrimaryInputEventHandler func(o []Event[I])

	syncer Syncer
}
