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

package util_test

import (
	"fmt"
)

import (
	. "github.com/onsi/ginkgo/v2"

	. "github.com/onsi/gomega"
)

import (
	"github.com/apache/dubbo-kubernetes/pkg/dds/util"
	test_model "github.com/apache/dubbo-kubernetes/pkg/test/resources/model"
)

var _ = Describe("TrimSuffixFromName", func() {
	type testCase struct {
		name   string
		suffix string
	}

	name := func(given testCase) string {
		return fmt.Sprintf("%s.%s", given.name, given.suffix)
	}

	DescribeTable("should remove provided suffix from the name of "+
		"the provided resource",
		func(given testCase) {
			// given
			meta := &test_model.ResourceMeta{Name: name(given)}
			resource := &test_model.Resource{Meta: meta}

			// when
			util.TrimSuffixFromName(resource, given.suffix)

			// then
			Expect(resource.GetMeta().GetName()).To(Equal(given.name))
		},
		// entry description generator
		func(given testCase) string {
			return fmt.Sprintf("name: %q, suffix: %q", name(given), given.suffix)
		},
		Entry(nil, testCase{name: "foo", suffix: "bar"}),
		Entry(nil, testCase{name: "bar", suffix: "baz"}),
		Entry(nil, testCase{name: "baz", suffix: "dubbo-system"}),
		Entry(nil, testCase{name: "faz", suffix: "daz.dubbo-system"}),
	)
})
