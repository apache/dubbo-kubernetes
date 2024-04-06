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

package reconcile

import (
	"context"
)

import (
	envoy_core "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	envoy_cache "github.com/envoyproxy/go-control-plane/pkg/cache/v3"

	"github.com/go-logr/logr"
)

import (
	"github.com/apache/dubbo-kubernetes/pkg/core/resources/model"
	cache_dds "github.com/apache/dubbo-kubernetes/pkg/dds/cache"
)

// Reconciler re-computes configuration for a given node.
type Reconciler interface {
	// Reconcile reconciles state of node given changed resource types.
	// Returns error and bool which is true if any resource was changed.
	Reconcile(context.Context, *envoy_core.Node, map[model.ResourceType]struct{}, logr.Logger) (error, bool)
	Clear(context.Context, *envoy_core.Node) error
}

// Generates a snapshot of xDS resources for a given node.
type SnapshotGenerator interface {
	GenerateSnapshot(context.Context, *envoy_core.Node, cache_dds.SnapshotBuilder, map[model.ResourceType]struct{}) (envoy_cache.ResourceSnapshot, error)
}
