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

	"github.com/apache/dubbo-kubernetes/dubbod/discovery/pkg/model"
	"github.com/apache/dubbo-kubernetes/dubbod/discovery/pkg/util/protoconv"
	"github.com/apache/dubbo-kubernetes/dubbod/discovery/pkg/xds/endpoints"
	"github.com/apache/dubbo-kubernetes/pkg/config/schema/kind"
	"github.com/apache/dubbo-kubernetes/pkg/util/sets"
	endpoint "github.com/kdubbo/xds-api/endpoint/v1"
	discovery "github.com/kdubbo/xds-api/service/discovery/v1"
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
		log.Debugf("ProxyRequest detected, pushing EDS even without config updates")
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
	log.Debugf("proxy=%s, watchedResources=%d, req.Full=%v, req.ConfigsUpdated=%v",
		proxy.ID, len(w.ResourceNames), req.Full, len(req.ConfigsUpdated))

	if !edsNeedsPush(req, proxy) {
		log.Debugf("edsNeedsPush returned false, skipping EDS generation for proxy=%s", proxy.ID)
		return nil, model.DefaultXdsLogDetails, nil
	}
	resources, logDetails := eds.buildEndpoints(proxy, req, w)
	log.Debugf("built %d resources for proxy=%s (%s)",
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
	edsUpdatedServices, targetedServices := edsUpdatedServicesForRequest(req)
	if targetedServices {
		log.Debugf("edsUpdatedServices count=%d (Full=%v)", len(edsUpdatedServices), req.Full)
	}
	var resources model.Resources
	empty := 0
	cached := 0
	regenerated := 0
	for clusterName := range w.ResourceNames {
		hostname := model.ParseSubsetKeyHostname(clusterName)
		serviceWasUpdated := true
		if targetedServices {
			if _, ok := edsUpdatedServices[hostname]; !ok {
				continue
			}
			log.Debugf("service %s was updated, forcing regeneration", hostname)
		}
		builder := endpoints.NewEndpointBuilder(clusterName, proxy, req.Push)
		if builder == nil {
			continue
		}

		// Targeted endpoint updates must regenerate the matching service so clients receive empty
		// endpoint lists as well as newly healthy endpoints.
		shouldRegenerate := serviceWasUpdated
		if !shouldRegenerate && !req.Full && req.ConfigsUpdated != nil {
			for ckey := range req.ConfigsUpdated {
				if ckey.Name == hostname {
					shouldRegenerate = true
					log.Debugf("forcing regeneration for cluster %s (hostname=%s) due to ConfigsUpdated", clusterName, hostname)
					break
				}
			}
		}

		// For proxyless gRPC, if serviceWasUpdated is true, we must regenerate
		if shouldRegenerate && proxy.IsProxylessGrpc() {
			log.Debugf("proxyless gRPC, forcing regeneration for cluster %s (hostname=%s, serviceWasUpdated=%v)", clusterName, hostname, serviceWasUpdated)
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
			log.Warnf("BuildClusterLoadAssignment returned nil for cluster %s (hostname: %s), created empty CLA", clusterName, hostname)
		}
		regenerated++

		if len(l.Endpoints) == 0 {
			empty++
			if proxy.IsProxylessGrpc() {
				log.Debugf("proxyless gRPC, pushing empty EDS for cluster %s (hostname=%s, serviceWasUpdated=%v, shouldRegenerate=%v)",
					clusterName, hostname, serviceWasUpdated, shouldRegenerate)
			}
		} else {
			// Count total endpoints in the ClusterLoadAssignment
			totalEndpointsInCLA := 0
			for _, localityLbEp := range l.Endpoints {
				totalEndpointsInCLA += len(localityLbEp.LbEndpoints)
			}
			if proxy.IsProxylessGrpc() {
				log.Debugf("proxyless gRPC, pushing EDS for cluster %s (hostname=%s, endpoints=%d, serviceWasUpdated=%v, shouldRegenerate=%v)",
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
		Incremental:    targetedServices,
		AdditionalInfo: fmt.Sprintf("empty:%v cached:%v/%v", empty, cached, cached+regenerated),
	}
}

// buildDeltaEndpoints builds endpoints for delta XDS
func (eds *EdsGenerator) buildDeltaEndpoints(proxy *model.Proxy, req *model.PushRequest, w *model.WatchedResource) (model.Resources, []string, model.XdsLogDetails) {
	edsUpdatedServices, targetedServices := edsUpdatedServicesForRequest(req)
	var resources model.Resources
	var removed []string
	empty := 0
	cached := 0
	regenerated := 0

	for clusterName := range w.ResourceNames {
		hostname := model.ParseSubsetKeyHostname(clusterName)

		serviceWasUpdated := true
		if targetedServices {
			if _, ok := edsUpdatedServices[hostname]; !ok {
				continue
			}
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

		// Targeted endpoint updates must regenerate the matching service so clients receive empty
		// endpoint lists as well as newly healthy endpoints.
		shouldRegenerate := serviceWasUpdated
		if !shouldRegenerate && !req.Full && req.ConfigsUpdated != nil {
			for ckey := range req.ConfigsUpdated {
				if ckey.Name == hostname {
					shouldRegenerate = true
					log.Debugf("forcing regeneration for cluster %s (hostname=%s) due to ConfigsUpdated", clusterName, hostname)
					break
				}
			}
		}

		// For proxyless gRPC, if serviceWasUpdated is true, we must regenerate
		if shouldRegenerate && proxy.IsProxylessGrpc() {
			log.Debugf("proxyless gRPC, forcing regeneration for cluster %s (hostname=%s, serviceWasUpdated=%v)", clusterName, hostname, serviceWasUpdated)
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
			log.Warnf("BuildClusterLoadAssignment returned nil for cluster %s (hostname: %s), created empty CLA", clusterName, hostname)
		}
		regenerated++

		if len(l.Endpoints) == 0 {
			empty++
			if proxy.IsProxylessGrpc() {
				log.Debugf("proxyless gRPC, pushing empty EDS for cluster %s (hostname=%s, serviceWasUpdated=%v, shouldRegenerate=%v)",
					clusterName, hostname, serviceWasUpdated, shouldRegenerate)
			}
		} else {
			// Count total endpoints in the ClusterLoadAssignment
			totalEndpointsInCLA := 0
			for _, localityLbEp := range l.Endpoints {
				totalEndpointsInCLA += len(localityLbEp.LbEndpoints)
			}
			if proxy.IsProxylessGrpc() {
				log.Debugf("proxyless gRPC, pushing EDS for cluster %s (hostname=%s, endpoints=%d, serviceWasUpdated=%v, shouldRegenerate=%v)",
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
		Incremental:    targetedServices,
		AdditionalInfo: fmt.Sprintf("empty:%v cached:%v/%v", empty, cached, cached+regenerated),
	}
}

func edsUpdatedServicesForRequest(req *model.PushRequest) (sets.String, bool) {
	if req == nil || len(req.ConfigsUpdated) == 0 {
		return nil, false
	}
	services := sets.New[string]()
	for cfg := range req.ConfigsUpdated {
		if skippedEdsConfigs.Contains(cfg.Kind) {
			continue
		}
		if cfg.Kind != kind.Service || cfg.Name == "" {
			return nil, false
		}
		services.Insert(cfg.Name)
	}
	return services, len(services) > 0
}

// canSendPartialFullPushes checks if a request contains *only* endpoints updates except `skippedEdsConfigs`.
// This allows us to perform more efficient pushes where we only update the endpoints that did change.
func canSendPartialFullPushes(req *model.PushRequest) bool {
	if req.Forced {
		return false
	}
	_, targetedServices := edsUpdatedServicesForRequest(req)
	return targetedServices
}

func shouldUseDeltaEds(req *model.PushRequest) bool {
	if !req.Full {
		return false
	}
	return canSendPartialFullPushes(req)
}
