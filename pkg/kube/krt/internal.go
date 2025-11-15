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
	"github.com/apache/dubbo-kubernetes/pkg/config"
	"github.com/apache/dubbo-kubernetes/pkg/ptr"
	"go.uber.org/atomic"
	"google.golang.org/protobuf/proto"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"reflect"
)

var globalUIDCounter = atomic.NewUint64(1)

type collectionUID uint64

type erasedEventHandler = func(o []Event[any])

type registerDependency interface {
	registerDependency(*dependency, Syncer, func(f erasedEventHandler) Syncer)
	name() string
}

type collectionOptions struct {
	name         string
	augmentation func(o any) any
	stop         <-chan struct{}
	metadata     Metadata
	debugger     *DebugHandler
}

type dependency struct {
	id             collectionUID
	collectionName string
	filter         *filter
}

type indexedDependency struct {
	id  collectionUID
	key string
	typ indexedDependencyType
}

func GetStop(opts ...CollectionOption) <-chan struct{} {
	o := buildCollectionOptions(opts...)
	return o.stop
}

func Equal[O any](a, b O) bool {
	if ak, ok := any(a).(Equaler[O]); ok {
		return ak.Equals(b)
	}
	if ak, ok := any(a).(Equaler[*O]); ok {
		return ak.Equals(&b)
	}
	if pk, ok := any(&a).(Equaler[O]); ok {
		return pk.Equals(b)
	}
	if pk, ok := any(&a).(Equaler[*O]); ok {
		return pk.Equals(&b)
	}

	ap, ok := any(a).(proto.Message)
	if ok {
		if reflect.TypeOf(ap.ProtoReflect().Interface()) == reflect.TypeOf(ap) {
			return proto.Equal(ap, any(b).(proto.Message))
		}
		panic(fmt.Sprintf("unable to compare object %T; perhaps it is embedding a protobuf? Provide an Equaler implementation", a))
	}
	return reflect.DeepEqual(a, b)
}

func buildCollectionOptions(opts ...CollectionOption) collectionOptions {
	c := &collectionOptions{}
	for _, o := range opts {
		o(c)
	}
	if c.stop == nil {
		c.stop = make(chan struct{})
	}
	return *c
}

func registerHandlerAsBatched[T any](c internalCollection[T], f func(o Event[T])) HandlerRegistration {
	return c.RegisterBatch(func(events []Event[T]) {
		for _, o := range events {
			f(o)
		}
	}, true)
}

func nextUID() collectionUID {
	return collectionUID(globalUIDCounter.Inc())
}

func getLabels(a any) map[string]string {
	al, ok := a.(Labeler)
	if ok {
		return al.GetLabels()
	}
	pal, ok := any(&a).(Labeler)
	if ok {
		return pal.GetLabels()
	}
	ak, ok := a.(metav1.Object)
	if ok {
		return ak.GetLabels()
	}
	ac, ok := a.(config.Config)
	if ok {
		return ac.Labels
	}
	panic(fmt.Sprintf("No Labels, got %T", a))
}

func castEvent[I, O any](o Event[I]) Event[O] {
	e := Event[O]{
		Event: o.Event,
	}
	if o.Old != nil {
		e.Old = ptr.Of(any(*o.Old).(O))
	}
	if o.New != nil {
		e.New = ptr.Of(any(*o.New).(O))
	}
	return e
}
