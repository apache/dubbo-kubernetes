package kclient

import (
	"github.com/apache/dubbo-kubernetes/pkg/kube/controllers"
	klabels "k8s.io/apimachinery/pkg/labels"
	apitypes "k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/cache"
)

type Untyped = Informer[controllers.Object]

type Reader[T controllers.Object] interface {
	Get(name, namespace string) T
	List(namespace string, selector klabels.Selector) []T
}

type Writer[T controllers.Object] interface {
	Create(object T) (T, error)
	Update(object T) (T, error)
	UpdateStatus(object T) (T, error)
	Patch(name, namespace string, pt apitypes.PatchType, data []byte) (T, error)
	PatchStatus(name, namespace string, pt apitypes.PatchType, data []byte) (T, error)
	ApplyStatus(name, namespace string, pt apitypes.PatchType, data []byte, fieldManager string) (T, error)
	Delete(name, namespace string) error
}

type ReadWriter[T controllers.Object] interface {
	Reader[T]
	Writer[T]
}

type Informer[T controllers.Object] interface {
	Reader[T]
	ListUnfiltered(namespace string, selector klabels.Selector) []T
	Start(stop <-chan struct{})
	ShutdownHandlers()
	ShutdownHandler(registration cache.ResourceEventHandlerRegistration)
	HasSyncedIgnoringHandlers() bool
	AddEventHandler(h cache.ResourceEventHandler) cache.ResourceEventHandlerRegistration
	Index(name string, extract func(o T) []string) RawIndexer
}

type Client[T controllers.Object] interface {
	Reader[T]
	Writer[T]
	Informer[T]
}

type RawIndexer interface {
	Lookup(key string) []any
}
