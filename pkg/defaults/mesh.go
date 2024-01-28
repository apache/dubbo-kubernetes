package defaults

import (
	"context"
	core_mesh "github.com/apache/dubbo-kubernetes/pkg/core/resources/apis/mesh"
	core_manager "github.com/apache/dubbo-kubernetes/pkg/core/resources/manager"
	core_model "github.com/apache/dubbo-kubernetes/pkg/core/resources/model"
	core_store "github.com/apache/dubbo-kubernetes/pkg/core/resources/store"
)

var defaultMeshKey = core_model.ResourceKey{
	Name: core_model.DefaultMesh,
}

func CreateMeshIfNotExist(
	ctx context.Context,
	resManager core_manager.ResourceManager,
	extensions context.Context,
) (*core_mesh.MeshResource, error) {
	mesh := core_mesh.NewMeshResource()
	err := resManager.Get(ctx, mesh, core_store.GetBy(defaultMeshKey))
	if err == nil {
		log.Info("default Mesh already exists. Skip creating default Mesh.")
		return mesh, nil
	}
	if !core_store.IsResourceNotFound(err) {
		return nil, err
	}
	if err := resManager.Create(ctx, mesh, core_store.CreateBy(defaultMeshKey)); err != nil {
		log.Info("could not create default mesh", "err", err)
		return nil, err
	}
	return mesh, nil
}
