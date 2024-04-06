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

package defaults

import (
	"context"
)

import (
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
