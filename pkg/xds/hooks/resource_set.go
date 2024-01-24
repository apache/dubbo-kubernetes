package hooks

import (
	core_xds "github.com/apache/dubbo-kubernetes/pkg/core/xds"
	xds_context "github.com/apache/dubbo-kubernetes/pkg/xds/context"
)

type ResourceSetHook interface {
	Modify(resourceSet *core_xds.ResourceSet, ctx xds_context.Context, proxy *core_xds.Proxy) error
}
