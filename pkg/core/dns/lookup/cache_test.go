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

package lookup_test

import (
	"net"
	"time"
)

import (
	. "github.com/onsi/ginkgo/v2"

	. "github.com/onsi/gomega"
)

import (
	"github.com/apache/dubbo-kubernetes/pkg/core"
	"github.com/apache/dubbo-kubernetes/pkg/core/dns/lookup"
)

var _ = Describe("DNS with cache", func() {
	var counter int
	var table map[string][]net.IP
	var lookupFunc lookup.LookupIPFunc = func(host string) ([]net.IP, error) {
		counter++
		return table[host], nil
	}
	var cachingLookupFunc lookup.LookupIPFunc

	BeforeEach(func() {
		cachingLookupFunc = lookup.CachedLookupIP(lookupFunc, 1*time.Second)
		table = map[string][]net.IP{}
		counter = 0
	})

	It("should use cache on the second call", func() {
		_, _ = cachingLookupFunc("example.com")
		_, _ = cachingLookupFunc("example.com")
		Expect(counter).To(Equal(1))
	})

	It("should avoid cache after TTL", func() {
		table["example.com"] = []net.IP{net.ParseIP("192.168.0.1")}

		ip, _ := cachingLookupFunc("example.com")
		Expect(ip[0]).To(Equal(net.ParseIP("192.168.0.1")))

		ip, _ = cachingLookupFunc("example.com")
		Expect(ip[0]).To(Equal(net.ParseIP("192.168.0.1")))

		table["example.com"] = []net.IP{net.ParseIP("10.20.0.1")}
		core.Now = func() time.Time {
			return time.Now().Add(2 * time.Second)
		}
		ip, _ = cachingLookupFunc("example.com")
		Expect(ip[0]).To(Equal(net.ParseIP("10.20.0.1")))
		Expect(counter).To(Equal(2))
	})
})
