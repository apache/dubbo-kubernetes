//go:build !agent
// +build !agent

package collections

import (
	"github.com/apache/dubbo-kubernetes/pkg/config/schema/collection"
)

var (
	Sail = collection.NewSchemasBuilder().
		// TODO MustAdd
		Build()
)
