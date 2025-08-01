package krt

import "github.com/apache/dubbo-kubernetes/pkg/kube/controllers"

type FetchOption func(*dependency)

type HandlerContext interface {
	// DiscardResult triggers the result of this invocation to be skipped
	// This allows a collection to mark that the current state is *invalid* and should use the last-known state.
	//
	// Note: this differs from returning `nil`, which would otherwise wipe out the last known state.
	//
	// Note: if the current resource has never been computed, the result will not be discarded if it is non-nil. This allows
	// setting a default. For example, you may always return a static default config if the initial results are invalid,
	// but not revert to this config if later results are invalid. Results can unconditionally be discarded by returning nil.
	DiscardResult()
	// _internalHandler is an interface that can only be implemented by this package.
	_internalHandler()
}

type (
	TransformationEmpty[T any] func(ctx HandlerContext) *T
)

type Metadata map[string]any

type Event[T any] struct {
	// Old object, set on Update or Delete.
	Old *T
	// New object, set on Add or Update
	New *T
	// Event is the change type
	Event controllers.EventType
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
	// GetKey returns an object by its key, if present. Otherwise, nil is returned.
	GetKey(k string) *T

	// List returns all objects in the collection.
	// Order of the list is undefined.
	List() []T

	// EventStream provides event handling capabilities for the collection, allowing clients to subscribe to changes
	// and receive notifications when objects are added, modified, or removed.
	EventStream[T]

	// Metadata returns the metadata associated with this collection.
	// This can be used to store and retrieve arbitrary key-value pairs
	// that provide additional context or configuration for the collection.
	Metadata() Metadata
}

type Singleton[T any] interface {
	// Get returns the object, or nil if there is none.
	Get() *T
	// Register adds an event watcher to the object. Any time it changes, the handler will be called
	Register(f func(o Event[T])) HandlerRegistration
	AsCollection() Collection[T]
	Metadata() Metadata
}
