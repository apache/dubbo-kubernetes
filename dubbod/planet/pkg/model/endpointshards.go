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

package model

import (
	"sync"

	"github.com/apache/dubbo-kubernetes/pkg/config/schema/kind"

	"github.com/apache/dubbo-kubernetes/dubbod/planet/pkg/serviceregistry/provider"
	"github.com/apache/dubbo-kubernetes/pkg/cluster"
	"github.com/apache/dubbo-kubernetes/pkg/util/sets"
)

type PushType int

const (
	NoPush PushType = iota
	IncrementalPush
	FullPush
)

type shardRegistry interface {
	Cluster() cluster.ID
	Provider() provider.ID
}

type ShardKey struct {
	Cluster  cluster.ID
	Provider provider.ID
}

type EndpointShards struct {
	sync.RWMutex
	Shards          map[ShardKey][]*DubboEndpoint
	ServiceAccounts sets.String
}

type EndpointIndex struct {
	mu          sync.RWMutex
	shardsBySvc map[string]map[string]*EndpointShards
	cache       XdsCache
}

func endpointUpdateRequiresPush(oldDubboEndpoints []*DubboEndpoint, incomingEndpoints []*DubboEndpoint) ([]*DubboEndpoint, bool) {
	// If old endpoints are empty (nil or empty slice), and we have new endpoints, we must push
	// Note: len() for nil slices is defined as zero, so we can use len() directly
	oldWasEmpty := len(oldDubboEndpoints) == 0
	newIsEmpty := len(incomingEndpoints) == 0

	if oldWasEmpty {
		// If there are no old endpoints, we should push with incoming endpoints as there is nothing to compare.
		// Even if new endpoints are unhealthy, we must push to notify clients that endpoints are now available
		// The client will handle unhealthy endpoints appropriately (e.g., retry, circuit breaker)
		if !newIsEmpty {
			return incomingEndpoints, true
		}
		// If both old and new are empty, no push needed
		return incomingEndpoints, false
	}
	// If old endpoints exist but new endpoints are empty, we must push
	if newIsEmpty {
		return incomingEndpoints, true
	}
	needPush := false
	newDubboEndpoints := make([]*DubboEndpoint, 0, len(incomingEndpoints))
	omap := make(map[string]*DubboEndpoint, len(oldDubboEndpoints))
	nmap := make(map[string]*DubboEndpoint, len(newDubboEndpoints))
	for _, oie := range oldDubboEndpoints {
		omap[oie.Key()] = oie
	}
	for _, nie := range incomingEndpoints {
		nmap[nie.Key()] = nie
	}
	for _, nie := range incomingEndpoints {
		if oie, exists := omap[nie.Key()]; exists {
			if !needPush && !oie.Equals(nie) {
				needPush = true
			}
			newDubboEndpoints = append(newDubboEndpoints, nie)
		} else {
			// New endpoint that didn't exist before - always push if it's healthy or should be sent
			if nie.HealthStatus != UnHealthy || nie.SendUnhealthyEndpoints {
				needPush = true
			}
			newDubboEndpoints = append(newDubboEndpoints, nie)
		}
	}
	if !needPush {
		// Check if any old endpoints were removed
		for _, oie := range oldDubboEndpoints {
			if _, f := nmap[oie.Key()]; !f {
				needPush = true
				break
			}
		}
	}

	return newDubboEndpoints, needPush
}

func updateShardServiceAccount(shards *EndpointShards, serviceName string) bool {
	oldServiceAccount := shards.ServiceAccounts
	serviceAccounts := sets.String{}
	for _, epShards := range shards.Shards {
		for _, ep := range epShards {
			if ep.ServiceAccount != "" {
				serviceAccounts.Insert(ep.ServiceAccount)
			}
		}
	}

	if !oldServiceAccount.Equals(serviceAccounts) {
		shards.ServiceAccounts = serviceAccounts
		log.Debugf("Updating service accounts now, svc %v, before service account %v, after %v",
			serviceName, oldServiceAccount, serviceAccounts)
		return true
	}

	return false
}

func (e *EndpointIndex) UpdateServiceEndpoints(shard ShardKey, hostname string, namespace string, dubboEndpoints []*DubboEndpoint, logPushType bool) PushType {
	if len(dubboEndpoints) == 0 {
		// Should delete the service EndpointShards when endpoints become zero to prevent memory leak,
		// but we should not delete the keys from EndpointIndex map - that will trigger
		// unnecessary full push which can become a real problem if a pod is in crashloop and thus endpoints
		// flip flopping between 1 and 0.
		e.DeleteServiceShard(shard, hostname, namespace, true)
		if logPushType {
			log.Infof("Incremental push, service %s at shard %v has no endpoints", hostname, shard)
		} else {
			log.Infof("Cache Update, Service %s at shard %v has no endpoints", hostname, shard)
		}
		return IncrementalPush
	}

	pushType := IncrementalPush
	// Find endpoint shard for this service, if it is available - otherwise create a new one.
	ep, created := e.GetOrCreateEndpointShard(hostname, namespace)
	// If we create a new endpoint shard, that means we have not seen the service earlier. We should do a full push.
	if created {
		if logPushType {
			log.Infof("Full push, new service %s/%s", namespace, hostname)
		} else {
			log.Infof("Cache Update, new service %s/%s", namespace, hostname)
		}
		pushType = FullPush
	}

	ep.Lock()
	defer ep.Unlock()
	oldDubboEndpoints := ep.Shards[shard]
	// Check if this is a transition from empty to non-empty BEFORE calling endpointUpdateRequiresPush
	// This ensures we always push when endpoints become available, even if they're unhealthy
	// Note: len() for nil slices is defined as zero, so we can use len() directly
	oldWasEmpty := len(oldDubboEndpoints) == 0
	newIsEmpty := len(dubboEndpoints) == 0
	transitionFromEmptyToNonEmpty := oldWasEmpty && !newIsEmpty

	newDubboEndpoints, needPush := endpointUpdateRequiresPush(oldDubboEndpoints, dubboEndpoints)

	// If endpoints transition from empty to non-empty, we MUST push
	// even if they're initially unhealthy (client will handle retries/circuit breaking)
	if transitionFromEmptyToNonEmpty {
		needPush = true
		log.Debugf("UpdateServiceEndpoints: service=%s, shard=%v, endpoints transitioned from empty to non-empty, forcing push", hostname, shard)
	}

	if logPushType {
		oldHealthyCount := 0
		oldUnhealthyCount := 0
		newHealthyCount := 0
		newUnhealthyCount := 0
		for _, ep := range oldDubboEndpoints {
			if ep.HealthStatus == Healthy {
				oldHealthyCount++
			} else {
				oldUnhealthyCount++
			}
		}
		for _, ep := range newDubboEndpoints {
			if ep.HealthStatus == Healthy {
				newHealthyCount++
			} else {
				newUnhealthyCount++
			}
		}
		log.Debugf("UpdateServiceEndpoints: service=%s, shard=%v, oldEndpoints=%d (healthy=%d, unhealthy=%d), newEndpoints=%d (healthy=%d, unhealthy=%d), needPush=%v, pushType=%v, transitionFromEmptyToNonEmpty=%v",
			hostname, shard, len(oldDubboEndpoints), oldHealthyCount, oldUnhealthyCount, len(newDubboEndpoints), newHealthyCount, newUnhealthyCount, needPush, pushType, transitionFromEmptyToNonEmpty)
	}

	if pushType != FullPush && !needPush {
		log.Warnf("No push, either old endpoint health status did not change or new endpoint came with unhealthy status, %v (oldEndpoints=%d, newEndpoints=%d)", hostname, len(oldDubboEndpoints), len(newDubboEndpoints))
		pushType = NoPush
	}

	ep.Shards[shard] = newDubboEndpoints

	// Check if ServiceAccounts have changed. We should do a full push if they have changed.
	saUpdated := updateShardServiceAccount(ep, hostname)

	// For existing endpoints, we need to do full push if service accounts change.
	if saUpdated && pushType != FullPush {
		// Avoid extra logging if already a full push
		if logPushType {
			log.Infof("Full push, service accounts changed, %v", hostname)
		} else {
			log.Infof("Cache Update, service accounts changed, %v", hostname)
		}
		pushType = FullPush
	}
	e.clearCacheForService(hostname, namespace)
	return pushType
}

func (e *EndpointIndex) GetOrCreateEndpointShard(serviceName, namespace string) (*EndpointShards, bool) {
	e.mu.Lock()
	defer e.mu.Unlock()

	if _, exists := e.shardsBySvc[serviceName]; !exists {
		e.shardsBySvc[serviceName] = map[string]*EndpointShards{}
	}
	if ep, exists := e.shardsBySvc[serviceName][namespace]; exists {
		return ep, false
	}
	// This endpoint is for a service that was not previously loaded.
	ep := &EndpointShards{
		Shards:          map[ShardKey][]*DubboEndpoint{},
		ServiceAccounts: sets.String{},
	}
	e.shardsBySvc[serviceName][namespace] = ep
	e.clearCacheForService(serviceName, namespace)
	return ep, true
}

func (e *EndpointIndex) DeleteServiceShard(shard ShardKey, serviceName, namespace string, preserveKeys bool) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.deleteServiceInner(shard, serviceName, namespace, preserveKeys)
}

func (e *EndpointIndex) deleteServiceInner(shard ShardKey, serviceName, namespace string, preserveKeys bool) {
	if e.shardsBySvc[serviceName] == nil ||
		e.shardsBySvc[serviceName][namespace] == nil {
		return
	}
	epShards := e.shardsBySvc[serviceName][namespace]
	epShards.Lock()
	delete(epShards.Shards, shard)
	e.clearCacheForService(serviceName, namespace)
	if !preserveKeys {
		if len(epShards.Shards) == 0 {
			delete(e.shardsBySvc[serviceName], namespace)
		}
		if len(e.shardsBySvc[serviceName]) == 0 {
			delete(e.shardsBySvc, serviceName)
		}
	}
	epShards.Unlock()
}

func (e *EndpointIndex) clearCacheForService(svc, ns string) {
	e.cache.Clear(sets.Set[ConfigKey]{{
		Kind:      kind.ServiceEntry,
		Name:      svc,
		Namespace: ns,
	}: {}})
}

func (e *EndpointIndex) ShardsForService(serviceName, namespace string) (*EndpointShards, bool) {
	e.mu.RLock()
	defer e.mu.RUnlock()
	byNs, ok := e.shardsBySvc[serviceName]
	if !ok {
		return nil, false
	}
	shards, ok := byNs[namespace]
	return shards, ok
}

// ServicesInNamespace returns all service names in the given namespace
func (e *EndpointIndex) ServicesInNamespace(namespace string) []string {
	e.mu.RLock()
	defer e.mu.RUnlock()
	services := make([]string, 0)
	for svcName, byNs := range e.shardsBySvc {
		if _, ok := byNs[namespace]; ok {
			services = append(services, svcName)
		}
	}
	return services
}

// AllServices returns all service names registered in the EndpointIndex
func (e *EndpointIndex) AllServices() []string {
	e.mu.RLock()
	defer e.mu.RUnlock()
	services := make([]string, 0, len(e.shardsBySvc))
	for svcName := range e.shardsBySvc {
		services = append(services, svcName)
	}
	return services
}

func (es *EndpointShards) CopyEndpoints(portMap map[string]int, ports sets.Set[int]) map[int][]*DubboEndpoint {
	es.RLock()
	defer es.RUnlock()
	res := map[int][]*DubboEndpoint{}
	for _, v := range es.Shards {
		for _, ep := range v {
			// use the port name as the key, unless LegacyClusterPortKey is set and takes precedence
			// In EDS we match on port *name*. But for historical reasons, we match on port number for CDS.
			var portNum int
			if ep.LegacyClusterPortKey != 0 {
				if !ports.Contains(ep.LegacyClusterPortKey) {
					continue
				}
				portNum = ep.LegacyClusterPortKey
			} else {
				pn, f := portMap[ep.ServicePortName]
				if !f {
					continue
				}
				portNum = pn
			}
			res[portNum] = append(res[portNum], ep)
		}
	}
	return res
}

func ShardKeyFromRegistry(instance shardRegistry) ShardKey {
	return ShardKey{Cluster: instance.Cluster(), Provider: instance.Provider()}
}
