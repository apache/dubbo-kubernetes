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

package test

import (
	"time"
)

import (
	"github.com/onsi/ginkgo/v2"

	"github.com/onsi/gomega"
)

// Within returns a function that executes the given test task in
// a dedicated goroutine, asserting that it must complete within
// the given timeout.
//
// See https://github.com/onsi/ginkgo/blob/v2/docs/MIGRATING_TO_V2.md#removed-async-testing
func Within(timeout time.Duration, task func()) func() {
	return func() {
		done := make(chan interface{})

		go func() {
			defer ginkgo.GinkgoRecover()
			defer close(done)
			task()
		}()

		gomega.Eventually(done, timeout).Should(gomega.BeClosed())
	}
}
