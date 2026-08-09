//
// Licensed to the Apache Software Foundation (ASF) under one or more
// contributor license agreements.  See the NOTICE file distributed with
// this work for additional information regarding copyright ownership.
// The ASF licenses this file to You under the Apache License, Version 2.0
// (the "License"); you may not use this file except in compliance with
// the License.  You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package grpcgen

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/apache/dubbo-kubernetes/dubbod/discovery/pkg/util/protoconv"
	discovery "github.com/kdubbo/xds-api/service/discovery/v1"

	"github.com/apache/dubbo-kubernetes/dubbod/discovery/pkg/model"
	"github.com/apache/dubbo-kubernetes/pkg/config"
	"github.com/apache/dubbo-kubernetes/pkg/config/host"
	"github.com/apache/dubbo-kubernetes/pkg/util/sets"
	route "github.com/kdubbo/xds-api/route/v1"
	matcher "github.com/kdubbo/xds-api/type/matcher/v1"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/wrapperspb"
	sigsk8siogatewayapiapisv1 "sigs.k8s.io/gateway-api/apis/v1"
)

func (g *GrpcConfigGenerator) BuildHTTPRoutes(node *model.Proxy, req *model.PushRequest, routeNames []string) ([]*discovery.Resource, model.XdsLogDetails) {
	resp := []*discovery.Resource{}
	log.Debugf("node=%s, isRouter=%v, routeNames=%v", node.ID, node.IsRouter(), routeNames)
	if len(routeNames) == 0 {
		log.Warnf("no routeNames requested for node=%s", node.ID)
	}
	for _, routeName := range routeNames {
		if rc := buildHTTPRoute(node, req.Push, routeName); rc != nil {
			log.Debugf("built route config for routeName=%s, VirtualHosts=%d", routeName, len(rc.VirtualHosts))
			if len(rc.VirtualHosts) > 0 {
				log.Debugf("VirtualHost[0] domains=%v, routes=%d", rc.VirtualHosts[0].Domains, len(rc.VirtualHosts[0].Routes))
			}
			resp = append(resp, &discovery.Resource{
				Name:     routeName,
				Resource: protoconv.MessageToAny(rc),
			})
		} else {
			log.Warnf("failed to build route config for routeName=%s", routeName)
		}
	}
	return resp, model.DefaultXdsLogDetails
}

func buildHTTPRoute(node *model.Proxy, push *model.PushContext, routeName string) *route.RouteConfiguration {
	log.Debugf("called with routeName=%s, node.ID=%s, node.Type=%v, node.IsRouter()=%v", routeName, node.ID, node.Type, node.IsRouter())
	// For Gateway Pod inbound listeners, routeName is just the port number (e.g., "80")
	// For Inherent gRPC inbound routes, routeName is just the port number (e.g., "17070")
	// For outbound routes, routeName is cluster format (outbound|port||hostname)
	parsedPort, err := strconv.Atoi(routeName)
	if err != nil {
		// Try to parse as cluster naming format (outbound|port||hostname)
		_, _, hostname, parsedPort := model.ParseSubsetKey(routeName)
		if hostname == "" || parsedPort == 0 {
			log.Warnf("failed to parse route name %v", routeName)
			return nil
		}

		// Build outbound route configuration for gRPC inherent
		// This is used by ApiListener to route traffic to the correct cluster
		svc := push.ServiceForHostname(node, hostname)
		if svc == nil {
			log.Warnf("service not found for hostname %s", hostname)
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

		faultPolicy := serviceFaultPolicy(push, svc, parsedPort)
		outboundRoutes := []*route.Route{
			defaultSingleClusterRoute(routeName, faultPolicy),
		}

		if node.IsRouter() {
			// Gateway routers use Gateway API HTTPRoute; service-to-service Inherent clients
			// must not let an unrelated north-south HTTPRoute shadow their service route.
			httpRoutes := push.HTTPRouteForHost(host.Name("*"))
			if len(httpRoutes) == 0 {
				log.Debugf("no HTTPRoute found for router host %s, using default route", hostStr)
			} else {
				log.Infof("found %d HTTPRoute(s) for Gateway Pod (using wildcard match)", len(httpRoutes))
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
				for hostnameStr := range httpRouteHostnames {
					if hostnameStr != "*" {
						domains = append(domains, hostnameStr)
						domains = append(domains, fmt.Sprintf("%s:%d", hostnameStr, parsedPort))
					}
				}

				if routes := buildRoutesFromGatewayHTTPRoute(httpRoutes, host.Name("*"), parsedPort, faultPolicy); len(routes) > 0 {
					log.Infof("built %d routes from Gateway API HTTPRoute", len(routes))
					outboundRoutes = routes
				} else {
					log.Warnf("HTTPRoute found but no routes built")
				}
			}
		} else if httpRoutes := filterHTTPRoutesByService(push.HTTPRouteForHost(host.Name(hostStr)), svc, parsedPort); len(httpRoutes) > 0 {
			log.Infof("found %d service-attached HTTPRoute(s) for host %s", len(httpRoutes), hostStr)
			if routes := buildRoutesFromGatewayHTTPRoute(httpRoutes, host.Name(hostStr), parsedPort, faultPolicy); len(routes) > 0 {
				log.Infof("built %d routes from service-attached HTTPRoute for host %s", len(routes), hostStr)
				outboundRoutes = routes
			} else {
				log.Warnf("service-attached HTTPRoute found but no routes built for host %s", hostStr)
			}
		} else {
			log.Debugf("no service-attached HTTPRoute found for host %s, using default route", hostStr)
		}

		virtualHosts := []*route.VirtualHost{
			{
				Name:    fmt.Sprintf("%s|http|%d", hostStr, parsedPort),
				Domains: domains,
				Routes:  outboundRoutes,
			},
		}
		if node.IsRouter() && svc.Attributes.Name == model.ActivationGatewayServiceName {
			virtualHosts = appendNonConflictingVirtualHosts(
				buildActivationVirtualHosts(push, svc.Attributes.Namespace),
				virtualHosts,
			)
		}
		return &route.RouteConfiguration{Name: routeName, VirtualHosts: virtualHosts}
	}

	// Build route configuration for inbound listener
	// For Gateway Pods (router type), this should route traffic based on HTTPRoute rules
	// For regular service Pods, use NonForwardingAction to handle requests directly
	// Also check if this is a Gateway Pod by checking service name (fallback for when node.Type is not Router)
	isGatewayPod := false
	isGatewayDataPort := parsedPort == 80
	gatewayListenerPort := parsedPort
	var gatewayName, gatewayNamespace string
	// Resolve the listener's Service port from the generated Gateway Service.
	// dxgate binds the Service targetPort (15080 by default), while Gateway API
	// parentRefs and HTTPRoutes refer to the public listener port (for example 80).
	for _, st := range node.ServiceTargets {
		if st.Service == nil {
			continue
		}
		if name, ok := st.Service.Attributes.Labels["gateway.networking.k8s.io/gateway-name"]; ok {
			isGatewayPod = true
			if st.Port.ServicePort != nil && st.Port.TargetPort == uint32(parsedPort) {
				gatewayName = name
				gatewayNamespace = st.Service.Attributes.Namespace
				gatewayListenerPort = st.Port.Port
				isGatewayDataPort = true
				break
			}
		}
	}
	if node.IsRouter() || isGatewayPod {
		if isGatewayPod && !node.IsRouter() {
			log.Warnf("Gateway Pod detected but node.Type is not Router (node.Type=%v, node.ID=%s), treating as router anyway", node.Type, node.ID)
		}
		log.Infof("Gateway Pod inbound listener, routeName=%s, port=%d, gateway=%s/%s", routeName, parsedPort, gatewayNamespace, gatewayName)

		if !isGatewayDataPort {
			log.Debugf("Gateway Pod inbound listener port %d is not a Gateway Service targetPort, skipping HTTPRoute", parsedPort)
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
		if gatewayNamespace == "" {
			gatewayNamespace = node.ConfigNamespace
			if node.Metadata != nil && node.Metadata.Namespace != "" {
				gatewayNamespace = node.Metadata.Namespace
			}
		}
		activationVirtualHosts := buildActivationVirtualHosts(push, gatewayNamespace)

		// Try to find HTTPRoutes for Gateway Pod
		// Gateway Pods receive traffic with arbitrary hostnames, so we need to collect all HTTPRoutes
		// For Gateway Pod inbound listener, we need ALL HTTPRoutes that could route traffic
		// First try wildcard match to get HTTPRoutes with no hostnames or wildcard hostnames
		allHTTPRoutes := push.HTTPRouteForHost(host.Name("*"))
		log.Debugf("Gateway Pod inbound listener, found %d HTTPRoute(s) with wildcard match", len(allHTTPRoutes))

		// Filter HTTPRoutes by parentRef to match this Gateway
		httpRoutes := filterHTTPRoutesByGateway(allHTTPRoutes, gatewayName, gatewayNamespace, gatewayListenerPort)
		agentConfig := buildAgentConfig(push, httpRoutes)
		log.Debugf("Gateway Pod inbound listener, filtered to %d HTTPRoute(s) matching gateway %s/%s listener port %d", len(httpRoutes), gatewayNamespace, gatewayName, gatewayListenerPort)

		// For Gateway Pod, we also need to collect HTTPRoutes with specific hostnames
		// because Gateway Pods route traffic based on HTTPRoute hostnames in the request
		if len(httpRoutes) > 0 {
			log.Infof("Gateway Pod inbound listener found %d HTTPRoute(s) for port %s", len(httpRoutes), routeName)
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

			if routes := buildRoutesFromGatewayHTTPRoute(httpRoutes, host.Name("*"), gatewayListenerPort, nil); len(routes) > 0 {
				log.Infof("Gateway Pod inbound listener built %d routes from HTTPRoute", len(routes))
				outboundRoutes = routes
			} else {
				log.Warnf("Gateway Pod inbound listener HTTPRoute found but no routes built")
			}
		} else {
			log.Warnf("Gateway Pod inbound listener no HTTPRoute found for port %s", routeName)
			if len(activationVirtualHosts) > 0 {
				return &route.RouteConfiguration{
					Name:         routeName,
					VirtualHosts: activationVirtualHosts,
				}
			}
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

		log.Infof("Gateway Pod inbound listener returning route config with %d domains, %d routes", len(domains), len(outboundRoutes))
		virtualHosts := []*route.VirtualHost{
			{
				Name:    "inbound|http|" + routeName,
				Domains: domains,
				Routes:  outboundRoutes,
			},
		}
		virtualHosts = appendNonConflictingVirtualHosts(activationVirtualHosts, virtualHosts)
		return &route.RouteConfiguration{
			Name:         routeName,
			VirtualHosts: virtualHosts,
			AgentConfig:  agentConfig,
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

func buildActivationVirtualHosts(push *model.PushContext, namespace string) []*route.VirtualHost {
	if push == nil || namespace == "" {
		return nil
	}
	var out []*route.VirtualHost
	for _, svc := range push.ActivatedServices(namespace) {
		for portIndex, port := range svc.Ports {
			hostName := string(svc.Hostname)
			domains := []string{fmt.Sprintf("%s:%d", hostName, port.Port)}
			shortName := strings.Split(hostName, ".")[0]
			if shortName != hostName {
				domains = append(domains, fmt.Sprintf("%s:%d", shortName, port.Port))
			}
			if portIndex == 0 {
				domains = append(domains, hostName)
				if shortName != hostName {
					domains = append(domains, shortName)
				}
			}
			clusterName := model.BuildSubsetKey(model.TrafficDirectionOutbound, "", svc.Hostname, port.Port)
			out = append(out, &route.VirtualHost{
				Name:    fmt.Sprintf("activation|%s|%d", hostName, port.Port),
				Domains: domains,
				Routes:  []*route.Route{defaultSingleClusterRoute(clusterName, nil)},
			})
		}
	}
	return out
}

func appendNonConflictingVirtualHosts(base, candidates []*route.VirtualHost) []*route.VirtualHost {
	claimed := sets.New[string]()
	for _, virtualHost := range base {
		claimed.InsertAll(virtualHost.GetDomains()...)
	}
	for _, candidate := range candidates {
		domains := make([]string, 0, len(candidate.GetDomains()))
		for _, domain := range candidate.GetDomains() {
			if !claimed.Contains(domain) {
				domains = append(domains, domain)
				claimed.Insert(domain)
			}
		}
		if len(domains) == 0 {
			continue
		}
		cloned := proto.Clone(candidate).(*route.VirtualHost)
		cloned.Domains = domains
		base = append(base, cloned)
	}
	return base
}

func defaultSingleClusterRoute(clusterName string, faultPolicy *route.FaultPolicy) *route.Route {
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
				FaultPolicy: faultPolicy,
			},
		},
	}
}

// buildRoutesFromGatewayHTTPRoute converts Gateway API HTTPRoute resources to XDS Route configurations
func buildRoutesFromGatewayHTTPRoute(httpRoutes []config.Config, hostName host.Name, defaultPort int, faultPolicy *route.FaultPolicy) []*route.Route {
	if len(httpRoutes) == 0 {
		return nil
	}

	var allRoutes []*route.Route
	for _, hrConfig := range httpRoutes {
		hrSpec, ok := hrConfig.Spec.(*sigsk8siogatewayapiapisv1.HTTPRouteSpec)
		if !ok {
			log.Warnf("HTTPRoute %s/%s spec is not HTTPRouteSpec", hrConfig.Namespace, hrConfig.Name)
			continue
		}

		// Process each rule in the HTTPRoute
		for ruleIdx, rule := range hrSpec.Rules {
			if len(rule.BackendRefs) == 0 {
				log.Debugf("HTTPRoute %s/%s rule[%d] has no backendRefs, skipping", hrConfig.Namespace, hrConfig.Name, ruleIdx)
				continue
			}
			if ruleUsesDxgateService(rule) {
				// Mesh-native LLM/MCP/A2A backends are carried in AgentConfig.
				// Emitting them as ordinary clusters would fabricate a
				// Kubernetes Service with the same name and bypass protocol
				// translation.
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

				log.Debugf("HTTPRoute %s/%s rule[%d] backend[%d] -> cluster=%s, weight=%d, host=%s, port=%d",
					hrConfig.Namespace, hrConfig.Name, ruleIdx, backendIdx, clusterName, weight, backendHost, backendPort)

				weights = append(weights, &route.WeightedCluster_ClusterWeight{
					Name:   clusterName,
					Weight: wrapperspb.UInt32(weight),
				})
			}

			if len(weights) == 0 {
				log.Warnf("HTTPRoute %s/%s rule[%d] has no valid backends", hrConfig.Namespace, hrConfig.Name, ruleIdx)
				continue
			}

			weightedClusters := &route.WeightedCluster{
				Clusters: weights,
			}
			if totalWeight > 0 {
				weightedClusters.TotalWeight = wrapperspb.UInt32(totalWeight)
			}

			routeAction := &route.RouteAction{
				ClusterSpecifier: &route.RouteAction_WeightedClusters{
					WeightedClusters: weightedClusters,
				},
			}
			if rule.Timeouts != nil {
				if timeout := gatewayAPIDurationToProto(rule.Timeouts.Request); timeout != nil {
					routeAction.Timeout = timeout
				}
			}
			routeAction.RetryPolicy = gatewayAPIRetryPolicy(rule.Retry, rule.Timeouts)
			routeAction.FaultPolicy = faultPolicy

			routeMatches := buildRouteMatchesFromHTTPRouteMatches(rule.Matches)
			for _, routeMatch := range routeMatches {
				allRoutes = append(allRoutes, &route.Route{
					Match: routeMatch,
					Action: &route.Route_Route{
						Route: routeAction,
					},
				})
			}

			log.Infof("HTTPRoute %s/%s rule[%d] -> built %d routes with %d clusters, totalWeight=%d",
				hrConfig.Namespace, hrConfig.Name, ruleIdx, len(routeMatches), len(weights), totalWeight)
		}
	}

	return allRoutes
}

func serviceFaultPolicy(push *model.PushContext, svc *model.Service, port int) *route.FaultPolicy {
	if push == nil || svc == nil {
		return nil
	}
	portName := ""
	if servicePort, found := svc.Ports.GetByPort(port); found && servicePort != nil {
		portName = servicePort.Name
	}
	settings, found := push.FaultInjectionForService(svc.Attributes.Namespace, svc.Attributes.Name, portName)
	if !found {
		return nil
	}
	policy := &route.FaultPolicy{}
	if settings.Delay > 0 {
		policy.Delay = &route.FaultDelay{
			FixedDelay: durationpb.New(settings.Delay),
			Percentage: wrapperspb.UInt32(settings.DelayPercentage),
		}
	}
	if settings.AbortStatus != 0 {
		policy.Abort = &route.FaultAbort{
			HttpStatus: settings.AbortStatus,
			Percentage: wrapperspb.UInt32(settings.AbortPercentage),
		}
	}
	if policy.Delay == nil && policy.Abort == nil {
		return nil
	}
	return policy
}

func gatewayAPIRetryPolicy(retry *sigsk8siogatewayapiapisv1.HTTPRouteRetry, timeouts *sigsk8siogatewayapiapisv1.HTTPRouteTimeouts) *route.RetryPolicy {
	if retry == nil {
		return nil
	}

	attempts := uint32(1)
	if retry.Attempts != nil {
		if *retry.Attempts <= 0 {
			return nil
		}
		attempts = uint32(*retry.Attempts)
	}

	retryOn := []string{"connect-failure", "reset"}
	statusCodes := make([]uint32, 0, len(retry.Codes))
	for _, code := range retry.Codes {
		if code < 400 || code > 599 {
			continue
		}
		statusCodes = append(statusCodes, uint32(code))
	}
	if len(statusCodes) > 0 {
		retryOn = append(retryOn, "retriable-status-codes")
	}

	policy := &route.RetryPolicy{
		RetryOn:              strings.Join(retryOn, ","),
		NumRetries:           wrapperspb.UInt32(attempts),
		RetriableStatusCodes: statusCodes,
	}
	if timeouts != nil {
		policy.PerTryTimeout = gatewayAPIDurationToProto(timeouts.BackendRequest)
	}
	if base := gatewayAPIDurationToProto(retry.Backoff); base != nil && base.AsDuration() > 0 {
		policy.RetryBackOff = &route.RetryPolicy_RetryBackOff{
			BaseInterval: base,
			MaxInterval:  durationpb.New(10 * base.AsDuration()),
		}
	}
	return policy
}

func gatewayAPIDurationToProto(duration *sigsk8siogatewayapiapisv1.Duration) *durationpb.Duration {
	if duration == nil {
		return nil
	}
	parsed, err := time.ParseDuration(string(*duration))
	if err != nil {
		log.Warnf("invalid HTTPRoute timeout duration %q: %v", *duration, err)
		return nil
	}
	return durationpb.New(parsed)
}

// filterHTTPRoutesByGateway filters HTTPRoutes by parentRef to match the given Gateway
func filterHTTPRoutesByGateway(httpRoutes []config.Config, gatewayName, gatewayNamespace string, port int) []config.Config {
	if gatewayName == "" {
		// If we can't determine the Gateway name, return all HTTPRoutes
		// This is a fallback for when Gateway Pod doesn't have proper labels
		log.Warnf("gateway name is empty, returning all HTTPRoutes")
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
					log.Debugf("HTTPRoute %s/%s matches Gateway %s/%s with listener %s",
						hr.Namespace, hr.Name, gatewayNamespace, gatewayName, *parentRef.SectionName)
				}
				matches = true
				break
			}
		}

		if matches {
			filtered = append(filtered, hr)
			log.Debugf("HTTPRoute %s/%s matches Gateway %s/%s",
				hr.Namespace, hr.Name, gatewayNamespace, gatewayName)
		}
	}

	return filtered
}

func filterHTTPRoutesByService(httpRoutes []config.Config, svc *model.Service, port int) []config.Config {
	if svc == nil {
		return nil
	}
	var filtered []config.Config
	for _, hr := range httpRoutes {
		hrSpec, ok := hr.Spec.(*sigsk8siogatewayapiapisv1.HTTPRouteSpec)
		if !ok {
			continue
		}
		if httpRouteReferencesService(hrSpec, hr.Namespace, svc, port) {
			filtered = append(filtered, hr)
			log.Debugf("HTTPRoute %s/%s matches Service %s/%s port %d",
				hr.Namespace, hr.Name, svc.Attributes.Namespace, svc.Attributes.Name, port)
		}
	}
	return filtered
}

func httpRouteReferencesService(hrSpec *sigsk8siogatewayapiapisv1.HTTPRouteSpec, routeNamespace string, svc *model.Service, port int) bool {
	for _, parentRef := range hrSpec.ParentRefs {
		if !isServiceParentRef(parentRef) {
			continue
		}
		refNamespace := routeNamespace
		if parentRef.Namespace != nil {
			refNamespace = string(*parentRef.Namespace)
		}
		if refNamespace != svc.Attributes.Namespace || string(parentRef.Name) != svc.Attributes.Name {
			continue
		}
		if parentRef.Port != nil && *parentRef.Port != int32(port) {
			continue
		}
		if parentRef.SectionName != nil {
			svcPort, found := svc.Ports.GetByPort(port)
			if !found || svcPort.Name != string(*parentRef.SectionName) {
				continue
			}
		}
		return true
	}
	return false
}

func isServiceParentRef(parentRef sigsk8siogatewayapiapisv1.ParentReference) bool {
	if parentRef.Kind == nil || string(*parentRef.Kind) != "Service" {
		return false
	}
	return parentRef.Group == nil || string(*parentRef.Group) == ""
}

// buildRouteMatchesFromHTTPRouteMatches preserves Gateway API OR semantics:
// every HTTPRouteMatch in a rule becomes an independent xDS route.
func buildRouteMatchesFromHTTPRouteMatches(matches []sigsk8siogatewayapiapisv1.HTTPRouteMatch) []*route.RouteMatch {
	if len(matches) == 0 {
		matches = []sigsk8siogatewayapiapisv1.HTTPRouteMatch{{}}
	}

	out := make([]*route.RouteMatch, 0, len(matches))
	for _, match := range matches {
		out = append(out, buildRouteMatchFromHTTPRouteMatch(match))
	}
	return out
}

func buildRouteMatchFromHTTPRouteMatch(match sigsk8siogatewayapiapisv1.HTTPRouteMatch) *route.RouteMatch {
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
