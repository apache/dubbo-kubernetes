package virtualhosts

import (
	envoy_common "github.com/apache/dubbo-kubernetes/pkg/xds/envoy"
	envoy_config_route_v3 "github.com/envoyproxy/go-control-plane/envoy/config/route/v3"
)

type RoutesConfigurer struct {
	Routes envoy_common.Routes
}

func (c RoutesConfigurer) Configure(virtualHost *envoy_config_route_v3.VirtualHost) error {
	return nil
}
