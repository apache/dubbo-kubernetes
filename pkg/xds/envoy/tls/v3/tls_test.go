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

package v3_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	mesh_proto "github.com/apache/dubbo-kubernetes/api/mesh/v1alpha1"
	core_mesh "github.com/apache/dubbo-kubernetes/pkg/core/resources/apis/mesh"
	test_model "github.com/apache/dubbo-kubernetes/pkg/test/resources/model"
	util_proto "github.com/apache/dubbo-kubernetes/pkg/util/proto"
	v3 "github.com/apache/dubbo-kubernetes/pkg/xds/envoy/tls/v3"
)

type caRequest struct {
	mesh string
}

func (r *caRequest) MeshName() []string {
	return []string{r.mesh}
}

func (r *caRequest) Name() string {
	return "mesh_ca:secret:" + r.mesh
}

type identityRequest struct {
	mesh string
}

func (r *identityRequest) Name() string {
	return "identity_cert:secret:" + r.mesh
}

var _ = Describe("CreateDownstreamTlsContext()", func() {
	Context("when mTLS is enabled on a given Mesh", func() {
		type testCase struct {
			expected string
		}

		DescribeTable("should generate proper Envoy config",
			func(given testCase) {
				// given
				mesh := &core_mesh.MeshResource{
					Meta: &test_model.ResourceMeta{
						Name: "default",
					},
					Spec: &mesh_proto.Mesh{
						Mtls: &mesh_proto.Mesh_Mtls{
							EnabledBackend: "builtin",
							Backends: []*mesh_proto.CertificateAuthorityBackend{
								{
									Name: "builtin",
									Type: "builtin",
								},
							},
						},
					},
				}

				// when
				snippet, err := v3.CreateDownstreamTlsContext(
					&caRequest{mesh: mesh.GetMeta().GetName()},
					&identityRequest{mesh: mesh.GetMeta().GetName()},
				)
				// then
				Expect(err).ToNot(HaveOccurred())
				// when
				actual, err := util_proto.ToYAML(snippet)
				// then
				Expect(err).ToNot(HaveOccurred())
				// and
				Expect(actual).To(MatchYAML(given.expected))
			},
			Entry("metadata is `nil`", testCase{
				expected: `
                commonTlsContext:
                  combinedValidationContext:
                    defaultValidationContext:
                      matchTypedSubjectAltNames:
                      - matcher:
                          prefix: spiffe://default/
                        sanType: URI
                    validationContextSdsSecretConfig:
                      name: mesh_ca:secret:default
                      sdsConfig:
                        ads: {}
                        resourceApiVersion: V3
                  tlsCertificateSdsSecretConfigs:
                  - name: identity_cert:secret:default
                    sdsConfig:
                      ads: {}
                      resourceApiVersion: V3
                requireClientCertificate: true`,
			}),
		)
	})
})

var _ = Describe("CreateUpstreamTlsContext()", func() {
	Context("when mTLS is enabled on a given Mesh", func() {
		type testCase struct {
			upstreamService string
			expected        string
		}

		DescribeTable("should generate proper Envoy config",
			func(given testCase) {
				// given
				mesh := "default"

				// when
				snippet, err := v3.CreateUpstreamTlsContext(
					&identityRequest{mesh: mesh},
					&caRequest{mesh: mesh},
					given.upstreamService,
					"",
					nil,
				)
				// then
				Expect(err).ToNot(HaveOccurred())
				// when
				actual, err := util_proto.ToYAML(snippet)
				// then
				Expect(err).ToNot(HaveOccurred())
				// and
				Expect(actual).To(MatchYAML(given.expected))
			},
			Entry("metadata is `nil`", testCase{
				upstreamService: "backend",
				expected: `
                commonTlsContext:
                  alpnProtocols:
                  - dubbo
                  combinedValidationContext:
                    defaultValidationContext:
                      matchTypedSubjectAltNames:
                      - matcher:
                          exact: spiffe://default/backend
                        sanType: URI
                    validationContextSdsSecretConfig:
                      name: mesh_ca:secret:default
                      sdsConfig:
                        ads: {}
                        resourceApiVersion: V3
                  tlsCertificateSdsSecretConfigs:
                  - name: identity_cert:secret:default
                    sdsConfig:
                      ads: {}
                      resourceApiVersion: V3`,
			}),
		)
	})
})
