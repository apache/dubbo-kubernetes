package v3

import (
	"context"
	"errors"
	"github.com/apache/dubbo-kubernetes/pkg/core"
	envoy_core "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	envoy_cache "github.com/envoyproxy/go-control-plane/pkg/cache/v3"
)

var reconcileLog = core.Log.WithName("xds-server").WithName("reconcile")

type reconciler struct {
}

type snapshotGenerator interface {
	GenerateSnapshot(ctx context.Context, node *envoy_core.Node) (*envoy_cache.Snapshot, error)
}

type TemplateSnapshotGenerator struct {
}

type snapshotCacher interface {
	Get(node *envoy_core.Node) (*envoy_cache.Snapshot, error)
	Cache(ctx context.Context, node *envoy_core.Node, snapshot *envoy_cache.Snapshot) error
	Clear(node *envoy_core.Node)
}

type simpleSnapshotCacher struct {
	hasher envoy_cache.NodeHash
	store  envoy_cache.SnapshotCache
}

func (s *simpleSnapshotCacher) Get(node *envoy_core.Node) (*envoy_cache.Snapshot, error) {
	snap, err := s.store.GetSnapshot(s.hasher.ID(node))
	if snap != nil {
		snapshot, ok := snap.(*envoy_cache.Snapshot)
		if !ok {
			return nil, errors.New("couldn't convert snapshot from cache to envoy Snapshot")
		}
		return snapshot, nil
	}
	return nil, err
}

func (s *simpleSnapshotCacher) Cache(ctx context.Context, node *envoy_core.Node, snapshot *envoy_cache.Snapshot) error {
	return s.store.SetSnapshot(ctx, s.hasher.ID(node), snapshot)
}

func (s *simpleSnapshotCacher) Clear(node *envoy_core.Node) {
	s.store.ClearSnapshot(s.hasher.ID(node))
}
