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

package triplegen

import (
	"fmt"
	"net"
	"sort"
	"strconv"
	"strings"

	"github.com/apache/dubbo-kubernetes/dubbod/discovery/pkg/model"
	"github.com/apache/dubbo-kubernetes/dubbod/discovery/pkg/networking/util"
	"github.com/apache/dubbo-kubernetes/dubbod/discovery/pkg/util/protoconv"
	"github.com/apache/dubbo-kubernetes/pkg/config/host"
	"github.com/apache/dubbo-kubernetes/pkg/util/sets"
	"github.com/apache/dubbo-kubernetes/pkg/wellknown"
	"github.com/apache/dubbo-kubernetes/pkg/xdsbootstrap"
	core "github.com/dubml/xds-api/core/v1"
	routerv1 "github.com/dubml/xds-api/extensions/filters/v1/http/router"
	hcmv1 "github.com/dubml/xds-api/extensions/filters/v1/network/http_connection_manager"
	listener "github.com/dubml/xds-api/listener/v1"
	discovery "github.com/dubml/xds-api/service/discovery/v1"
)

type listenerNames map[string]listenerName

type listenerName struct {
	RequestedNames sets.String
	Ports          sets.String
}

func (g *TripleConfigGenerator) BuildListeners(node *model.Proxy, push *model.PushContext, names []string) model.Resources {
	nodeID := ""
	if node != nil {
		nodeID = node.ID
	}
	if len(names) == 0 && push != nil {
		allServices := push.GetAllServices()
		candidateNames := sets.New[string]()
		for _, svc := range allServices {
			if svc == nil || svc.Hostname == "" {
				continue
			}
			for _, p := range svc.Ports {
				if p == nil || p.Port <= 0 {
					continue
				}
				candidateNames.Insert(fmt.Sprintf("%s:%d", svc.Hostname, p.Port))
			}
		}
		names = sets.SortedList(candidateNames)
		log.Debugf("wildcard request for %s, generating %d outbound listeners: %v", nodeID, len(names), names)
	} else if len(names) > 0 {
		log.Debugf("specific request for %s, requested listeners: %v", nodeID, names)
	}

	if len(names) == 0 {
		return nil
	}

	filter := newListenerNameFilter(names, node)
	resp := buildOutboundListeners(node, push, filter)

	requestedSet := sets.New(names...)
	filtered := make(model.Resources, 0, len(resp))
	seen := sets.New[string]()
	for _, r := range resp {
		if requestedSet.Contains(r.Name) && !seen.Contains(r.Name) {
			seen.Insert(r.Name)
			filtered = append(filtered, r)
		} else {
			log.Debugf("filtering out unrequested or duplicate listener %s (requested: %v)", r.Name, names)
		}
	}
	return filtered
}

func tryFindFQDN(name string, node *model.Proxy) string {
	if node == nil {
		return ""
	}
	// no "." - assuming this is a shortname "foo" -> "foo.ns.svc.cluster.local"
	if !strings.Contains(name, ".") {
		return fmt.Sprintf("%s.%s", name, node.DNSDomain)
	}
	if node.Metadata == nil {
		return ""
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

func newListenerNameFilter(names []string, node *model.Proxy) listenerNames {
	filter := make(listenerNames, len(names))
	for _, name := range names {
		// inbound, create a simple entry and move on
		if strings.HasPrefix(name, xdsbootstrap.ServerListenerNamePrefix) {
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

	filterKeys := make([]string, 0, len(filter))
	for name := range filter {
		filterKeys = append(filterKeys, name)
	}
	sort.Strings(filterKeys)

	for _, name := range filterKeys {
		ln := filter[name]
		if strings.HasPrefix(name, xdsbootstrap.ServerListenerNamePrefix) {
			continue // Skip inbound listeners
		}

		// If this entry has ports, use them; otherwise use the name as-is (might already have port)
		if len(ln.Ports) > 0 {
			ports := make([]string, 0, len(ln.Ports))
			for port := range ln.Ports {
				ports = append(ports, port)
			}
			sort.Strings(ports)
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
			if err != nil || port <= 0 {
				log.Warnf("buildOutboundListeners: invalid port %q: %v", portStr, err)
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
			// Format: outbound|port||hostname (e.g., outbound|7070||consumer.app.svc.cluster.local)
			// Use svc.Hostname (FQDN) instead of svc.Attributes.Name (short name) to match CDS expectations
			clusterName := model.BuildSubsetKey(model.TrafficDirectionOutbound, "", svc.Hostname, port)

			// Build route name (same format as cluster name) for RDS
			routeName := clusterName

			// For Triple inherent mode, outbound listeners MUST use ApiListener with RDS.
			hcm := &hcmv1.HttpConnectionManager{
				CodecType:  hcmv1.HttpConnectionManager_AUTO,
				StatPrefix: fmt.Sprintf("outbound_%d_%s", port, svc.Attributes.Name),
				RouteSpecifier: &hcmv1.HttpConnectionManager_Rds{
					Rds: &hcmv1.Rds{
						ConfigSource: &core.ConfigSource{
							ConfigSourceSpecifier: &core.ConfigSource_Ads{
								Ads: &core.AggregatedConfigSource{},
							},
						},
						RouteConfigName: routeName,
					},
				},
				HttpFilters: []*hcmv1.HttpFilter{
					{
						Name: wellknown.HTTPRouter,
						ConfigType: &hcmv1.HttpFilter_TypedConfig{
							TypedConfig: protoconv.MessageToAny(&routerv1.Router{}),
						},
					},
				},
			}

			// Build outbound listener with ApiListener and empty FilterChains for Triple xDS clients.
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
