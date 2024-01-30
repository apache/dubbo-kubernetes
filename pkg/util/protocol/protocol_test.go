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

package protocol_test

import (
	. "github.com/onsi/ginkgo/v2"

	. "github.com/onsi/gomega"
)

import (
	core_mesh "github.com/apache/dubbo-kubernetes/pkg/core/resources/apis/mesh"
	. "github.com/apache/dubbo-kubernetes/pkg/util/protocol"
)

var _ = Describe("GetCommonProtocol()", func() {
	type testCase struct {
		one      core_mesh.Protocol
		another  core_mesh.Protocol
		expected core_mesh.Protocol
	}

	DescribeTable("should correctly determine common protocol",
		func(given testCase) {
			// when
			actual := GetCommonProtocol(given.one, given.another)
			// then
			Expect(actual).To(Equal(given.expected))
		},
		Entry("`unknown` and `unknown`", testCase{
			one:      core_mesh.ProtocolUnknown,
			another:  core_mesh.ProtocolUnknown,
			expected: core_mesh.ProtocolUnknown,
		}),
		Entry("`unknown` and `http`", testCase{
			one:      core_mesh.ProtocolUnknown,
			another:  core_mesh.ProtocolHTTP,
			expected: core_mesh.ProtocolUnknown,
		}),
		Entry("`http` and `unknown`", testCase{
			one:      core_mesh.ProtocolHTTP,
			another:  core_mesh.ProtocolUnknown,
			expected: core_mesh.ProtocolUnknown,
		}),
		Entry("`unknown` and `tcp`", testCase{
			one:      core_mesh.ProtocolUnknown,
			another:  core_mesh.ProtocolTCP,
			expected: core_mesh.ProtocolUnknown,
		}),
		Entry("`tcp` and `unknown`", testCase{
			one:      core_mesh.ProtocolTCP,
			another:  core_mesh.ProtocolUnknown,
			expected: core_mesh.ProtocolUnknown,
		}),
		Entry("`http` and `tcp`", testCase{
			one:      core_mesh.ProtocolHTTP,
			another:  core_mesh.ProtocolTCP,
			expected: core_mesh.ProtocolTCP,
		}),
		Entry("`tcp` and `http`", testCase{
			one:      core_mesh.ProtocolTCP,
			another:  core_mesh.ProtocolHTTP,
			expected: core_mesh.ProtocolTCP,
		}),
		Entry("`http` and `http`", testCase{
			one:      core_mesh.ProtocolHTTP,
			another:  core_mesh.ProtocolHTTP,
			expected: core_mesh.ProtocolHTTP,
		}),
		Entry("`tcp` and `tcp`", testCase{
			one:      core_mesh.ProtocolTCP,
			another:  core_mesh.ProtocolTCP,
			expected: core_mesh.ProtocolTCP,
		}),
		Entry("`http2` and `http2`", testCase{
			one:      core_mesh.ProtocolHTTP2,
			another:  core_mesh.ProtocolHTTP2,
			expected: core_mesh.ProtocolHTTP2,
		}),
		Entry("`http2` and `http`", testCase{
			one:      core_mesh.ProtocolHTTP2,
			another:  core_mesh.ProtocolHTTP,
			expected: core_mesh.ProtocolTCP,
		}),
		Entry("`http2` and `tcp`", testCase{
			one:      core_mesh.ProtocolHTTP2,
			another:  core_mesh.ProtocolTCP,
			expected: core_mesh.ProtocolTCP,
		}),
		Entry("`grpc` and `grpc`", testCase{
			one:      core_mesh.ProtocolGRPC,
			another:  core_mesh.ProtocolGRPC,
			expected: core_mesh.ProtocolGRPC,
		}),
		Entry("`grpc` and `http`", testCase{
			one:      core_mesh.ProtocolGRPC,
			another:  core_mesh.ProtocolHTTP,
			expected: core_mesh.ProtocolTCP,
		}),
		Entry("`grpc` and `http2`", testCase{
			one:      core_mesh.ProtocolGRPC,
			another:  core_mesh.ProtocolHTTP2,
			expected: core_mesh.ProtocolHTTP2,
		}),
		Entry("`grpc` and `tcp`", testCase{
			one:      core_mesh.ProtocolGRPC,
			another:  core_mesh.ProtocolTCP,
			expected: core_mesh.ProtocolTCP,
		}),
		Entry("`kafka` and `tcp`", testCase{
			one:      core_mesh.ProtocolKafka,
			another:  core_mesh.ProtocolTCP,
			expected: core_mesh.ProtocolTCP,
		}),
	)
})
