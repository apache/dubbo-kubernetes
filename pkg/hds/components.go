package hds

import (
	"context"
	config_core "github.com/apache/dubbo-kubernetes/pkg/config/core"
	"github.com/apache/dubbo-kubernetes/pkg/core"
	core_runtime "github.com/apache/dubbo-kubernetes/pkg/core/runtime"
	hds_callbacks "github.com/apache/dubbo-kubernetes/pkg/hds/callbacks"
	hds_server "github.com/apache/dubbo-kubernetes/pkg/hds/server"
	"github.com/apache/dubbo-kubernetes/pkg/hds/tracker"
	util_xds "github.com/apache/dubbo-kubernetes/pkg/util/xds"
	util_xds_v3 "github.com/apache/dubbo-kubernetes/pkg/util/xds/v3"
	envoy_core "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	envoy_service_health "github.com/envoyproxy/go-control-plane/envoy/service/health/v3"
)

var hdsServerLog = core.Log.WithName("hds-server")

func Setup(rt core_runtime.Runtime) error {
	if rt.Config().Mode == config_core.Global {
		return nil
	}
	if !rt.Config().DpServer.Hds.Enabled {
		return nil
	}

	snapshotCache := util_xds_v3.NewSnapshotCache(false, hasher{}, util_xds.NewLogger(hdsServerLog))

	callbacks, err := DefaultCallbacks(rt, snapshotCache)
	if err != nil {
		return err
	}

	srv := hds_server.New(context.Background(), snapshotCache, callbacks)

	hdsServerLog.Info("registering Health Discovery Service in Dataplane Server")
	envoy_service_health.RegisterHealthDiscoveryServiceServer(rt.DpServer().GrpcServer(), srv)
	return nil
}

func DefaultCallbacks(rt core_runtime.Runtime, cache util_xds_v3.SnapshotCache) (hds_callbacks.Callbacks, error) {
	return hds_callbacks.Chain{
		tracker.NewCallbacks(
			hdsServerLog,
			rt.ResourceManager(),
			rt.ReadOnlyResourceManager(),
			cache,
			rt.Config().DpServer.Hds,
			hasher{},
			rt.Config().GetEnvoyAdminPort()),
	}, nil
}

type hasher struct{}

func (_ hasher) ID(node *envoy_core.Node) string {
	return node.Id
}
