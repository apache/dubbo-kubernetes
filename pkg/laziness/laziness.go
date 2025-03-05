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

package laziness

import (
	"sync"
	"sync/atomic"
)

type lazinessImpl[T any] struct {
	getter func() (T, error)
	retry  bool
	res    T
	err    error
	done   uint32
	m      sync.Mutex
}

type Laziness[T any] interface {
	Get() (T, error)
}

var _ Laziness[any] = &lazinessImpl[any]{}

func New[T any](f func() (T, error)) Laziness[T] {
	return &lazinessImpl[T]{
		getter: f,
	}
}

func NewWithRetry[T any](f func() (T, error)) Laziness[T] {
	return &lazinessImpl[T]{
		getter: f,
		retry:  true,
	}
}

func (l *lazinessImpl[T]) Get() (T, error) {
	if atomic.LoadUint32(&l.done) == 0 {
		return l.doSlow()
	}
	return l.res, l.err
}

func (l *lazinessImpl[T]) doSlow() (T, error) {
	l.m.Lock()
	defer l.m.Unlock()
	if l.done == 0 {
		done := uint32(1)
		defer func() {
			atomic.StoreUint32(&l.done, done)
		}()
		res, err := l.getter()
		if l.retry && err != nil {
			done = 0
		} else {
			l.res, l.err = res, err
		}
		return res, err
	}
	return l.res, l.err
}
