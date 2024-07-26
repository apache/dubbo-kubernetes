package store

import (
	core_store "github.com/apache/dubbo-kubernetes/pkg/core/resources/store"
)

type SecretStore interface {
	core_store.ResourceStore
}
