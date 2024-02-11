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

package cache_test

import (
	"fmt"
	mesh_proto "github.com/apache/dubbo-kubernetes/api/mesh/v1alpha1"
	core_mesh "github.com/apache/dubbo-kubernetes/pkg/core/resources/apis/mesh"
	"github.com/apache/dubbo-kubernetes/pkg/dds/cache"
	envoy_types "github.com/envoyproxy/go-control-plane/pkg/cache/types"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
)

var _ = Describe("Snapshot", func() {
	mustMarshalAny := func(pb proto.Message) *anypb.Any {
		a, err := anypb.New(pb)
		if err != nil {
			panic(err)
		}
		return a
	}

	Describe("GetResources()", func() {
		It("should handle `nil`", func() {
			// when
			var snapshot *cache.Snapshot
			// then
			Expect(snapshot.GetResources(string(core_mesh.MeshType))).To(BeNil())
		})

		It("should return Meshes", func() {
			// given
			resources := &mesh_proto.DubboResource{
				Meta: &mesh_proto.DubboResource_Meta{Name: "mesh1", Mesh: "mesh1"},
				Spec: mustMarshalAny(&mesh_proto.Mesh{}),
			}
			// when
			snapshot := cache.NewSnapshotBuilder().
				With(core_mesh.MeshType, []envoy_types.Resource{resources}).
				Build("v1")
			// then
			expected := map[string]envoy_types.Resource{
				"mesh1.mesh1": resources,
			}
			Expect(snapshot.GetResources(string(core_mesh.MeshType))).To(Equal(expected))
		})

		It("should return `nil` for unsupported resource types", func() {
			// given
			resources := &mesh_proto.DubboResource{
				Meta: &mesh_proto.DubboResource_Meta{Name: "mesh1", Mesh: "mesh1"},
				Spec: mustMarshalAny(&mesh_proto.Mesh{}),
			}
			// when
			snapshot := cache.NewSnapshotBuilder().
				With(core_mesh.MeshType, []envoy_types.Resource{resources}).
				Build("v1")
			// then
			Expect(snapshot.GetResources("UnsupportedType")).To(BeNil())
		})
	})

	Describe("GetVersion()", func() {
		It("should handle `nil`", func() {
			// when
			var snapshot *cache.Snapshot
			// then
			Expect(snapshot.GetVersion(string(core_mesh.MeshType))).To(Equal(""))
		})

		It("should return proper version for a supported resource type", func() {
			// given
			resources := &mesh_proto.DubboResource{
				Meta: &mesh_proto.DubboResource_Meta{Name: "mesh1", Mesh: "mesh1"},
				Spec: mustMarshalAny(&mesh_proto.Mesh{}),
			}
			// when
			snapshot := cache.NewSnapshotBuilder().
				With(core_mesh.MeshType, []envoy_types.Resource{resources}).
				Build("v1")
			// then
			Expect(snapshot.GetVersion(string(core_mesh.MeshType))).To(Equal("v1"))
		})

		It("should return an empty string for unsupported resource type", func() {
			// given
			resources := &mesh_proto.DubboResource{
				Meta: &mesh_proto.DubboResource_Meta{Name: "mesh1", Mesh: "mesh1"},
				Spec: mustMarshalAny(&mesh_proto.Mesh{}),
			}
			// when
			snapshot := cache.NewSnapshotBuilder().
				With(core_mesh.MeshType, []envoy_types.Resource{resources}).
				Build("v1")
			// then
			Expect(snapshot.GetVersion("unsupported type")).To(Equal(""))
		})
	})

	Describe("ConstructVersionMap()", func() {
		It("should handle `nil`", func() {
			// when
			var snapshot *cache.Snapshot
			// then
			Expect(snapshot.ConstructVersionMap()).To(Equal(fmt.Errorf("missing snapshot")))
		})

		It("should construct version map for resource", func() {
			// given
			resources := &mesh_proto.DubboResource{
				Meta: &mesh_proto.DubboResource_Meta{Name: "mesh1", Mesh: "mesh1"},
				Spec: mustMarshalAny(&mesh_proto.Mesh{}),
			}
			snapshot := cache.NewSnapshotBuilder().
				With(core_mesh.MeshType, []envoy_types.Resource{resources}).
				Build("v1")

			// when
			Expect(snapshot.ConstructVersionMap()).ToNot(HaveOccurred())

			// then
			Expect(snapshot.GetVersionMap(string(core_mesh.MeshType))["mesh1.mesh1"]).ToNot(BeEmpty())
		})

		It("should change version when resource has changed", func() {
			// given
			resources := &mesh_proto.DubboResource{
				Meta: &mesh_proto.DubboResource_Meta{Name: "mesh1", Mesh: "mesh1"},
				Spec: mustMarshalAny(&mesh_proto.Mesh{}),
			}
			snapshot := cache.NewSnapshotBuilder().
				With(core_mesh.MeshType, []envoy_types.Resource{resources}).
				Build("v1")

			// when
			Expect(snapshot.ConstructVersionMap()).ToNot(HaveOccurred())

			// then
			Expect(snapshot.GetVersionMap(string(core_mesh.MeshType))["mesh1.mesh1"]).ToNot(BeEmpty())

			// when
			previousVersion := snapshot.GetVersionMap(string(core_mesh.MeshType))["mesh1.mesh1"]

			// given
			resources = &mesh_proto.DubboResource{
				Meta: &mesh_proto.DubboResource_Meta{Name: "mesh1", Mesh: "mesh1"},
				Spec: mustMarshalAny(&mesh_proto.Mesh{
					Mtls: &mesh_proto.Mesh_Mtls{
						EnabledBackend: "ca",
						Backends: []*mesh_proto.CertificateAuthorityBackend{
							{
								Name: "ca",
								Type: "builtin",
							},
						},
					},
				}),
			}
			snapshot = cache.NewSnapshotBuilder().
				With(core_mesh.MeshType, []envoy_types.Resource{resources}).
				Build("v1")

			// when
			Expect(snapshot.ConstructVersionMap()).ToNot(HaveOccurred())

			// then
			Expect(snapshot.GetVersionMap(string(core_mesh.MeshType))["mesh1.mesh1"]).ToNot(Equal(previousVersion))
		})

		It("should not change version when resource has not changed", func() {
			// given
			resources := &mesh_proto.DubboResource{
				Meta: &mesh_proto.DubboResource_Meta{Name: "mesh1", Mesh: "mesh1"},
				Spec: mustMarshalAny(&mesh_proto.Mesh{}),
			}
			snapshot := cache.NewSnapshotBuilder().
				With(core_mesh.MeshType, []envoy_types.Resource{resources}).
				Build("v1")

			// when
			Expect(snapshot.ConstructVersionMap()).ToNot(HaveOccurred())

			// then
			Expect(snapshot.GetVersionMap(string(core_mesh.MeshType))["mesh1.mesh1"]).ToNot(BeEmpty())

			// when
			previousVersion := snapshot.GetVersionMap(string(core_mesh.MeshType))["mesh1.mesh1"]

			// given
			resources = &mesh_proto.DubboResource{
				Meta: &mesh_proto.DubboResource_Meta{Name: "mesh1", Mesh: "mesh1"},
				Spec: mustMarshalAny(&mesh_proto.Mesh{}),
			}
			snapshot = cache.NewSnapshotBuilder().
				With(core_mesh.MeshType, []envoy_types.Resource{resources}).
				Build("v1")

			// when
			Expect(snapshot.ConstructVersionMap()).ToNot(HaveOccurred())

			// then
			Expect(snapshot.GetVersionMap(string(core_mesh.MeshType))["mesh1.mesh1"]).To(Equal(previousVersion))
		})
	})
})
