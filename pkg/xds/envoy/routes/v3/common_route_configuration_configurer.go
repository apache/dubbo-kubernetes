package v3

import (
	util_proto "github.com/apache/dubbo-kubernetes/pkg/util/proto"
	envoy_config_route_v3 "github.com/envoyproxy/go-control-plane/envoy/config/route/v3"
)

type CommonRouteConfigurationConfigurer struct{}

func (c CommonRouteConfigurationConfigurer) Configure(routeConfiguration *envoy_config_route_v3.RouteConfiguration) error {
	routeConfiguration.ValidateClusters = util_proto.Bool(false)
	return nil
}
