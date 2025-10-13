package multicluster

import (
	"crypto/sha256"
	"github.com/apache/dubbo-kubernetes/pkg/cluster"
	"github.com/apache/dubbo-kubernetes/pkg/kube"
	"go.uber.org/atomic"
)

type Cluster struct {
	// ID of the cluster.
	ID cluster.ID
	// Client for accessing the cluster.
	Client kube.Client

	kubeConfigSha [sha256.Size]byte

	stop chan struct{}
	// initialSync is marked when RunAndWait completes
	initialSync *atomic.Bool
	// initialSyncTimeout is set when RunAndWait timed out
	initialSyncTimeout *atomic.Bool
}
