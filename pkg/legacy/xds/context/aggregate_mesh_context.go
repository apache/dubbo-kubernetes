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

package context

import (
	"context"
)

import (
	core_mesh "github.com/apache/dubbo-kubernetes/pkg/core/resources/apis/mesh"
	"github.com/apache/dubbo-kubernetes/pkg/core/resources/manager"
	core_store "github.com/apache/dubbo-kubernetes/pkg/core/resources/store"
	"github.com/apache/dubbo-kubernetes/pkg/xds/cache/sha256"
)

type meshContextFetcher = func(ctx context.Context, meshName string) (MeshContext, error)

func AggregateMeshContexts(
	ctx context.Context,
	resManager manager.ReadOnlyResourceManager,
	fetcher meshContextFetcher,
) (AggregatedMeshContexts, error) {
	var meshList core_mesh.MeshResourceList
	if err := resManager.List(ctx, &meshList, core_store.ListOrdered()); err != nil {
		return AggregatedMeshContexts{}, err
	}

	var meshContexts []MeshContext
	meshContextsByName := map[string]MeshContext{}
	for _, mesh := range meshList.Items {
		meshCtx, err := fetcher(ctx, mesh.GetMeta().GetName())
		if err != nil {
			if core_store.IsResourceNotFound(err) {
				// When the mesh no longer exists it's likely because it was removed since, let's just skip it.
				continue
			}
			return AggregatedMeshContexts{}, err
		}
		meshContexts = append(meshContexts, meshCtx)
		meshContextsByName[mesh.Meta.GetName()] = meshCtx
	}

	hash := aggregatedHash(meshContexts)

	result := AggregatedMeshContexts{
		Hash:               hash,
		Meshes:             meshList.Items,
		MeshContextsByName: meshContextsByName,
	}
	return result, nil
}

func aggregatedHash(meshContexts []MeshContext) string {
	var hash string
	for _, meshCtx := range meshContexts {
		hash += meshCtx.Hash
	}
	return sha256.Hash(hash)
}
