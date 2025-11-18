/*
 * Licensed to the Apache Software Foundation (ASF) under one or more
 * contributor license agreements.  See the NOTICE file distributed with
 * this work for additional information regarding copyright ownership.
 * The ASF licenses this file to You under the Apache License, Version 2.0
 * (the "License"); you may not use this file except in compliance with
 * the License.  You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package grpcgen

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/apache/dubbo-kubernetes/dubbod/planet/pkg/util/protoconv"
	discovery "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v3"

	"github.com/apache/dubbo-kubernetes/dubbod/planet/pkg/model"
	"github.com/apache/dubbo-kubernetes/pkg/config/host"
	route "github.com/envoyproxy/go-control-plane/envoy/config/route/v3"
	"google.golang.org/protobuf/types/known/wrapperspb"
	networking "istio.io/api/networking/v1alpha3"
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
			log.Warnf("failed to parse route name %v", routeName)
			return nil
		}

		// Build outbound route configuration for gRPC proxyless
		// This is used by ApiListener to route traffic to the correct cluster
		svc := push.ServiceForHostname(node, hostname)
		if svc == nil {
			log.Warnf("buildHTTPRoute: service not found for hostname %s", hostname)
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

		outboundRoutes := []*route.Route{
			defaultSingleClusterRoute(routeName),
		}
		if vs := push.ServiceRouteForHost(host.Name(hostStr)); vs != nil {
			log.Infof("buildHTTPRoute: found ServiceRoute for host %s with %d HTTP routes", hostStr, len(vs.Http))
			if routes := buildRoutesFromServiceRoute(vs, host.Name(hostStr), parsedPort); len(routes) > 0 {
				log.Infof("buildHTTPRoute: built %d weighted routes from ServiceRoute for host %s", len(routes), hostStr)
				outboundRoutes = routes
			} else {
				log.Warnf("buildHTTPRoute: ServiceRoute found but no routes built for host %s", hostStr)
			}
		} else {
			log.Debugf("buildHTTPRoute: no ServiceRoute found for host %s, using default route", hostStr)
		}

		return &route.RouteConfiguration{
			Name: routeName,
			VirtualHosts: []*route.VirtualHost{
				{
					Name:    fmt.Sprintf("%s|http|%d", hostStr, parsedPort),
					Domains: domains,
					Routes:  outboundRoutes,
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

func defaultSingleClusterRoute(clusterName string) *route.Route {
	return &route.Route{
		Match: &route.RouteMatch{
			PathSpecifier: &route.RouteMatch_Prefix{
				Prefix: "/",
			},
		},
		Action: &route.Route_Route{
			Route: &route.RouteAction{
				ClusterSpecifier: &route.RouteAction_Cluster{
					Cluster: clusterName,
				},
			},
		},
	}
}

func buildRoutesFromServiceRoute(vs *networking.VirtualService, hostName host.Name, defaultPort int) []*route.Route {
	if vs == nil || len(vs.Http) == 0 {
		return nil
	}
	var routes []*route.Route
	for _, httpRoute := range vs.Http {
		if httpRoute == nil {
			continue
		}
		if built := buildRouteFromHTTPRoute(httpRoute, hostName, defaultPort); built != nil {
			routes = append(routes, built)
		}
	}
	return routes
}

func buildRouteFromHTTPRoute(httpRoute *networking.HTTPRoute, hostName host.Name, defaultPort int) *route.Route {
	if httpRoute == nil || len(httpRoute.Route) == 0 {
		log.Warnf("buildRouteFromHTTPRoute: httpRoute is nil or has no routes")
		return nil
	}
	log.Infof("buildRouteFromHTTPRoute: processing HTTPRoute with %d route destinations", len(httpRoute.Route))
	weights := make([]*route.WeightedCluster_ClusterWeight, 0, len(httpRoute.Route))
	var totalWeight uint32
	for i, dest := range httpRoute.Route {
		if dest == nil {
			log.Warnf("buildRouteFromHTTPRoute: route[%d] is nil", i)
			continue
		}
		destination := dest.Destination
		if destination == nil {
			log.Warnf("buildRouteFromHTTPRoute: route[%d] has nil Destination (weight=%d), creating default destination with host=%s",
				i, dest.Weight, hostName)
			destination = &networking.Destination{
				Host: string(hostName),
			}
		} else {
			log.Debugf("buildRouteFromHTTPRoute: route[%d] Destination: host=%s, subset=%s, port=%v, weight=%d",
				i, destination.Host, destination.Subset, destination.Port, dest.Weight)
		}
		targetHost := destination.Host
		if targetHost == "" {
			targetHost = string(hostName)
		}
		targetPort := defaultPort
		if destination.Port != nil && destination.Port.Number != 0 {
			targetPort = int(destination.Port.Number)
		}
		subsetName := destination.Subset
		clusterName := model.BuildSubsetKey(model.TrafficDirectionOutbound, subsetName, host.Name(targetHost), targetPort)
		weight := dest.Weight
		if weight <= 0 {
			weight = 1
		}
		totalWeight += uint32(weight)
		log.Infof("buildRouteFromHTTPRoute: route[%d] -> cluster=%s, subset=%s, weight=%d, host=%s, port=%d",
			i, clusterName, subsetName, weight, targetHost, targetPort)
		weights = append(weights, &route.WeightedCluster_ClusterWeight{
			Name:   clusterName,
			Weight: wrapperspb.UInt32(uint32(weight)),
		})
	}
	if len(weights) == 0 {
		log.Warnf("buildRouteFromHTTPRoute: no valid weights generated")
		return nil
	}
	weightedClusters := &route.WeightedCluster{
		Clusters: weights,
	}
	if totalWeight > 0 {
		weightedClusters.TotalWeight = wrapperspb.UInt32(totalWeight)
	}
	log.Infof("buildRouteFromHTTPRoute: built WeightedCluster with %d clusters, totalWeight=%d", len(weights), totalWeight)

	return &route.Route{
		Match: &route.RouteMatch{
			PathSpecifier: &route.RouteMatch_Prefix{
				Prefix: "/",
			},
		},
		Action: &route.Route_Route{
			Route: &route.RouteAction{
				ClusterSpecifier: &route.RouteAction_WeightedClusters{
					WeightedClusters: weightedClusters,
				},
			},
		},
	}
}
