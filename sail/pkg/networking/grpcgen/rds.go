package grpcgen

import (
	"fmt"
	"strconv"
	"strings"

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
	// For proxyless gRPC inbound routes, routeName is just the port number (e.g., "17070")
	// For outbound routes, routeName is cluster format (outbound|port||hostname)
	_, err := strconv.Atoi(routeName)
	if err != nil {
		// Try to parse as cluster naming format (outbound|port||hostname)
		_, _, hostname, parsedPort := model.ParseSubsetKey(routeName)
		if hostname == "" || parsedPort == 0 {
			klog.Warningf("failed to parse route name %v", routeName)
			return nil
		}

		// Build outbound route configuration for gRPC proxyless
		// This is used by ApiListener to route traffic to the correct cluster
		svc := push.ServiceForHostname(node, hostname)
		if svc == nil {
			klog.Warningf("buildHTTPRoute: service not found for hostname %s", hostname)
			return nil
		}

		// Build VirtualHost Domains for outbound route
		// CRITICAL: Domains must match the xDS URL hostname for gRPC xDS client
		hostStr := string(hostname)
		domains := []string{
			fmt.Sprintf("%s:%d", hostStr, parsedPort), // FQDN with port - MOST SPECIFIC
			hostStr, // Full FQDN
		}
		// Add short name if different from FQDN
		hostParts := strings.Split(hostStr, ".")
		if len(hostParts) > 0 && hostParts[0] != hostStr {
			shortName := hostParts[0]
			domains = append(domains, fmt.Sprintf("%s:%d", shortName, parsedPort)) // Short name with port
			domains = append(domains, shortName)                                   // Short name
		}
		domains = append(domains, "*") // Wildcard for any domain - LEAST SPECIFIC

		return &route.RouteConfiguration{
			Name: routeName,
			VirtualHosts: []*route.VirtualHost{
				{
					Name:    fmt.Sprintf("%s|http|%d", hostStr, parsedPort),
					Domains: domains,
					Routes: []*route.Route{
						{
							Match: &route.RouteMatch{
								PathSpecifier: &route.RouteMatch_Prefix{
									Prefix: "/",
								},
							},
							Action: &route.Route_Route{
								Route: &route.RouteAction{
									ClusterSpecifier: &route.RouteAction_Cluster{
										Cluster: routeName, // Use routeName (cluster name)
									},
								},
							},
						},
					},
				},
			},
		}
	}

	// Build minimal route configuration for proxyless gRPC inbound listener
	// NonForwardingAction indicates this is an inbound listener that should handle requests directly
	return &route.RouteConfiguration{
		Name: routeName,
		VirtualHosts: []*route.VirtualHost{
			{
				Name:    "inbound|http|" + routeName,
				Domains: []string{"*"},
				Routes: []*route.Route{
					{
						Match: &route.RouteMatch{
							PathSpecifier: &route.RouteMatch_Prefix{
								Prefix: "/",
							},
						},
						Action: &route.Route_NonForwardingAction{},
					},
				},
			},
		},
	}
}
