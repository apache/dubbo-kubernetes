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

package admin_test

import (
	"os"
	"path"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/apache/dubbo-kubernetes/pkg/envoy/admin"
	"github.com/apache/dubbo-kubernetes/pkg/test/matchers"
	_ "github.com/apache/dubbo-kubernetes/pkg/xds/envoy"
)

var _ = Describe("Sanitize ConfigDump", func() {
	type testCase struct {
		configFile string
		goldenFile string
	}

	DescribeTable("should redact sensitive information",
		func(given testCase) {
			// given
			rawConfigDump, err := os.ReadFile(filepath.Join("testdata", given.configFile))
			Expect(err).ToNot(HaveOccurred())

			// when
			sanitizedDump, err := admin.Sanitize(rawConfigDump)
			Expect(err).ToNot(HaveOccurred())

			// then
			Expect(sanitizedDump).To(matchers.MatchGoldenJSON(path.Join("testdata", given.goldenFile)))
		},
		Entry("full config", testCase{
			configFile: "full_config.json",
			goldenFile: "golden.full_config.json",
		}),
		Entry("no hds", testCase{
			configFile: "no_hds.json",
			goldenFile: "golden.no_hds.json",
		}),
	)
})
