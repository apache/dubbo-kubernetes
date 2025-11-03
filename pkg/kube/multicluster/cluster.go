package multicluster

import (
	"crypto/sha256"
	"github.com/apache/dubbo-kubernetes/pkg/cluster"
	"github.com/apache/dubbo-kubernetes/pkg/kube"
	"go.uber.org/atomic"
)

type Cluster struct {
	ID     cluster.ID
	Client kube.Client

	kubeConfigSha [sha256.Size]byte

	stop chan struct{}

	initialSync        *atomic.Bool
	initialSyncTimeout *atomic.Bool
}

func (c *Cluster) HasSynced() bool {
	// It could happen when a wrong credential provide, this cluster has no chance to run.
	// In this case, the `initialSyncTimeout` will never be set
	// In order not block istiod start up, check close as well.
	if c.Closed() {
		return true
	}
	return c.initialSync.Load() || c.initialSyncTimeout.Load()
}

func (c *Cluster) Closed() bool {
	select {
	case <-c.stop:
		return true
	default:
		return false
	}
}
