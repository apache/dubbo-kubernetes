package grpcgen

import (
	"github.com/apache/dubbo-kubernetes/sail/pkg/model"
	"github.com/apache/dubbo-kubernetes/sail/pkg/util/protoconv"
	route "github.com/envoyproxy/go-control-plane/envoy/config/route/v3"
	discovery "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v3"
	"k8s.io/klog/v2"
)

func (g *GrpcConfigGenerator) BuildHTTPRoutes(node *model.Proxy, push *model.PushContext, routeNames []string) model.Resources {
	resp := model.Resources{}
	for _, routeName := range routeNames {
		if rc := buildHTTPRoute(node, push, routeName); rc != nil {
			resp = append(resp, &discovery.Resource{
				Name:     routeName,
				Resource: protoconv.MessageToAny(rc),
			})
		}
	}
	return resp
}

func buildHTTPRoute(node *model.Proxy, push *model.PushContext, routeName string) *route.RouteConfiguration {
	// TODO use route-style naming instead of cluster naming
	_, _, hostname, port := model.ParseSubsetKey(routeName)
	if hostname == "" || port == 0 {
		klog.Warningf("failed to parse %v", routeName)
		return nil
	}

	// virtualHosts, _, _ := core.BuildSidecarOutboundVirtualHosts(node, push, routeName, port, nil, &model.DisabledCache{})

	// Only generate the required route for grpc. Will need to generate more
	// as GRPC adds more features.
	return &route.RouteConfiguration{
		Name:         routeName,
		VirtualHosts: nil,
	}
}
