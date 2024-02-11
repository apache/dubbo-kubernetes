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

package cache

import (
	core_model "github.com/apache/dubbo-kubernetes/pkg/core/resources/model"
	util_dds "github.com/apache/dubbo-kubernetes/pkg/dds/util"
	envoy_types "github.com/envoyproxy/go-control-plane/pkg/cache/types"

	envoy_cache "github.com/envoyproxy/go-control-plane/pkg/cache/v3"
)

type ResourceBuilder interface{}

type SnapshotBuilder interface {
	With(typ core_model.ResourceType, resources []envoy_types.Resource) SnapshotBuilder
	Build(version string) envoy_cache.ResourceSnapshot
}

type builder struct {
	resources map[core_model.ResourceType][]envoy_types.ResourceWithTTL
}

func (b *builder) With(typ core_model.ResourceType, resources []envoy_types.Resource) SnapshotBuilder {
	ttlResources := make([]envoy_types.ResourceWithTTL, len(resources))
	for i, res := range resources {
		ttlResources[i] = envoy_types.ResourceWithTTL{
			Resource: res,
			TTL:      nil,
		}
	}
	b.resources[typ] = ttlResources
	return b
}

func (b *builder) Build(version string) envoy_cache.ResourceSnapshot {
	snapshot := &Snapshot{Resources: map[core_model.ResourceType]envoy_cache.Resources{}}
	for _, typ := range util_dds.GetSupportedTypes() {
		snapshot.Resources[core_model.ResourceType(typ)] = envoy_cache.NewResources(version, nil)
	}
	for typ, items := range b.resources {
		snapshot.Resources[typ] = envoy_cache.Resources{Version: version, Items: IndexResourcesByName(items)}
	}
	return snapshot
}

func NewSnapshotBuilder() SnapshotBuilder {
	return &builder{
		resources: map[core_model.ResourceType][]envoy_types.ResourceWithTTL{},
	}
}
