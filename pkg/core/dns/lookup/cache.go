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

package lookup

import (
	"net"
	"sync"
	"time"
)

import (
	"github.com/apache/dubbo-kubernetes/pkg/core"
)

type cacheRecord struct {
	ips          []net.IP
	creationTime time.Time
}

func CacheLookupIP(f LookupIPFunc, ttl time.Duration) LookupIPFunc {
	cache := map[string]*cacheRecord{}
	var rwmux sync.RWMutex
	return func(host string) ([]net.IP, error) {
		rwmux.RLock()
		r, ok := cache[host]
		rwmux.RUnlock()

		if ok && r.creationTime.Add(ttl).After(core.Now()) {
			return r.ips, nil
		}

		ips, err := f(host)
		if err != nil {
			return nil, err
		}

		rwmux.Lock()
		cache[host] = &cacheRecord{ips: ips, creationTime: core.Now()}
		rwmux.Unlock()

		return ips, nil
	}
}
