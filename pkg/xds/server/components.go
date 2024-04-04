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

package server

import (
	util_xds "github.com/apache/dubbo-kubernetes/pkg/util/xds"
	"github.com/pkg/errors"
)

import (
	core_mesh "github.com/apache/dubbo-kubernetes/pkg/core/resources/apis/mesh"
	core_model "github.com/apache/dubbo-kubernetes/pkg/core/resources/model"
	"github.com/apache/dubbo-kubernetes/pkg/core/resources/registry"
	core_runtime "github.com/apache/dubbo-kubernetes/pkg/core/runtime"
	"github.com/apache/dubbo-kubernetes/pkg/xds/cache/cla"
	xds_context "github.com/apache/dubbo-kubernetes/pkg/xds/context"
	v3 "github.com/apache/dubbo-kubernetes/pkg/xds/server/v3"
)

var (
	// HashMeshExcludedResources defines Mesh-scoped resources that are not used in XDS therefore when counting hash mesh we can skip them
	HashMeshExcludedResources = map[core_model.ResourceType]bool{
		core_mesh.DataplaneInsightType: true,
	}
	HashMeshIncludedGlobalResources = map[core_model.ResourceType]bool{
		core_mesh.ZoneIngressType: true,
	}
)

func RegisterXDS(rt core_runtime.Runtime) error {
	statsCallbacks, err := util_xds.NewStatsCallbacks(nil, "xds")
	claCache, err := cla.NewCache(rt.Config().Store.Cache.ExpirationTime.Duration)
	if err != nil {
		return err
	}

	envoyCpCtx := &xds_context.ControlPlaneContext{
		CLACache: claCache,
		Zone:     "",
	}
	if err := v3.RegisterXDS(statsCallbacks, envoyCpCtx, rt); err != nil {
		return errors.Wrap(err, "could not register V3 XDS")
	}
	return nil
}

func MeshResourceTypes() []core_model.ResourceType {
	types := []core_model.ResourceType{}
	for _, desc := range registry.Global().ObjectDescriptors() {
		if desc.Scope == core_model.ScopeMesh && !HashMeshExcludedResources[desc.Name] {
			types = append(types, desc.Name)
		}
	}
	for typ := range HashMeshIncludedGlobalResources {
		types = append(types, typ)
	}
	return types
}
