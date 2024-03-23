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

package datasource_test

import (
	"context"
)

import (
	. "github.com/onsi/ginkgo/v2"

	. "github.com/onsi/gomega"
)

import (
	system_proto "github.com/apache/dubbo-kubernetes/api/system/v1alpha1"
	"github.com/apache/dubbo-kubernetes/pkg/core/datasource"
	"github.com/apache/dubbo-kubernetes/pkg/core/resources/apis/system"
	"github.com/apache/dubbo-kubernetes/pkg/test/resources/model"
	util_proto "github.com/apache/dubbo-kubernetes/pkg/util/proto"
)

var _ = Describe("DataSource Loader", func() {
	var dataSourceLoader datasource.Loader

	BeforeEach(func() {
		secrets := []*system.SecretResource{
			{
				Meta: &model.ResourceMeta{
					Mesh: "default",
					Name: "test-secret",
				},
				Spec: &system_proto.Secret{
					Data: util_proto.Bytes([]byte("abc")),
				},
			},
		}
		dataSourceLoader = datasource.NewStaticLoader(secrets)
	})

	Context("Secret", func() {
		It("should load secret", func() {
			// when
			data, err := dataSourceLoader.Load(context.Background(), "default", &system_proto.DataSource{
				Type: &system_proto.DataSource_Secret{
					Secret: "test-secret",
				},
			})

			// then
			Expect(err).ToNot(HaveOccurred())
			Expect(data).To(Equal([]byte("abc")))
		})

		It("should throw an error when secret is not found", func() {
			// when
			_, err := dataSourceLoader.Load(context.Background(), "default", &system_proto.DataSource{
				Type: &system_proto.DataSource_Secret{
					Secret: "test-secret-2",
				},
			})

			// then
			Expect(err).To(MatchError(`could not load data: Resource not found: type="Secret" name="test-secret-2" mesh="default"`))
		})
	})

	Context("Inline", func() {
		It("should load from inline", func() {
			// when
			data, err := dataSourceLoader.Load(context.Background(), "default", &system_proto.DataSource{
				Type: &system_proto.DataSource_Inline{
					Inline: util_proto.Bytes([]byte("abc")),
				},
			})

			// then
			Expect(err).ToNot(HaveOccurred())
			Expect(data).To(Equal([]byte("abc")))
		})
	})

	Context("Inline string", func() {
		It("should load from inline string", func() {
			// when
			data, err := dataSourceLoader.Load(context.Background(), "default", &system_proto.DataSource{
				Type: &system_proto.DataSource_InlineString{
					InlineString: "abc",
				},
			})

			// then
			Expect(err).ToNot(HaveOccurred())
			Expect(data).To(Equal([]byte("abc")))
		})
	})
})
