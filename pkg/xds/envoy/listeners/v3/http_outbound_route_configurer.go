package v3

import (
	mesh_proto "github.com/apache/dubbo-kubernetes/api/mesh/v1alpha1"
	core_xds "github.com/apache/dubbo-kubernetes/pkg/core/xds"
	envoy_common "github.com/apache/dubbo-kubernetes/pkg/xds/envoy"
	envoy_names "github.com/apache/dubbo-kubernetes/pkg/xds/envoy/names"
	envoy_routes "github.com/apache/dubbo-kubernetes/pkg/xds/envoy/routes"
	envoy_virtual_hosts "github.com/apache/dubbo-kubernetes/pkg/xds/envoy/virtualhosts"
	envoy_listener "github.com/envoyproxy/go-control-plane/envoy/config/listener/v3"
)

type HttpOutboundRouteConfigurer struct {
	Service string
	Routes  envoy_common.Routes
	DpTags  mesh_proto.MultiValueTagSet
}

var _ FilterChainConfigurer = &HttpOutboundRouteConfigurer{}

func (c *HttpOutboundRouteConfigurer) Configure(filterChain *envoy_listener.FilterChain) error {
	static := HttpStaticRouteConfigurer{
		Builder: envoy_routes.NewRouteConfigurationBuilder(core_xds.APIVersion(envoy_common.APIV3), envoy_names.GetOutboundRouteName(c.Service)).
			Configure(envoy_routes.CommonRouteConfiguration()).
			Configure(envoy_routes.TagsHeader(c.DpTags)).
			Configure(envoy_routes.VirtualHost(envoy_virtual_hosts.NewVirtualHostBuilder(core_xds.APIVersion(envoy_common.APIV3), c.Service).
				Configure(envoy_virtual_hosts.Routes(c.Routes)))),
	}

	return static.Configure(filterChain)
}
