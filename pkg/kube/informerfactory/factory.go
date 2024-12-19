package informerfactory

import (
	"github.com/apache/dubbo-kubernetes/pkg/util/sets"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"k8s.io/client-go/tools/cache"
	"sync"
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

type informerKey struct {
	gvr           schema.GroupVersionResource
	labelSelector string
	fieldSelector string
	informerType  kubetypes.InformerType
	namespace     string
}

type informerFactory struct {
	lock             sync.Mutex
	informers        map[informerKey]builtInformer
	startedInformers sets.Set[informerKey]
	wg               sync.WaitGroup
	shuttingDown     bool
}

type builtInformer struct {
	informer        cache.SharedIndexInformer
	objectTransform func(obj any) (any, error)
}

func NewSharedInformerFactory() InformerFactory {
	return &informerFactory{
		informers:        map[informerKey]builtInformer{},
		startedInformers: sets.New[informerKey](),
	}
}

func (s StartableInformer) Start(stopCh <-chan struct{}) {
	s.start(stopCh)
}

func (f *informerFactory) Start(stopCh <-chan struct{}) {
	f.lock.Lock()
	defer f.lock.Unlock()

	if f.shuttingDown {
		return
	}

	for informerType, informer := range f.informers {
		if !f.startedInformers.Contains(informerType) {
			informer := informer
			f.wg.Add(1)
			go func() {
				defer f.wg.Done()
				informer.informer.Run(stopCh)
			}()
			f.startedInformers.Insert(informerType)
		}
	}
}

func (f *informerFactory) WaitForCacheSync(stopCh <-chan struct{}) bool {
	informers := func() []cache.SharedIndexInformer {
		f.lock.Lock()
		defer f.lock.Unlock()
		informers := make([]cache.SharedIndexInformer, 0, len(f.informers))
		for informerKey, informer := range f.informers {
			if f.startedInformers.Contains(informerKey) {
				informers = append(informers, informer.informer)
			}
		}
		return informers
	}()

	for _, informer := range informers {
		if !cache.WaitForCacheSync(stopCh, informer.HasSynced) {
			return false
		}
	}
	return true
}

func (f *informerFactory) Shutdown() {
	defer f.wg.Wait()

	f.lock.Lock()
	defer f.lock.Unlock()
	f.shuttingDown = true
}

func (f *informerFactory) InformerFor(resource schema.GroupVersionResource, opts kubetypes.InformerOptions, newFunc NewInformerFunc) StartableInformer {
	f.lock.Lock()
	defer f.lock.Unlock()

	key := informerKey{
		gvr:           resource,
		labelSelector: opts.LabelSelector,
		fieldSelector: opts.FieldSelector,
		informerType:  opts.InformerType,
		namespace:     opts.Namespace,
	}
	inf, exists := f.informers[key]
	if exists {
		checkInformerOverlap(inf, resource, opts)
		return f.makeStartableInformer(inf.informer, key)
	}

	informer := newFunc()
	f.informers[key] = builtInformer{
		informer:        informer,
		objectTransform: opts.ObjectTransform,
	}

	return f.makeStartableInformer(informer, key)
}
