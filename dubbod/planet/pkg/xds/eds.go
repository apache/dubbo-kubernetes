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

package xds

import (
	"fmt"

	"github.com/apache/dubbo-kubernetes/dubbod/planet/pkg/model"
	"github.com/apache/dubbo-kubernetes/dubbod/planet/pkg/util/protoconv"
	"github.com/apache/dubbo-kubernetes/dubbod/planet/pkg/xds/endpoints"
	"github.com/apache/dubbo-kubernetes/pkg/config/schema/kind"
	"github.com/apache/dubbo-kubernetes/pkg/util/sets"
	endpoint "github.com/envoyproxy/go-control-plane/envoy/config/endpoint/v3"
	discovery "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v3"
)

type EdsGenerator struct {
	Cache         model.XdsCache
	EndpointIndex *model.EndpointIndex
}

var _ model.XdsDeltaResourceGenerator = &EdsGenerator{}

var skippedEdsConfigs = sets.New(
	kind.VirtualService,
	kind.PeerAuthentication,
	kind.Secret,
)

func edsNeedsPush(req *model.PushRequest, proxy *model.Proxy) bool {
	if res, ok := xdsNeedsPush(req, proxy); ok {
		return res
	}
	// For proxy requests (ProxyRequest reason), we should always push EDS
	// that request specific EDS resources.
	if req.Reason != nil && req.Reason.Has(model.ProxyRequest) {
		log.Debugf("edsNeedsPush: ProxyRequest detected, pushing EDS even without config updates")
		return true
	}
	for config := range req.ConfigsUpdated {
		if !skippedEdsConfigs.Contains(config.Kind) {
			return true
		}
	}
	return false
}

func (eds *EdsGenerator) Generate(proxy *model.Proxy, w *model.WatchedResource, req *model.PushRequest) (model.Resources, model.XdsLogDetails, error) {
	log.Debugf("EDS Generate: proxy=%s, watchedResources=%d, req.Full=%v, req.ConfigsUpdated=%v",
		proxy.ID, len(w.ResourceNames), req.Full, len(req.ConfigsUpdated))

	if !edsNeedsPush(req, proxy) {
		log.Debugf("EDS Generate: edsNeedsPush returned false, skipping EDS generation for proxy=%s", proxy.ID)
		return nil, model.DefaultXdsLogDetails, nil
	}
	resources, logDetails := eds.buildEndpoints(proxy, req, w)
	log.Debugf("EDS Generate: built %d resources for proxy=%s (%s)",
		len(resources), proxy.ID, logDetails.AdditionalInfo)
	return resources, logDetails, nil
}

func (eds *EdsGenerator) GenerateDeltas(proxy *model.Proxy, req *model.PushRequest, w *model.WatchedResource) (model.Resources, model.DeletedResources, model.XdsLogDetails, bool, error) {
	if !edsNeedsPush(req, proxy) {
		return nil, nil, model.DefaultXdsLogDetails, false, nil
	}
	if !shouldUseDeltaEds(req) {
		resources, logDetails := eds.buildEndpoints(proxy, req, w)
		return resources, nil, logDetails, false, nil
	}

	resources, removed, logs := eds.buildDeltaEndpoints(proxy, req, w)
	return resources, removed, logs, true, nil
}

func (eds *EdsGenerator) buildEndpoints(proxy *model.Proxy, req *model.PushRequest, w *model.WatchedResource) (model.Resources, model.XdsLogDetails) {
	var edsUpdatedServices map[string]struct{}
	// canSendPartialFullPushes determines if we can send a partial push (ie a subset of known CLAs).
	// This is safe when only Services has changed, as this implies that only the CLAs for the
	// associated Service changed. Note when a multi-network Service changes it triggers a push with
	// ConfigsUpdated=ALL, so in this case we would not enable a partial push.
	// Despite this code existing on the SotW code path, sending these partial pushes is still allowed;
	// see https://www.envoyproxy.io/docs/envoy/latest/api-docs/xds_protocol#grouping-resources-into-responses
	if !req.Full || canSendPartialFullPushes(req) {
		edsUpdatedServices = model.ConfigNamesOfKind(req.ConfigsUpdated, kind.ServiceEntry)
		if len(edsUpdatedServices) > 0 {
			log.Debugf("buildEndpoints: edsUpdatedServices count=%d (Full=%v)", len(edsUpdatedServices), req.Full)
		}
	}
	var resources model.Resources
	empty := 0
	cached := 0
	regenerated := 0
	for clusterName := range w.ResourceNames {
		hostname := model.ParseSubsetKeyHostname(clusterName)
		// Always check if this service was updated, even if edsUpdatedServices is nil
		// For incremental pushes, we need to check ConfigsUpdated directly to ensure cache is cleared
		serviceWasUpdated := false
		if req.ConfigsUpdated != nil {
			configKeys := make([]string, 0, len(req.ConfigsUpdated))
			for ckey := range req.ConfigsUpdated {
				configKeys = append(configKeys, fmt.Sprintf("%s/%s/%s", ckey.Kind, ckey.Namespace, ckey.Name))
				if ckey.Kind == kind.ServiceEntry {
					// Match ConfigsUpdated.Name with hostname
					// Both should be FQDN (e.g., "consumer.grpc-app.svc.cluster.local")
					if ckey.Name == hostname {
						serviceWasUpdated = true
						log.Debugf("buildEndpoints: service %s was updated, forcing regeneration", hostname)
						break
					}
				}
			}
		}

		// For proxyless gRPC, we MUST always process all watched clusters
		// This ensures clients receive EDS updates even when endpoints become available or unavailable
		if edsUpdatedServices != nil {
			if _, ok := edsUpdatedServices[hostname]; !ok {
				// Cluster was not in edsUpdatedServices
				if !serviceWasUpdated {
					// For proxyless gRPC, always process all clusters to ensure EDS is up-to-date
					// even if the service wasn't explicitly updated in this push
					if proxy.IsProxylessGrpc() {
						log.Debugf("buildEndpoints: proxyless gRPC, processing cluster %s even though not in edsUpdatedServices (hostname=%s)", clusterName, hostname)
						serviceWasUpdated = true
					} else {
						// For non-proxyless, skip if not updated
						continue
					}
				}
			} else {
				serviceWasUpdated = true
			}
		} else {
			// If edsUpdatedServices is nil, this is a full push - process all clusters
			serviceWasUpdated = true
		}
		builder := endpoints.NewEndpointBuilder(clusterName, proxy, req.Push)
		if builder == nil {
			continue
		}

		// For proxyless gRPC, we must always regenerate EDS when serviceWasUpdated is true
		// to ensure empty endpoints are sent when they become unavailable, and endpoints are sent when available.
		// For incremental pushes (Full=false), we must always regenerate EDS
		// if ConfigsUpdated contains this service, because endpoint health status may have changed.
		// The cache may contain stale empty endpoints from when the endpoint was unhealthy.
		// For full pushes, we can use cache if the service was not updated.
		// If this is an incremental push and ConfigsUpdated contains any ServiceEntry,
		// we must check if it matches this hostname. If it does, force regeneration.
		shouldRegenerate := serviceWasUpdated
		if !shouldRegenerate && !req.Full && req.ConfigsUpdated != nil {
			// For incremental pushes, check if any ServiceEntry in ConfigsUpdated matches this hostname
			for ckey := range req.ConfigsUpdated {
				if ckey.Kind == kind.ServiceEntry && ckey.Name == hostname {
					shouldRegenerate = true
					log.Debugf("buildEndpoints: forcing regeneration for cluster %s (hostname=%s) due to ConfigsUpdated", clusterName, hostname)
					break
				}
			}
		}

		// For proxyless gRPC, if serviceWasUpdated is true, we must regenerate
		if shouldRegenerate && proxy.IsProxylessGrpc() {
			log.Debugf("buildEndpoints: proxyless gRPC, forcing regeneration for cluster %s (hostname=%s, serviceWasUpdated=%v)", clusterName, hostname, serviceWasUpdated)
		}

		// Try to get from cache only if we don't need to regenerate
		if !shouldRegenerate && eds.Cache != nil {
			cachedEndpoint := eds.Cache.Get(builder)
			if cachedEndpoint != nil {
				resources = append(resources, cachedEndpoint)
				cached++
				continue
			}
		}

		// generate eds from beginning
		l := builder.BuildClusterLoadAssignment(eds.EndpointIndex)
		// NOTE: BuildClusterLoadAssignment never returns nil - it always returns either:
		// 1. A valid ClusterLoadAssignment with endpoints, or
		// 2. An empty ClusterLoadAssignment (via buildEmptyClusterLoadAssignment)
		// The empty ClusterLoadAssignment has an explicit empty Endpoints list,
		// and prevent "weighted-target: no targets to pick from" errors.
		// The nil check below is defensive programming, but should never be true in practice.
		if l == nil {
			// Defensive: Build an empty ClusterLoadAssignment if somehow nil is returned
			// This should never happen, but ensures we always send a valid response
			l = &endpoint.ClusterLoadAssignment{
				ClusterName: clusterName,
				Endpoints:   []*endpoint.LocalityLbEndpoints{}, // Empty endpoints list
			}
			log.Warnf("buildEndpoints: BuildClusterLoadAssignment returned nil for cluster %s (hostname: %s), created empty CLA", clusterName, hostname)
		}
		regenerated++

		if len(l.Endpoints) == 0 {
			empty++
			if proxy.IsProxylessGrpc() {
				log.Infof("buildEndpoints: proxyless gRPC, pushing empty EDS for cluster %s (hostname=%s, serviceWasUpdated=%v, shouldRegenerate=%v)",
					clusterName, hostname, serviceWasUpdated, shouldRegenerate)
			}
		} else {
			// Count total endpoints in the ClusterLoadAssignment
			totalEndpointsInCLA := 0
			for _, localityLbEp := range l.Endpoints {
				totalEndpointsInCLA += len(localityLbEp.LbEndpoints)
			}
			if proxy.IsProxylessGrpc() {
				log.Infof("buildEndpoints: proxyless gRPC, pushing EDS for cluster %s (hostname=%s, endpoints=%d, serviceWasUpdated=%v, shouldRegenerate=%v)",
					clusterName, hostname, totalEndpointsInCLA, serviceWasUpdated, shouldRegenerate)
			}
		}
		resource := &discovery.Resource{
			Name:     l.ClusterName,
			Resource: protoconv.MessageToAny(l),
		}
		resources = append(resources, resource)
		if eds.Cache != nil {
			eds.Cache.Add(builder, req, resource)
		}
	}
	return resources, model.XdsLogDetails{
		Incremental:    len(edsUpdatedServices) != 0,
		AdditionalInfo: fmt.Sprintf("empty:%v cached:%v/%v", empty, cached, cached+regenerated),
	}
}

// buildDeltaEndpoints builds endpoints for delta XDS
func (eds *EdsGenerator) buildDeltaEndpoints(proxy *model.Proxy, req *model.PushRequest, w *model.WatchedResource) (model.Resources, []string, model.XdsLogDetails) {
	edsUpdatedServices := model.ConfigNamesOfKind(req.ConfigsUpdated, kind.ServiceEntry)
	var resources model.Resources
	var removed []string
	empty := 0
	cached := 0
	regenerated := 0

	for clusterName := range w.ResourceNames {
		hostname := model.ParseSubsetKeyHostname(clusterName)

		// For proxyless gRPC, we MUST always process all watched clusters
		// This ensures clients receive EDS updates even when endpoints become available or unavailable
		serviceWasUpdated := false
		if len(edsUpdatedServices) > 0 {
			if _, ok := edsUpdatedServices[hostname]; !ok {
				// Cluster was not in edsUpdatedServices
				// For proxyless gRPC, always process all clusters to ensure EDS is up-to-date
				// even if the service wasn't explicitly updated in this push
				if proxy.IsProxylessGrpc() {
					log.Debugf("buildDeltaEndpoints: proxyless gRPC, processing cluster %s even though not in edsUpdatedServices (hostname=%s)", clusterName, hostname)
					serviceWasUpdated = true
				} else {
					// For non-proxyless, skip if not updated
					continue
				}
			} else {
				serviceWasUpdated = true
			}
		} else {
			// If edsUpdatedServices is nil, this is a full push - process all clusters
			serviceWasUpdated = true
		}

		builder := endpoints.NewEndpointBuilder(clusterName, proxy, req.Push)
		if builder == nil {
			removed = append(removed, clusterName)
			continue
		}

		// if a service is not found, it means the cluster is removed
		if !builder.ServiceFound() {
			removed = append(removed, clusterName)
			continue
		}

		// For proxyless gRPC, we must always regenerate EDS when serviceWasUpdated is true
		// to ensure empty endpoints are sent when they become unavailable, and endpoints are sent when available.
		// For incremental pushes (Full=false), we must always regenerate EDS
		// if ConfigsUpdated contains this service, because endpoint health status may have changed.
		// The cache may contain stale empty endpoints from when the endpoint was unhealthy.
		shouldRegenerate := serviceWasUpdated
		if !shouldRegenerate && !req.Full && req.ConfigsUpdated != nil {
			// For incremental pushes, check if any ServiceEntry in ConfigsUpdated matches this hostname
			for ckey := range req.ConfigsUpdated {
				if ckey.Kind == kind.ServiceEntry && ckey.Name == hostname {
					shouldRegenerate = true
					log.Debugf("buildDeltaEndpoints: forcing regeneration for cluster %s (hostname=%s) due to ConfigsUpdated", clusterName, hostname)
					break
				}
			}
		}

		// For proxyless gRPC, if serviceWasUpdated is true, we must regenerate
		if shouldRegenerate && proxy.IsProxylessGrpc() {
			log.Debugf("buildDeltaEndpoints: proxyless gRPC, forcing regeneration for cluster %s (hostname=%s, serviceWasUpdated=%v)", clusterName, hostname, serviceWasUpdated)
		}

		// Try to get from cache only if we don't need to regenerate
		if !shouldRegenerate && eds.Cache != nil {
			cachedEndpoint := eds.Cache.Get(builder)
			if cachedEndpoint != nil {
				resources = append(resources, cachedEndpoint)
				cached++
				continue
			}
		}

		// generate new eds cache
		l := builder.BuildClusterLoadAssignment(eds.EndpointIndex)
		// NOTE: BuildClusterLoadAssignment never returns nil - it always returns either:
		// 1. A valid ClusterLoadAssignment with endpoints, or
		// 2. An empty ClusterLoadAssignment (via buildEmptyClusterLoadAssignment)
		// The empty ClusterLoadAssignment has an explicit empty Endpoints list,
		// and prevent "weighted-target: no targets to pick from" errors.
		// The nil check below is defensive programming, but should never be true in practice.
		if l == nil {
			// Defensive: Build an empty ClusterLoadAssignment if somehow nil is returned
			// This should never happen, but ensures we always send a valid response
			l = &endpoint.ClusterLoadAssignment{
				ClusterName: clusterName,
				Endpoints:   []*endpoint.LocalityLbEndpoints{}, // Empty endpoints list
			}
			log.Warnf("buildDeltaEndpoints: BuildClusterLoadAssignment returned nil for cluster %s (hostname: %s), created empty CLA", clusterName, hostname)
		}
		regenerated++

		if len(l.Endpoints) == 0 {
			empty++
			if proxy.IsProxylessGrpc() {
				log.Infof("buildDeltaEndpoints: proxyless gRPC, pushing empty EDS for cluster %s (hostname=%s, serviceWasUpdated=%v, shouldRegenerate=%v)",
					clusterName, hostname, serviceWasUpdated, shouldRegenerate)
			}
		} else {
			// Count total endpoints in the ClusterLoadAssignment
			totalEndpointsInCLA := 0
			for _, localityLbEp := range l.Endpoints {
				totalEndpointsInCLA += len(localityLbEp.LbEndpoints)
			}
			if proxy.IsProxylessGrpc() {
				log.Infof("buildDeltaEndpoints: proxyless gRPC, pushing EDS for cluster %s (hostname=%s, endpoints=%d, serviceWasUpdated=%v, shouldRegenerate=%v)",
					clusterName, hostname, totalEndpointsInCLA, serviceWasUpdated, shouldRegenerate)
			}
		}

		// Always send ClusterLoadAssignment, even if empty
		// Removing the cluster from the response causes "weighted-target: no targets to pick from" errors
		// because the client never receives an update indicating endpoints are empty
		resource := &discovery.Resource{
			Name:     l.ClusterName,
			Resource: protoconv.MessageToAny(l),
		}
		resources = append(resources, resource)
		if eds.Cache != nil {
			eds.Cache.Add(builder, req, resource)
		}
	}
	return resources, removed, model.XdsLogDetails{
		Incremental:    len(edsUpdatedServices) != 0,
		AdditionalInfo: fmt.Sprintf("empty:%v cached:%v/%v", empty, cached, cached+regenerated),
	}
}

// canSendPartialFullPushes checks if a request contains *only* endpoints updates except `skippedEdsConfigs`.
// This allows us to perform more efficient pushes where we only update the endpoints that did change.
func canSendPartialFullPushes(req *model.PushRequest) bool {
	// If we don't know what configs are updated, just send a full push
	if req.Forced {
		return false
	}
	for cfg := range req.ConfigsUpdated {
		if skippedEdsConfigs.Contains(cfg.Kind) {
			// the updated config does not impact EDS, skip it
			// this happens when push requests are merged due to debounce
			continue
		}
		if cfg.Kind != kind.ServiceEntry {
			return false
		}
	}
	return true
}

func shouldUseDeltaEds(req *model.PushRequest) bool {
	if !req.Full {
		return false
	}
	return canSendPartialFullPushes(req)
}
