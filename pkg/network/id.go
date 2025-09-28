package network

import "github.com/apache/dubbo-kubernetes/pkg/util/identifier"

// ID is the unique identifier for a network.
type ID string

func (id ID) Equals(other ID) bool {
	return identifier.IsSameOrEmpty(string(id), string(other))
}

func (id ID) String() string {
	return string(id)
}
