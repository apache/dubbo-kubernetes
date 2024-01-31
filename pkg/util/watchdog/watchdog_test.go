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

package watchdog_test

import (
	"context"
	"fmt"
	"time"
)

import (
	. "github.com/onsi/ginkgo/v2"

	. "github.com/onsi/gomega"
)

import (
	"github.com/apache/dubbo-kubernetes/pkg/test"
	. "github.com/apache/dubbo-kubernetes/pkg/util/watchdog"
)

var _ = Describe("SimpleWatchdog", func() {
	var timeTicks chan time.Time
	var onTickCalls chan struct{}
	var onErrorCalls chan error

	BeforeEach(func() {
		timeTicks = make(chan time.Time)
		onTickCalls = make(chan struct{})
		onErrorCalls = make(chan error)
	})

	var stopCh, doneCh chan struct{}

	BeforeEach(func() {
		stopCh = make(chan struct{})
		doneCh = make(chan struct{})
	})

	It("should call OnTick() on timer ticks", test.Within(5*time.Second, func() {
		// given
		watchdog := SimpleWatchdog{
			NewTicker: func() *time.Ticker {
				return &time.Ticker{
					C: timeTicks,
				}
			},
			OnTick: func(context.Context) error {
				onTickCalls <- struct{}{}
				return nil
			},
		}

		// setup
		go func() {
			watchdog.Start(stopCh)

			close(doneCh)
		}()

		By("simulating 1st tick")
		// when
		timeTicks <- time.Time{}

		// then
		<-onTickCalls

		By("simulating 2nd tick")
		// when
		timeTicks <- time.Time{}

		// then
		<-onTickCalls

		By("simulating Dataplane disconnect")
		// when
		close(stopCh)

		// then
		<-doneCh
	}))

	It("should call OnError() when OnTick() returns an error", test.Within(5*time.Second, func() {
		// given
		expectedErr := fmt.Errorf("expected error")
		// and
		watchdog := SimpleWatchdog{
			NewTicker: func() *time.Ticker {
				return &time.Ticker{
					C: timeTicks,
				}
			},
			OnTick: func(context.Context) error {
				return expectedErr
			},
			OnError: func(err error) {
				onErrorCalls <- err
			},
		}

		// setup
		go func() {
			watchdog.Start(stopCh)

			close(doneCh)
		}()

		By("simulating 1st tick")
		// when
		timeTicks <- time.Time{}

		// then
		actualErr := <-onErrorCalls
		Expect(actualErr).To(MatchError(expectedErr))

		By("simulating Dataplane disconnect")
		// when
		close(stopCh)

		// then
		<-doneCh
	}))

	It("should not crash the whole application when watchdog crashes", test.Within(5*time.Second, func() {
		// given
		watchdog := SimpleWatchdog{
			NewTicker: func() *time.Ticker {
				return &time.Ticker{
					C: timeTicks,
				}
			},
			OnTick: func(context.Context) error {
				panic("xyz")
			},
			OnError: func(err error) {
				onErrorCalls <- err
			},
		}

		// when
		go func() {
			watchdog.Start(stopCh)
			close(doneCh)
		}()
		timeTicks <- time.Time{}

		// then watchdog returned an error
		Expect(<-onErrorCalls).To(HaveOccurred())
		close(stopCh)
		<-doneCh
	}))
})
