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

package manager

import (
	"context"
	"fmt"
	"sync"
	"time"
)

import (
	"github.com/patrickmn/go-cache"
)

import (
	"github.com/apache/dubbo-kubernetes/pkg/core/resources/model"
	"github.com/apache/dubbo-kubernetes/pkg/core/resources/store"
)

// Cached version of the ReadOnlyResourceManager designed to be used only for use cases of eventual consistency.
// This cache is NOT consistent across instances of the control plane.
//
// When retrieving elements from cache, they point to the same instances of the resources.
// We cannot do deep copies because it would consume lots of memory, therefore you need to be extra careful to NOT modify the resources.
type cachedManager struct {
	delegate ReadOnlyResourceManager
	cache    *cache.Cache

	mutexes  map[string]*sync.Mutex
	mapMutex sync.Mutex // guards "mutexes" field
}

var _ ReadOnlyResourceManager = &cachedManager{}

func NewCachedManager(delegate ReadOnlyResourceManager, expirationTime time.Duration) (ReadOnlyResourceManager, error) {
	return &cachedManager{
		delegate: delegate,
		cache:    cache.New(expirationTime, time.Duration(int64(float64(expirationTime)*0.9))),
		mutexes:  map[string]*sync.Mutex{},
	}, nil
}

func (c *cachedManager) Get(ctx context.Context, res model.Resource, fs ...store.GetOptionsFunc) error {
	opts := store.NewGetOptions(fs...)
	cacheKey := fmt.Sprintf("GET:%s:%s", res.Descriptor().Name, opts.HashCode())
	obj, found := c.cache.Get(cacheKey)
	if !found {
		// 可能存在缓存刚刚过期, 而这里并发的goroutine较多的情况
		// 我们应该只让其中一个填满缓存, 让其余的等待. 否则我们将重复昂贵的工作
		mutex := c.mutexFor(cacheKey)
		mutex.Lock()
		obj, found = c.cache.Get(cacheKey)
		if !found {
			// 当多个goroutine被一一解锁之后, 只有一个goroutine执行该分支, 其余当则从缓存中获取对象
			if err := c.delegate.Get(ctx, res, fs...); err != nil {
				mutex.Unlock()
				return err
			}
			c.cache.SetDefault(cacheKey, res)
		}
		mutex.Unlock()
		// 我们需要从映射中清楚互斥体, 否则我们会看到内存泄漏
		c.cleanMutexFor(cacheKey)
	}

	if found {
		cached := obj.(model.Resource)
		if err := res.SetSpec(cached.GetSpec()); err != nil {
			return err
		}
		res.SetMeta(cached.GetMeta())
	}
	return nil
}

func (c *cachedManager) List(ctx context.Context, list model.ResourceList, fs ...store.ListOptionsFunc) error {
	opts := store.NewListOptions(fs...)
	if !opts.IsCacheable() {
		return fmt.Errorf("filter functions are not allowed for cached store")
	}
	cacheKey := fmt.Sprintf("LIST:%s:%s", list.GetItemType(), opts.HashCode())
	obj, found := c.cache.Get(cacheKey)
	if !found {
		// There might be a situation when cache just expired and there are many concurrent goroutines here.
		// We should only let one fill the cache and let the rest of them wait for it. Otherwise we will be repeating expensive work.
		mutex := c.mutexFor(cacheKey)
		mutex.Lock()
		obj, found = c.cache.Get(cacheKey)
		if !found {
			// After many goroutines are unlocked one by one, only one should execute this branch, the rest should retrieve object from the cache
			if err := c.delegate.List(ctx, list, fs...); err != nil {
				mutex.Unlock()
				return err
			}
			c.cache.SetDefault(cacheKey, list.GetItems())
		}
		mutex.Unlock()
		c.cleanMutexFor(cacheKey) // We need to cleanup mutexes from the map, otherwise we can see the memory leak.
	}

	if found {
		resources := obj.([]model.Resource)
		for _, res := range resources {
			if err := list.AddItem(res); err != nil {
				return err
			}
		}
	}
	return nil
}

func (c *cachedManager) mutexFor(key string) *sync.Mutex {
	c.mapMutex.Lock()
	defer c.mapMutex.Unlock()
	mutex, exist := c.mutexes[key]
	if !exist {
		mutex = &sync.Mutex{}
		c.mutexes[key] = mutex
	}
	return mutex
}

func (c *cachedManager) cleanMutexFor(key string) {
	c.mapMutex.Lock()
	delete(c.mutexes, key)
	c.mapMutex.Unlock()
}
