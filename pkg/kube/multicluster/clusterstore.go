package multicluster

import (
	"github.com/apache/dubbo-kubernetes/pkg/util/sets"
	"sync"
)

type ClusterStore struct {
	sync.RWMutex
	clusters sets.String
}

func (c *ClusterStore) HasSynced() bool {
	c.RLock()
	defer c.RUnlock()
	return true
}
