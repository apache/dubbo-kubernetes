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
