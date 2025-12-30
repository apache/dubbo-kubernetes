//
// Licensed to the Apache Software Foundation (ASF) under one or more
// contributor license agreements.  See the NOTICE file distributed with
// this work for additional information regarding copyright ownership.
// The ASF licenses this file to You under the Apache License, Version 2.0
// (the "License"); you may not use this file except in compliance with
// the License.  You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package model

import (
	"github.com/apache/dubbo-kubernetes/pkg/cluster"
	"github.com/apache/dubbo-kubernetes/pkg/slices"
)

func (m *AddressMap) GetAddresses() map[cluster.ID][]string {
	if m == nil {
		return nil
	}

	m.mutex.RLock()
	defer m.mutex.RUnlock()

	if m.Addresses == nil {
		return nil
	}

	out := make(map[cluster.ID][]string)
	for k, v := range m.Addresses {
		out[k] = slices.Clone(v)
	}
	return out
}

func (m *AddressMap) AddAddressesFor(c cluster.ID, addresses []string) *AddressMap {
	if len(addresses) == 0 {
		return m
	}

	m.mutex.Lock()
	defer m.mutex.Unlock()

	// Create the map if nil.
	if m.Addresses == nil {
		m.Addresses = make(map[cluster.ID][]string)
	}

	m.Addresses[c] = append(m.Addresses[c], addresses...)
	return m
}

func (m *AddressMap) GetAddressesFor(c cluster.ID) []string {
	if m == nil {
		return nil
	}

	m.mutex.RLock()
	defer m.mutex.RUnlock()

	if m.Addresses == nil {
		return nil
	}

	// Copy the Addresses array.
	return append([]string{}, m.Addresses[c]...)
}

func (m *AddressMap) SetAddressesFor(c cluster.ID, addresses []string) *AddressMap {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if len(addresses) == 0 {
		// Setting an empty array for the cluster. Remove the entry for the cluster if it exists.
		if m.Addresses != nil {
			delete(m.Addresses, c)

			// Delete the map if there's nothing left.
			if len(m.Addresses) == 0 {
				m.Addresses = nil
			}
		}
	} else {
		// Create the map if it doesn't already exist.
		if m.Addresses == nil {
			m.Addresses = make(map[cluster.ID][]string)
		}
		m.Addresses[c] = addresses
	}
	return m
}

func (m *AddressMap) Len() int {
	if m == nil {
		return 0
	}
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	return len(m.Addresses)
}
