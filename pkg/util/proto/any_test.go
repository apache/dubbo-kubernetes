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

package proto_test

import (
	. "github.com/onsi/ginkgo/v2"

	. "github.com/onsi/gomega"
)

import (
	"github.com/apache/dubbo-kubernetes/pkg/test/matchers"
	util_proto "github.com/apache/dubbo-kubernetes/pkg/util/proto"
	envoy_metadata "github.com/apache/dubbo-kubernetes/pkg/xds/envoy/metadata/v3"
)

var _ = Describe("MarshalAnyDeterministic", func() {
	It("should marshal deterministically", func() {
		tags := map[string]string{
			"service": "backend",
			"version": "v1",
			"cloud":   "aws",
		}
		metadata := envoy_metadata.EndpointMetadata(tags)
		for i := 0; i < 100; i++ {
			any1, _ := util_proto.MarshalAnyDeterministic(metadata)
			any2, _ := util_proto.MarshalAnyDeterministic(metadata)
			Expect(any1).To(matchers.MatchProto(any2))
		}
	})
})
