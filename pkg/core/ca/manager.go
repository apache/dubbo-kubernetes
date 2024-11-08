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

package ca

import mesh_proto "github.com/apache/dubbo-kubernetes/api/mesh/v1alpha1"

import (
	"context"

	"github.com/apache/dubbo-kubernetes/pkg/core/resources/model"
	"github.com/apache/dubbo-kubernetes/pkg/tls"
)

type Cert = []byte

type KeyPair = tls.KeyPair

// Manager manages CAs by creating CAs and generating certificate. It is created per CA type and then may be used for different CA instances of the same type
type Manager interface {
	// ValidateBackend validates that backend configuration is correct
	ValidateBackend(ctx context.Context, mesh string, backend *mesh_proto.CertificateAuthorityBackend) error
	// EnsureBackends ensures the given CA backends managed by this manager are available.
	// Since the secrets are now explicitly children of mesh we need to pass the whole mesh object so that we can properly set the owner.
	EnsureBackends(ctx context.Context, mesh model.Resource, backends []*mesh_proto.CertificateAuthorityBackend) error
	// UsedSecrets returns a list of secrets that are used by the manager
	UsedSecrets(mesh string, backend *mesh_proto.CertificateAuthorityBackend) ([]string, error)

	// GetRootCert returns root certificates of the CA
	GetRootCert(ctx context.Context, mesh string, backend *mesh_proto.CertificateAuthorityBackend) ([]Cert, error)
	// GenerateDataplaneCert generates cert for a dataplane with service tags
	GenerateDataplaneCert(ctx context.Context, mesh string, backend *mesh_proto.CertificateAuthorityBackend, tags mesh_proto.MultiValueTagSet) (KeyPair, error)
}

// Managers hold Manager instance for each type of backend available (by default: builtin, provided)
type Managers = map[string]Manager
