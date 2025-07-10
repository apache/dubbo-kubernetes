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

package defaults_test

import (
	"context"
)

import (
	. "github.com/onsi/ginkgo/v2"

	. "github.com/onsi/gomega"
)

import (
	"github.com/apache/dubbo-kubernetes/api/mesh/v1alpha1"
	dubbo_cp "github.com/apache/dubbo-kubernetes/pkg/config/app/dubbo-cp"
	core_mesh "github.com/apache/dubbo-kubernetes/pkg/core/resources/apis/mesh"
	core_manager "github.com/apache/dubbo-kubernetes/pkg/core/resources/manager"
	core_model "github.com/apache/dubbo-kubernetes/pkg/core/resources/model"
	core_store "github.com/apache/dubbo-kubernetes/pkg/core/resources/store"
	core_component "github.com/apache/dubbo-kubernetes/pkg/core/runtime/component"
	"github.com/apache/dubbo-kubernetes/pkg/defaults"
	resources_memory "github.com/apache/dubbo-kubernetes/pkg/plugins/resources/memory"
)

var _ = Describe("Defaults Component", func() {
	Describe("when skip mesh creation is set to false", func() {
		var component core_component.Component
		var manager core_manager.ResourceManager

		BeforeEach(func() {
			cfg := &dubbo_cp.Defaults{
				SkipMeshCreation: false,
			}
			store := resources_memory.NewStore()
			manager = core_manager.NewResourceManager(store)
			component = defaults.NewDefaultsComponent(cfg, manager, context.Background())
		})

		It("should create default mesh", func() {
			// when
			err := component.Start(nil)

			// then
			Expect(err).ToNot(HaveOccurred())
			err = manager.Get(context.Background(), core_mesh.NewMeshResource(), core_store.GetByKey(core_model.DefaultMesh, core_model.NoMesh))
			Expect(err).ToNot(HaveOccurred())
		})

		It("should not override already created mesh", func() {
			// given
			mesh := &core_mesh.MeshResource{
				Spec: &v1alpha1.Mesh{
					Mtls: &v1alpha1.Mesh_Mtls{
						EnabledBackend: "builtin",
						Backends: []*v1alpha1.CertificateAuthorityBackend{
							{
								Name: "builtin",
								Type: "builtin",
							},
						},
					},
				},
			}
			err := manager.Create(context.Background(), mesh, core_store.CreateByKey(core_model.DefaultMesh, core_model.NoMesh))
			Expect(err).ToNot(HaveOccurred())

			// when
			err = component.Start(nil)

			// then
			mesh = core_mesh.NewMeshResource()
			Expect(err).ToNot(HaveOccurred())
			err = manager.Get(context.Background(), mesh, core_store.GetByKey(core_model.DefaultMesh, core_model.NoMesh))
			Expect(err).ToNot(HaveOccurred())
			Expect(mesh.Spec.Mtls.EnabledBackend).To(Equal("builtin"))
		})
	})

	Describe("when skip mesh creation is set to true", func() {
		var component core_component.Component
		var manager core_manager.ResourceManager

		BeforeEach(func() {
			cfg := &dubbo_cp.Defaults{
				SkipMeshCreation: true,
			}
			store := resources_memory.NewStore()
			manager = core_manager.NewResourceManager(store)
			component = defaults.NewDefaultsComponent(cfg, manager, context.Background())
		})

		It("should not create default mesh", func() {
			// when
			err := component.Start(nil)

			// then
			Expect(err).ToNot(HaveOccurred())
			err = manager.Get(context.Background(), core_mesh.NewMeshResource(), core_store.GetByKey("default", "default"))
			Expect(core_store.IsResourceNotFound(err)).To(BeTrue())
		})
	})
})
