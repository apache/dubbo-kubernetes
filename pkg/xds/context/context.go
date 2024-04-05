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
	"encoding/base64"
)

import (
	core_mesh "github.com/apache/dubbo-kubernetes/pkg/core/resources/apis/mesh"
	"github.com/apache/dubbo-kubernetes/pkg/core/xds"
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

// ControlPlaneContext contains shared global data and components that are required for generating XDS
// This data is the same regardless of a data plane proxy and mesh we are generating the data for.
type ControlPlaneContext struct {
	CLACache envoy.CLACache
	Zone     string
}

// GlobalContext holds resources that are Global
type GlobalContext struct {
	ResourceMap ResourceMap
	hash        []byte
}

// Hash base64 version of the hash mostly used for testing
func (g GlobalContext) Hash() string {
	return base64.StdEncoding.EncodeToString(g.hash)
}

// BaseMeshContext holds for a Mesh a set of resources that are changing less often (policies, external services...)
type BaseMeshContext struct {
	Mesh        *core_mesh.MeshResource
	ResourceMap ResourceMap
	hash        []byte
}

// Hash base64 version of the hash mostly useed for testing
func (g BaseMeshContext) Hash() string {
	return base64.StdEncoding.EncodeToString(g.hash)
}

type MeshContext struct {
	Hash                string
	Resource            *core_mesh.MeshResource
	Resources           Resources
	DataplanesByName    map[string]*core_mesh.DataplaneResource
	EndpointMap         xds.EndpointMap
	ServicesInformation map[string]*ServiceInformation
}

type ServiceInformation struct {
	TLSReadiness      bool
	Protocol          core_mesh.Protocol
	IsExternalService bool
}

func (mc *MeshContext) GetServiceProtocol(serviceName string) core_mesh.Protocol {
	if info, found := mc.ServicesInformation[serviceName]; found {
		return info.Protocol
	}
	return core_mesh.ProtocolUnknown
}
