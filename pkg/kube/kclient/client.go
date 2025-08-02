package kclient

import (
	"context"
	"fmt"
	"github.com/apache/dubbo-kubernetes/navigator/pkg/features"
	dubbogvr "github.com/apache/dubbo-kubernetes/pkg/config/schema/gvr"
	"github.com/apache/dubbo-kubernetes/pkg/config/schema/kubeclient"
	types "github.com/apache/dubbo-kubernetes/pkg/config/schema/kubetypes"
	"github.com/apache/dubbo-kubernetes/pkg/kube"
	"github.com/apache/dubbo-kubernetes/pkg/kube/controllers"
	"github.com/apache/dubbo-kubernetes/pkg/kube/informerfactory"
	"github.com/apache/dubbo-kubernetes/pkg/kube/kubetypes"
	"github.com/apache/dubbo-kubernetes/pkg/ptr"
	"github.com/apache/dubbo-kubernetes/pkg/util/sets"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	klabels "k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime/schema"
	apitypes "k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/cache"
	"sync"
)

type fullClient[T controllers.Object] struct {
	writeClient[T]
	Informer[T]
}

type writeClient[T controllers.Object] struct {
	client kube.Client
}

func NewFiltered[T controllers.ComparableObject](c kube.Client, filter Filter) Client[T] {
	gvr := types.MustToGVR[T](types.MustGVKFromType[T]())
	inf := kubeclient.GetInformerFiltered[T](c, ToOpts(c, gvr, filter), gvr)
	return &fullClient[T]{
		writeClient: writeClient[T]{client: c},
		Informer:    newInformerClient[T](gvr, inf, filter),
	}
}

type Filter = kubetypes.Filter

func ToOpts(c kube.Client, gvr schema.GroupVersionResource, filter Filter) kubetypes.InformerOptions {
	ns := filter.Namespace
	if !dubbogvr.IsClusterScoped(gvr) && ns == "" {
		ns = features.InformerWatchNamespace
	}
	return kubetypes.InformerOptions{
		LabelSelector:   filter.LabelSelector,
		FieldSelector:   filter.FieldSelector,
		Namespace:       ns,
		ObjectTransform: filter.ObjectTransform,
		Cluster:         c.ClusterID(),
	}
}

type handlerRegistration struct {
	registration cache.ResourceEventHandlerRegistration
	// handler is the actual handler. Note this does NOT have the filtering applied.
	handler cache.ResourceEventHandler
}

type informerClient[T controllers.Object] struct {
	informer      cache.SharedIndexInformer
	startInformer func(stopCh <-chan struct{})
	filter        func(t any) bool

	handlerMu          sync.RWMutex
	registeredHandlers []handlerRegistration
}

func newInformerClient[T controllers.ComparableObject](
	gvr schema.GroupVersionResource,
	inf informerfactory.StartableInformer,
	filter Filter,
) Informer[T] {
	ic := &informerClient[T]{
		informer:      inf.Informer,
		startInformer: inf.Start,
	}
	if filter.ObjectFilter != nil {
		applyDynamicFilter(filter, gvr, ic)
	}
	return ic
}

func applyDynamicFilter[T controllers.ComparableObject](filter Filter, gvr schema.GroupVersionResource, ic *informerClient[T]) {
	if filter.ObjectFilter != nil {
		ic.filter = filter.ObjectFilter.Filter
		filter.ObjectFilter.AddHandler(func(added, removed sets.String) {
			ic.handlerMu.RLock()
			defer ic.handlerMu.RUnlock()
			if gvr == dubbogvr.Namespace {
				for _, item := range ic.ListUnfiltered(metav1.NamespaceAll, klabels.Everything()) {
					if !added.Contains(item.GetName()) {
						continue
					}
					for _, c := range ic.registeredHandlers {
						c.handler.OnAdd(item, false)
					}
				}
			} else {
				for ns := range added {
					for _, item := range ic.ListUnfiltered(ns, klabels.Everything()) {
						for _, c := range ic.registeredHandlers {
							c.handler.OnAdd(item, false)
						}
					}
				}
				for ns := range removed {
					for _, item := range ic.ListUnfiltered(ns, klabels.Everything()) {
						for _, c := range ic.registeredHandlers {
							c.handler.OnDelete(item)
						}
					}
				}
			}
		})
	}
}

func (n *informerClient[T]) ListUnfiltered(namespace string, selector klabels.Selector) []T {
	var res []T
	err := cache.ListAllByNamespace(n.informer.GetIndexer(), namespace, selector, func(i any) {
		cast := i.(T)
		res = append(res, cast)
	})

	// Should never happen
	if err != nil && features.EnableUnsafeAssertions {
		fmt.Printf("lister returned err for %v: %v", namespace, err)
	}
	return res
}

func (n *informerClient[T]) Start(stopCh <-chan struct{}) {
	n.startInformer(stopCh)
}

func (n *informerClient[T]) Get(name, namespace string) T {
	obj, exists, err := n.informer.GetIndexer().GetByKey(keyFunc(name, namespace))
	if err != nil {
		return ptr.Empty[T]()
	}
	if !exists {
		return ptr.Empty[T]()
	}
	cast := obj.(T)
	if !n.applyFilter(cast) {
		return ptr.Empty[T]()
	}
	return cast
}

func keyFunc(name, namespace string) string {
	if len(namespace) == 0 {
		return name
	}
	return namespace + "/" + name
}

func (n *informerClient[T]) applyFilter(t T) bool {
	if n.filter == nil {
		return true
	}
	return n.filter(t)
}

func (n *writeClient[T]) Create(object T) (T, error) {
	api := kubeclient.GetWriteClient[T](n.client, object.GetNamespace())
	return api.Create(context.Background(), object, metav1.CreateOptions{})
}

func (n *writeClient[T]) Update(object T) (T, error) {
	api := kubeclient.GetWriteClient[T](n.client, object.GetNamespace())
	return api.Update(context.Background(), object, metav1.UpdateOptions{})
}

func (n *writeClient[T]) Patch(name, namespace string, pt apitypes.PatchType, data []byte) (T, error) {
	api := kubeclient.GetWriteClient[T](n.client, namespace)
	return api.Patch(context.Background(), name, pt, data, metav1.PatchOptions{})
}

func (n *writeClient[T]) PatchStatus(name, namespace string, pt apitypes.PatchType, data []byte) (T, error) {
	api := kubeclient.GetWriteClient[T](n.client, namespace)
	return api.Patch(context.Background(), name, pt, data, metav1.PatchOptions{}, "status")
}

func (n *writeClient[T]) ApplyStatus(name, namespace string, pt apitypes.PatchType, data []byte, fieldManager string) (T, error) {
	api := kubeclient.GetWriteClient[T](n.client, namespace)
	return api.Patch(context.Background(), name, pt, data, metav1.PatchOptions{
		Force:        ptr.Of(true),
		FieldManager: fieldManager,
	}, "status")
}

func (n *writeClient[T]) UpdateStatus(object T) (T, error) {
	api, ok := kubeclient.GetWriteClient[T](n.client, object.GetNamespace()).(kubetypes.WriteStatusAPI[T])
	if !ok {
		return ptr.Empty[T](), fmt.Errorf("%T does not support UpdateStatus", object)
	}
	return api.UpdateStatus(context.Background(), object, metav1.UpdateOptions{})
}

func (n *writeClient[T]) Delete(name, namespace string) error {
	api := kubeclient.GetWriteClient[T](n.client, namespace)
	return api.Delete(context.Background(), name, metav1.DeleteOptions{})
}
