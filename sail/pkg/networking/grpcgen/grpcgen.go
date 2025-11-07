package grpcgen

import (
	"github.com/apache/dubbo-kubernetes/sail/pkg/model"
	v3 "github.com/apache/dubbo-kubernetes/sail/pkg/xds/v3"
)

type GrpcConfigGenerator struct{}

func (g *GrpcConfigGenerator) Generate(proxy *model.Proxy, w *model.WatchedResource, req *model.PushRequest) (model.Resources, model.XdsLogDetails, error) {
	// Extract requested resource names from WatchedResource
	// If ResourceNames is empty (wildcard request), pass empty slice
	// BuildListeners will handle empty names as wildcard request and generate all listeners
	var requestedNames []string
	if w != nil && w.ResourceNames != nil && len(w.ResourceNames) > 0 {
		requestedNames = w.ResourceNames.UnsortedList()
	}

	switch w.TypeUrl {
	case v3.ListenerType:
		// Pass requested names to BuildListeners to ensure consistent behavior
		// This is critical for gRPC proxyless clients to avoid resource count oscillation
		// When requestedNames is empty (wildcard), BuildListeners generates all listeners
		// When requestedNames is non-empty, BuildListeners only generates requested listeners
		return g.BuildListeners(proxy, req.Push, requestedNames), model.DefaultXdsLogDetails, nil
	case v3.ClusterType:
		return g.BuildClusters(proxy, req.Push, requestedNames), model.DefaultXdsLogDetails, nil
	case v3.RouteType:
		return g.BuildHTTPRoutes(proxy, req.Push, requestedNames), model.DefaultXdsLogDetails, nil
	}

	return nil, model.DefaultXdsLogDetails, nil
}
