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

package krt

import (
	"fmt"

	"github.com/apache/dubbo-kubernetes/pkg/util/ptr"

	"github.com/apache/dubbo-kubernetes/pkg/kube"
	"github.com/apache/dubbo-kubernetes/pkg/kube/controllers"
	"github.com/apache/dubbo-kubernetes/pkg/kube/kclient"
	"github.com/apache/dubbo-kubernetes/pkg/slices"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	klabels "k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/tools/cache"
)

var _ internalCollection[controllers.Object] = &informer[controllers.Object]{}

type informer[I controllers.ComparableObject] struct {
	inf            kclient.Informer[I]
	collectionName string
	id             collectionUID
	eventHandlers  *handlers[I]
	augmentation   func(a any) any
	synced         chan struct{}
	baseSyncer     Syncer
	metadata       Metadata
}

type informerIndex[I any] struct {
	idx kclient.RawIndexer
}

type informerHandlerRegistration struct {
	Syncer
	remove func()
}

func WrapClient[I controllers.ComparableObject](c kclient.Informer[I], opts ...CollectionOption) Collection[I] {
	o := buildCollectionOptions(opts...)
	if o.name == "" {
		o.name = fmt.Sprintf("Informer[%v]", ptr.TypeName[I]())
	}
	h := &informer[I]{
		inf:            c,
		collectionName: o.name,
		id:             nextUID(),
		eventHandlers:  &handlers[I]{},
		augmentation:   o.augmentation,
		synced:         make(chan struct{}),
	}
	h.baseSyncer = channelSyncer{
		name:   h.collectionName,
		synced: h.synced,
	}

	if o.metadata != nil {
		h.metadata = o.metadata
	}
	go func() {
		defer c.ShutdownHandlers()
		if !kube.WaitForCacheSync(o.name, o.stop, c.HasSyncedIgnoringHandlers) {
			return
		}
		close(h.synced)
		log.Infof("%v synced", h.name())

		<-o.stop
	}()
	return h
}

func informerEventHandler[I controllers.ComparableObject](handler func(o Event[I], initialSync bool)) cache.ResourceEventHandler {
	return controllers.EventHandler[I]{
		AddExtendedFunc: func(obj I, initialSync bool) {
			handler(Event[I]{
				New:   &obj,
				Event: controllers.EventAdd,
			}, initialSync)
		},
		UpdateFunc: func(oldObj, newObj I) {
			handler(Event[I]{
				Old:   &oldObj,
				New:   &newObj,
				Event: controllers.EventUpdate,
			}, false)
		},
		DeleteFunc: func(obj I) {
			handler(Event[I]{
				Old:   &obj,
				Event: controllers.EventDelete,
			}, false)
		},
	}
}

func (i informerHandlerRegistration) UnregisterHandler() {
	i.remove()
}

func (i *informer[I]) name() string {
	return i.collectionName
}

func (i *informer[I]) WaitUntilSynced(stop <-chan struct{}) bool {
	return i.baseSyncer.WaitUntilSynced(stop)
}

func (i *informer[I]) Synced() Syncer {
	return channelSyncer{
		name:   i.collectionName,
		synced: i.synced,
	}
}

func (i *informer[I]) HasSynced() bool {
	return i.baseSyncer.HasSynced()
}

func (i *informer[I]) Metadata() Metadata {
	return i.metadata
}

func (i *informer[I]) GetKey(k string) *I {
	if got := i.inf.Get(k, ""); !controllers.IsNil(got) {
		return &got
	}
	return nil
}

func (i *informer[I]) List() []I {
	res := i.inf.List(metav1.NamespaceAll, klabels.Everything())
	return res
}

func (i *informer[I]) Register(f func(o Event[I])) HandlerRegistration {
	return registerHandlerAsBatched[I](i, f)
}

func (i *informer[I]) RegisterBatch(f func(o []Event[I]), runExistingState bool) HandlerRegistration {
	synced := i.inf.AddEventHandler(informerEventHandler[I](func(o Event[I], initialSync bool) {
		// Only process events if runExistingState is true OR this is not an initial sync event
		// This matches Istio's behavior: runExistingState=false means skip initial sync events
		if runExistingState || !initialSync {
			// Log all events to help diagnose missing events
			var nameStr, nsStr string
			if o.New != nil {
				if objWithNs, ok := any(*o.New).(interface {
					GetNamespace() string
					GetName() string
				}); ok {
					nsStr = objWithNs.GetNamespace()
					nameStr = objWithNs.GetName()
				}
			} else if o.Old != nil {
				if objWithNs, ok := any(*o.Old).(interface {
					GetNamespace() string
					GetName() string
				}); ok {
					nsStr = objWithNs.GetNamespace()
					nameStr = objWithNs.GetName()
				}
			}
			log.Debugf("informer.RegisterBatch: processing event %s for %s/%s (initialSync=%v, runExistingState=%v)", o.Event, nsStr, nameStr, initialSync, runExistingState)
			f([]Event[I]{o})
		} else {
			// Log skipped events to help diagnose
			var nameStr, nsStr string
			if o.New != nil {
				if objWithNs, ok := any(*o.New).(interface {
					GetNamespace() string
					GetName() string
				}); ok {
					nsStr = objWithNs.GetNamespace()
					nameStr = objWithNs.GetName()
				}
			}
			log.Debugf("informer.RegisterBatch: skipping initial sync event for %s/%s (initialSync=%v, runExistingState=%v)", nsStr, nameStr, initialSync, runExistingState)
		}
	}))
	base := i.baseSyncer
	handler := pollSyncer{
		name: fmt.Sprintf("%v handler", i.name()),
		f:    synced.HasSynced,
	}
	sync := multiSyncer{syncers: []Syncer{base, handler}}
	return informerHandlerRegistration{
		Syncer: sync,
		remove: func() {
			i.inf.ShutdownHandler(synced)
		},
	}
}

func (i *informer[I]) uid() collectionUID {
	return i.id
}

func (i *informer[I]) dump() CollectionDump {
	return CollectionDump{
		Outputs: eraseMap(slices.GroupUnique(i.inf.List(metav1.NamespaceAll, klabels.Everything()), getTypedKey)),
		Synced:  i.HasSynced(),
	}
}

func (i *informer[I]) augment(a any) any {
	if i.augmentation != nil {
		return i.augmentation(a)
	}
	return a
}

func (i *informer[I]) index(name string, extract func(o I) []string) indexer[I] {
	idx := i.inf.Index(name, extract)
	return &informerIndex[I]{
		idx: idx,
	}
}

func (ii *informerIndex[I]) Lookup(key string) []I {
	return slices.Map(ii.idx.Lookup(key), func(i any) I {
		return i.(I)
	})
}
