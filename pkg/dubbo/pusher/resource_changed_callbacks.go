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

package pusher

import (
	"sync"
)

import (
	core_model "github.com/apache/dubbo-kubernetes/pkg/core/resources/model"
)

type (
	ResourceChangedCallbackFn  func(items PushedItems)
	ResourceChangedEventFilter func(resourceList core_model.ResourceList) core_model.ResourceList
	ResourceRequestFilter      func(request interface{}, resourceList core_model.ResourceList) core_model.ResourceList
)

type ResourceChangedCallback struct {
	mu       sync.Mutex // Only one can run at a time
	Callback ResourceChangedCallbackFn
	Filters  []ResourceChangedEventFilter
}

func (c *ResourceChangedCallback) Invoke(items PushedItems) {
	c.mu.Lock()
	defer c.mu.Unlock()

	pushed := items.resourceList
	for _, filter := range c.Filters {
		pushed = filter(items.resourceList)
	}
	if len(pushed.GetItems()) == 0 {
		return
	}

	callback := c.Callback
	callback(items)
}

type ResourceChangedCallbacks struct {
	// mu to protect resourceChangedCallbacks
	mu          sync.RWMutex
	callbackMap map[core_model.ResourceType]map[string]*ResourceChangedCallback
}

func NewResourceChangedCallbacks() *ResourceChangedCallbacks {
	return &ResourceChangedCallbacks{
		callbackMap: make(map[core_model.ResourceType]map[string]*ResourceChangedCallback),
	}
}

func (callbacks *ResourceChangedCallbacks) InvokeCallbacks(resourceType core_model.ResourceType, items PushedItems) {
	callbacks.mu.RLock()
	defer callbacks.mu.RUnlock()

	var wg sync.WaitGroup
	for _, c := range callbacks.callbackMap[resourceType] {
		tmpCallback := c
		wg.Add(1)
		go func() {
			tmpCallback.Invoke(items)
			wg.Done()
		}()
	}

	wg.Wait()
}

func (callbacks *ResourceChangedCallbacks) AddCallBack(
	resourceType core_model.ResourceType,
	id string,
	callback ResourceChangedCallbackFn,
	filters ...ResourceChangedEventFilter,
) {
	callbacks.mu.Lock()
	defer callbacks.mu.Unlock()

	if _, ok := callbacks.callbackMap[resourceType]; !ok {
		callbacks.callbackMap[resourceType] = make(map[string]*ResourceChangedCallback)
	}

	callbacks.callbackMap[resourceType][id] = &ResourceChangedCallback{Callback: callback, Filters: filters}
}

func (callbacks *ResourceChangedCallbacks) RemoveCallBack(resourceType core_model.ResourceType, id string) {
	callbacks.mu.Lock()
	defer callbacks.mu.Unlock()

	if _, ok := callbacks.callbackMap[resourceType]; !ok {
		return
	}

	delete(callbacks.callbackMap[resourceType], id)
}

func (callbacks *ResourceChangedCallbacks) GetCallBack(resourceType core_model.ResourceType, id string) (*ResourceChangedCallback, bool) {
	callbacks.mu.RLock()
	defer callbacks.mu.RUnlock()

	if _, ok := callbacks.callbackMap[resourceType]; !ok {
		return nil, false
	}

	cb, ok := callbacks.callbackMap[resourceType][id]
	return cb, ok
}
