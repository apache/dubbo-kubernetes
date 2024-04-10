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
