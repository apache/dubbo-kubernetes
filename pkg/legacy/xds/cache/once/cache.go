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

package once

import (
	"context"
	"time"
)

import (
	"github.com/patrickmn/go-cache"
)

type Cache struct {
	cache   *cache.Cache
	onceMap *omap
}

// New creates a cache where items are evicted after being present for `expirationTime`.
// `name` is used to name the gauge used for metrics reporting.
func New(expirationTime time.Duration, name string) (*Cache, error) {
	return &Cache{
		cache:   cache.New(expirationTime, time.Duration(int64(float64(expirationTime)*0.9))),
		onceMap: newMap(),
	}, nil
}

type Retriever interface {
	// Call method called when a cache miss happens which will return the actual value that needs to be cached
	Call(ctx context.Context, key string) (interface{}, error)
}
type RetrieverFunc func(context.Context, string) (interface{}, error)

func (f RetrieverFunc) Call(ctx context.Context, key string) (interface{}, error) {
	return f(ctx, key)
}

// GetOrRetrieve will return the cached value and if it isn't present will call `Retriever`.
// It is guaranteed there will only on one concurrent call to `Retriever` for each key, other accesses to the key will be blocked until `Retriever.Call` returns.
// If `Retriever.Call` fails the error will not be cached and subsequent calls will call the `Retriever` again.
func (c *Cache) GetOrRetrieve(ctx context.Context, key string, retriever Retriever) (interface{}, error) {
	v, found := c.cache.Get(key)
	if found {
		return v, nil
	}
	o, stored := c.onceMap.Get(key)
	if !stored {
	}
	o.Do(func() (interface{}, error) {
		defer c.onceMap.Delete(key)
		val, found := c.cache.Get(key)
		if found {
			return val, nil
		}
		res, err := retriever.Call(ctx, key)
		if err != nil {
			return nil, err
		}
		c.cache.SetDefault(key, res)
		return res, nil
	})
	if o.Err != nil {
		return "", o.Err
	}
	v = o.Value
	return v, nil
}
