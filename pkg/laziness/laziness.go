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

// Package laziness is a package to expose lazily computed values.
package laziness

import (
	"sync"
	"sync/atomic"
)

type lazinessImpl[T any] struct {
	getter func() (T, error)
	// retry, if true, will ensure getter() is called for each Get() until a non-nil error is returned.
	retry bool
	// Cached responses. Note: with retry enabled, this will be unset until a non-nil error
	res  T
	err  error
	done uint32
	m    sync.Mutex
}

// Laziness represents a value whose computation is deferred until the first access.
type Laziness[T any] interface {
	Get() (T, error)
}

var _ Laziness[any] = &lazinessImpl[any]{}

func New[T any](f func() (T, error)) Laziness[T] {
	return &lazinessImpl[T]{
		getter: f,
	}
}

// NewWithRetry returns a new lazily computed value. The value will be computed on each call until a
// non-nil error is returned.
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
		// Defer in case of panic
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
