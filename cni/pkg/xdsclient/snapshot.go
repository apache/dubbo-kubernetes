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

package xdsclient

import (
	"fmt"
	"net"
	"sort"
	"strconv"
	"strings"
	"time"

	clusterv1 "github.com/dubml/xds-api/cluster/v1"
	corev1 "github.com/dubml/xds-api/core/v1"
	endpointv1 "github.com/dubml/xds-api/endpoint/v1"
	hcmv1 "github.com/dubml/xds-api/extensions/filters/v1/network/http_connection_manager"
	tlsv1 "github.com/dubml/xds-api/extensions/transport_sockets/tls/v1"
	listenerv1 "github.com/dubml/xds-api/listener/v1"
	routev1 "github.com/dubml/xds-api/route/v1"
)

// TargetEndpoint represents a healthy selectable backend endpoint.
type TargetEndpoint struct {
	Address string
	Port    uint32
	Weight  uint32
}

// HostPort returns net.JoinHostPort for this endpoint.
func (e TargetEndpoint) HostPort() string {
	return net.JoinHostPort(e.Address, strconv.Itoa(int(e.Port)))
}

// TargetCluster holds the compiled cluster routing and TLS parameters.
type TargetCluster struct {
	Name           string
	ServiceName    string
	Endpoints      []TargetEndpoint
	UseMTLS        bool
	SNI            string
	ConnectTimeout time.Duration
}

// Snapshot is an immutable, atomically published routing table compiled from LDS/RDS/CDS/EDS.
type Snapshot struct {
	Generation uint64
	Ready      bool
	Listeners  map[string]*routev1.RouteConfiguration
	Clusters   map[string]*TargetCluster
}

// Selection is the resolved routing decision for a single connection or request.
type Selection struct {
	ClusterName    string
	Endpoint       TargetEndpoint
	UseMTLS        bool
	SNI            string
	ConnectTimeout time.Duration
}

// resourceCache maintains the latest validated xDS resources per type.
type resourceCache struct {
	listeners map[string]*listenerv1.Listener
	routes    map[string]*routev1.RouteConfiguration
	clusters  map[string]*clusterv1.Cluster
	endpoints map[string]*endpointv1.ClusterLoadAssignment
}

func newResourceCache() *resourceCache {
	return &resourceCache{
		listeners: map[string]*listenerv1.Listener{},
		routes:    map[string]*routev1.RouteConfiguration{},
		clusters:  map[string]*clusterv1.Cluster{},
		endpoints: map[string]*endpointv1.ClusterLoadAssignment{},
	}
}

type dependencies struct {
	routes    []string
	clusters  []string
	endpoints []string
}

func (c *resourceCache) compile(generation uint64) (*Snapshot, dependencies) {
	snap := &Snapshot{
		Generation: generation,
		Listeners:  make(map[string]*routev1.RouteConfiguration, len(c.listeners)),
		Clusters:   make(map[string]*TargetCluster, len(c.clusters)),
	}

	routeSet := map[string]struct{}{}
	clusterSet := map[string]struct{}{}
	endpointSet := map[string]struct{}{}
	missing := false

	for name, l := range c.listeners {
		if l == nil {
			continue
		}
		hcm := extractHCM(l)
		if hcm == nil {
			continue
		}
		switch rs := hcm.GetRouteSpecifier().(type) {
		case *hcmv1.HttpConnectionManager_RouteConfig:
			if rs.RouteConfig != nil {
				snap.Listeners[name] = rs.RouteConfig
				collectRouteClusters(rs.RouteConfig, clusterSet)
			}
		case *hcmv1.HttpConnectionManager_Rds:
			if rs.Rds != nil && rs.Rds.RouteConfigName != "" {
				routeName := rs.Rds.RouteConfigName
				routeSet[routeName] = struct{}{}
				if rc, ok := c.routes[routeName]; ok && rc != nil {
					snap.Listeners[name] = rc
					collectRouteClusters(rc, clusterSet)
				} else {
					missing = true
				}
			}
		}
	}

	for name, cl := range c.clusters {
		if cl == nil {
			continue
		}
		tc := compileCluster(cl, c.endpoints)
		snap.Clusters[name] = tc
		if cl.GetType() == clusterv1.Cluster_EDS {
			svcName := cl.GetEdsClusterConfig().GetServiceName()
			if svcName == "" {
				svcName = name
			}
			endpointSet[svcName] = struct{}{}
		}
	}

	for clName := range clusterSet {
		if _, ok := c.clusters[clName]; !ok {
			missing = true
		}
	}
	for epName := range endpointSet {
		if _, ok := c.endpoints[epName]; !ok {
			missing = true
		}
	}

	snap.Ready = len(c.listeners) > 0 && !missing
	return snap, dependencies{
		routes:    sortedKeys(routeSet),
		clusters:  sortedKeys(clusterSet),
		endpoints: sortedKeys(endpointSet),
	}
}

// Select resolves an endpoint by listener name (or host:port fallback), authority, path, and deterministic ticket.
func (s *Snapshot) Select(listenerName, authority, path string, ticket uint64) (Selection, error) {
	if s == nil {
		return Selection{}, fmt.Errorf("xDS snapshot unavailable")
	}

	rc := s.Listeners[listenerName]
	if rc == nil && authority != "" {
		rc = s.Listeners[authority]
	}
	if rc == nil {
		// Fallback: direct cluster lookup if listenerName matches a known cluster/service.
		if tc, ok := s.Clusters[listenerName]; ok {
			return selectFromCluster(tc, ticket)
		}
		return Selection{}, fmt.Errorf("no xDS listener for %q", listenerName)
	}

	clusterName := matchRouteCluster(rc, authority, path, ticket)
	if clusterName == "" {
		return Selection{}, fmt.Errorf("no matching xDS route for listener %q authority %q path %q", listenerName, authority, path)
	}
	tc, ok := s.Clusters[clusterName]
	if !ok {
		return Selection{}, fmt.Errorf("xDS cluster %q not found", clusterName)
	}
	return selectFromCluster(tc, ticket)
}

// SelectByDestinationIPPort matches a Cluster or Listener using a destination ClusterIP:port or host:port.
func (s *Snapshot) SelectByDestinationIPPort(destAddr string, ticket uint64) (Selection, error) {
	if s == nil {
		return Selection{}, fmt.Errorf("xDS snapshot unavailable")
	}
	if sel, err := s.Select(destAddr, destAddr, "/", ticket); err == nil {
		return sel, nil
	}
	host, portStr, err := net.SplitHostPort(destAddr)
	if err != nil {
		return Selection{}, fmt.Errorf("invalid destination %q: %w", destAddr, err)
	}
	// Check if destAddr is already a known healthy endpoint in any cluster (direct pod IP).
	for _, tc := range s.Clusters {
		for _, ep := range tc.Endpoints {
			if ep.Address == host && strconv.Itoa(int(ep.Port)) == portStr {
				return Selection{
					ClusterName:    tc.Name,
					Endpoint:       ep,
					UseMTLS:        tc.UseMTLS,
					SNI:            tc.SNI,
					ConnectTimeout: tc.ConnectTimeout,
				}, nil
			}
		}
	}
	return Selection{}, fmt.Errorf("no xDS route or endpoint matches destination %q", destAddr)
}

func selectFromCluster(tc *TargetCluster, ticket uint64) (Selection, error) {
	if tc == nil || len(tc.Endpoints) == 0 {
		name := ""
		if tc != nil {
			name = tc.Name
		}
		return Selection{}, fmt.Errorf("no healthy endpoints for cluster %q", name)
	}
	var totalWeight uint64
	for _, ep := range tc.Endpoints {
		w := uint64(ep.Weight)
		if w == 0 {
			w = 1
		}
		totalWeight += w
	}
	slot := ticket % totalWeight
	var acc uint64
	chosen := tc.Endpoints[0]
	for _, ep := range tc.Endpoints {
		w := uint64(ep.Weight)
		if w == 0 {
			w = 1
		}
		acc += w
		if slot < acc {
			chosen = ep
			break
		}
	}
	return Selection{
		ClusterName:    tc.Name,
		Endpoint:       chosen,
		UseMTLS:        tc.UseMTLS,
		SNI:            tc.SNI,
		ConnectTimeout: tc.ConnectTimeout,
	}, nil
}

func extractHCM(l *listenerv1.Listener) *hcmv1.HttpConnectionManager {
	if api := l.GetApiListener().GetApiListener(); api != nil {
		hcm := &hcmv1.HttpConnectionManager{}
		if err := api.UnmarshalTo(hcm); err == nil {
			return hcm
		}
	}
	for _, fc := range l.GetFilterChains() {
		for _, f := range fc.GetFilters() {
			if typed := f.GetTypedConfig(); typed != nil {
				hcm := &hcmv1.HttpConnectionManager{}
				if err := typed.UnmarshalTo(hcm); err == nil {
					return hcm
				}
			}
		}
	}
	return nil
}

func collectRouteClusters(rc *routev1.RouteConfiguration, out map[string]struct{}) {
	for _, vh := range rc.GetVirtualHosts() {
		for _, r := range vh.GetRoutes() {
			ra := r.GetRoute()
			if ra == nil {
				continue
			}
			switch cs := ra.GetClusterSpecifier().(type) {
			case *routev1.RouteAction_Cluster:
				if cs.Cluster != "" {
					out[cs.Cluster] = struct{}{}
				}
			case *routev1.RouteAction_WeightedClusters:
				for _, wc := range cs.WeightedClusters.GetClusters() {
					if wc.GetName() != "" {
						out[wc.GetName()] = struct{}{}
					}
				}
			}
		}
	}
}

func compileCluster(cl *clusterv1.Cluster, eds map[string]*endpointv1.ClusterLoadAssignment) *TargetCluster {
	tc := &TargetCluster{
		Name:           cl.GetName(),
		ConnectTimeout: 5 * time.Second,
	}
	if d := cl.GetConnectTimeout(); d != nil && d.AsDuration() > 0 {
		tc.ConnectTimeout = d.AsDuration()
	}
	if ts := cl.GetTransportSocket().GetTypedConfig(); ts != nil {
		tlsCtx := &tlsv1.UpstreamTlsContext{}
		if err := ts.UnmarshalTo(tlsCtx); err == nil {
			tc.UseMTLS = true
			tc.SNI = tlsCtx.GetSni()
		}
	}

	var cla *endpointv1.ClusterLoadAssignment
	if cl.GetType() == clusterv1.Cluster_EDS {
		svcName := cl.GetEdsClusterConfig().GetServiceName()
		if svcName == "" {
			svcName = cl.GetName()
		}
		tc.ServiceName = svcName
		cla = eds[svcName]
	} else {
		cla = cl.GetLoadAssignment()
	}

	if cla != nil {
		tc.Endpoints = extractHealthyEndpoints(cla)
	}
	return tc
}

func extractHealthyEndpoints(cla *endpointv1.ClusterLoadAssignment) []TargetEndpoint {
	var minPriority uint32 = ^uint32(0)
	var byPriority []struct {
		priority uint32
		ep       TargetEndpoint
	}
	for _, loc := range cla.GetEndpoints() {
		locWeight := uint32(1)
		if loc.GetLoadBalancingWeight() != nil && loc.GetLoadBalancingWeight().GetValue() > 0 {
			locWeight = loc.GetLoadBalancingWeight().GetValue()
		}
		for _, lb := range loc.GetLbEndpoints() {
			if lb.GetHealthStatus() != corev1.HealthStatus_UNKNOWN && lb.GetHealthStatus() != corev1.HealthStatus_HEALTHY {
				continue
			}
			sa := lb.GetEndpoint().GetAddress().GetSocketAddress()
			if sa == nil || sa.GetAddress() == "" || sa.GetPortValue() == 0 {
				continue
			}
			epWeight := uint32(1)
			if lb.GetLoadBalancingWeight() != nil && lb.GetLoadBalancingWeight().GetValue() > 0 {
				epWeight = lb.GetLoadBalancingWeight().GetValue()
			}
			p := loc.GetPriority()
			if p < minPriority {
				minPriority = p
			}
			byPriority = append(byPriority, struct {
				priority uint32
				ep       TargetEndpoint
			}{
				priority: p,
				ep: TargetEndpoint{
					Address: sa.GetAddress(),
					Port:    sa.GetPortValue(),
					Weight:  locWeight * epWeight,
				},
			})
		}
	}
	out := make([]TargetEndpoint, 0, len(byPriority))
	for _, item := range byPriority {
		if item.priority == minPriority {
			out = append(out, item.ep)
		}
	}
	return out
}

func matchRouteCluster(rc *routev1.RouteConfiguration, authority, path string, ticket uint64) string {
	vh := matchVirtualHost(rc.GetVirtualHosts(), authority)
	if vh == nil {
		return ""
	}
	if path == "" {
		path = "/"
	}
	for _, r := range vh.GetRoutes() {
		if !routeMatchesPath(r.GetMatch(), path) {
			continue
		}
		ra := r.GetRoute()
		if ra == nil {
			continue
		}
		switch cs := ra.GetClusterSpecifier().(type) {
		case *routev1.RouteAction_Cluster:
			return cs.Cluster
		case *routev1.RouteAction_WeightedClusters:
			return selectWeightedCluster(cs.WeightedClusters, ticket)
		}
	}
	return ""
}

func matchVirtualHost(vhs []*routev1.VirtualHost, authority string) *routev1.VirtualHost {
	hostOnly := authority
	if h, _, err := net.SplitHostPort(authority); err == nil {
		hostOnly = h
	}
	var wildcard *routev1.VirtualHost
	for _, vh := range vhs {
		for _, d := range vh.GetDomains() {
			if d == "*" {
				if wildcard == nil {
					wildcard = vh
				}
				continue
			}
			if strings.EqualFold(d, authority) || strings.EqualFold(d, hostOnly) {
				return vh
			}
			if strings.HasPrefix(d, "*.") && strings.HasSuffix(strings.ToLower(hostOnly), strings.ToLower(d[1:])) {
				return vh
			}
		}
	}
	return wildcard
}

func routeMatchesPath(m *routev1.RouteMatch, path string) bool {
	if m == nil {
		return true
	}
	switch ps := m.GetPathSpecifier().(type) {
	case *routev1.RouteMatch_Prefix:
		return strings.HasPrefix(path, ps.Prefix)
	case *routev1.RouteMatch_Path:
		return path == ps.Path
	default:
		return true
	}
}

func selectWeightedCluster(wc *routev1.WeightedCluster, ticket uint64) string {
	clusters := wc.GetClusters()
	if len(clusters) == 0 {
		return ""
	}
	var total uint64
	for _, c := range clusters {
		w := uint64(c.GetWeight().GetValue())
		if w == 0 {
			w = 1
		}
		total += w
	}
	slot := ticket % total
	var acc uint64
	for _, c := range clusters {
		w := uint64(c.GetWeight().GetValue())
		if w == 0 {
			w = 1
		}
		acc += w
		if slot < acc {
			return c.GetName()
		}
	}
	return clusters[0].GetName()
}

func sortedKeys(m map[string]struct{}) []string {
	if len(m) == 0 {
		return nil
	}
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
