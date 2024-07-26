package store

import (
	core_store "github.com/apache/dubbo-kubernetes/pkg/core/resources/store"
)

// todo consider unifing SecretStore with ResourceStore
func NewSecretStore(resourceStore core_store.ResourceStore) SecretStore {
	return resourceStore
}
