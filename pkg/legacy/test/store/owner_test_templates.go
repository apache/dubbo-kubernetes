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

package store

import (
	"context"
	"fmt"
	"time"
)

import (
	. "github.com/onsi/ginkgo/v2"

	. "github.com/onsi/gomega"
)

import (
	mesh_proto "github.com/apache/dubbo-kubernetes/api/mesh/v1alpha1"
	core_mesh "github.com/apache/dubbo-kubernetes/pkg/core/resources/apis/mesh"
	"github.com/apache/dubbo-kubernetes/pkg/core/resources/model"
	"github.com/apache/dubbo-kubernetes/pkg/core/resources/store"
)

func ExecuteOwnerTests(
	createStore func() store.ResourceStore,
	storeName string,
) {
	const mesh = "default-mesh"
	var s store.ClosableResourceStore

	BeforeEach(func() {
		s = store.NewStrictResourceStore(createStore())
	})

	AfterEach(func() {
		err := s.Close()
		Expect(err).ToNot(HaveOccurred())
	})

	Context("Store: "+storeName, func() {
		It("should delete secret when its owner is deleted", func() {
			// setup
			meshRes := core_mesh.NewMeshResource()
			err := s.Create(context.Background(), meshRes, store.CreateByKey(mesh, model.NoMesh))
			Expect(err).ToNot(HaveOccurred())

			name := "secret-1"
			secretRes := core_mesh.NewDataplaneResource()
			err = s.Create(context.Background(), secretRes,
				store.CreateByKey(name, mesh),
				store.CreatedAt(time.Now()),
				store.CreateWithOwner(meshRes))
			Expect(err).ToNot(HaveOccurred())

			// when
			err = s.Delete(context.Background(), meshRes, store.DeleteByKey(mesh, model.NoMesh))
			Expect(err).ToNot(HaveOccurred())

			// then
			actual := core_mesh.NewDataplaneResource()
			err = s.Get(context.Background(), actual, store.GetByKey(name, mesh))
			Expect(store.IsResourceNotFound(err)).To(BeTrue())
		})

		It("should delete resource when its owner is deleted", func() {
			// setup
			meshRes := core_mesh.NewMeshResource()
			err := s.Create(context.Background(), meshRes, store.CreateByKey(mesh, model.NoMesh))
			Expect(err).ToNot(HaveOccurred())

			name := "resource-1"
			trRes := core_mesh.DataplaneResource{
				Spec: &mesh_proto.Dataplane{
					Networking: &mesh_proto.Dataplane_Networking{
						Address: "0.0.0.0",
					},
				},
			}
			err = s.Create(context.Background(), &trRes,
				store.CreateByKey(name, mesh),
				store.CreatedAt(time.Now()),
				store.CreateWithOwner(meshRes))
			Expect(err).ToNot(HaveOccurred())

			// when
			err = s.Delete(context.Background(), meshRes, store.DeleteByKey(mesh, model.NoMesh))
			Expect(err).ToNot(HaveOccurred())

			// then
			actual := core_mesh.NewDataplaneResource()
			err = s.Get(context.Background(), actual, store.GetByKey(name, mesh))
			Expect(store.IsResourceNotFound(err)).To(BeTrue())
		})

		It("should delete resource when its owner is deleted after owner update", func() {
			// setup
			meshRes := core_mesh.NewMeshResource()
			err := s.Create(context.Background(), meshRes, store.CreateByKey(mesh, model.NoMesh))
			Expect(err).ToNot(HaveOccurred())

			name := "resource-1"
			trRes := core_mesh.DataplaneResource{
				Spec: &mesh_proto.Dataplane{
					Networking: &mesh_proto.Dataplane_Networking{
						Address: "0.0.0.0",
					},
				},
			}
			err = s.Create(context.Background(), &trRes,
				store.CreateByKey(name, mesh),
				store.CreatedAt(time.Now()),
				store.CreateWithOwner(meshRes))
			Expect(err).ToNot(HaveOccurred())

			// when owner is updated
			Expect(s.Update(context.Background(), meshRes)).To(Succeed())

			// and only then deleted
			err = s.Delete(context.Background(), meshRes, store.DeleteByKey(mesh, model.NoMesh))
			Expect(err).ToNot(HaveOccurred())

			// then
			actual := core_mesh.NewDataplaneResource()
			err = s.Get(context.Background(), actual, store.GetByKey(name, mesh))
			Expect(store.IsResourceNotFound(err)).To(BeTrue())
		})

		It("should delete several resources when their owner is deleted", func() {
			// setup
			meshRes := core_mesh.NewMeshResource()
			err := s.Create(context.Background(), meshRes, store.CreateByKey(mesh, model.NoMesh))
			Expect(err).ToNot(HaveOccurred())

			for i := 0; i < 10; i++ {
				tr := core_mesh.DataplaneResource{
					Spec: &mesh_proto.Dataplane{
						Networking: &mesh_proto.Dataplane_Networking{
							Address: "0.0.0.0",
						},
					},
				}
				err = s.Create(context.Background(), &tr,
					store.CreateByKey(fmt.Sprintf("resource-%d", i), mesh),
					store.CreatedAt(time.Now()),
					store.CreateWithOwner(meshRes))
				Expect(err).ToNot(HaveOccurred())
			}
			actual := core_mesh.DataplaneResourceList{}
			err = s.List(context.Background(), &actual, store.ListByMesh(mesh))
			Expect(err).ToNot(HaveOccurred())
			Expect(actual.Items).To(HaveLen(10))

			// when
			err = s.Delete(context.Background(), meshRes, store.DeleteByKey(mesh, model.NoMesh))
			Expect(err).ToNot(HaveOccurred())

			// then
			actual = core_mesh.DataplaneResourceList{}
			err = s.List(context.Background(), &actual, store.ListByMesh(mesh))
			Expect(err).ToNot(HaveOccurred())
			Expect(actual.Items).To(BeEmpty())
		})

		It("should delete owners chain", func() {
			// setup
			meshRes := core_mesh.NewMeshResource()
			err := s.Create(context.Background(), meshRes, store.CreateByKey(mesh, model.NoMesh))
			Expect(err).ToNot(HaveOccurred())

			var prev model.Resource = meshRes
			for i := 0; i < 10; i++ {
				tr := &core_mesh.DataplaneResource{
					Spec: &mesh_proto.Dataplane{
						Networking: &mesh_proto.Dataplane_Networking{
							Address: "0.0.0.0",
						},
					},
				}
				err := s.Create(context.Background(), tr,
					store.CreateByKey(fmt.Sprintf("resource-%d", i), mesh),
					store.CreatedAt(time.Now()),
					store.CreateWithOwner(prev))
				Expect(err).ToNot(HaveOccurred())
				prev = tr
			}

			actual := core_mesh.DataplaneResourceList{}
			err = s.List(context.Background(), &actual, store.ListByMesh(mesh))
			Expect(err).ToNot(HaveOccurred())
			Expect(actual.Items).To(HaveLen(10))

			// when
			err = s.Delete(context.Background(), meshRes, store.DeleteByKey(mesh, model.NoMesh))
			Expect(err).ToNot(HaveOccurred())

			// then
			actual = core_mesh.DataplaneResourceList{}
			err = s.List(context.Background(), &actual, store.ListByMesh(mesh))
			Expect(err).ToNot(HaveOccurred())
			Expect(actual.Items).To(BeEmpty())
		})

		It("should delete a parent after children is deleted", func() {
			// given
			meshRes := core_mesh.NewMeshResource()
			err := s.Create(context.Background(), meshRes, store.CreateByKey(mesh, model.NoMesh))
			Expect(err).ToNot(HaveOccurred())

			name := "resource-1"
			tr := &core_mesh.DataplaneResource{
				Spec: &mesh_proto.Dataplane{
					Networking: &mesh_proto.Dataplane_Networking{
						Address: "0.0.0.0",
					},
				},
			}
			err = s.Create(context.Background(), tr,
				store.CreateByKey(name, mesh),
				store.CreatedAt(time.Now()),
				store.CreateWithOwner(meshRes))
			Expect(err).ToNot(HaveOccurred())

			// when children is deleted
			err = s.Delete(context.Background(), core_mesh.NewDataplaneResource(), store.DeleteByKey(name, mesh))

			// then
			Expect(err).ToNot(HaveOccurred())

			// when parent is deleted
			err = s.Delete(context.Background(), core_mesh.NewMeshResource(), store.DeleteByKey(mesh, model.NoMesh))

			// then
			Expect(err).ToNot(HaveOccurred())
		})
	})
}
