//
// Licensed to the Apache Software Foundation (ASF) under one or more
// contributor license agreements.  See the NOTICE file distributed with
// this work for additional information regarding copyright ownership.
// The ASF licenses this file to You under the Apache License, Version 2.0
// (the "License"); you may not use this file except in compliance with
// the License.  You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package kubetypes

import (
	"context"
	"github.com/apache/dubbo-kubernetes/pkg/cluster"
	"github.com/apache/dubbo-kubernetes/pkg/slices"
	"github.com/apache/dubbo-kubernetes/pkg/util/sets"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
)

type InformerType int

const (
	StandardInformer InformerType = iota
	DynamicInformer
	MetadataInformer
)

type WriteAPI[T runtime.Object] interface {
	Create(ctx context.Context, object T, opts metav1.CreateOptions) (T, error)
	Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts metav1.PatchOptions, subresources ...string) (result T, err error)
	Update(ctx context.Context, object T, opts metav1.UpdateOptions) (T, error)
	Delete(ctx context.Context, name string, opts metav1.DeleteOptions) error
}

type WriteStatusAPI[T runtime.Object] interface {
	UpdateStatus(ctx context.Context, object T, opts metav1.UpdateOptions) (T, error)
}

type DynamicObjectFilter interface {
	// Filter returns true if the input object or namespace string resides in a namespace selected for discovery
	Filter(obj any) bool
	// AddHandler registers a handler on namespace, which will be triggered when namespace selected or deselected.
	AddHandler(func(selected, deselected sets.String))
}

type CrdWatcher interface {
	HasSynced() bool
	KnownOrCallback(s schema.GroupVersionResource, f func(stop <-chan struct{})) bool
	WaitForCRD(s schema.GroupVersionResource, stop <-chan struct{}) bool
	Run(stop <-chan struct{})
}

type DelayedFilter interface {
	HasSynced() bool
	KnownOrCallback(f func(stop <-chan struct{})) bool
}

type staticFilter struct {
	f func(obj interface{}) bool
}

var _ DynamicObjectFilter = staticFilter{}

type Filter struct {
	LabelSelector   string
	FieldSelector   string
	Namespace       string
	ObjectFilter    DynamicObjectFilter
	ObjectTransform func(obj any) (any, error)
}

type composedFilter struct {
	filter DynamicObjectFilter
	extra  []func(obj any) bool
}

type InformerOptions struct {
	LabelSelector   string
	FieldSelector   string
	Namespace       string
	ObjectTransform func(obj any) (any, error)
	Cluster         cluster.ID
	InformerType    InformerType
}

func NewStaticObjectFilter(f func(obj any) bool) DynamicObjectFilter {
	return staticFilter{f}
}

func ComposeFilters(filter DynamicObjectFilter, extra ...func(obj any) bool) DynamicObjectFilter {
	return composedFilter{
		filter: filter,
		extra: slices.FilterInPlace(extra, func(f func(obj any) bool) bool {
			return f != nil
		}),
	}
}

func (s staticFilter) Filter(obj any) bool {
	return s.f(obj)
}

func (s staticFilter) AddHandler(func(selected, deselected sets.String)) {
	// Do nothing
}

func (f composedFilter) AddHandler(fn func(selected, deselected sets.String)) {
	if f.filter != nil {
		f.filter.AddHandler(fn)
	}
}

func (f composedFilter) Filter(obj any) bool {
	for _, filter := range f.extra {
		if !filter(obj) {
			return false
		}
	}
	if f.filter != nil {
		return f.filter.Filter(obj)
	}
	return true
}
