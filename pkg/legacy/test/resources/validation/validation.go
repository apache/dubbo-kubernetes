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

package validation

import (
	. "github.com/onsi/ginkgo/v2"

	. "github.com/onsi/gomega"
)

import (
	core_model "github.com/apache/dubbo-kubernetes/pkg/core/resources/model"
	"github.com/apache/dubbo-kubernetes/pkg/core/validators"
)

// ResourceGenerator creates a resource of a pre-defined type.
type ResourceGenerator interface {
	New() core_model.Resource
}

// ResourceValidationCase captures a resource YAML and any corresponding validation error.
type ResourceValidationCase struct {
	Resource   string
	Violations []validators.Violation
}

// DescribeValidCases creates a Ginkgo table test for the given entries,
// where each entry is a valid YAML resource. It ensures that each entry
// can be successfully validated.
func DescribeValidCases[T core_model.Resource](generator func() T, cases ...TableEntry) {
	DescribeTable(
		"should pass validation",
		func(given string) {
			// setup
			resource := generator()

			// when
			err := core_model.FromYAML([]byte(given), resource.GetSpec())

			// then
			Expect(err).ToNot(HaveOccurred())

			// when
			verr := core_model.Validate(resource)

			// then
			Expect(verr).ToNot(HaveOccurred())
		},
		cases)
}

// DescribeErrorCases creates a Ginkgo table test for the given entries, where each entry
// is a ResourceValidationCase that contains an invalid resource YAML and the corresponding
// validation error.
func DescribeErrorCases[T core_model.Resource](generator func() T, cases ...TableEntry) {
	DescribeTable(
		"should validate all fields and return as many individual errors as possible",
		func(given ResourceValidationCase) {
			// setup
			resource := generator()

			// when
			Expect(
				core_model.FromYAML([]byte(given.Resource), resource.GetSpec()),
			).ToNot(HaveOccurred())

			expected := validators.ValidationError{
				Violations: given.Violations,
			}

			// then
			err := core_model.Validate(resource)
			Expect(err).To(HaveOccurred())
			verr := err.(*validators.ValidationError)
			Expect(verr.Violations).To(ConsistOf(expected.Violations))
		},
		cases,
	)
}

// ErrorCase is a helper that generates a table entry for DescribeErrorCases.
func ErrorCase(description string, err validators.Violation, yaml string) TableEntry {
	return Entry(
		description,
		ResourceValidationCase{
			Violations: []validators.Violation{err},
			Resource:   yaml,
		},
	)
}

func ErrorCases(description string, errs []validators.Violation, yaml string) TableEntry {
	return Entry(
		description,
		ResourceValidationCase{
			Violations: errs,
			Resource:   yaml,
		},
	)
}
