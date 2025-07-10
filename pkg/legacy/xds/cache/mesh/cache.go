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

package mesh

import (
	"context"
	"time"
)

import (
	"github.com/patrickmn/go-cache"
)

import (
	"github.com/apache/dubbo-kubernetes/pkg/xds/cache/once"
	xds_context "github.com/apache/dubbo-kubernetes/pkg/xds/context"
)

// Cache is needed to share and cache Hashes among goroutines which
// reconcile Dataplane's state. Calculating hash is a heavy operation
// that requires fetching all the resources belonging to the Mesh.
type Cache struct {
	// cache is used for caching a context and ignoring mesh changes for up to a
	// short expiration time.
	cache *once.Cache
	// hashCache keeps a cached context, for a much longer time, that is only reused
	// when the mesh hasn't changed.
	hashCache *cache.Cache

	meshContextBuilder xds_context.MeshContextBuilder
}

// cleanupTime is the time after which the mesh context is removed from
// the longer TTL cache.
// It exists to ensure contexts of deleted Meshes are eventually cleaned up.
const cleanupTime = time.Minute

func NewCache(
	expirationTime time.Duration,
	meshContextBuilder xds_context.MeshContextBuilder,
) (*Cache, error) {
	c, err := once.New(expirationTime, "mesh_cache")
	if err != nil {
		return nil, err
	}
	return &Cache{
		cache:              c,
		meshContextBuilder: meshContextBuilder,
		hashCache:          cache.New(cleanupTime, time.Duration(int64(float64(cleanupTime)*0.9))),
	}, nil
}

func (c *Cache) GetMeshContext(ctx context.Context, mesh string) (xds_context.MeshContext, error) {
	// Check our short TTL cache for a context, ignoring whether there have been
	// changes since it was generated.
	elt, err := c.cache.GetOrRetrieve(ctx, mesh, once.RetrieverFunc(func(ctx context.Context, key string) (interface{}, error) {
		// Check hashCache first for an existing mesh latestContext
		var latestContext *xds_context.MeshContext
		if cached, ok := c.hashCache.Get(mesh); ok {
			latestContext = cached.(*xds_context.MeshContext)
		}

		// Rebuild the context only if the hash has changed
		var err error
		latestContext, err = c.meshContextBuilder.BuildIfChanged(ctx, mesh, latestContext)
		if err != nil {
			return xds_context.MeshContext{}, err
		}

		// By always setting the mesh context, we refresh the TTL
		// with the effect that often used contexts remain in the cache while no
		// longer used contexts are evicted.
		c.hashCache.SetDefault(mesh, latestContext)
		return *latestContext, nil
	}))
	if err != nil {
		return xds_context.MeshContext{}, err
	}
	return elt.(xds_context.MeshContext), nil
}
