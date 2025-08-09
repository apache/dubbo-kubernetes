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

package controllers

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/cache"
)

type EventType int

const (
	// EventAdd is sent when an object is added
	EventAdd EventType = iota

	// EventUpdate is sent when an object is modified
	// Captures the modified object
	EventUpdate

	// EventDelete is sent when an object is deleted
	// Captures the object at the last known state
	EventDelete
)

func (event EventType) String() string {
	out := "unknown"
	switch event {
	case EventAdd:
		out = "add"
	case EventUpdate:
		out = "update"
	case EventDelete:
		out = "delete"
	}
	return out
}

type ComparableObject interface {
	comparable
	Object
}

type Object interface {
	metav1.Object
	runtime.Object
}

type EventHandler[T Object] struct {
	AddFunc         func(obj T)
	AddExtendedFunc func(obj T, initialSync bool)
	UpdateFunc      func(oldObj, newObj T)
	DeleteFunc      func(obj T)
}

func (e EventHandler[T]) OnAdd(obj interface{}, initialSync bool) {
	if e.AddExtendedFunc != nil {
		e.AddExtendedFunc(Extract[T](obj), initialSync)
	} else if e.AddFunc != nil {
		e.AddFunc(Extract[T](obj))
	}
}

func (e EventHandler[T]) OnUpdate(oldObj, newObj interface{}) {
	if e.UpdateFunc != nil {
		e.UpdateFunc(Extract[T](oldObj), Extract[T](newObj))
	}
}

func (e EventHandler[T]) OnDelete(obj interface{}) {
	if e.DeleteFunc != nil {
		e.DeleteFunc(Extract[T](obj))
	}
}

func IsNil[O comparable](o O) bool {
	var t O
	return o == t
}

func Extract[T Object](obj any) T {
	var empty T
	if obj == nil {
		return empty
	}
	o, ok := obj.(T)
	if !ok {
		tombstone, ok := obj.(cache.DeletedFinalStateUnknown)
		if !ok {
			return empty
		}
		o, ok = tombstone.Obj.(T)
		if !ok {
			return empty
		}
	}
	return o
}
