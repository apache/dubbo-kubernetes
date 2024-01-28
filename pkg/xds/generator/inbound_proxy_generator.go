package generator

import (
	"context"
	core_mesh "github.com/apache/dubbo-kubernetes/pkg/core/resources/apis/mesh"
	"github.com/apache/dubbo-kubernetes/pkg/core/validators"
	core_xds "github.com/apache/dubbo-kubernetes/pkg/core/xds"
	"github.com/apache/dubbo-kubernetes/pkg/util/net"
	xds_context "github.com/apache/dubbo-kubernetes/pkg/xds/context"
	envoy_common "github.com/apache/dubbo-kubernetes/pkg/xds/envoy"
	envoy_clusters "github.com/apache/dubbo-kubernetes/pkg/xds/envoy/clusters"
	envoy_listeners "github.com/apache/dubbo-kubernetes/pkg/xds/envoy/listeners"
	envoy_names "github.com/apache/dubbo-kubernetes/pkg/xds/envoy/names"

	"github.com/pkg/errors"
)

const OriginInbound = "inbound"

type InboundProxyGenerator struct{}

func (g InboundProxyGenerator) Generator(ctx context.Context, _ *core_xds.ResourceSet, xdsCtx xds_context.Context, proxy *core_xds.Proxy) (*core_xds.ResourceSet, error) {
	resources := core_xds.NewResourceSet()
	for i, endpoint := range proxy.Dataplane.Spec.Networking.GetInboundInterfaces() {
		// we do not create inbounds for serviceless
		if endpoint.IsServiceLess() {
			continue
		}

		iface := proxy.Dataplane.Spec.Networking.Inbound[i]
		protocol := core_mesh.ParseProtocol(iface.GetProtocol())

		// generate CDS resource
		localClusterName := envoy_names.GetLocalClusterName(endpoint.WorkloadPort)
		clusterBuilder := envoy_clusters.NewClusterBuilder(proxy.APIVersion, localClusterName).
			Configure(envoy_clusters.ProvidedEndpointCluster(false, core_xds.Endpoint{Target: endpoint.WorkloadIP, Port: endpoint.DataplanePort}))
		// localhost traffic is routed dirrectly to the application, in case of other interface we are going to set source address to
		// 127.0.0.6 to avoid redirections and thanks to first iptables rule just return fast
		if endpoint.WorkloadIP != core_mesh.IPv4Loopback.String() && endpoint.WorkloadIP != core_mesh.IPv6Loopback.String() {
			switch net.IsAddressIPv6(endpoint.WorkloadIP) {
			case true:
				clusterBuilder.Configure(envoy_clusters.UpstreamBindConfig(InPassThroughIPv6, 0))
			case false:
				clusterBuilder.Configure(envoy_clusters.UpstreamBindConfig(InPassThroughIPv4, 0))
			}
		}

		switch protocol {
		case core_mesh.ProtocolHTTP:
			clusterBuilder.Configure(envoy_clusters.Http())
		case core_mesh.ProtocolHTTP2, core_mesh.ProtocolGRPC:
			clusterBuilder.Configure(envoy_clusters.Http2())
		}
		envoyCluster, err := clusterBuilder.Build()
		if err != nil {
			return nil, errors.Wrapf(err, "%s: could not generate cluster %s", validators.RootedAt("dataplane").Field("networking").Field("inbound").Index(i), localClusterName)
		}
		resources.Add(&core_xds.Resource{
			Name:     localClusterName,
			Resource: envoyCluster,
			Origin:   OriginInbound,
		})

		cluster := envoy_common.NewCluster(envoy_common.WithService(localClusterName))
		routes := envoy_common.Routes{}

		// Add the default fall-back route
		routes = append(routes, envoy_common.NewRoute(envoy_common.WithCluster(cluster)))

		// generate LDS resource
		service := iface.GetService()
		inboundListenerName := envoy_names.GetInboundListenerName(endpoint.DataplaneIP, endpoint.DataplanePort)
		filterChainBuilder := func(serverSideMTLS bool) *envoy_listeners.FilterChainBuilder {
			filterChainBuilder := envoy_listeners.NewFilterChainBuilder(proxy.APIVersion, envoy_common.AnonymousResource)
			switch protocol {
			// configuration for HTTP case
			case core_mesh.ProtocolHTTP, core_mesh.ProtocolHTTP2:
				filterChainBuilder.
					Configure(envoy_listeners.HttpConnectionManager(localClusterName, true)).
					Configure(envoy_listeners.HttpInboundRoutes(service, routes))
			case core_mesh.ProtocolGRPC:
				filterChainBuilder.
					Configure(envoy_listeners.HttpConnectionManager(localClusterName, true)).
					Configure(envoy_listeners.GrpcStats()).
					Configure(envoy_listeners.HttpInboundRoutes(service, routes))
			case core_mesh.ProtocolKafka:
				filterChainBuilder.
					Configure(envoy_listeners.Kafka(localClusterName)).
					Configure(envoy_listeners.TcpProxyDeprecated(localClusterName, envoy_common.NewCluster(envoy_common.WithService(localClusterName))))
			case core_mesh.ProtocolTCP:
				fallthrough
			default:
				// configuration for non-HTTP cases
				filterChainBuilder.Configure(envoy_listeners.TcpProxyDeprecated(localClusterName, envoy_common.NewCluster(envoy_common.WithService(localClusterName))))
			}
			return filterChainBuilder
		}

		listenerBuilder := envoy_listeners.NewInboundListenerBuilder(proxy.APIVersion, endpoint.DataplaneIP, endpoint.DataplanePort, core_xds.SocketAddressProtocolTCP).
			Configure(envoy_listeners.TagsMetadata(iface.GetTags()))

		listenerBuilder.Configure(envoy_listeners.FilterChain(filterChainBuilder(false)))

		inboundListener, err := listenerBuilder.Build()
		if err != nil {
			return nil, errors.Wrapf(err, "%s: could not generate listener %s", validators.RootedAt("dataplane").Field("networking").Field("inbound").Index(i), inboundListenerName)
		}
		resources.Add(&core_xds.Resource{
			Name:     inboundListenerName,
			Resource: inboundListener,
			Origin:   OriginInbound,
		})
	}
	return resources, nil
}
