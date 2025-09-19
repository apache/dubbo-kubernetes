/*
 * Licensed to the Apache Software Foundation (ASF) under one or more
 * contributor license agreements.  See the NOTICE file distributed with
 * this work for additional information regarding copyright ownership.
 * The ASF licenses this file to You under the Apache License, Version 2.0
 * (the "License"); you may not use this file except in compliance with
 * the License.  You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package kclient

import (
	"context"
	"fmt"
	dubbogvr "github.com/apache/dubbo-kubernetes/pkg/config/schema/gvr"
	"github.com/apache/dubbo-kubernetes/pkg/config/schema/kubeclient"
	types "github.com/apache/dubbo-kubernetes/pkg/config/schema/kubetypes"
	"github.com/apache/dubbo-kubernetes/pkg/kube"
	"github.com/apache/dubbo-kubernetes/pkg/kube/controllers"
	"github.com/apache/dubbo-kubernetes/pkg/kube/informerfactory"
	"github.com/apache/dubbo-kubernetes/pkg/kube/kubetypes"
	"github.com/apache/dubbo-kubernetes/pkg/ptr"
	"github.com/apache/dubbo-kubernetes/pkg/util/sets"
	"github.com/apache/dubbo-kubernetes/pkg/util/slices"
	"github.com/apache/dubbo-kubernetes/sail/pkg/features"
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

func New[T controllers.ComparableObject](c kube.Client) Client[T] {
	return NewFiltered[T](c, Filter{})
}

type Filter = kubetypes.Filter

func NewFiltered[T controllers.ComparableObject](c kube.Client, filter Filter) Client[T] {
	gvr := types.MustToGVR[T](types.MustGVKFromType[T]())
	inf := kubeclient.GetInformerFiltered[T](c, ToOpts(c, gvr, filter), gvr)
	return &fullClient[T]{
		writeClient: writeClient[T]{client: c},
		Informer:    newInformerClient[T](gvr, inf, filter),
	}
}

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
	handler      cache.ResourceEventHandler
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

func (n *informerClient[T]) List(namespace string, selector klabels.Selector) []T {
	var res []T
	err := cache.ListAllByNamespace(n.informer.GetIndexer(), namespace, selector, func(i any) {
		cast := i.(T)
		if n.applyFilter(cast) {
			res = append(res, cast)
		}
	})

	if err != nil && features.EnableUnsafeAssertions {
		fmt.Printf("lister returned err for %v: %v", namespace, err)
	}
	return res
}

func (n *informerClient[T]) ListUnfiltered(namespace string, selector klabels.Selector) []T {
	var res []T
	err := cache.ListAllByNamespace(n.informer.GetIndexer(), namespace, selector, func(i any) {
		cast := i.(T)
		res = append(res, cast)
	})

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

func (n *informerClient[T]) HasSynced() bool {
	if !n.informer.HasSynced() {
		return false
	}
	n.handlerMu.RLock()
	defer n.handlerMu.RUnlock()
	for _, g := range n.registeredHandlers {
		if !g.registration.HasSynced() {
			return false
		}
	}
	return true
}

func (n *informerClient[T]) HasSyncedIgnoringHandlers() bool {
	return n.informer.HasSynced()
}

func (n *informerClient[T]) ShutdownHandlers() {
	n.handlerMu.Lock()
	defer n.handlerMu.Unlock()
	for _, c := range n.registeredHandlers {
		_ = n.informer.RemoveEventHandler(c.registration)
	}
	n.registeredHandlers = nil
}

func (n *informerClient[T]) AddEventHandler(h cache.ResourceEventHandler) cache.ResourceEventHandlerRegistration {
	fh := cache.FilteringResourceEventHandler{
		FilterFunc: func(obj interface{}) bool {
			if n.filter == nil {
				return true
			}
			return n.filter(obj)
		},
		Handler: h,
	}
	n.handlerMu.Lock()
	defer n.handlerMu.Unlock()
	reg, err := n.informer.AddEventHandler(fh)
	if err != nil {
		return neverReady{}
	}
	n.registeredHandlers = append(n.registeredHandlers, handlerRegistration{registration: reg, handler: h})
	return reg
}

type neverReady struct{}

func (a neverReady) HasSynced() bool {
	return false
}

func (n *informerClient[T]) Index(name string, extract func(o T) []string) RawIndexer {
	if _, ok := n.informer.GetIndexer().GetIndexers()[name]; !ok {
		if err := n.informer.AddIndexers(map[string]cache.IndexFunc{
			name: func(obj any) ([]string, error) {
				t := controllers.Extract[T](obj)
				return extract(t), nil
			},
		}); err != nil {
		}
	}
	ret := internalIndex{
		key:     name,
		indexer: n.informer.GetIndexer(),
		filter:  n.filter,
	}
	return ret
}

type internalIndex struct {
	key     string
	indexer cache.Indexer
	filter  func(t any) bool
}

func (n *informerClient[T]) ShutdownHandler(registration cache.ResourceEventHandlerRegistration) {
	n.handlerMu.Lock()
	defer n.handlerMu.Unlock()
	n.registeredHandlers = slices.FilterInPlace(n.registeredHandlers, func(h handlerRegistration) bool {
		return h.registration != registration
	})
	_ = n.informer.RemoveEventHandler(registration)
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
func (i internalIndex) Lookup(key string) []any {
	res, err := i.indexer.ByIndex(i.key, key)
	if err != nil {
	}
	if i.filter != nil {
		return slices.FilterInPlace(res, i.filter)
	}
	return res
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
