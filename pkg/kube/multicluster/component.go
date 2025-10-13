package multicluster

import (
	"github.com/apache/dubbo-kubernetes/pkg/cluster"
	"github.com/apache/dubbo-kubernetes/pkg/maps"
	"sync"
)

type ComponentConstraint interface {
	Close()
	HasSynced() bool
}

type Component[T ComponentConstraint] struct {
	mu          sync.RWMutex
	constructor func(cluster *Cluster) T
	clusters    map[cluster.ID]T
}

func (m *Component[T]) clusterAdded(cluster *Cluster) ComponentConstraint {
	comp := m.constructor(cluster)
	m.mu.Lock()
	defer m.mu.Unlock()
	m.clusters[cluster.ID] = comp
	return comp
}

func (m *Component[T]) clusterUpdated(cluster *Cluster) ComponentConstraint {
	// Build outside of the lock, in case its slow
	comp := m.constructor(cluster)
	old, f := m.clusters[cluster.ID]
	m.mu.Lock()
	m.clusters[cluster.ID] = comp
	m.mu.Unlock()
	// Close outside of the lock, in case its slow
	if f {
		old.Close()
	}
	return comp
}

func (m *Component[T]) clusterDeleted(cluster cluster.ID) {
	m.mu.Lock()
	defer m.mu.Unlock()
	// If there is an old one, close it
	if old, f := m.clusters[cluster]; f {
		old.Close()
	}
	delete(m.clusters, cluster)
}

func (m *Component[T]) All() []T {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return maps.Values(m.clusters)
}

func (m *Component[T]) HasSynced() bool {
	for _, c := range m.All() {
		if !c.HasSynced() {
			return false
		}
	}
	return true
}
