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
	"github.com/apache/dubbo-kubernetes/pkg/util/ptr"

	"github.com/apache/dubbo-kubernetes/dubbod/planet/pkg/features"
	dubbogvr "github.com/apache/dubbo-kubernetes/pkg/config/schema/gvr"
	"github.com/apache/dubbo-kubernetes/pkg/config/schema/kubeclient"
	types "github.com/apache/dubbo-kubernetes/pkg/config/schema/kubetypes"
	"github.com/apache/dubbo-kubernetes/pkg/kube"
	"github.com/apache/dubbo-kubernetes/pkg/kube/controllers"
	"github.com/apache/dubbo-kubernetes/pkg/kube/informerfactory"
	"github.com/apache/dubbo-kubernetes/pkg/kube/kubetypes"
	"github.com/apache/dubbo-kubernetes/pkg/slices"
	"github.com/apache/dubbo-kubernetes/pkg/util/sets"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	klabels "k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime/schema"
	apitypes "k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/cache"

	"sync"
	"sync/atomic"

	dubbolog "github.com/apache/dubbo-kubernetes/pkg/log"
)

var log = dubbolog.RegisterScope("kclient", "kclient debugging")

type Filter = kubetypes.Filter

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

type fullClient[T controllers.Object] struct {
	writeClient[T]
	Informer[T]
}

type writeClient[T controllers.Object] struct {
	client kube.Client
}

type internalIndex struct {
	key     string
	indexer cache.Indexer
	filter  func(t any) bool
}

type neverReady struct{}

func New[T controllers.ComparableObject](c kube.Client) Client[T] {
	return NewFiltered[T](c, Filter{})
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

func NewMetadata(c kube.Client, gvr schema.GroupVersionResource, filter Filter) Informer[*metav1.PartialObjectMetadata] {
	opts := ToOpts(c, gvr, filter)
	opts.InformerType = kubetypes.MetadataInformer
	inf := kubeclient.GetInformerFilteredFromGVR(c, opts, gvr)
	return newInformerClient[*metav1.PartialObjectMetadata](gvr, inf, filter)
}

func NewUntypedInformer(c kube.Client, gvr schema.GroupVersionResource, filter Filter) Untyped {
	inf := kubeclient.GetInformerFilteredFromGVR(c, ToOpts(c, gvr, filter), gvr)
	return newInformerClient[controllers.Object](gvr, inf, filter)
}

func NewDelayedInformer[T controllers.ComparableObject](c kube.Client, gvr schema.GroupVersionResource, informerType kubetypes.InformerType, filter Filter) Informer[T] {
	watcher := c.CrdWatcher()
	if watcher == nil {
		log.Info("NewDelayedInformer called without a CrdWatcher enabled")
	}
	delay := newDelayedFilter(gvr, watcher)
	inf := func() informerfactory.StartableInformer {
		opts := ToOpts(c, gvr, filter)
		opts.InformerType = informerType
		return kubeclient.GetInformerFiltered[T](c, opts, gvr)
	}
	return newDelayedInformer[T](gvr, inf, delay, filter)
}

func newDelayedInformer[T controllers.ComparableObject](gvr schema.GroupVersionResource, getInf func() informerfactory.StartableInformer, delay kubetypes.DelayedFilter, filter Filter) Informer[T] {
	delayedClient := &delayedClient[T]{
		inf:     new(atomic.Pointer[Informer[T]]),
		delayed: delay,
	}

	// If resource is not yet known, we will use the delayedClient.
	// When the resource is later loaded, the callback will trigger and swap our dummy delayedClient
	// with a full client
	readyNow := delay.KnownOrCallback(func(stop <-chan struct{}) {
		// The inf() call is responsible for starting the informer
		inf := getInf()
		fc := &informerClient[T]{
			informer:      inf.Informer,
			startInformer: inf.Start,
		}
		applyDynamicFilter(filter, gvr, fc)
		inf.Start(stop)
		log.Infof("%v is now ready, building client", gvr.GroupResource())
		// Swap out the dummy client with the full one
		delayedClient.set(fc)
	})
	if !readyNow {
		log.Debugf("%v is not ready now, building delayed client", gvr.GroupResource())
		return delayedClient
	}
	log.Debugf("%v ready now, building client", gvr.GroupResource())
	return newInformerClient[T](gvr, getInf(), filter)
}

func NewFiltered[T controllers.ComparableObject](c kube.Client, filter Filter) Client[T] {
	gvr := types.MustToGVR[T](types.MustGVKFromType[T]())
	inf := kubeclient.GetInformerFiltered[T](c, ToOpts(c, gvr, filter), gvr)
	return &fullClient[T]{
		writeClient: writeClient[T]{client: c},
		Informer:    newInformerClient[T](gvr, inf, filter),
	}
}

func newInformerClient[T controllers.ComparableObject](gvr schema.GroupVersionResource, inf informerfactory.StartableInformer, filter Filter) Informer[T] {
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

	if err != nil {
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

	if err != nil {
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

func (n *informerClient[T]) ShutdownHandler(registration cache.ResourceEventHandlerRegistration) {
	n.handlerMu.Lock()
	defer n.handlerMu.Unlock()
	n.registeredHandlers = slices.FilterInPlace(n.registeredHandlers, func(h handlerRegistration) bool {
		return h.registration != registration
	})
	_ = n.informer.RemoveEventHandler(registration)
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

func (a neverReady) HasSynced() bool {
	return false
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

func keyFunc(name, namespace string) string {
	if len(namespace) == 0 {
		return name
	}
	return namespace + "/" + name
}
