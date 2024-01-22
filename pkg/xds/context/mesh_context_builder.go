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
	"github.com/apache/dubbo-kubernetes/pkg/core/resources/manager"
	core_model "github.com/apache/dubbo-kubernetes/pkg/core/resources/model"
)

type meshContextBuilder struct {
	rm      manager.ReadOnlyResourceManager
	typeSet map[core_model.ResourceType]struct{}
	zone    string
}

type MeshContextBuilder interface {
	Build(ctx context.Context, meshName string) (MeshContext, error)

	// BuildGlobalContextIfChanged builds GlobalContext only if `latest` is nil or hash is different
	// If hash is the same, the return `latest`
	BuildGlobalContextIfChanged(ctx context.Context, latest *GlobalContext) (*GlobalContext, error)

	// BuildBaseMeshContextIfChanged builds BaseMeshContext only if `latest` is nil or hash is different
	// If hash is the same, the return `latest`
	BuildBaseMeshContextIfChanged(ctx context.Context, meshName string, latest *BaseMeshContext) (*BaseMeshContext, error)

	// BuildIfChanged builds MeshContext only if latestMeshCtx is nil or hash of
	// latestMeshCtx is different.
	// If hash is the same, then the function returns the passed latestMeshCtx.
	// Hash returned in MeshContext can never be empty.
	BuildIfChanged(ctx context.Context, meshName string, latestMeshCtx *MeshContext) (*MeshContext, error)
}
