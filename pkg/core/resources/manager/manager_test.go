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

package manager_test

import (
	"context"
)

import (
	. "github.com/onsi/ginkgo/v2"

	. "github.com/onsi/gomega"
)

import (
	mesh_proto "github.com/apache/dubbo-kubernetes/api/mesh/v1alpha1"
	core_mesh "github.com/apache/dubbo-kubernetes/pkg/core/resources/apis/mesh"
	"github.com/apache/dubbo-kubernetes/pkg/core/resources/manager"
	"github.com/apache/dubbo-kubernetes/pkg/core/resources/model"
	"github.com/apache/dubbo-kubernetes/pkg/core/resources/store"
	"github.com/apache/dubbo-kubernetes/pkg/plugins/resources/memory"
)

var _ = Describe("Resource Manager", func() {
	var resStore store.ResourceStore
	var resManager manager.ResourceManager

	BeforeEach(func() {
		resStore = memory.NewStore()
		resManager = manager.NewResourceManager(resStore)
	})

	createSampleMesh := func(name string) error {
		meshRes := core_mesh.MeshResource{
			Spec: &mesh_proto.Mesh{},
		}
		return resManager.Create(context.Background(), &meshRes, store.CreateByKey(name, model.NoMesh))
	}

	createSampleResource := func(mesh string) (*core_mesh.DataplaneResource, error) {
		trRes := core_mesh.DataplaneResource{
			Spec: &mesh_proto.Dataplane{
				Networking: &mesh_proto.Dataplane_Networking{
					Address: "10.10.10.10",
					Inbound: []*mesh_proto.Dataplane_Networking_Inbound{
						{
							Port: 8080,
							Tags: map[string]string{
								"dubbo.io/service": "true",
							},
						},
					},
				},
			},
		}
		err := resManager.Create(context.Background(), &trRes, store.CreateByKey("tr-1", mesh))
		return &trRes, err
	}

	Describe("Create()", func() {
		It("should let create when mesh exists", func() {
			// given
			err := createSampleMesh("mesh-1")
			Expect(err).ToNot(HaveOccurred())

			// when
			_, err = createSampleResource("mesh-1")

			// then
			Expect(err).ToNot(HaveOccurred())
		})

		It("should not let to create a resource when mesh not exists", func() {
			// given no mesh for resource

			// when
			_, err := createSampleResource("mesh-1")

			// then
			Expect(err.Error()).To(Equal("mesh of name mesh-1 is not found"))
		})
	})

	Describe("DeleteAll()", func() {
		It("should delete all resources within a mesh", func() {
			// setup
			Expect(createSampleMesh("mesh-1")).To(Succeed())
			Expect(createSampleMesh("mesh-2")).To(Succeed())
			_, err := createSampleResource("mesh-1")
			Expect(err).ToNot(HaveOccurred())
			_, err = createSampleResource("mesh-2")
			Expect(err).ToNot(HaveOccurred())

			tlKey := model.ResourceKey{
				Mesh: "mesh-1",
				Name: "tl-1",
			}
			zoneIngress := &core_mesh.ZoneIngressResource{
				Spec: &mesh_proto.ZoneIngress{
					Networking: &mesh_proto.ZoneIngress_Networking{
						AdvertisedAddress: "localhost",
						AdvertisedPort:    8888,
						Port:              8889,
					},
				},
			}
			err = resManager.Create(context.Background(), zoneIngress, store.CreateBy(tlKey))
			Expect(err).ToNot(HaveOccurred())

			// when
			err = resManager.DeleteAll(context.Background(), &core_mesh.DataplaneResourceList{}, store.DeleteAllByMesh("mesh-1"))

			// then
			Expect(err).ToNot(HaveOccurred())

			// and resource from mesh-1 is deleted
			res1 := core_mesh.NewDataplaneResource()
			err = resManager.Get(context.Background(), res1, store.GetByKey("tr-1", "mesh-1"))
			Expect(store.IsResourceNotFound(err)).To(BeTrue())

			// and only TrafficRoutes are deleted
			Expect(resManager.Get(context.Background(), core_mesh.NewZoneIngressResource(), store.GetBy(tlKey))).To(Succeed())

			// and resource from mesh-2 is retained
			res2 := core_mesh.NewDataplaneResource()
			err = resManager.Get(context.Background(), res2, store.GetByKey("tr-1", "mesh-2"))
			Expect(err).ToNot(HaveOccurred())
		})
	})
})
