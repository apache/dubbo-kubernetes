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

package kubeclient

import (
	"context"
	"github.com/apache/dubbo-kubernetes/pkg/config/schema/kubetypes"
	"github.com/apache/dubbo-kubernetes/pkg/kube/informerfactory"
	ktypes "github.com/apache/dubbo-kubernetes/pkg/kube/kubetypes"
	"github.com/apache/dubbo-kubernetes/pkg/typemap"
	istioclient "istio.io/client-go/pkg/clientset/versioned"
	kubeext "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/metadata"
	"k8s.io/client-go/tools/cache"
)

type ClientGetter interface {
	// Ext returns the API extensions client.
	Ext() kubeext.Interface

	// Kube returns the core kube client
	Kube() kubernetes.Interface

	// Dynamic client.
	Dynamic() dynamic.Interface

	Dubbo() istioclient.Interface

	// Metadata returns the Metadata kube client.
	Metadata() metadata.Interface

	// Informers returns an informer factory.
	Informers() informerfactory.InformerFactory
}

type TypeRegistration[T runtime.Object] interface {
	kubetypes.RegisterType[T]

	// ListWatchFunc provides the necessary methods for list and
	// watch for the informer
	ListWatch(c ClientGetter, opts ktypes.InformerOptions) cache.ListerWatcher
}

var registerTypes = typemap.NewTypeMap()

func GetInformerFiltered[T runtime.Object](c ClientGetter, opts ktypes.InformerOptions, gvr schema.GroupVersionResource) informerfactory.StartableInformer {
	reg := typemap.Get[TypeRegistration[T]](registerTypes)
	if reg != nil {
		// This is registered type
		tr := *reg
		return c.Informers().InformerFor(tr.GetGVR(), opts, func() cache.SharedIndexInformer {
			inf := cache.NewSharedIndexInformer(
				tr.ListWatch(c, opts),
				tr.Object(),
				0,
				cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc},
			)
			setupInformer(opts, inf)
			return inf
		})
	}
	return GetInformerFilteredFromGVR(c, opts, gvr)
}

func setupInformer(opts ktypes.InformerOptions, inf cache.SharedIndexInformer) {
	if opts.ObjectTransform != nil {
		_ = inf.SetTransform(opts.ObjectTransform)
	} else {
		_ = inf.SetTransform(stripUnusedFields)
	}
}

func GetInformerFilteredFromGVR(c ClientGetter, opts ktypes.InformerOptions, g schema.GroupVersionResource) informerfactory.StartableInformer {
	switch opts.InformerType {
	case ktypes.DynamicInformer:
		return getInformerFilteredDynamic(c, opts, g)
	case ktypes.MetadataInformer:
		return getInformerFilteredMetadata(c, opts, g)
	default:
		return getInformerFiltered(c, opts, g)
	}
}

func getInformerFilteredDynamic(c ClientGetter, opts ktypes.InformerOptions, g schema.GroupVersionResource) informerfactory.StartableInformer {
	return c.Informers().InformerFor(g, opts, func() cache.SharedIndexInformer {
		inf := cache.NewSharedIndexInformerWithOptions(
			&cache.ListWatch{
				ListFunc: func(options metav1.ListOptions) (runtime.Object, error) {
					options.FieldSelector = opts.FieldSelector
					options.LabelSelector = opts.LabelSelector
					return c.Dynamic().Resource(g).Namespace(opts.Namespace).List(context.Background(), options)
				},
				WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
					options.FieldSelector = opts.FieldSelector
					options.LabelSelector = opts.LabelSelector
					return c.Dynamic().Resource(g).Namespace(opts.Namespace).Watch(context.Background(), options)
				},
			},
			&unstructured.Unstructured{},
			cache.SharedIndexInformerOptions{
				ResyncPeriod:      0,
				Indexers:          cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc},
				ObjectDescription: g.String(),
			},
		)
		setupInformer(opts, inf)
		return inf
	})
}

func getInformerFilteredMetadata(c ClientGetter, opts ktypes.InformerOptions, g schema.GroupVersionResource) informerfactory.StartableInformer {
	return c.Informers().InformerFor(g, opts, func() cache.SharedIndexInformer {
		inf := cache.NewSharedIndexInformerWithOptions(
			&cache.ListWatch{
				ListFunc: func(options metav1.ListOptions) (runtime.Object, error) {
					options.FieldSelector = opts.FieldSelector
					options.LabelSelector = opts.LabelSelector
					return c.Metadata().Resource(g).Namespace(opts.Namespace).List(context.Background(), options)
				},
				WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
					options.FieldSelector = opts.FieldSelector
					options.LabelSelector = opts.LabelSelector
					return c.Metadata().Resource(g).Namespace(opts.Namespace).Watch(context.Background(), options)
				},
			},
			&metav1.PartialObjectMetadata{},
			cache.SharedIndexInformerOptions{
				ResyncPeriod:      0,
				Indexers:          cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc},
				ObjectDescription: g.String(),
			},
		)
		setupInformer(opts, inf)
		return inf
	})
}

func stripUnusedFields(obj any) (any, error) {
	t, ok := obj.(metav1.ObjectMetaAccessor)
	if !ok {
		// shouldn't happen
		return obj, nil
	}
	// ManagedFields is large and we never use it
	t.GetObjectMeta().SetManagedFields(nil)
	return obj, nil
}
