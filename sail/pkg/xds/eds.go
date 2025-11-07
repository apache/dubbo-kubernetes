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
	"fmt"

	"github.com/apache/dubbo-kubernetes/pkg/config/schema/kind"
	"github.com/apache/dubbo-kubernetes/pkg/util/sets"
	"github.com/apache/dubbo-kubernetes/sail/pkg/model"
	"github.com/apache/dubbo-kubernetes/sail/pkg/util/protoconv"
	"github.com/apache/dubbo-kubernetes/sail/pkg/xds/endpoints"
	endpoint "github.com/envoyproxy/go-control-plane/envoy/config/endpoint/v3"
	discovery "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v3"
	"k8s.io/klog/v2"
)

var _ model.XdsDeltaResourceGenerator = &EdsGenerator{}

type EdsGenerator struct {
	Cache         model.XdsCache
	EndpointIndex *model.EndpointIndex
}

func (s *DiscoveryServer) ServiceUpdate(shard model.ShardKey, hostname string, namespace string, event model.Event) {
	if event == model.EventDelete {
		s.Env.EndpointIndex.DeleteServiceShard(shard, hostname, namespace, false)
	} else {
	}
}

func (s *DiscoveryServer) EDSUpdate(shard model.ShardKey, serviceName string, namespace string,
	dubboEndpoints []*model.DubboEndpoint,
) {
	pushType := s.Env.EndpointIndex.UpdateServiceEndpoints(shard, serviceName, namespace, dubboEndpoints, true)
	if pushType == model.IncrementalPush || pushType == model.FullPush {
		s.ConfigUpdate(&model.PushRequest{
			Full:           pushType == model.FullPush,
			ConfigsUpdated: sets.New(model.ConfigKey{Kind: kind.ServiceEntry, Name: serviceName, Namespace: namespace}),
			Reason:         model.NewReasonStats(model.EndpointUpdate),
		})
	}
}

func (s *DiscoveryServer) EDSCacheUpdate(shard model.ShardKey, serviceName string, namespace string,
	istioEndpoints []*model.DubboEndpoint,
) {
	s.Env.EndpointIndex.UpdateServiceEndpoints(shard, serviceName, namespace, istioEndpoints, false)
}

var skippedEdsConfigs = sets.New(
	kind.VirtualService,
	kind.RequestAuthentication,
	kind.PeerAuthentication,
	kind.Secret,
	kind.DNSName,
)

func edsNeedsPush(req *model.PushRequest, proxy *model.Proxy) bool {
	if res, ok := xdsNeedsPush(req, proxy); ok {
		return res
	}
	for config := range req.ConfigsUpdated {
		if !skippedEdsConfigs.Contains(config.Kind) {
			return true
		}
	}
	return false
}

func (eds *EdsGenerator) Generate(proxy *model.Proxy, w *model.WatchedResource, req *model.PushRequest) (model.Resources, model.XdsLogDetails, error) {
	if !edsNeedsPush(req, proxy) {
		return nil, model.DefaultXdsLogDetails, nil
	}
	resources, logDetails := eds.buildEndpoints(proxy, req, w)
	return resources, logDetails, nil
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

func (eds *EdsGenerator) buildEndpoints(proxy *model.Proxy,
	req *model.PushRequest,
	w *model.WatchedResource,
) (model.Resources, model.XdsLogDetails) {
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
			klog.V(3).Infof("buildEndpoints: edsUpdatedServices count=%d (Full=%v)", len(edsUpdatedServices), req.Full)
		}
	}
	var resources model.Resources
	empty := 0
	cached := 0
	regenerated := 0
	for clusterName := range w.ResourceNames {
		hostname := model.ParseSubsetKeyHostname(clusterName)
		// CRITICAL FIX: Always check if this service was updated, even if edsUpdatedServices is nil
		// For incremental pushes, we need to check ConfigsUpdated directly to ensure cache is cleared
		serviceWasUpdated := false
		if req.ConfigsUpdated != nil {
			// CRITICAL: Log all ConfigsUpdated entries for debugging hostname matching
			configKeys := make([]string, 0, len(req.ConfigsUpdated))
			for ckey := range req.ConfigsUpdated {
				configKeys = append(configKeys, fmt.Sprintf("%s/%s/%s", ckey.Kind, ckey.Namespace, ckey.Name))
				if ckey.Kind == kind.ServiceEntry {
					// CRITICAL: Match ConfigsUpdated.Name with hostname
					// Both should be FQDN (e.g., "consumer.grpc-proxyless.svc.cluster.local")
					if ckey.Name == hostname {
						serviceWasUpdated = true
						klog.V(2).Infof("buildEndpoints: service %s was updated, forcing regeneration", hostname)
						break
					}
				}
			}
		}

		if edsUpdatedServices != nil {
			if _, ok := edsUpdatedServices[hostname]; !ok {
				// Cluster was not updated, skip recomputing. This happens when we get an incremental update for a
				// specific Hostname. On connect or for full push edsUpdatedServices will be empty.
				if !serviceWasUpdated {
					continue
				}
			} else {
				serviceWasUpdated = true
			}
		}
		builder := endpoints.NewEndpointBuilder(clusterName, proxy, req.Push)
		if builder == nil {
			continue
		}

		// CRITICAL FIX: For incremental pushes (Full=false), we must always regenerate EDS
		// if ConfigsUpdated contains this service, because endpoint health status may have changed.
		// The cache may contain stale empty endpoints from when the endpoint was unhealthy.
		// For full pushes, we can use cache if the service was not updated.
		// CRITICAL: If this is an incremental push and ConfigsUpdated contains any ServiceEntry,
		// we must check if it matches this hostname. If it does, force regeneration.
		shouldRegenerate := serviceWasUpdated
		if !shouldRegenerate && !req.Full && req.ConfigsUpdated != nil {
			// For incremental pushes, check if any ServiceEntry in ConfigsUpdated matches this hostname
			for ckey := range req.ConfigsUpdated {
				if ckey.Kind == kind.ServiceEntry && ckey.Name == hostname {
					shouldRegenerate = true
					break
				}
			}
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
		// CRITICAL FIX: Even if endpoints are empty (l == nil), we should still create an empty ClusterLoadAssignment
		// to ensure the client receives the EDS response. This is necessary for proxyless gRPC clients
		// to know the endpoint state, even if it's empty initially.
		// Note: BuildClusterLoadAssignment returns nil when no endpoints are found,
		// but we still need to send an empty ClusterLoadAssignment to the client.
		if l == nil {
			// Build an empty ClusterLoadAssignment for this cluster
			// Use the same logic as endpoint_builder.go:buildEmptyClusterLoadAssignment
			l = &endpoint.ClusterLoadAssignment{
				ClusterName: clusterName,
			}
		}
		regenerated++

		if len(l.Endpoints) == 0 {
			empty++
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

func shouldUseDeltaEds(req *model.PushRequest) bool {
	if !req.Full {
		return false
	}
	return canSendPartialFullPushes(req)
}

func (eds *EdsGenerator) GenerateDeltas(proxy *model.Proxy, req *model.PushRequest,
	w *model.WatchedResource,
) (model.Resources, model.DeletedResources, model.XdsLogDetails, bool, error) {
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

// buildDeltaEndpoints builds endpoints for delta XDS
func (eds *EdsGenerator) buildDeltaEndpoints(proxy *model.Proxy,
	req *model.PushRequest,
	w *model.WatchedResource,
) (model.Resources, []string, model.XdsLogDetails) {
	edsUpdatedServices := model.ConfigNamesOfKind(req.ConfigsUpdated, kind.ServiceEntry)
	var resources model.Resources
	var removed []string
	empty := 0
	cached := 0
	regenerated := 0

	for clusterName := range w.ResourceNames {
		// filter out eds that are not updated for clusters
		if len(edsUpdatedServices) > 0 {
			if _, ok := edsUpdatedServices[model.ParseSubsetKeyHostname(clusterName)]; !ok {
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

		// Try to get from cache
		if eds.Cache != nil {
			cachedEndpoint := eds.Cache.Get(builder)
			if cachedEndpoint != nil {
				resources = append(resources, cachedEndpoint)
				cached++
				continue
			}
		}

		// generate new eds cache
		l := builder.BuildClusterLoadAssignment(eds.EndpointIndex)
		if l == nil || len(l.Endpoints) == 0 {
			removed = append(removed, clusterName)
			continue
		}
		regenerated++
		if len(l.Endpoints) == 0 {
			empty++
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
	return resources, removed, model.XdsLogDetails{
		Incremental:    len(edsUpdatedServices) != 0,
		AdditionalInfo: fmt.Sprintf("empty:%v cached:%v/%v", empty, cached, cached+regenerated),
	}
}
