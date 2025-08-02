package krt

import "github.com/apache/dubbo-kubernetes/pkg/kube/controllers"

type FetchOption func(*dependency)

type HandlerContext interface {
	DiscardResult()
	_internalHandler()
}

type (
	TransformationEmpty[T any]     func(ctx HandlerContext) *T
	TransformationMulti[I, O any]  func(ctx HandlerContext, i I) []O
	TransformationSingle[I, O any] func(ctx HandlerContext, i I) *O
)

type Metadata map[string]any

type Event[T any] struct {
	Old   *T
	New   *T
	Event controllers.EventType
}

type indexer[T any] interface {
	Lookup(key string) []T
}

type EventStream[T any] interface {
	Syncer
	Register(f func(o Event[T])) HandlerRegistration
	RegisterBatch(f func(o []Event[T]), runExistingState bool) HandlerRegistration
}

type CollectionOption func(*collectionOptions)

type collectionOptions struct {
	name          string
	augmentation  func(o any) any
	stop          <-chan struct{}
	joinUnchecked bool

	indexCollectionFromString func(string) any
	metadata                  Metadata
}

type HandlerRegistration interface {
	Syncer
	UnregisterHandler()
}

type Collection[T any] interface {
	GetKey(k string) *T
	List() []T
	EventStream[T]
	Metadata() Metadata
}

type Singleton[T any] interface {
	Get() *T
	Register(f func(o Event[T])) HandlerRegistration
	AsCollection() Collection[T]
	Metadata() Metadata
}
