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

package component_test

import (
	. "github.com/onsi/ginkgo/v2"

	. "github.com/onsi/gomega"
)

import (
	"github.com/apache/dubbo-kubernetes/pkg/core/runtime/component"
	leader_memory "github.com/apache/dubbo-kubernetes/pkg/plugins/leader/memory"
)

var _ = Describe("Component Manager", func() {
	Context("Component Manager is running", func() {
		var manager component.Manager
		var stopCh chan struct{}

		BeforeAll(func() {
			// given
			manager = component.NewManager(leader_memory.NewNeverLeaderElector())
			chComponentBeforeStart := make(chan int)
			err := manager.Add(component.ComponentFunc(func(_ <-chan struct{}) error {
				close(chComponentBeforeStart)
				return nil
			}))

			// when component manager is started
			stopCh = make(chan struct{})
			go func() {
				defer GinkgoRecover()
				Expect(manager.Start(stopCh)).To(Succeed())
			}()

			// then component added before Start() runs
			Expect(err).ToNot(HaveOccurred())
			Eventually(chComponentBeforeStart, "30s", "50ms").Should(BeClosed())
		})

		AfterAll(func() {
			close(stopCh)
		})

		It("should be able to add component in runtime", func() {
			// when component is added after Start()
			chComponentAfterStart := make(chan int)
			err := manager.Add(component.ComponentFunc(func(_ <-chan struct{}) error {
				close(chComponentAfterStart)
				return nil
			}))

			// then it runs
			Expect(err).ToNot(HaveOccurred())
			Eventually(chComponentAfterStart, "30s", "50ms").Should(BeClosed())
		})

		It("should not be able to add leader component", func() {
			// when leader component is added after Start()
			err := manager.Add(component.LeaderComponentFunc(func(_ <-chan struct{}) error {
				return nil
			}))

			// then
			Expect(err).To(Equal(component.LeaderComponentAddAfterStartErr))
		})
	})
}, Ordered)
