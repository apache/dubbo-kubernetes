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
	"hash/fnv"

	core_mesh "github.com/apache/dubbo-kubernetes/pkg/core/resources/apis/mesh"
	core_model "github.com/apache/dubbo-kubernetes/pkg/core/resources/model"
	"github.com/apache/dubbo-kubernetes/pkg/core/resources/registry"
	"github.com/apache/dubbo-kubernetes/pkg/util/maps"
)

type ResourceMap map[core_model.ResourceType]core_model.ResourceList

func (rm ResourceMap) listOrEmpty(resourceType core_model.ResourceType) core_model.ResourceList {
	list, ok := rm[resourceType]
	if !ok {
		list, err := registry.Global().NewList(resourceType)
		if err != nil {
			panic(err)
		}
		return list
	}
	return list
}

func (rm ResourceMap) Hash() []byte {
	hasher := fnv.New128a()
	for _, k := range maps.SortedKeys(rm) {
		hasher.Write(core_model.ResourceListHash(rm[k]))
	}
	return hasher.Sum(nil)
}

// Resources mulity mesh soon
type Resources struct {
	MeshLocalResources ResourceMap
}

func NewResources() Resources {
	return Resources{
		MeshLocalResources: map[core_model.ResourceType]core_model.ResourceList{},
	}
}

func (r Resources) ListOrEmpty(resourceType core_model.ResourceType) core_model.ResourceList {
	return r.MeshLocalResources.listOrEmpty(resourceType)
}

func (r Resources) ZoneIngresses() *core_mesh.ZoneIngressResourceList {
	return r.ListOrEmpty(core_mesh.ZoneIngressType).(*core_mesh.ZoneIngressResourceList)
}

func (r Resources) Dataplanes() *core_mesh.DataplaneResourceList {
	return r.ListOrEmpty(core_mesh.DataplaneType).(*core_mesh.DataplaneResourceList)
}

func (r Resources) OtherMeshes() *core_mesh.MeshResourceList {
	return r.ListOrEmpty(core_mesh.MeshType).(*core_mesh.MeshResourceList)
}

func (r Resources) Meshes() *core_mesh.MeshResourceList {
	return r.ListOrEmpty(core_mesh.MeshType).(*core_mesh.MeshResourceList)
}
