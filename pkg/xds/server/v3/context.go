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

package v3

import (
	envoy_core "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	envoy_cache "github.com/envoyproxy/go-control-plane/pkg/cache/v3"
	envoy_log "github.com/envoyproxy/go-control-plane/pkg/log"

	"github.com/go-logr/logr"
)

import (
	"github.com/apache/dubbo-kubernetes/pkg/core"
	"github.com/apache/dubbo-kubernetes/pkg/core/xds"
	util_xds "github.com/apache/dubbo-kubernetes/pkg/util/xds"
)

type XdsContext interface {
	Hasher() envoy_cache.NodeHash
	Cache() envoy_cache.SnapshotCache
}

func NewXdsContext() XdsContext {
	return newXdsContext("xds-server", true)
}

func newXdsContext(name string, ads bool) XdsContext {
	log := core.Log.WithName(name)
	hasher := hasher{log}
	logger := util_xds.NewLogger(log)
	cache := envoy_cache.NewSnapshotCache(ads, hasher, logger)
	return &xdsContext{
		NodeHash:      hasher,
		Logger:        logger,
		SnapshotCache: cache,
	}
}

type xdsContext struct {
	envoy_cache.NodeHash
	envoy_log.Logger
	envoy_cache.SnapshotCache
}

func (c *xdsContext) Hasher() envoy_cache.NodeHash {
	return c.NodeHash
}

func (c *xdsContext) Cache() envoy_cache.SnapshotCache {
	return c.SnapshotCache
}

type hasher struct {
	log logr.Logger
}

func (h hasher) ID(node *envoy_core.Node) string {
	if node == nil {
		return "unknown"
	}
	proxyId, err := xds.ParseProxyIdFromString(node.GetId())
	if err != nil {
		h.log.Error(err, "failed to parse Proxy ID", "node", node)
		return "unknown"
	}
	return proxyId.String()
}
