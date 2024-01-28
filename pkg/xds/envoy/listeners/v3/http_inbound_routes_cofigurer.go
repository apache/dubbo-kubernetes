package v3

import (
	core_xds "github.com/apache/dubbo-kubernetes/pkg/core/xds"
	envoy_common "github.com/apache/dubbo-kubernetes/pkg/xds/envoy"
	envoy_names "github.com/apache/dubbo-kubernetes/pkg/xds/envoy/names"
	envoy_routes "github.com/apache/dubbo-kubernetes/pkg/xds/envoy/routes"
	envoy_virtual_hosts "github.com/apache/dubbo-kubernetes/pkg/xds/envoy/virtualhosts"
	envoy_listener "github.com/envoyproxy/go-control-plane/envoy/config/listener/v3"
)

type HttpInboundRouteConfigurer struct {
	Service string
	Routes  envoy_common.Routes
}

var _ FilterChainConfigurer = &HttpInboundRouteConfigurer{}

func (c *HttpInboundRouteConfigurer) Configure(filterChain *envoy_listener.FilterChain) error {
	routeName := envoy_names.GetInboundRouteName(c.Service)

	static := HttpStaticRouteConfigurer{
		Builder: envoy_routes.NewRouteConfigurationBuilder(core_xds.APIVersion(envoy_common.APIV3), routeName).
			Configure(envoy_routes.CommonRouteConfiguration()).
			Configure(envoy_routes.ResetTagsHeader()).
			Configure(envoy_routes.VirtualHost(envoy_virtual_hosts.NewVirtualHostBuilder(core_xds.APIVersion(envoy_common.APIV3), c.Service).
				Configure(envoy_virtual_hosts.Routes(c.Routes)))),
	}

	return static.Configure(filterChain)
}
