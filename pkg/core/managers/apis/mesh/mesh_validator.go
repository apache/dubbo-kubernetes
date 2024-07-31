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

package mesh

import (
	"context"
	core_ca "github.com/apache/dubbo-kubernetes/pkg/core/ca"
	core_store "github.com/apache/dubbo-kubernetes/pkg/core/resources/store"
	"github.com/apache/dubbo-kubernetes/pkg/core/validators"
	"github.com/pkg/errors"
)

import (
	core_mesh "github.com/apache/dubbo-kubernetes/pkg/core/resources/apis/mesh"
)

type MeshValidator interface {
	ValidateCreate(ctx context.Context, name string, resource *core_mesh.MeshResource) error
	ValidateUpdate(ctx context.Context, previousMesh *core_mesh.MeshResource, newMesh *core_mesh.MeshResource) error
	ValidateDelete(ctx context.Context, name string) error
}

type meshValidator struct {
	CaManagers core_ca.Managers
	Store      core_store.ResourceStore
}

func NewMeshValidator(caManagers core_ca.Managers, store core_store.ResourceStore) MeshValidator {
	return &meshValidator{
		CaManagers: caManagers,
		Store:      store,
	}
}

func (m *meshValidator) ValidateCreate(ctx context.Context, name string, resource *core_mesh.MeshResource) error {
	var verr validators.ValidationError
	if len(name) > 63 {
		verr.AddViolation("name", "cannot be longer than 63 characters")
	}
	verr.Add(ValidateMTLSBackends(ctx, m.CaManagers, name, resource))
	return verr.OrNil()
}

func (m *meshValidator) ValidateDelete(ctx context.Context, name string) error {
	if err := ValidateNoActiveDP(ctx, name, m.Store); err != nil {
		return err
	}
	return nil
}

func (m *meshValidator) ValidateUpdate(ctx context.Context, previousMesh *core_mesh.MeshResource, newMesh *core_mesh.MeshResource) error {
	var verr validators.ValidationError
	verr.Add(m.validateMTLSBackendChange(previousMesh, newMesh))
	verr.Add(ValidateMTLSBackends(ctx, m.CaManagers, newMesh.Meta.GetName(), newMesh))
	return verr.OrNil()
}

func ValidateNoActiveDP(ctx context.Context, name string, store core_store.ResourceStore) error {
	dps := core_mesh.DataplaneResourceList{}
	validationErr := &validators.ValidationError{}
	if err := store.List(ctx, &dps, core_store.ListByMesh(name)); err != nil {
		return errors.Wrap(err, "unable to list Dataplanes")
	}
	if len(dps.Items) != 0 {
		validationErr.AddViolation("mesh", "unable to delete mesh, there are still some dataplanes attached")
		return validationErr
	}
	return nil
}

func (m *meshValidator) validateMTLSBackendChange(previousMesh *core_mesh.MeshResource, newMesh *core_mesh.MeshResource) validators.ValidationError {
	verr := validators.ValidationError{}
	if previousMesh.MTLSEnabled() && newMesh.MTLSEnabled() && previousMesh.Spec.GetMtls().GetEnabledBackend() != newMesh.Spec.GetMtls().GetEnabledBackend() {
		verr.AddViolation("mtls.enabledBackend", "Changing CA when mTLS is enabled is forbidden. Disable mTLS first and then change the CA")
	}
	return verr
}
