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
	"net"
	"strconv"
	"strings"

	dubbolog "github.com/apache/dubbo-kubernetes/pkg/log"

	"github.com/apache/dubbo-kubernetes/dubbod/planet/pkg/model"
	"github.com/apache/dubbo-kubernetes/dubbod/planet/pkg/networking/util"
	"github.com/apache/dubbo-kubernetes/dubbod/planet/pkg/util/protoconv"
	"github.com/apache/dubbo-kubernetes/pkg/config/host"
	"github.com/apache/dubbo-kubernetes/pkg/dubbo-agent/grpcxds"
	"github.com/apache/dubbo-kubernetes/pkg/util/sets"
	"github.com/apache/dubbo-kubernetes/pkg/wellknown"
	core "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	listener "github.com/envoyproxy/go-control-plane/envoy/config/listener/v3"
	route "github.com/envoyproxy/go-control-plane/envoy/config/route/v3"
	routerv3 "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/http/router/v3"
	hcmv3 "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/network/http_connection_manager/v3"
	discovery "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v3"
)

var log = dubbolog.RegisterScope("grpcgen", "xDS Generator for Proxyless gRPC")

type listenerNames map[string]listenerName

type listenerName struct {
	RequestedNames sets.String
	Ports          sets.String
}

func newListenerNameFilter(names []string, node *model.Proxy) listenerNames {
	filter := make(listenerNames, len(names))
	for _, name := range names {
		// inbound, create a simple entry and move on
		if strings.HasPrefix(name, grpcxds.ServerListenerNamePrefix) {
			filter[name] = listenerName{RequestedNames: sets.New(name)}
			continue
		}

		host, port, err := net.SplitHostPort(name)
		hasPort := err == nil

		// attempt to expand shortname to FQDN
		requestedName := name
		if hasPort {
			requestedName = host
		}
		allNames := []string{requestedName}
		if fqdn := tryFindFQDN(requestedName, node); fqdn != "" {
			allNames = append(allNames, fqdn)
		}

		for _, name := range allNames {
			ln, ok := filter[name]
			if !ok {
				ln = listenerName{RequestedNames: sets.New[string]()}
			}
			ln.RequestedNames.Insert(requestedName)

			// only build the portmap if we aren't filtering this name yet, or if the existing filter is non-empty
			if hasPort && (!ok || len(ln.Ports) != 0) {
				if ln.Ports == nil {
					ln.Ports = map[string]struct{}{}
				}
				ln.Ports.Insert(port)
			} else if !hasPort {
				// if we didn't have a port, we should clear the portmap
				ln.Ports = nil
			}
			filter[name] = ln
		}
	}
	return filter
}

func (g *GrpcConfigGenerator) BuildListeners(node *model.Proxy, push *model.PushContext, names []string) model.Resources {
	// For LDS (wildcard type), empty ResourceNames means request all listeners
	// If names is empty, generate listeners for all ServiceTargets to ensure consistent behavior
	// This prevents the client from receiving different numbers of listeners on different requests
	if len(names) == 0 && len(node.ServiceTargets) > 0 {
		// Build listener names from ServiceTargets for wildcard/initial request
		names = make([]string, 0, len(node.ServiceTargets))
		for _, st := range node.ServiceTargets {
			if st.Service != nil && st.Port.ServicePort != nil {
				// Generate inbound listener name for each service target port
				listenerName := fmt.Sprintf("%s0.0.0.0:%d", grpcxds.ServerListenerNamePrefix, st.Port.TargetPort)
				names = append(names, listenerName)
			}
		}
		log.Debugf("BuildListeners: wildcard request for %s, generating %d listeners from ServiceTargets: %v", node.ID, len(names), names)
	} else if len(names) > 0 {
		log.Debugf("BuildListeners: specific request for %s, requested listeners: %v", node.ID, names)
	}

	// CRITICAL: If names is provided (non-empty), we MUST only generate those specific listeners
	// This prevents the push loop where client requests 1 listener but receives 14
	// The filter ensures we only generate requested listeners
	filter := newListenerNameFilter(names, node)
	resp := make(model.Resources, 0, len(filter))

	// Build outbound listeners first (they may reference clusters that need CDS)
	outboundRes := buildOutboundListeners(node, push, filter)
	resp = append(resp, outboundRes...)
	resp = append(resp, buildInboundListeners(node, push, filter.inboundNames())...)

	// Final safety check: ensure we only return resources that were requested
	// This is critical for preventing push loops in proxyless gRPC
	if len(names) > 0 {
		requestedSet := sets.New(names...)
		filtered := make(model.Resources, 0, len(resp))
		for _, r := range resp {
			if requestedSet.Contains(r.Name) {
				filtered = append(filtered, r)
			} else {
				log.Debugf("BuildListeners: filtering out unrequested listener %s (requested: %v)", r.Name, names)
			}
		}
		return filtered
	}

	return resp
}

func buildOutboundListeners(node *model.Proxy, push *model.PushContext, filter listenerNames) model.Resources {
	out := make(model.Resources, 0, len(filter))

	// Extract outbound listener names (not inbound)
	// The filter map uses hostname as key, and stores ports in listenerName.Ports
	// We need to reconstruct the full listener name (hostname:port) for each port
	type listenerInfo struct {
		hostname string
		ports    []string
	}
	var outboundListeners []listenerInfo

	for name, ln := range filter {
		if strings.HasPrefix(name, grpcxds.ServerListenerNamePrefix) {
			continue // Skip inbound listeners
		}

		// If this entry has ports, use them; otherwise use the name as-is (might already have port)
		if len(ln.Ports) > 0 {
			ports := make([]string, 0, len(ln.Ports))
			for port := range ln.Ports {
				ports = append(ports, port)
			}
			outboundListeners = append(outboundListeners, listenerInfo{
				hostname: name,
				ports:    ports,
			})
		} else {
			// No ports in filter, try to parse name as hostname:port
			_, _, err := net.SplitHostPort(name)
			if err == nil {
				// Name already contains port, use it directly
				outboundListeners = append(outboundListeners, listenerInfo{
					hostname: name,
					ports:    []string{name}, // Use full name as-is
				})
			} else {
				// No port, skip (shouldn't happen for outbound listeners)
				log.Debugf("buildOutboundListeners: skipping listener %s (no port information)", name)
			}
		}
	}

	if len(outboundListeners) == 0 {
		return out
	}

	// Build outbound listeners for each requested service
	for _, li := range outboundListeners {
		// For each port, build a listener
		for _, portInfo := range li.ports {
			var hostStr, portStr string
			var err error

			// Check if portInfo is already "hostname:port" format
			hostStr, portStr, err = net.SplitHostPort(portInfo)
			if err != nil {
				// portInfo is just the port number, use hostname from filter key
				hostStr = li.hostname
				portStr = portInfo
			}

			port, err := strconv.Atoi(portStr)
			if err != nil {
				log.Warnf("buildOutboundListeners: failed to parse port from %s: %v", portStr, err)
				continue
			}

			// Reconstruct full listener name for logging and final check
			fullListenerName := fmt.Sprintf("%s:%s", hostStr, portStr)

			// Find service in PushContext
			// Try different hostname formats: FQDN, short name, etc.
			hostname := host.Name(hostStr)
			svc := push.ServiceForHostname(node, hostname)

			// Extract short name from FQDN for fallback lookup
			parts := strings.Split(hostStr, ".")
			shortName := ""
			if len(parts) > 0 {
				shortName = parts[0]
			}

			// If not found with FQDN, try short name (e.g., "consumer" from "consumer.grpc-app.svc.cluster.local")
			if svc == nil && shortName != "" {
				svc = push.ServiceForHostname(node, host.Name(shortName))
				if svc != nil {
					log.Debugf("buildOutboundListeners: found service %s using short name %s", svc.Hostname, shortName)
				}
			}

			// If still not found, try to find service by iterating all services
			if svc == nil {
				// Try to find service by matching hostname in all namespaces
				allServices := push.GetAllServices()
				for _, s := range allServices {
					if s.Hostname == hostname || (shortName != "" && strings.HasPrefix(string(s.Hostname), shortName+".")) {
						svc = s
						log.Debugf("buildOutboundListeners: found service %s/%s by iterating", svc.Attributes.Namespace, svc.Hostname)
						break
					}
				}
			}

			if svc == nil {
				log.Warnf("buildOutboundListeners: service not found for hostname %s (tried FQDN and short name)", hostStr)
				continue
			}

			// Verify service has the requested port (Service port, not targetPort)
			hasPort := false
			var matchedPort *model.Port
			for _, p := range svc.Ports {
				if p.Port == port {
					hasPort = true
					matchedPort = p
					break
				}
			}
			if !hasPort {
				log.Warnf("buildOutboundListeners: port %d not found in service %s (available ports: %v)",
					port, hostStr, func() []int {
						ports := make([]int, 0, len(svc.Ports))
						for _, p := range svc.Ports {
							ports = append(ports, p.Port)
						}
						return ports
					}())
				continue
			}

			log.Debugf("buildOutboundListeners: building outbound listener for %s:%d (service: %s/%s, port: %s)",
				hostStr, port, svc.Attributes.Namespace, svc.Attributes.Name, matchedPort.Name)

			// Build cluster name using BuildSubsetKey to ensure correct format
			// Format: outbound|port||hostname (e.g., outbound|7070||consumer.grpc-app.svc.cluster.local)
			// Use svc.Hostname (FQDN) instead of svc.Attributes.Name (short name) to match CDS expectations
			clusterName := model.BuildSubsetKey(model.TrafficDirectionOutbound, "", svc.Hostname, port)

			// Build route name (same format as cluster name) for RDS
			routeName := clusterName

			// CRITICAL: For gRPC proxyless, outbound listeners MUST use ApiListener with RDS
			// This is the correct pattern used by Istio for gRPC xDS clients
			// Using FilterChain with inline RouteConfig causes the gRPC client to remain in IDLE state
			hcm := &hcmv3.HttpConnectionManager{
				CodecType:  hcmv3.HttpConnectionManager_AUTO,
				StatPrefix: fmt.Sprintf("outbound_%d_%s", port, svc.Attributes.Name),
				RouteSpecifier: &hcmv3.HttpConnectionManager_Rds{
					Rds: &hcmv3.Rds{
						ConfigSource: &core.ConfigSource{
							ConfigSourceSpecifier: &core.ConfigSource_Ads{
								Ads: &core.AggregatedConfigSource{},
							},
						},
						RouteConfigName: routeName,
					},
				},
				HttpFilters: []*hcmv3.HttpFilter{
					{
						Name: "envoy.filters.http.router",
						ConfigType: &hcmv3.HttpFilter_TypedConfig{
							TypedConfig: protoconv.MessageToAny(&routerv3.Router{}),
						},
					},
				},
			}

			// Build outbound listener with ApiListener (Istio pattern)
			// CRITICAL: gRPC xDS clients expect ApiListener for outbound, not FilterChain
			ll := &listener.Listener{
				Name: fullListenerName,
				Address: &core.Address{Address: &core.Address_SocketAddress{
					SocketAddress: &core.SocketAddress{
						Address: svc.GetAddressForProxy(node), // Use service VIP
						PortSpecifier: &core.SocketAddress_PortValue{
							PortValue: uint32(port),
						},
					},
				}},
				ApiListener: &listener.ApiListener{
					ApiListener: protoconv.MessageToAny(hcm),
				},
			}

			// Add extra addresses if available
			extrAddresses := svc.GetExtraAddressesForProxy(node)
			if len(extrAddresses) > 0 {
				ll.AdditionalAddresses = util.BuildAdditionalAddresses(extrAddresses, uint32(port))
			}

			// CRITICAL: Log listener details for debugging gRPC xDS client connection issues
			log.Debugf("buildOutboundListeners: created ApiListener name=%s, address=%s:%d, routeName=%s",
				ll.Name, ll.Address.GetSocketAddress().Address, ll.Address.GetSocketAddress().GetPortValue(), routeName)

			out = append(out, &discovery.Resource{
				Name:     ll.Name,
				Resource: protoconv.MessageToAny(ll),
			})
		}
	}

	return out
}

func (f listenerNames) inboundNames() []string {
	var out []string
	for key := range f {
		if strings.HasPrefix(key, grpcxds.ServerListenerNamePrefix) {
			out = append(out, key)
		}
	}
	return out
}

func buildInboundListeners(node *model.Proxy, push *model.PushContext, names []string) model.Resources {
	var out model.Resources
	// TODO NewMtlsPolicy
	serviceInstancesByPort := map[uint32]model.ServiceTarget{}
	for _, si := range node.ServiceTargets {
		if si.Port.ServicePort != nil {
			serviceInstancesByPort[si.Port.TargetPort] = si
		}
	}

	// Use the provided names - at this point names should not be empty
	// (empty names are handled in BuildListeners to generate all listeners)
	// This ensures we only generate listeners that were requested, preventing inconsistent behavior
	listenerNames := names
	if len(listenerNames) == 0 {
		// This should not happen if BuildListeners logic is correct, but handle gracefully
		return out
	}

	for _, name := range listenerNames {
		listenAddress := strings.TrimPrefix(name, grpcxds.ServerListenerNamePrefix)
		listenHost, listenPortStr, err := net.SplitHostPort(listenAddress)
		if err != nil {
			log.Errorf("failed parsing address from gRPC listener name %s: %v", name, err)
			continue
		}
		listenPort, err := strconv.Atoi(listenPortStr)
		if err != nil {
			log.Errorf("failed parsing port from gRPC listener name %s: %v", name, err)
			continue
		}
		si, ok := serviceInstancesByPort[uint32(listenPort)]
		if !ok {
			// If no service target found for this port, don't create a listener
			// This prevents creating invalid listeners that would cause SERVING/NOT_SERVING cycles
			// The client should only request listeners for ports that are actually exposed by the service
			log.Warnf("%s has no service instance for port %s, skipping listener %s. "+
				"This usually means the requested port doesn't match any service port in the pod's ServiceTargets.",
				node.ID, listenPortStr, name)
			continue
		}

		// For proxyless gRPC inbound listeners, we need a FilterChain with HttpConnectionManager filter
		// to satisfy gRPC client requirements. According to grpc-go issue #7691 and the error
		// "missing HttpConnectionManager filter", gRPC proxyless clients require HttpConnectionManager
		// in the FilterChain for inbound listeners.
		// Use inline RouteConfig instead of RDS to avoid triggering additional RDS requests that cause push loops
		// For proxyless gRPC, inline configuration is preferred to minimize round-trips
		routeName := fmt.Sprintf("%d", listenPort)
		hcm := &hcmv3.HttpConnectionManager{
			CodecType:  hcmv3.HttpConnectionManager_AUTO,
			StatPrefix: fmt.Sprintf("inbound_%d", listenPort),
			RouteSpecifier: &hcmv3.HttpConnectionManager_RouteConfig{
				RouteConfig: &route.RouteConfiguration{
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
				},
			},
			HttpFilters: []*hcmv3.HttpFilter{
				{
					Name: "envoy.filters.http.router",
					ConfigType: &hcmv3.HttpFilter_TypedConfig{
						TypedConfig: protoconv.MessageToAny(&routerv3.Router{}),
					},
				},
			},
		}

		ll := &listener.Listener{
			Name: name,
			Address: &core.Address{Address: &core.Address_SocketAddress{
				SocketAddress: &core.SocketAddress{
					Address: listenHost,
					PortSpecifier: &core.SocketAddress_PortValue{
						PortValue: uint32(listenPort),
					},
				},
			}},
			// Create FilterChain with HttpConnectionManager filter for proxyless gRPC
			FilterChains: []*listener.FilterChain{
				{
					Filters: []*listener.Filter{
						{
							Name: wellknown.HTTPConnectionManager,
							ConfigType: &listener.Filter_TypedConfig{
								TypedConfig: protoconv.MessageToAny(hcm),
							},
						},
					},
				},
			},
			// the following must not be set or the client will NACK
			ListenerFilters: nil,
			UseOriginalDst:  nil,
		}
		// add extra addresses for the listener
		extrAddresses := si.Service.GetExtraAddressesForProxy(node)
		if len(extrAddresses) > 0 {
			ll.AdditionalAddresses = util.BuildAdditionalAddresses(extrAddresses, uint32(listenPort))
		}

		out = append(out, &discovery.Resource{
			Name:     ll.Name,
			Resource: protoconv.MessageToAny(ll),
		})
	}
	return out
}

func tryFindFQDN(name string, node *model.Proxy) string {
	// no "." - assuming this is a shortname "foo" -> "foo.ns.svc.cluster.local"
	if !strings.Contains(name, ".") {
		return fmt.Sprintf("%s.%s", name, node.DNSDomain)
	}
	for _, suffix := range []string{
		node.Metadata.Namespace,
		node.Metadata.Namespace + ".svc",
	} {
		shortname := strings.TrimSuffix(name, "."+suffix)
		if shortname != name && strings.HasPrefix(node.DNSDomain, suffix) {
			return fmt.Sprintf("%s.%s", shortname, node.DNSDomain)
		}
	}
	return ""
}
