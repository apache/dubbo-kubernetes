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
	"github.com/apache/dubbo-kubernetes/pkg/xds/envoy"
)

type Context struct {
	ControlPlane *ControlPlaneContext
	Mesh         MeshContext
}

type ConnectionInfo struct {
	// Authority defines the URL that was used by the data plane to connect to the control plane
	Authority string
}

// MeshContext contains shared data within one mesh that is required for generating XDS config.
// This data is the same for all data plane proxies within one mesh.
// If there is an information that can be precomputed and shared between all data plane proxies
// it should be put here. This way we can save CPU cycles of computing the same information.
type MeshContext struct {
	Hash      string
	Resources Resources
}

type ControlPlaneContext struct {
	CLACache envoy.CLACache
	Zone     string
}

type AggregatedMeshContexts struct {
	Hash               string
	MeshContextsByName map[string]MeshContext
}

// GlobalContext holds resources that are Global
type GlobalContext struct {
	ResourceMap ResourceMap
	hash        []byte
}

// BaseMeshContext holds for a Mesh a set of resources that are changing less often (policies, external services...)
type BaseMeshContext struct {
	ResourceMap ResourceMap
	hash        []byte
}
