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
	"github.com/apache/dubbo-kubernetes/pkg/config"
	"github.com/apache/dubbo-kubernetes/pkg/config/host"
	route "github.com/envoyproxy/go-control-plane/envoy/config/route/v3"
	matcher "github.com/envoyproxy/go-control-plane/envoy/type/matcher/v3"
	"google.golang.org/protobuf/types/known/wrapperspb"
	networking "istio.io/api/networking/v1alpha3"
	sigsk8siogatewayapiapisv1 "sigs.k8s.io/gateway-api/apis/v1"
)

func (g *GrpcConfigGenerator) BuildHTTPRoutes(node *model.Proxy, req *model.PushRequest, routeNames []string) ([]*discovery.Resource, model.XdsLogDetails) {
	resp := []*discovery.Resource{}
	log.Infof("BuildHTTPRoutes: node=%s, isRouter=%v, routeNames=%v", node.ID, node.IsRouter(), routeNames)
	if len(routeNames) == 0 {
		log.Warnf("BuildHTTPRoutes: no routeNames requested for node=%s", node.ID)
	}
	for _, routeName := range routeNames {
		if rc := buildHTTPRoute(node, req.Push, routeName); rc != nil {
			log.Infof("BuildHTTPRoutes: built route config for routeName=%s, VirtualHosts=%d", routeName, len(rc.VirtualHosts))
			if len(rc.VirtualHosts) > 0 {
				log.Infof("BuildHTTPRoutes: VirtualHost[0] domains=%v, routes=%d", rc.VirtualHosts[0].Domains, len(rc.VirtualHosts[0].Routes))
			}
			resp = append(resp, &discovery.Resource{
				Name:     routeName,
				Resource: protoconv.MessageToAny(rc),
			})
		} else {
			log.Warnf("BuildHTTPRoutes: failed to build route config for routeName=%s", routeName)
		}
	}
	return resp, model.DefaultXdsLogDetails
}

func buildHTTPRoute(node *model.Proxy, push *model.PushContext, routeName string) *route.RouteConfiguration {
	log.Debugf("buildHTTPRoute: called with routeName=%s, node.ID=%s, node.Type=%v, node.IsRouter()=%v", routeName, node.ID, node.Type, node.IsRouter())
	// For Gateway Pod inbound listeners, routeName is just the port number (e.g., "80")
	// For proxyless gRPC inbound routes, routeName is just the port number (e.g., "17070")
	// For outbound routes, routeName is cluster format (outbound|port||hostname)
	parsedPort, err := strconv.Atoi(routeName)
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
		// Domains must match the xDS URL hostname for gRPC xDS client
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

		// First try Gateway API HTTPRoute
		// For Gateway Pods, we need to match HTTPRoute hostnames
		// Since Gateway Pods receive traffic with arbitrary hostnames (from HTTP requests),
		// we need to use wildcard "*" to match all HTTPRoutes
		// The VirtualHost domains already include "*" which will match any hostname
		httpRoutes := push.HTTPRouteForHost(host.Name("*"))
		if len(httpRoutes) > 0 {
			log.Infof("buildHTTPRoute: found %d HTTPRoute(s) for Gateway Pod (using wildcard match)", len(httpRoutes))
			// For Gateway Pods, collect all HTTPRoute hostnames and add them to domains
			httpRouteHostnames := make(map[string]bool)
			for _, hr := range httpRoutes {
				hrSpec, ok := hr.Spec.(*sigsk8siogatewayapiapisv1.HTTPRouteSpec)
				if !ok {
					continue
				}
				if len(hrSpec.Hostnames) == 0 {
					// No hostnames means match all
					httpRouteHostnames["*"] = true
				} else {
					for _, hostname := range hrSpec.Hostnames {
						hostnameStr := string(hostname)
						if hostnameStr == "" || hostnameStr == "*" {
							httpRouteHostnames["*"] = true
						} else {
							httpRouteHostnames[hostnameStr] = true
						}
					}
				}
			}
			// Add HTTPRoute hostnames to domains (they will be matched by Envoy)
			for hostnameStr := range httpRouteHostnames {
				if hostnameStr != "*" {
					domains = append(domains, hostnameStr)
					domains = append(domains, fmt.Sprintf("%s:%d", hostnameStr, parsedPort))
				}
			}

			if routes := buildRoutesFromGatewayHTTPRoute(httpRoutes, host.Name("*"), parsedPort); len(routes) > 0 {
				log.Infof("buildHTTPRoute: built %d routes from Gateway API HTTPRoute", len(routes))
				outboundRoutes = routes
			} else {
				log.Warnf("buildHTTPRoute: HTTPRoute found but no routes built")
			}
		} else if vs := push.ServiceRouteForHost(host.Name(hostStr)); vs != nil {
			// Fallback to ServiceRoute if no HTTPRoute found
			log.Infof("buildHTTPRoute: found ServiceRoute for host %s with %d HTTP routes", hostStr, len(vs.Http))
			if routes := buildRoutesFromServiceRoute(vs, host.Name(hostStr), parsedPort); len(routes) > 0 {
				log.Infof("buildHTTPRoute: built %d weighted routes from ServiceRoute for host %s", len(routes), hostStr)
				outboundRoutes = routes
			} else {
				log.Warnf("buildHTTPRoute: ServiceRoute found but no routes built for host %s", hostStr)
			}
		} else {
			log.Debugf("buildHTTPRoute: no HTTPRoute or ServiceRoute found for host %s, using default route", hostStr)
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

	// Build route configuration for inbound listener
	// For Gateway Pods (router type), this should route traffic based on HTTPRoute rules
	// For regular service Pods, use NonForwardingAction to handle requests directly
	// Also check if this is a Gateway Pod by checking service name (fallback for when node.Type is not Router)
	isGatewayPod := false
	var gatewayName, gatewayNamespace string
	// Try to find Gateway Pod's service and extract Gateway name from labels
	for _, st := range node.ServiceTargets {
		if strings.Contains(strings.ToLower(st.Service.Attributes.Name), "gateway") {
			isGatewayPod = true
			// Try to get Gateway name from service labels
			if len(st.Service.Attributes.Labels) > 0 {
				if name, ok := st.Service.Attributes.Labels["gateway.networking.k8s.io/gateway-name"]; ok {
					gatewayName = name
					gatewayNamespace = st.Service.Attributes.Namespace
					break
				}
			}
		}
	}
	if node.IsRouter() || isGatewayPod {
		if isGatewayPod && !node.IsRouter() {
			log.Warnf("buildHTTPRoute: Gateway Pod detected but node.Type is not Router (node.Type=%v, node.ID=%s), treating as router anyway", node.Type, node.ID)
		}
		log.Infof("buildHTTPRoute: Gateway Pod inbound listener, routeName=%s, port=%d, gateway=%s/%s", routeName, parsedPort, gatewayNamespace, gatewayName)

		// CRITICAL: Only apply HTTPRoute to the Gateway listener port (80)
		// Other ports (15012, 15021, etc.) are service ports and should not use HTTPRoute
		if parsedPort != 80 {
			log.Debugf("buildHTTPRoute: Gateway Pod inbound listener port %d is not 80, skipping HTTPRoute (this is a service port, not Gateway listener)", parsedPort)
			// Return empty route config for non-Gateway ports
			return &route.RouteConfiguration{
				Name: routeName,
				VirtualHosts: []*route.VirtualHost{
					{
						Name:    "inbound|http|" + routeName,
						Domains: []string{"*"},
						Routes:  []*route.Route{}, // Empty routes for service ports
					},
				},
			}
		}

		// Gateway Pod inbound listener: route external traffic based on HTTPRoute
		domains := []string{"*"}           // Match all hostnames by default
		outboundRoutes := []*route.Route{} // Don't use fallback route, only use HTTPRoute routes

		// Try to find HTTPRoutes for Gateway Pod
		// Gateway Pods receive traffic with arbitrary hostnames, so we need to collect all HTTPRoutes
		// For Gateway Pod inbound listener, we need ALL HTTPRoutes that could route traffic
		// First try wildcard match to get HTTPRoutes with no hostnames or wildcard hostnames
		allHTTPRoutes := push.HTTPRouteForHost(host.Name("*"))
		log.Debugf("buildHTTPRoute: Gateway Pod inbound listener, found %d HTTPRoute(s) with wildcard match", len(allHTTPRoutes))

		// Filter HTTPRoutes by parentRef to match this Gateway
		httpRoutes := filterHTTPRoutesByGateway(allHTTPRoutes, gatewayName, gatewayNamespace, parsedPort)
		log.Debugf("buildHTTPRoute: Gateway Pod inbound listener, filtered to %d HTTPRoute(s) matching gateway %s/%s port %d", len(httpRoutes), gatewayNamespace, gatewayName, parsedPort)

		// For Gateway Pod, we also need to collect HTTPRoutes with specific hostnames
		// because Gateway Pods route traffic based on HTTPRoute hostnames in the request
		// We'll add all HTTPRoutes to the domains list so Envoy can match them
		if len(httpRoutes) > 0 {
			log.Infof("buildHTTPRoute: Gateway Pod inbound listener found %d HTTPRoute(s) for port %s", len(httpRoutes), routeName)
			// Collect all HTTPRoute hostnames and add them to domains
			httpRouteHostnames := make(map[string]bool)
			for _, hr := range httpRoutes {
				hrSpec, ok := hr.Spec.(*sigsk8siogatewayapiapisv1.HTTPRouteSpec)
				if !ok {
					continue
				}
				if len(hrSpec.Hostnames) == 0 {
					httpRouteHostnames["*"] = true
				} else {
					for _, hostname := range hrSpec.Hostnames {
						hostnameStr := string(hostname)
						if hostnameStr == "" || hostnameStr == "*" {
							httpRouteHostnames["*"] = true
						} else {
							httpRouteHostnames[hostnameStr] = true
						}
					}
				}
			}
			// Add HTTPRoute hostnames to domains
			for hostnameStr := range httpRouteHostnames {
				if hostnameStr != "*" {
					domains = append(domains, hostnameStr)
					domains = append(domains, fmt.Sprintf("%s:%d", hostnameStr, parsedPort))
				}
			}

			if routes := buildRoutesFromGatewayHTTPRoute(httpRoutes, host.Name("*"), parsedPort); len(routes) > 0 {
				log.Infof("buildHTTPRoute: Gateway Pod inbound listener built %d routes from HTTPRoute", len(routes))
				outboundRoutes = routes
			} else {
				log.Warnf("buildHTTPRoute: Gateway Pod inbound listener HTTPRoute found but no routes built")
			}
		} else {
			log.Warnf("buildHTTPRoute: Gateway Pod inbound listener no HTTPRoute found for port %s, returning empty route config", routeName)
			// Return empty route config - Gateway Pod must have HTTPRoute to route traffic
			return &route.RouteConfiguration{
				Name: routeName,
				VirtualHosts: []*route.VirtualHost{
					{
						Name:    "inbound|http|" + routeName,
						Domains: []string{"*"},
						Routes:  []*route.Route{}, // Empty routes - no HTTPRoute found
					},
				},
			}
		}

		log.Infof("buildHTTPRoute: Gateway Pod inbound listener returning route config with %d domains, %d routes", len(domains), len(outboundRoutes))
		return &route.RouteConfiguration{
			Name: routeName,
			VirtualHosts: []*route.VirtualHost{
				{
					Name:    "inbound|http|" + routeName,
					Domains: domains,
					Routes:  outboundRoutes,
				},
			},
		}
	}

	// Regular service Pod inbound listener: NonForwardingAction indicates this is an inbound listener that should handle requests directly
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

// buildRoutesFromGatewayHTTPRoute converts Gateway API HTTPRoute resources to XDS Route configurations
func buildRoutesFromGatewayHTTPRoute(httpRoutes []config.Config, hostName host.Name, defaultPort int) []*route.Route {
	if len(httpRoutes) == 0 {
		return nil
	}

	var allRoutes []*route.Route
	for _, hrConfig := range httpRoutes {
		hrSpec, ok := hrConfig.Spec.(*sigsk8siogatewayapiapisv1.HTTPRouteSpec)
		if !ok {
			log.Warnf("buildRoutesFromGatewayHTTPRoute: HTTPRoute %s/%s spec is not HTTPRouteSpec", hrConfig.Namespace, hrConfig.Name)
			continue
		}

		// Process each rule in the HTTPRoute
		for ruleIdx, rule := range hrSpec.Rules {
			if len(rule.BackendRefs) == 0 {
				log.Debugf("buildRoutesFromGatewayHTTPRoute: HTTPRoute %s/%s rule[%d] has no backendRefs, skipping", hrConfig.Namespace, hrConfig.Name, ruleIdx)
				continue
			}

			// Build weighted clusters from backendRefs
			weights := make([]*route.WeightedCluster_ClusterWeight, 0, len(rule.BackendRefs))
			var totalWeight uint32

			for backendIdx, backendRef := range rule.BackendRefs {
				// Get backend service name and namespace
				backendName := string(backendRef.Name)
				backendNamespace := hrConfig.Namespace
				if backendRef.Namespace != nil {
					backendNamespace = string(*backendRef.Namespace)
				}

				// Get backend port
				backendPort := defaultPort
				if backendRef.Port != nil {
					backendPort = int(*backendRef.Port)
				}

				// Build service FQDN
				backendHost := fmt.Sprintf("%s.%s.svc.cluster.local", backendName, backendNamespace)
				clusterName := model.BuildSubsetKey(model.TrafficDirectionOutbound, "", host.Name(backendHost), backendPort)

				// Get weight (default to 1 if not specified)
				weight := uint32(1)
				if backendRef.Weight != nil {
					weight = uint32(*backendRef.Weight)
				}
				if weight == 0 {
					weight = 1
				}
				totalWeight += weight

				log.Debugf("buildRoutesFromGatewayHTTPRoute: HTTPRoute %s/%s rule[%d] backend[%d] -> cluster=%s, weight=%d, host=%s, port=%d",
					hrConfig.Namespace, hrConfig.Name, ruleIdx, backendIdx, clusterName, weight, backendHost, backendPort)

				weights = append(weights, &route.WeightedCluster_ClusterWeight{
					Name:   clusterName,
					Weight: wrapperspb.UInt32(weight),
				})
			}

			if len(weights) == 0 {
				log.Warnf("buildRoutesFromGatewayHTTPRoute: HTTPRoute %s/%s rule[%d] has no valid backends", hrConfig.Namespace, hrConfig.Name, ruleIdx)
				continue
			}

			weightedClusters := &route.WeightedCluster{
				Clusters: weights,
			}
			if totalWeight > 0 {
				weightedClusters.TotalWeight = wrapperspb.UInt32(totalWeight)
			}

			// Build route match from HTTPRoute matches
			routeMatch := buildRouteMatchFromHTTPRouteMatches(rule.Matches)

			builtRoute := &route.Route{
				Match: routeMatch,
				Action: &route.Route_Route{
					Route: &route.RouteAction{
						ClusterSpecifier: &route.RouteAction_WeightedClusters{
							WeightedClusters: weightedClusters,
						},
					},
				},
			}

			log.Infof("buildRoutesFromGatewayHTTPRoute: HTTPRoute %s/%s rule[%d] -> built route with %d clusters, totalWeight=%d",
				hrConfig.Namespace, hrConfig.Name, ruleIdx, len(weights), totalWeight)
			allRoutes = append(allRoutes, builtRoute)
		}
	}

	return allRoutes
}

// filterHTTPRoutesByGateway filters HTTPRoutes by parentRef to match the given Gateway
func filterHTTPRoutesByGateway(httpRoutes []config.Config, gatewayName, gatewayNamespace string, port int) []config.Config {
	if gatewayName == "" {
		// If we can't determine the Gateway name, return all HTTPRoutes
		// This is a fallback for when Gateway Pod doesn't have proper labels
		log.Warnf("filterHTTPRoutesByGateway: gateway name is empty, returning all HTTPRoutes")
		return httpRoutes
	}

	var filtered []config.Config
	for _, hr := range httpRoutes {
		hrSpec, ok := hr.Spec.(*sigsk8siogatewayapiapisv1.HTTPRouteSpec)
		if !ok {
			continue
		}

		// Check if any parentRef matches this Gateway
		matches := false
		for _, parentRef := range hrSpec.ParentRefs {
			refName := string(parentRef.Name)
			refNamespace := hr.Namespace // Default to HTTPRoute namespace
			if parentRef.Namespace != nil {
				refNamespace = string(*parentRef.Namespace)
			}

			// Check if parentRef matches Gateway name and namespace
			if refName == gatewayName && refNamespace == gatewayNamespace {
				// Check if parentRef has a section name (listener name)
				// If section name is specified, we should match it, but for now we accept all listeners
				// TODO: Match listener name if specified
				if parentRef.SectionName != nil {
					log.Debugf("filterHTTPRoutesByGateway: HTTPRoute %s/%s matches Gateway %s/%s with listener %s",
						hr.Namespace, hr.Name, gatewayNamespace, gatewayName, *parentRef.SectionName)
				}
				matches = true
				break
			}
		}

		if matches {
			filtered = append(filtered, hr)
			log.Debugf("filterHTTPRoutesByGateway: HTTPRoute %s/%s matches Gateway %s/%s",
				hr.Namespace, hr.Name, gatewayNamespace, gatewayName)
		}
	}

	return filtered
}

// buildRouteMatchFromHTTPRouteMatches converts Gateway API HTTPRouteMatch to XDS RouteMatch
func buildRouteMatchFromHTTPRouteMatches(matches []sigsk8siogatewayapiapisv1.HTTPRouteMatch) *route.RouteMatch {
	if len(matches) == 0 {
		// No matches means match all
		return &route.RouteMatch{
			PathSpecifier: &route.RouteMatch_Prefix{
				Prefix: "/",
			},
		}
	}

	// For now, we'll use the first match. In a full implementation, we might need to merge multiple matches.
	match := matches[0]
	routeMatch := &route.RouteMatch{}

	// Handle path match
	if match.Path != nil {
		pathType := match.Path.Type
		pathValue := match.Path.Value
		if pathValue == nil {
			pathValue = ptr("")
		}

		// pathType is a pointer, need to dereference it
		if pathType != nil {
			switch *pathType {
			case sigsk8siogatewayapiapisv1.PathMatchExact:
				routeMatch.PathSpecifier = &route.RouteMatch_Path{
					Path: *pathValue,
				}
			case sigsk8siogatewayapiapisv1.PathMatchPathPrefix:
				routeMatch.PathSpecifier = &route.RouteMatch_Prefix{
					Prefix: *pathValue,
				}
			case sigsk8siogatewayapiapisv1.PathMatchRegularExpression:
				routeMatch.PathSpecifier = &route.RouteMatch_SafeRegex{
					SafeRegex: &matcher.RegexMatcher{
						Regex: *pathValue,
					},
				}
			default:
				// Default to prefix match
				routeMatch.PathSpecifier = &route.RouteMatch_Prefix{
					Prefix: "/",
				}
			}
		} else {
			// No path type specified, default to prefix match
			routeMatch.PathSpecifier = &route.RouteMatch_Prefix{
				Prefix: "/",
			}
		}
	} else {
		// No path match means match all paths
		routeMatch.PathSpecifier = &route.RouteMatch_Prefix{
			Prefix: "/",
		}
	}

	// Handle header matches (if any)
	if len(match.Headers) > 0 {
		headerMatchers := make([]*route.HeaderMatcher, 0, len(match.Headers))
		for _, headerMatch := range match.Headers {
			headerMatcher := &route.HeaderMatcher{
				Name: string(headerMatch.Name),
			}

			if headerMatch.Type != nil {
				switch *headerMatch.Type {
				case sigsk8siogatewayapiapisv1.HeaderMatchExact:
					if headerMatch.Value != "" {
						headerMatcher.HeaderMatchSpecifier = &route.HeaderMatcher_ExactMatch{
							ExactMatch: headerMatch.Value,
						}
					}
				case sigsk8siogatewayapiapisv1.HeaderMatchRegularExpression:
					if headerMatch.Value != "" {
						headerMatcher.HeaderMatchSpecifier = &route.HeaderMatcher_SafeRegexMatch{
							SafeRegexMatch: &matcher.RegexMatcher{
								Regex: headerMatch.Value,
							},
						}
					}
				}
			}

			if headerMatcher.HeaderMatchSpecifier != nil {
				headerMatchers = append(headerMatchers, headerMatcher)
			}
		}
		if len(headerMatchers) > 0 {
			routeMatch.Headers = headerMatchers
		}
	}

	return routeMatch
}

func ptr(s string) *string {
	return &s
}
