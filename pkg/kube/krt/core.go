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

import "github.com/apache/dubbo-kubernetes/pkg/kube/controllers"

type Metadata map[string]any

type FetchOption func(*dependency)

type HandlerContext interface {
	DiscardResult()
	_internalHandler()
}

type Key[O any] string

type (
	TransformationEmpty[T any]     func(ctx HandlerContext) *T
	TransformationMulti[I, O any]  func(ctx HandlerContext, i I) []O
	TransformationSingle[I, O any] func(ctx HandlerContext, i I) *O
)

type Event[T any] struct {
	Old   *T
	New   *T
	Event controllers.EventType
}

type Equaler[K any] interface {
	Equals(k K) bool
}

type indexer[T any] interface {
	Lookup(key string) []T
}

type EventStream[T any] interface {
	Syncer
	Register(f func(o Event[T])) HandlerRegistration
	RegisterBatch(f func(o []Event[T]), runExistingState bool) HandlerRegistration
}

type CollectionOption func(*collectionOptions)

type HandlerRegistration interface {
	Syncer
	UnregisterHandler()
}

type Collection[T any] interface {
	GetKey(k string) *T
	List() []T
	EventStream[T]
	Metadata() Metadata
}

type Singleton[T any] interface {
	Get() *T
	Register(f func(o Event[T])) HandlerRegistration
	AsCollection() Collection[T]
	Metadata() Metadata
}

type LabelSelectorer interface {
	GetLabelSelector() map[string]string
}

type Labeler interface {
	GetLabels() map[string]string
}

func (e Event[T]) Items() []T {
	res := make([]T, 0, 2)
	if e.Old != nil {
		res = append(res, *e.Old)
	}
	if e.New != nil {
		res = append(res, *e.New)
	}
	return res
}

func (e Event[T]) Latest() T {
	if e.New != nil {
		return *e.New
	}
	return *e.Old
}

type ResourceNamer interface {
	ResourceName() string
}

type uidable interface {
	uid() collectionUID
}

type internalCollection[T any] interface {
	Collection[T]

	// Name is a human facing name for this collection.
	// Note this may not be universally unique
	name() string
	// Uid is an internal unique ID for this collection. MUST be globally unique
	uid() collectionUID

	dump() CollectionDump

	// Augment mutates an object for use in various function calls. See WithObjectAugmentation
	augment(any) any

	// Create a new index into the collection
	index(name string, extract func(o T) []string) indexer[T]
}
