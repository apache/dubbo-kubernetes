package krt

import "github.com/apache/dubbo-kubernetes/pkg/kube/controllers"

type FetchOption func(*dependency)

type HandlerContext interface {
	DiscardResult()
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

type Equaler[K any] interface {
	Equals(k K) bool
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

type LabelSelectorer interface {
	GetLabelSelector() map[string]string
}

type Labeler interface {
	GetLabels() map[string]string
}

func (e Event[T]) Items() []T {
	res := make([]T, 0, 2)
	if e.Old != nil {
		res = append(res, *e.Old)
	}
	if e.New != nil {
		res = append(res, *e.New)
	}
	return res
}

func (e Event[T]) Latest() T {
	if e.New != nil {
		return *e.New
	}
	return *e.Old
}

type ResourceNamer interface {
	ResourceName() string
}

type uidable interface {
	uid() collectionUID
}
