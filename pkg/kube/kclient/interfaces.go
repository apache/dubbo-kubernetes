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
	HasSynced() bool
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
