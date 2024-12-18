package informerfactory

import (
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/tools/cache"
)

type NewInformerFunc func() cache.SharedIndexInformer

type StartableInformer struct {
	Informer cache.SharedIndexInformer
	start    func(stopCh <-chan struct{})
}

type InformerFactory interface {
	Start(stopCh <-chan struct{})
	InformerFor(resource schema.GroupVersionResource, opts kubetypes.InformerOptions, newFunc NewInformerFunc) StartableInformer
	WaitForCacheSync(stopCh <-chan struct{}) bool
	Shutdown()
}

func (s StartableInformer) Start(stopCh <-chan struct{}) {
	s.start(stopCh)
}
