// Licensed to the Apache Software Foundation (ASF) under one or more
// contributor license agreements.  See the NOTICE file distributed with
// this work for additional information regarding copyright ownership.
// The ASF licenses this file to You under the Apache License, Version 2.0
// (the "License"); you may not use this file except in compliance with
// the License.  You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package envoy

import (
	core_xds "github.com/apache/dubbo-kubernetes/pkg/core/xds"
	"github.com/apache/dubbo-kubernetes/pkg/xds/envoy/names"
	xds_tls "github.com/apache/dubbo-kubernetes/pkg/xds/envoy/tls"
)

type identityCertRequest struct {
	meshName string
}

func (r identityCertRequest) Name() string {
	return names.GetSecretName(xds_tls.IdentityCertResource, "secret", r.meshName)
}

type caRequest struct {
	meshName string
}

type allInOneCaRequest struct {
	meshNames []string
}

func (r caRequest) Name() string {
	return names.GetSecretName(xds_tls.MeshCaResource, "secret", r.meshName)
}

func (r caRequest) MeshName() []string {
	return []string{r.meshName}
}

func (r allInOneCaRequest) Name() string {
	return names.GetSecretName(xds_tls.MeshCaResource, "secret", "all")
}

func (r allInOneCaRequest) MeshName() []string {
	return r.meshNames
}

type secretsTracker struct {
	ownMesh   string
	allMeshes []string

	identity bool
	meshes   map[string]struct{}
	allInOne bool
}

func NewSecretsTracker(ownMesh string, allMeshes []string) core_xds.SecretsTracker {
	return &secretsTracker{
		ownMesh:   ownMesh,
		allMeshes: allMeshes,

		meshes: map[string]struct{}{},
	}
}

func (st *secretsTracker) RequestIdentityCert() core_xds.IdentityCertRequest {
	st.identity = true
	return &identityCertRequest{
		meshName: st.ownMesh,
	}
}

func (st *secretsTracker) RequestCa(mesh string) core_xds.CaRequest {
	st.meshes[mesh] = struct{}{}
	return &caRequest{
		meshName: mesh,
	}
}

func (st *secretsTracker) RequestAllInOneCa() core_xds.CaRequest {
	st.allInOne = true
	return &allInOneCaRequest{
		meshNames: st.allMeshes,
	}
}

func (st *secretsTracker) UsedIdentity() bool {
	return st.identity
}

func (st *secretsTracker) UsedCas() map[string]struct{} {
	return st.meshes
}

func (st *secretsTracker) UsedAllInOne() bool {
	return st.allInOne
}
