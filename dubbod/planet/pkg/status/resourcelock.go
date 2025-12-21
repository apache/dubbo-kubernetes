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

package status

import (
	"context"
	"strconv"
	"sync"

	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/apache/dubbo-kubernetes/pkg/config"
	"github.com/apache/dubbo-kubernetes/pkg/util/sets"
)

type WorkerQueue interface {
	// Push a task.
	Push(target Resource, controller *Controller, context any)
	// Run the loop until a signal on the context
	Run(ctx context.Context)
	// Delete a task
	Delete(target Resource)
}

type cacheEntry struct {
	cacheResource       Resource
	perControllerStatus map[*Controller]any
}

type lockResource struct {
	schema.GroupVersionResource
	Namespace string
	Name      string
}

func convert(i Resource) lockResource {
	return lockResource{
		GroupVersionResource: i.GroupVersionResource,
		Namespace:            i.Namespace,
		Name:                 i.Name,
	}
}

type WorkQueue struct {
	tasks []lockResource
	lock  sync.Mutex
	cache map[lockResource]cacheEntry

	OnPush func()
}

func (wq *WorkQueue) Push(target Resource, ctl *Controller, progress any) {
	wq.lock.Lock()
	key := convert(target)
	if item, inqueue := wq.cache[key]; inqueue {
		item.perControllerStatus[ctl] = progress
		wq.cache[key] = item
	} else {
		wq.cache[key] = cacheEntry{
			cacheResource:       target,
			perControllerStatus: map[*Controller]any{ctl: progress},
		}
		wq.tasks = append(wq.tasks, key)
	}
	wq.lock.Unlock()
	if wq.OnPush != nil {
		wq.OnPush()
	}
}

func (wq *WorkQueue) Pop(exclusion sets.Set[lockResource]) (target Resource, progress map[*Controller]any) {
	wq.lock.Lock()
	defer wq.lock.Unlock()
	for i := 0; i < len(wq.tasks); i++ {
		if !exclusion.Contains(wq.tasks[i]) {
			t, ok := wq.cache[wq.tasks[i]]
			wq.tasks = append(wq.tasks[:i], wq.tasks[i+1:]...)
			if !ok {
				return Resource{}, nil
			}
			return t.cacheResource, t.perControllerStatus
		}
	}
	return Resource{}, nil
}

func (wq *WorkQueue) Length() int {
	wq.lock.Lock()
	defer wq.lock.Unlock()
	return len(wq.tasks)
}

func (wq *WorkQueue) Delete(target Resource) {
	wq.lock.Lock()
	defer wq.lock.Unlock()
	delete(wq.cache, convert(target))
}

type WorkerPool struct {
	q                WorkQueue
	closing          bool
	write            func(*config.Config)
	get              func(Resource) *config.Config
	workerCount      uint
	maxWorkers       uint
	currentlyWorking sets.Set[lockResource]
	lock             sync.Mutex
}

func NewWorkerPool(write func(*config.Config), get func(Resource) *config.Config, maxWorkers uint) WorkerQueue {
	return &WorkerPool{
		write:            write,
		get:              get,
		maxWorkers:       maxWorkers,
		currentlyWorking: sets.New[lockResource](),
		q: WorkQueue{
			tasks:  make([]lockResource, 0),
			cache:  make(map[lockResource]cacheEntry),
			OnPush: nil,
		},
	}
}

func (wp *WorkerPool) Delete(target Resource) {
	wp.q.Delete(target)
}

func (wp *WorkerPool) Push(target Resource, controller *Controller, context any) {
	wp.q.Push(target, controller, context)
	wp.maybeAddWorker()
}

func (wp *WorkerPool) Run(ctx context.Context) {
	context.AfterFunc(ctx, func() {
		wp.lock.Lock()
		wp.closing = true
		wp.lock.Unlock()
	})
}

func (wp *WorkerPool) maybeAddWorker() {
	wp.lock.Lock()
	if wp.workerCount >= wp.maxWorkers || wp.q.Length() == 0 {
		wp.lock.Unlock()
		return
	}
	wp.workerCount++
	wp.lock.Unlock()
	go func() {
		for {
			wp.lock.Lock()
			if wp.closing || wp.q.Length() == 0 {
				wp.workerCount--
				wp.lock.Unlock()
				return
			}

			target, perControllerWork := wp.q.Pop(wp.currentlyWorking)

			if target == (Resource{}) {
				wp.lock.Unlock()
				return
			}
			wp.q.Delete(target)
			wp.currentlyWorking.Insert(convert(target))
			wp.lock.Unlock()
			cfg := wp.get(target)
			if cfg != nil {
				if strconv.FormatInt(cfg.Generation, 10) == target.Generation {
					sm := GetStatusManipulator(cfg.Status)
					sm.SetObservedGeneration(cfg.Generation)
					for c, i := range perControllerWork {
						c.fn(sm, i)
					}
					cfg.Status = sm.Unwrap()
					wp.write(cfg)
				}
			}
			wp.lock.Lock()
			wp.currentlyWorking.Delete(convert(target))
			wp.lock.Unlock()
		}
	}()
}

type Manipulator interface {
	SetObservedGeneration(int64)
	SetInner(c any)
	Unwrap() any
}

type NopStatusManipulator struct {
	inner any
}

func (n *NopStatusManipulator) SetObservedGeneration(i int64) {
}

func (n *NopStatusManipulator) Unwrap() any {
	return n.inner
}

func (n *NopStatusManipulator) SetInner(c any) {
	n.inner = c
}
