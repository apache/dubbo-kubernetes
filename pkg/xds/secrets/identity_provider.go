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

package secrets

import (
	"context"
	"time"

	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"

	mesh_proto "github.com/apache/dubbo-kubernetes/api/mesh/v1alpha1"
	core_ca "github.com/apache/dubbo-kubernetes/pkg/core/ca"
	core_mesh "github.com/apache/dubbo-kubernetes/pkg/core/resources/apis/mesh"
	core_xds "github.com/apache/dubbo-kubernetes/pkg/core/xds"
)

type Identity struct {
	Mesh     string
	Name     string
	Services mesh_proto.MultiValueTagSet
}

type IdentityProvider interface {
	// Get returns PEM encoded cert + key, backend that was used to generate this pair and an error.
	Get(context.Context, Identity, *core_mesh.MeshResource) (*core_xds.IdentitySecret, string, error)
}

type identityCertProvider struct {
	caManagers     core_ca.Managers
	latencyMetrics *prometheus.SummaryVec
}

func NewIdentityProvider(caManagers core_ca.Managers) (IdentityProvider, error) {
	return &identityCertProvider{
		caManagers: caManagers,
	}, nil
}

func (s *identityCertProvider) Get(ctx context.Context, requestor Identity, mesh *core_mesh.MeshResource) (*core_xds.IdentitySecret, string, error) {
	backend := mesh.GetEnabledCertificateAuthorityBackend()
	if backend == nil {
		return nil, "", errors.Errorf("CA default backend in mesh %q has to be defined", mesh.GetMeta().GetName())
	}

	timeout := backend.GetDpCert().GetRequestTimeout()
	if timeout != nil {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, timeout.AsDuration())
		defer cancel()
	}

	caManager, exist := s.caManagers[backend.Type]
	if !exist {
		return nil, "", errors.Errorf("CA manager of type %s not exist", backend.Type)
	}

	var pair core_ca.KeyPair
	var err error
	func() {
		start := time.Now()
		defer func() {
			s.latencyMetrics.WithLabelValues(backend.GetName()).Observe(float64(time.Since(start).Milliseconds()))
		}()
		pair, err = caManager.GenerateDataplaneCert(ctx, mesh.GetMeta().GetName(), backend, requestor.Services)
	}()

	if err != nil {
		return nil, "", errors.Wrapf(err, "could not generate dataplane cert for mesh: %q backend: %q services: %q", mesh.GetMeta().GetName(), backend.Name, requestor.Services)
	}

	return &core_xds.IdentitySecret{
		PemCerts: [][]byte{pair.CertPEM},
		PemKey:   pair.KeyPEM,
	}, backend.Name, nil
}
