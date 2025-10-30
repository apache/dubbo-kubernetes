package grpcgen

import (
	"github.com/apache/dubbo-kubernetes/sail/pkg/model"
	v3 "github.com/apache/dubbo-kubernetes/sail/pkg/xds/v3"
)

type GrpcConfigGenerator struct{}

func (g *GrpcConfigGenerator) Generate(proxy *model.Proxy, w *model.WatchedResource, req *model.PushRequest) (model.Resources, model.XdsLogDetails, error) {
	switch w.TypeUrl {
	case v3.ListenerType:
		return g.BuildListeners(proxy, req.Push, w.ResourceNames.UnsortedList()), model.DefaultXdsLogDetails, nil
	case v3.ClusterType:
		return g.BuildClusters(proxy, req.Push, w.ResourceNames.UnsortedList()), model.DefaultXdsLogDetails, nil
	case v3.RouteType:
		return g.BuildHTTPRoutes(proxy, req.Push, w.ResourceNames.UnsortedList()), model.DefaultXdsLogDetails, nil
	}

	return nil, model.DefaultXdsLogDetails, nil
}
