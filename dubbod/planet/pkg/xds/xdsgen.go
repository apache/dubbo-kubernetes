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

package xds

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/apache/dubbo-kubernetes/dubbod/planet/pkg/model"
	"github.com/apache/dubbo-kubernetes/dubbod/planet/pkg/networking/util"
	v3 "github.com/apache/dubbo-kubernetes/dubbod/planet/pkg/xds/v3"
	"github.com/apache/dubbo-kubernetes/pkg/config/host"
	"github.com/apache/dubbo-kubernetes/pkg/env"
	"github.com/apache/dubbo-kubernetes/pkg/lazy"
	"github.com/apache/dubbo-kubernetes/pkg/util/sets"
	dubboversion "github.com/apache/dubbo-kubernetes/pkg/version"
	"github.com/apache/dubbo-kubernetes/pkg/xds"
	cluster "github.com/envoyproxy/go-control-plane/envoy/config/cluster/v3"
	core "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	listener "github.com/envoyproxy/go-control-plane/envoy/config/listener/v3"
	hcmv3 "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/network/http_connection_manager/v3"
	discovery "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v3"
)

type DubboControlPlaneInstance struct {
	Component string
	ID        string
	Info      dubboversion.BuildInfo
}

var controlPlane = lazy.New(func() (*core.ControlPlane, error) {
	podName := env.Register("POD_NAME", "", "").Get()
	byVersion, err := json.Marshal(DubboControlPlaneInstance{
		Component: "dubbod",
		ID:        podName,
		Info:      dubboversion.Info,
	})
	if err != nil {
		log.Warnf("XDS: Could not serialize control plane id: %v", err)
	}
	return &core.ControlPlane{Identifier: string(byVersion)}, nil
})

func ControlPlane(typ string) *core.ControlPlane {
	if typ != TypeDebugSyncronization {
		// Currently only TypeDebugSyncronization utilizes this so don't both sending otherwise
		return nil
	}
	cp, _ := controlPlane.Get()
	return cp
}

func xdsNeedsPush(req *model.PushRequest, _ *model.Proxy) (needsPush, definitive bool) {
	if req == nil {
		return true, true
	}
	if req.Forced {
		return true, true
	}
	return false, false
}
func (s *DiscoveryServer) pushXds(con *Connection, w *model.WatchedResource, req *model.PushRequest) error {
	if w == nil {
		return nil
	}
	gen := s.findGenerator(w.TypeUrl, con)
	if gen == nil {
		return nil
	}

	// CRITICAL FIX: For proxyless gRPC, handle wildcard (empty ResourceNames) requests correctly
	// When client sends empty ResourceNames after receiving specific resources, it's likely an ACK
	// We should NOT generate all resources, but instead return the last sent resources
	// However, for initial wildcard requests, we need to extract resource names from parent resources
	var requestedResourceNames sets.String
	var useLastSentResources bool
	if con.proxy.IsProxylessGrpc() {
		// Check if this is a wildcard request (empty ResourceNames) but we have previously sent resources
		if len(w.ResourceNames) == 0 && w.NonceSent != "" {
			// This is likely an ACK after receiving specific resources
			// Use the last sent resources instead of generating all
			useLastSentResources = true
			// Get the last sent resource names from WatchedResource
			// We'll populate this from the last sent resources after generation
			log.Debugf("pushXds: proxyless gRPC wildcard request with NonceSent=%s, will use last sent resources", w.NonceSent)
		} else if len(w.ResourceNames) == 0 && w.NonceSent == "" {
			// CRITICAL FIX: Initial wildcard request - need to extract resource names from parent resources
			// For CDS: extract cluster names from LDS
			// For EDS: extract cluster names from CDS
			if w.TypeUrl == v3.ClusterType {
				// Extract cluster names from LDS response
				ldsWatched := con.proxy.GetWatchedResource(v3.ListenerType)
				if ldsWatched != nil && ldsWatched.NonceSent != "" {
					// LDS has been sent, extract cluster names from it
					// We need to regenerate LDS to extract cluster names, or store them
					// For now, let's extract from the proxy's ServiceTargets or from LDS generation
					log.Debugf("pushXds: CDS wildcard request, extracting cluster names from LDS")
					// Create a temporary request to generate LDS and extract cluster names
					ldsReq := &model.PushRequest{
						Full:   true,
						Push:   req.Push,
						Reason: model.NewReasonStats(model.DependentResource),
						Start:  con.proxy.LastPushTime,
						Forced: false,
					}
					ldsGen := s.findGenerator(v3.ListenerType, con)
					if ldsGen != nil {
						ldsRes, _, _ := ldsGen.Generate(con.proxy, ldsWatched, ldsReq)
						if len(ldsRes) > 0 {
							clusterNames := extractClusterNamesFromLDS(ldsRes)
							if len(clusterNames) > 0 {
								w.ResourceNames = sets.New(clusterNames...)
								requestedResourceNames = sets.New[string]()
								requestedResourceNames.InsertAll(clusterNames...)
								log.Debugf("pushXds: extracted %d cluster names from LDS: %v", len(clusterNames), clusterNames)
							}
						}
					}
				}
			} else if w.TypeUrl == v3.EndpointType {
				// Extract cluster names from CDS response
				cdsWatched := con.proxy.GetWatchedResource(v3.ClusterType)
				if cdsWatched != nil && cdsWatched.NonceSent != "" {
					// CDS has been sent, extract EDS cluster names from it
					log.Debugf("pushXds: EDS wildcard request, extracting cluster names from CDS")
					// Create a temporary request to generate CDS and extract EDS cluster names
					cdsReq := &model.PushRequest{
						Full:   true,
						Push:   req.Push,
						Reason: model.NewReasonStats(model.DependentResource),
						Start:  con.proxy.LastPushTime,
						Forced: false,
					}
					cdsGen := s.findGenerator(v3.ClusterType, con)
					if cdsGen != nil {
						cdsRes, _, _ := cdsGen.Generate(con.proxy, cdsWatched, cdsReq)
						if len(cdsRes) > 0 {
							edsClusterNames := extractEDSClusterNamesFromCDS(cdsRes)
							if len(edsClusterNames) > 0 {
								w.ResourceNames = sets.New(edsClusterNames...)
								requestedResourceNames = sets.New[string]()
								requestedResourceNames.InsertAll(edsClusterNames...)
								log.Debugf("pushXds: extracted %d EDS cluster names from CDS: %v", len(edsClusterNames), edsClusterNames)
							}
						}
					}
				}
			} else if w.TypeUrl == v3.RouteType {
				// CRITICAL FIX: Extract route names from LDS response for RDS wildcard requests
				// RDS is not a wildcard type, so when client sends empty ResourceNames,
				// we need to extract route names from LDS listeners that reference RDS
				ldsWatched := con.proxy.GetWatchedResource(v3.ListenerType)
				if ldsWatched != nil && ldsWatched.NonceSent != "" {
					// LDS has been sent, extract route names from it
					log.Debugf("pushXds: RDS wildcard request, extracting route names from LDS")
					// Create a temporary request to generate LDS and extract route names
					ldsReq := &model.PushRequest{
						Full:   true,
						Push:   req.Push,
						Reason: model.NewReasonStats(model.DependentResource),
						Start:  con.proxy.LastPushTime,
						Forced: false,
					}
					ldsGen := s.findGenerator(v3.ListenerType, con)
					if ldsGen != nil {
						ldsRes, _, _ := ldsGen.Generate(con.proxy, ldsWatched, ldsReq)
						if len(ldsRes) > 0 {
							routeNames := extractRouteNamesFromLDS(ldsRes)
							if len(routeNames) > 0 {
								w.ResourceNames = sets.New(routeNames...)
								requestedResourceNames = sets.New[string]()
								requestedResourceNames.InsertAll(routeNames...)
								log.Debugf("pushXds: extracted %d route names from LDS: %v", len(routeNames), routeNames)
							}
						}
					}
				}
			}
		} else if len(w.ResourceNames) > 0 {
			// Specific resource request
			requestedResourceNames = sets.New[string]()
			requestedResourceNames.InsertAll(w.ResourceNames.UnsortedList()...)
		}
	}

	// If delta is set, client is requesting new resources or removing old ones. We should just generate the
	// new resources it needs, rather than the entire set of known resources.
	// Note: we do not need to account for unsubscribed resources as these are handled by parent removal;
	// See https://www.envoyproxy.io/docs/envoy/latest/api-docs/xds_protocol#deleting-resources.
	// This means if there are only removals, we will not respond.
	var logFiltered string
	if !req.Delta.IsEmpty() && !con.proxy.IsProxylessGrpc() {
		logFiltered = " filtered:" + strconv.Itoa(len(w.ResourceNames)-len(req.Delta.Subscribed))
		w = &model.WatchedResource{
			TypeUrl:       w.TypeUrl,
			ResourceNames: req.Delta.Subscribed,
		}
	}

	// For proxyless gRPC wildcard requests with previous NonceSent, use last sent resources
	var res model.Resources
	var logdata model.XdsLogDetails
	var err error
	if useLastSentResources {
		// Don't generate new resources, return empty and we'll handle it below
		res = nil
		logdata = model.DefaultXdsLogDetails
		err = nil
	} else {
		res, logdata, err = gen.Generate(con.proxy, w, req)
	}

	info := ""
	if len(logdata.AdditionalInfo) > 0 {
		info = " " + logdata.AdditionalInfo
	}
	if len(logFiltered) > 0 {
		info += logFiltered
	}
	if err != nil {
		return err
	}

	// CRITICAL FIX: For proxyless gRPC wildcard requests with previous NonceSent, return last sent resources
	if useLastSentResources && res == nil {
		// This is a wildcard ACK - client is acknowledging previous push
		// We should NOT push again, as the client already has the resources
		// The ShouldRespond logic should have prevented this, but we handle it here as safety
		log.Debugf("pushXds: proxyless gRPC wildcard ACK with NonceSent=%s, skipping push (client already has resources from previous push)", w.NonceSent)
		return nil
	}

	if res == nil {
		return nil
	}

	// CRITICAL FIX: For proxyless gRPC, filter resources to only include requested ones
	// This prevents the push loop where client requests 1 resource but receives 13/14
	var filteredRes model.Resources
	if con.proxy.IsProxylessGrpc() {
		if len(requestedResourceNames) > 0 {
			// Filter to only requested resources
			filteredRes = make(model.Resources, 0, len(res))
			for _, r := range res {
				if requestedResourceNames.Contains(r.Name) {
					filteredRes = append(filteredRes, r)
				} else {
					log.Debugf("pushXds: filtering out unrequested resource %s for proxyless gRPC (requested: %v)", r.Name, requestedResourceNames.UnsortedList())
				}
			}
			if len(filteredRes) != len(res) {
				info += " filtered:" + strconv.Itoa(len(res)-len(filteredRes))
				res = filteredRes
			}
			// CRITICAL: If filtering resulted in 0 resources but client requested specific resources,
			// this means the requested resources don't exist. Don't send empty response to avoid loop.
			// Instead, log and return nil to prevent push.
			if len(res) == 0 && len(requestedResourceNames) > 0 {
				log.Warnf("pushXds: proxyless gRPC requested %d resources but none matched after filtering (requested: %v, generated before filter: %d). Skipping push to avoid loop.",
					len(requestedResourceNames), requestedResourceNames.UnsortedList(), len(filteredRes)+len(res))
				return nil
			}
		} else if len(w.ResourceNames) == 0 {
			// Wildcard request without previous NonceSent - this is initial request
			// Allow generating all resources for initial connection
			log.Debugf("pushXds: proxyless gRPC initial wildcard request, generating all resources")
		}
	}

	// CRITICAL: Never send empty response for proxyless gRPC - this causes push loops
	// If we have no resources to send, return nil instead of sending empty response
	if len(res) == 0 {
		log.Debugf("pushXds: no resources to send for %s (proxy: %s), skipping push", w.TypeUrl, con.proxy.ID)
		return nil
	}

	nonceValue := nonce(req.Push.PushVersion)
	resp := &discovery.DiscoveryResponse{
		ControlPlane: ControlPlane(w.TypeUrl),
		TypeUrl:      w.TypeUrl,
		VersionInfo:  req.Push.PushVersion,
		Nonce:        nonceValue,
		Resources:    xds.ResourcesToAny(res),
	}

	ptype := "PUSH"
	if logdata.Incremental {
		ptype = "PUSH INC"
	}

	if err := xds.Send(con, resp); err != nil {
		return err
	}

	// CRITICAL FIX: Update NonceSent after successfully sending the response
	// This is essential for tracking which nonce was sent and preventing push loops
	con.proxy.UpdateWatchedResource(w.TypeUrl, func(wr *model.WatchedResource) *model.WatchedResource {
		if wr == nil {
			return nil
		}
		wr.NonceSent = nonceValue
		// Also update ResourceNames to match what we actually sent (for proxyless gRPC)
		if con.proxy.IsProxylessGrpc() && res != nil {
			sentNames := sets.New[string]()
			for _, r := range res {
				sentNames.Insert(r.Name)
			}
			// Only update if we sent different resources than requested
			if requestedResourceNames != nil {
				// Compare sets by checking if they have the same size and all elements match
				if sentNames.Len() != requestedResourceNames.Len() {
					wr.ResourceNames = sentNames
				} else {
					// Check if all sent names are in requested names
					allMatch := true
					for name := range sentNames {
						if !requestedResourceNames.Contains(name) {
							allMatch = false
							break
						}
					}
					if !allMatch {
						wr.ResourceNames = sentNames
					}
				}
			}
		}
		return wr
	})

	switch {
	case !req.Full:
	default:
		// Log format matches Istio: "LDS: PUSH for node:xxx resources:1 size:342B"
		// CRITICAL: Always log resource names for better debugging, especially for proxyless gRPC
		resourceNamesStr := ""
		if len(res) > 0 {
			if len(res) <= 10 {
				// Log all resource names if there are few resources
				names := make([]string, 0, len(res))
				for _, r := range res {
					names = append(names, r.Name)
				}
				resourceNamesStr = fmt.Sprintf(" [%s]", strings.Join(names, ", "))
			} else {
				// If too many resources, show first 5 and count
				names := make([]string, 0, 5)
				for i := 0; i < 5 && i < len(res); i++ {
					names = append(names, res[i].Name)
				}
				resourceNamesStr = fmt.Sprintf(" [%s, ... and %d more]", strings.Join(names, ", "), len(res)-5)
			}
		} else {
			resourceNamesStr = " [empty]"
		}
		log.Infof("%s: %s for node:%s resources:%d size:%s%s%s", v3.GetShortType(w.TypeUrl), ptype, con.proxy.ID, len(res),
			util.ByteCount(ResourceSize(res)), info, resourceNamesStr)
	}

	// CRITICAL FIX: For proxyless gRPC, after pushing LDS with outbound listeners,
	// automatically trigger CDS and RDS push for the referenced clusters and routes
	// ONLY if this is a direct request push (not a push from pushConnection which would cause loops)
	// Only auto-push if CDS/RDS is not already being watched by the client (client will request it naturally)
	if w.TypeUrl == v3.ListenerType && con.proxy.IsProxylessGrpc() && len(res) > 0 {
		// Only auto-push CDS/RDS if this is a direct request (not a full push from pushConnection)
		// Check if this push was triggered by a direct client request using IsRequest()
		isDirectRequest := req.IsRequest()
		if isDirectRequest {
			clusterNames := extractClusterNamesFromLDS(res)
			routeNames := extractRouteNamesFromLDS(res)

			// Auto-push CDS for referenced clusters
			if len(clusterNames) > 0 {
				cdsWatched := con.proxy.GetWatchedResource(v3.ClusterType)
				// Only auto-push CDS if client hasn't already requested it
				// If client has already requested CDS (WatchedResource exists with ResourceNames),
				// the client's request will handle it, so we don't need to auto-push
				if cdsWatched == nil || cdsWatched.ResourceNames == nil || len(cdsWatched.ResourceNames) == 0 {
					// Client hasn't requested CDS yet, auto-push to ensure client gets the cluster config
					con.proxy.NewWatchedResource(v3.ClusterType, clusterNames)
					log.Debugf("pushXds: LDS push completed, auto-pushing CDS for clusters: %v", clusterNames)
					// Trigger CDS push directly without going through pushConnection to avoid loops
					cdsReq := &model.PushRequest{
						Full:   true,
						Push:   req.Push,
						Reason: model.NewReasonStats(model.ProxyRequest),
						Start:  con.proxy.LastPushTime,
						Forced: false,
					}
					if err := s.pushXds(con, con.proxy.GetWatchedResource(v3.ClusterType), cdsReq); err != nil {
						log.Warnf("pushXds: failed to push CDS after LDS: %v", err)
					}
				} else {
					// Client has already requested CDS, let the client's request handle it
					log.Debugf("pushXds: LDS push completed, client already requested CDS, skipping auto-push")
				}
			}

			// CRITICAL FIX: Auto-push RDS for referenced routes
			// This is essential for gRPC proxyless - client needs RDS to route traffic correctly
			if len(routeNames) > 0 {
				rdsWatched := con.proxy.GetWatchedResource(v3.RouteType)
				// Only auto-push RDS if client hasn't already requested it
				if rdsWatched == nil || rdsWatched.ResourceNames == nil || len(rdsWatched.ResourceNames) == 0 {
					// Client hasn't requested RDS yet, auto-push to ensure client gets the route config
					con.proxy.NewWatchedResource(v3.RouteType, routeNames)
					log.Debugf("pushXds: LDS push completed, auto-pushing RDS for routes: %v", routeNames)
					// Trigger RDS push directly without going through pushConnection to avoid loops
					rdsReq := &model.PushRequest{
						Full:   true,
						Push:   req.Push,
						Reason: model.NewReasonStats(model.ProxyRequest),
						Start:  con.proxy.LastPushTime,
						Forced: false,
					}
					if err := s.pushXds(con, con.proxy.GetWatchedResource(v3.RouteType), rdsReq); err != nil {
						log.Warnf("pushXds: failed to push RDS after LDS: %v", err)
					}
				} else {
					// Check if any route names are missing from the watched set
					existingNames := rdsWatched.ResourceNames
					if existingNames == nil {
						existingNames = sets.New[string]()
					}
					hasNewRoutes := false
					for _, rn := range routeNames {
						if !existingNames.Contains(rn) {
							hasNewRoutes = true
							break
						}
					}
					if hasNewRoutes {
						// Update RDS watched resource to include the new route names
						con.proxy.UpdateWatchedResource(v3.RouteType, func(wr *model.WatchedResource) *model.WatchedResource {
							if wr == nil {
								wr = &model.WatchedResource{TypeUrl: v3.RouteType, ResourceNames: sets.New[string]()}
							}
							existingNames := wr.ResourceNames
							if existingNames == nil {
								existingNames = sets.New[string]()
							}
							for _, rn := range routeNames {
								existingNames.Insert(rn)
							}
							wr.ResourceNames = existingNames
							return wr
						})
						log.Debugf("pushXds: LDS push completed, updating RDS watched resource with new routes: %v", routeNames)
						// Trigger RDS push for the new routes
						rdsReq := &model.PushRequest{
							Full:   true,
							Push:   req.Push,
							Reason: model.NewReasonStats(model.DependentResource),
							Start:  con.proxy.LastPushTime,
							Forced: false,
						}
						if err := s.pushXds(con, con.proxy.GetWatchedResource(v3.RouteType), rdsReq); err != nil {
							log.Warnf("pushXds: failed to push RDS after LDS: %v", err)
						}
					} else {
						log.Debugf("pushXds: LDS push completed, RDS routes already watched: %v", routeNames)
					}
				}
			}
		}
	}

	// CRITICAL FIX: For proxyless gRPC, after pushing CDS with EDS clusters,
	// we should NOT automatically push EDS before the client requests it.
	// The gRPC xDS client will automatically request EDS after receiving CDS with EDS clusters.
	// If we push EDS before the client requests it, the client will receive the response
	// without having a state, causing "no state exists for it" warnings and "weighted-target: no targets to pick from" errors.
	// Instead, we should:
	// 1. Update the watched resource to include the EDS cluster names (so when client requests EDS, we know what to send)
	// 2. Wait for the client to request EDS naturally
	// 3. When the client requests EDS, we will push it with the correct state
	if w.TypeUrl == v3.ClusterType && con.proxy.IsProxylessGrpc() && len(res) > 0 {
		// Extract EDS cluster names from CDS resources
		edsClusterNames := extractEDSClusterNamesFromCDS(res)
		if len(edsClusterNames) > 0 {
			edsWatched := con.proxy.GetWatchedResource(v3.EndpointType)
			if edsWatched == nil {
				// EDS not watched yet, create watched resource with cluster names
				// This ensures that when the client requests EDS, we know which clusters to send
				con.proxy.NewWatchedResource(v3.EndpointType, edsClusterNames)
				log.Debugf("pushXds: CDS push completed, created EDS watched resource for clusters: %v (waiting for client request)", edsClusterNames)
			} else {
				// Check if any cluster names are missing from the watched set
				existingNames := edsWatched.ResourceNames
				if existingNames == nil {
					existingNames = sets.New[string]()
				}
				hasNewClusters := false
				for _, cn := range edsClusterNames {
					if !existingNames.Contains(cn) {
						hasNewClusters = true
						break
					}
				}
				if hasNewClusters {
					// Update EDS watched resource to include the new cluster names
					con.proxy.UpdateWatchedResource(v3.EndpointType, func(wr *model.WatchedResource) *model.WatchedResource {
						if wr == nil {
							wr = &model.WatchedResource{TypeUrl: v3.EndpointType, ResourceNames: sets.New[string]()}
						}
						existingNames := wr.ResourceNames
						if existingNames == nil {
							existingNames = sets.New[string]()
						}
						for _, cn := range edsClusterNames {
							existingNames.Insert(cn)
						}
						wr.ResourceNames = existingNames
						return wr
					})
					log.Debugf("pushXds: CDS push completed, updated EDS watched resource with new clusters: %v (waiting for client request)", edsClusterNames)
				} else {
					log.Debugf("pushXds: CDS push completed, EDS clusters already watched: %v", edsClusterNames)
				}
			}
			// CRITICAL: Do NOT push EDS here - wait for the client to request it naturally
			// The gRPC xDS client will automatically request EDS after receiving CDS with EDS clusters
			// Pushing EDS before the client requests it causes "no state exists" warnings
		}
	}

	return nil
}

// extractEDSClusterNamesFromCDS extracts EDS cluster names from CDS cluster resources
// Only clusters with ClusterDiscoveryType=EDS are returned
func extractEDSClusterNamesFromCDS(clusters model.Resources) []string {
	clusterNames := sets.New[string]()
	for _, r := range clusters {
		// Unmarshal the cluster resource to check its type
		cl := &cluster.Cluster{}
		if err := r.Resource.UnmarshalTo(cl); err != nil {
			log.Debugf("extractEDSClusterNamesFromCDS: failed to unmarshal cluster %s: %v", r.Name, err)
			continue
		}
		// Check if this is an EDS cluster
		if cl.ClusterDiscoveryType != nil {
			if edsType, ok := cl.ClusterDiscoveryType.(*cluster.Cluster_Type); ok && edsType.Type == cluster.Cluster_EDS {
				// This is an EDS cluster, add to the list
				clusterNames.Insert(cl.Name)
			}
		}
	}
	return clusterNames.UnsortedList()
}

// extractClusterNamesFromLDS extracts cluster names referenced in LDS listener resources
// For outbound listeners with name format "hostname:port", the cluster name is "outbound|port||hostname"
func extractClusterNamesFromLDS(listeners model.Resources) []string {
	clusterNames := sets.New[string]()
	for _, r := range listeners {
		// Parse listener name to extract cluster name
		// Outbound listener format: "hostname:port" -> cluster: "outbound|port||hostname"
		// Inbound listener format: "xds.dubbo.apache.org/grpc/lds/inbound/[::]:port" -> no cluster
		listenerName := r.Name
		if strings.Contains(listenerName, ":") && !strings.HasPrefix(listenerName, "xds.dubbo.apache.org/grpc/lds/inbound/") {
			// This is an outbound listener
			parts := strings.Split(listenerName, ":")
			if len(parts) == 2 {
				hostname := parts[0]
				portStr := parts[1]
				port, err := strconv.Atoi(portStr)
				if err == nil {
					// Build cluster name: outbound|port||hostname
					clusterName := model.BuildSubsetKey(model.TrafficDirectionOutbound, "", host.Name(hostname), port)
					clusterNames.Insert(clusterName)
				}
			}
		}
	}
	return clusterNames.UnsortedList()
}

// extractRouteNamesFromLDS extracts route names referenced in LDS listener resources
// For outbound listeners with ApiListener, the route name is the RouteConfigName from Rds config
// Route name format is the same as cluster name: "outbound|port||hostname"
func extractRouteNamesFromLDS(listeners model.Resources) []string {
	routeNames := sets.New[string]()
	for _, r := range listeners {
		// Unmarshal the listener resource to extract route names
		ll := &listener.Listener{}
		if err := r.Resource.UnmarshalTo(ll); err != nil {
			log.Debugf("extractRouteNamesFromLDS: failed to unmarshal listener %s: %v", r.Name, err)
			continue
		}

		// Check if this listener has ApiListener (used by gRPC proxyless for outbound)
		if ll.ApiListener != nil && ll.ApiListener.ApiListener != nil {
			// Unmarshal ApiListener to get HttpConnectionManager
			hcm := &hcmv3.HttpConnectionManager{}
			if err := ll.ApiListener.ApiListener.UnmarshalTo(hcm); err != nil {
				log.Debugf("extractRouteNamesFromLDS: failed to unmarshal ApiListener for listener %s: %v", r.Name, err)
				continue
			}

			// Check if HttpConnectionManager uses RDS
			if hcm.RouteSpecifier != nil {
				if rds, ok := hcm.RouteSpecifier.(*hcmv3.HttpConnectionManager_Rds); ok && rds.Rds != nil {
					// Found RDS reference, extract route name
					routeName := rds.Rds.RouteConfigName
					if routeName != "" {
						routeNames.Insert(routeName)
						log.Debugf("extractRouteNamesFromLDS: found route name %s from listener %s", routeName, r.Name)
					}
				}
			}
		}

		// Fallback: If ApiListener extraction failed, try to extract from listener name
		// This is a backup method in case the listener structure is different
		// Outbound listener format: "hostname:port" -> route: "outbound|port||hostname"
		listenerName := r.Name
		if strings.Contains(listenerName, ":") && !strings.HasPrefix(listenerName, "xds.dubbo.apache.org/grpc/lds/inbound/") {
			// This is an outbound listener
			parts := strings.Split(listenerName, ":")
			if len(parts) == 2 {
				hostname := parts[0]
				portStr := parts[1]
				port, err := strconv.Atoi(portStr)
				if err == nil {
					// Build route name: outbound|port||hostname (same format as cluster name)
					routeName := model.BuildSubsetKey(model.TrafficDirectionOutbound, "", host.Name(hostname), port)
					routeNames.Insert(routeName)
					log.Debugf("extractRouteNamesFromLDS: extracted route name %s from listener name %s", routeName, listenerName)
				}
			}
		}
	}
	return routeNames.UnsortedList()
}

func ResourceSize(r model.Resources) int {
	size := 0
	for _, r := range r {
		size += len(r.Resource.Value)
	}
	return size
}

func (s *DiscoveryServer) findGenerator(typeURL string, con *Connection) model.XdsResourceGenerator {
	if g, f := s.Generators[con.proxy.Metadata.Generator+"/"+typeURL]; f {
		return g
	}
	if g, f := s.Generators[typeURL]; f {
		return g
	}

	// XdsResourceGenerator is the default generator for this connection. We want to allow
	// some types to use custom generators - for example EDS.
	g := con.proxy.XdsResourceGenerator
	if g == nil {
		if strings.HasPrefix(typeURL, TypeDebugPrefix) {
			g = s.Generators["event"]
		} else {
			// TODO move this to just directly using the resource TypeUrl
			g = s.Generators["api"] // default to "MCP" generators - any type supported by store
		}
	}
	return g
}

// atMostNJoin joins up to N items from data, appending "and X others" if more exist
func atMostNJoin(data []string, limit int) string {
	if limit == 0 || limit == 1 {
		// Assume limit >1, but make sure we don't crash if someone does pass those
		return strings.Join(data, ", ")
	}
	if len(data) == 0 {
		return ""
	}
	if len(data) < limit {
		return strings.Join(data, ", ")
	}
	return strings.Join(data[:limit-1], ", ") + fmt.Sprintf(", and %d others", len(data)-limit+1)
}
