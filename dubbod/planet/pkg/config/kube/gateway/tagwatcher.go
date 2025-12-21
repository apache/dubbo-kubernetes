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

package gateway

import (
	"sync"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// TagWatcher is a simplified implementation for Dubbo
// It checks if a Gateway belongs to the current revision
type TagWatcher interface {
	Run(stopCh <-chan struct{})
	HasSynced() bool
	AddHandler(handler TagHandler)
	IsMine(metav1.ObjectMeta) bool
}

// TagHandler is a callback for when the tags revision change.
type TagHandler func(any)

type simpleTagWatcher struct {
	revision string
	handlers []TagHandler
	mu       sync.RWMutex
	synced   bool
}

// NewSimpleTagWatcher creates a simple tag watcher that always returns true for IsMine
// This is a simplified version compared to Istio's TagWatcher
func NewSimpleTagWatcher(revision string) TagWatcher {
	return &simpleTagWatcher{
		revision: revision,
		synced:   true,
	}
}

func (t *simpleTagWatcher) Run(stopCh <-chan struct{}) {
	// Simple implementation - no background processing needed
	<-stopCh
}

func (t *simpleTagWatcher) HasSynced() bool {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.synced
}

func (t *simpleTagWatcher) AddHandler(handler TagHandler) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.handlers = append(t.handlers, handler)
}

func (t *simpleTagWatcher) IsMine(obj metav1.ObjectMeta) bool {
	// Simplified: always return true for now
	// In a full implementation, this would check labels/annotations for revision matching
	return true
}
