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

package builtin

import (
	"context"
	"fmt"

	"github.com/asaskevich/govalidator"
	"github.com/pkg/errors"

	mesh_proto "github.com/apache/dubbo-kubernetes/api/mesh/v1alpha1"
	system_proto "github.com/apache/dubbo-kubernetes/api/system/v1alpha1"
	"github.com/apache/dubbo-kubernetes/pkg/core"
	core_ca "github.com/apache/dubbo-kubernetes/pkg/core/ca"
	ca_issuer "github.com/apache/dubbo-kubernetes/pkg/core/ca/issuer"
	core_mesh "github.com/apache/dubbo-kubernetes/pkg/core/resources/apis/mesh"
	core_system "github.com/apache/dubbo-kubernetes/pkg/core/resources/apis/system"
	"github.com/apache/dubbo-kubernetes/pkg/core/resources/manager"
	core_model "github.com/apache/dubbo-kubernetes/pkg/core/resources/model"
	core_store "github.com/apache/dubbo-kubernetes/pkg/core/resources/store"
	core_validators "github.com/apache/dubbo-kubernetes/pkg/core/validators"
	"github.com/apache/dubbo-kubernetes/pkg/plugins/ca/builtin/config"
	util_proto "github.com/apache/dubbo-kubernetes/pkg/util/proto"
)

var log = core.Log.WithName("ca").WithName("builtin")

const MaxBackendNameLength = 255

type builtinCaManager struct {
	secretManager manager.ResourceManager
}

func NewBuiltinCaManager(secretManager manager.ResourceManager) core_ca.Manager {
	return &builtinCaManager{
		secretManager: secretManager,
	}
}

var _ core_ca.Manager = &builtinCaManager{}

// EnsureBackends ensures the CA secrets for the given mesh and backends.
func (b *builtinCaManager) EnsureBackends(ctx context.Context, mesh core_model.Resource, backends []*mesh_proto.CertificateAuthorityBackend) error {
	for _, backend := range backends {
		meshName := mesh.GetMeta().GetName()
		_, err := b.getCa(ctx, meshName, backend.Name)
		if err == nil {
			log.V(1).Info("CA secrets already exist. Nothing to create", "mesh", mesh, "backend", backend.Name)
			continue
		}

		if !core_store.IsResourceNotFound(err) {
			return err
		}

		if err := b.create(ctx, mesh, backend); err != nil {
			return errors.Wrapf(err, "failed to create CA for mesh %q and backend %q", mesh, backend.Name)
		}
		log.Info("CA created", "mesh", meshName)
	}
	return nil
}

// validate config,backend name
func (b *builtinCaManager) ValidateBackend(ctx context.Context, mesh string, backend *mesh_proto.CertificateAuthorityBackend) error {
	verr := core_validators.ValidationError{}
	cfg := &config.BuiltinCertificateAuthorityConfig{}
	if err := util_proto.ToTyped(backend.Conf, cfg); err != nil {
		verr.AddViolation("", "could not convert backend config: "+err.Error())
	}
	backendNameWithPrefix := createSecretName(mesh, backend.Name, "cert")
	if len(backendNameWithPrefix) > MaxBackendNameLength {
		verr.AddViolationAt(core_validators.RootedAt("mtls").Field("backends").Field("name"), fmt.Sprintf("Backend name is too long. Max length: %d", MaxBackendNameLength))
	}
	if !govalidator.IsDNSName(backendNameWithPrefix) || !govalidator.IsLowerCase(backendNameWithPrefix) {
		verr.AddViolationAt(core_validators.RootedAt("mtls").Field("backends").Field("name"), fmt.Sprintf("%q name must be valid dns name", backend.Name))
	}
	return verr.OrNil()
}

func (b *builtinCaManager) UsedSecrets(mesh string, backend *mesh_proto.CertificateAuthorityBackend) ([]string, error) {
	return []string{
		certSecretResKey(mesh, backend.Name).Name,
		keySecretResKey(mesh, backend.Name).Name,
	}, nil
}

func (b *builtinCaManager) create(ctx context.Context, mesh core_model.Resource, backend *mesh_proto.CertificateAuthorityBackend) error {
	meshName := mesh.GetMeta().GetName()
	cfg := &config.BuiltinCertificateAuthorityConfig{}
	if err := util_proto.ToTyped(backend.Conf, cfg); err != nil {
		return errors.Wrap(err, "could not convert backend config to BuiltinCertificateAuthorityConfig")
	}

	var opts []certOptsFn
	if cfg.GetCaCert().GetExpiration() != "" {
		duration, err := core_mesh.ParseDuration(cfg.GetCaCert().GetExpiration())
		if err != nil {
			return err
		}
		opts = append(opts, withExpirationTime(duration))
	}
	keyPair, err := newRootCa(meshName, int(cfg.GetCaCert().GetRSAbits().GetValue()), opts...)
	if err != nil {
		return errors.Wrapf(err, "failed to generate a Root CA cert for Mesh %q", meshName)
	}

	certSecret := &core_system.SecretResource{
		Spec: &system_proto.Secret{
			Data: util_proto.Bytes(keyPair.CertPEM),
		},
	}
	if err := b.secretManager.Create(ctx, certSecret, core_store.CreateWithOwner(mesh), core_store.CreateBy(certSecretResKey(meshName, backend.Name))); err != nil {
		if !errors.Is(err, &core_store.ResourceConflictError{}) {
			return err
		}
		log.V(1).Info("CA certificate already exists. Nothing to create", "mesh", meshName, "backend", backend.Name)
	}

	keySecret := &core_system.SecretResource{
		Spec: &system_proto.Secret{
			Data: util_proto.Bytes(keyPair.KeyPEM),
		},
	}
	if err := b.secretManager.Create(ctx, keySecret, core_store.CreateWithOwner(mesh), core_store.CreateBy(keySecretResKey(meshName, backend.Name))); err != nil {
		if !errors.Is(err, &core_store.ResourceConflictError{}) {
			return err
		}
		log.V(1).Info("CA secret key already exists. Nothing to create", "mesh", meshName, "backend", backend.Name)
	}
	return nil
}

func certSecretResKey(mesh string, backendName string) core_model.ResourceKey {
	return core_model.ResourceKey{
		Mesh: mesh,
		Name: createSecretName(mesh, backendName, "cert"),
	}
}

func keySecretResKey(mesh string, backendName string) core_model.ResourceKey {
	return core_model.ResourceKey{
		Mesh: mesh,
		Name: createSecretName(mesh, backendName, "key"),
	}
}

func createSecretName(mesh string, backendName string, secretType string) string {
	return fmt.Sprintf("%s.ca-builtin-%s-%s", mesh, secretType, backendName) // we add mesh as a prefix to have uniqueness of Secret names on K8S
}

func (b *builtinCaManager) GetRootCert(ctx context.Context, mesh string, backend *mesh_proto.CertificateAuthorityBackend) ([]core_ca.Cert, error) {
	ca, err := b.getCa(ctx, mesh, backend.Name)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to load CA key pair for Mesh %q and backend %q", mesh, backend.Name)
	}
	return []core_ca.Cert{ca.CertPEM}, nil
}

func (b *builtinCaManager) GenerateDataplaneCert(ctx context.Context, mesh string, backend *mesh_proto.CertificateAuthorityBackend, tags mesh_proto.MultiValueTagSet) (core_ca.KeyPair, error) {
	ca, err := b.getCa(ctx, mesh, backend.Name)
	if err != nil {
		return core_ca.KeyPair{}, errors.Wrapf(err, "failed to load CA key pair for Mesh %q and backend %q", mesh, backend.Name)
	}

	var opts []ca_issuer.CertOptsFn
	if backend.GetDpCert().GetRotation().GetExpiration() != "" {
		duration, err := core_mesh.ParseDuration(backend.GetDpCert().GetRotation().Expiration)
		if err != nil {
			return core_ca.KeyPair{}, err
		}
		opts = append(opts, ca_issuer.WithExpirationTime(duration))
	}
	keyPair, err := ca_issuer.NewWorkloadCert(ca, mesh, tags, opts...)
	if err != nil {
		return core_ca.KeyPair{}, errors.Wrapf(err, "failed to generate a Workload Identity cert for tags %q in Mesh %q using backend %q", tags.String(), mesh, backend)
	}
	return *keyPair, nil
}

func (b *builtinCaManager) getCa(ctx context.Context, mesh string, backendName string) (core_ca.KeyPair, error) {
	certSecret := core_system.NewSecretResource()
	if err := b.secretManager.Get(ctx, certSecret, core_store.GetBy(certSecretResKey(mesh, backendName))); err != nil {
		return core_ca.KeyPair{}, err
	}

	keySecret := core_system.NewSecretResource()
	if err := b.secretManager.Get(ctx, keySecret, core_store.GetBy(keySecretResKey(mesh, backendName))); err != nil {
		return core_ca.KeyPair{}, err
	}

	return core_ca.KeyPair{
		CertPEM: certSecret.Spec.Data.Value,
		KeyPEM:  keySecret.Spec.Data.Value,
	}, nil
}
