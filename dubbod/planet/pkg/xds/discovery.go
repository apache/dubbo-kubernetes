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
	"context"
	"fmt"
	"github.com/apache/dubbo-kubernetes/pkg/util/sets"
	"strconv"
	"sync"
	"time"

	"github.com/apache/dubbo-kubernetes/pkg/security"

	"github.com/apache/dubbo-kubernetes/dubbod/planet/pkg/features"
	"github.com/apache/dubbo-kubernetes/dubbod/planet/pkg/model"
	"github.com/apache/dubbo-kubernetes/pkg/cluster"
	"github.com/apache/dubbo-kubernetes/pkg/config/schema/kind"
	"github.com/apache/dubbo-kubernetes/pkg/kube/krt"
	discovery "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v3"
	"go.uber.org/atomic"
	"golang.org/x/time/rate"
	"google.golang.org/grpc"
)

var processStartTime = time.Now()

type DiscoveryServer struct {
	Env                 *model.Environment
	serverReady         atomic.Bool
	DiscoveryStartTime  time.Time
	ClusterAliases      map[cluster.ID]cluster.ID
	pushQueue           *PushQueue
	krtDebugger         *krt.DebugHandler
	InboundUpdates      *atomic.Int64
	CommittedUpdates    *atomic.Int64
	RequestRateLimit    *rate.Limiter
	ProxyNeedsPush      func(proxy *model.Proxy, req *model.PushRequest) (*model.PushRequest, bool)
	pushChannel         chan *model.PushRequest
	DebounceOptions     DebounceOptions
	concurrentPushLimit chan struct{}
	adsClientsMutex     sync.RWMutex
	adsClients          map[string]*Connection
	Cache               model.XdsCache
	Generators          map[string]model.XdsResourceGenerator
	pushVersion         atomic.Uint64
	Authenticators      []security.Authenticator
}

type DebounceOptions struct {
	DebounceAfter     time.Duration
	debounceMax       time.Duration
	enableEDSDebounce bool
}

func NewDiscoveryServer(env *model.Environment, clusterAliases map[string]string, debugger *krt.DebugHandler) *DiscoveryServer {
	out := &DiscoveryServer{
		Env:                 env,
		Generators:          map[string]model.XdsResourceGenerator{},
		krtDebugger:         debugger,
		InboundUpdates:      atomic.NewInt64(0),
		CommittedUpdates:    atomic.NewInt64(0),
		concurrentPushLimit: make(chan struct{}, features.PushThrottle),
		RequestRateLimit:    rate.NewLimiter(rate.Limit(features.RequestLimit), 1),
		pushChannel:         make(chan *model.PushRequest, 10),
		pushQueue:           NewPushQueue(),
		DebounceOptions: DebounceOptions{
			DebounceAfter:     features.DebounceAfter,
			debounceMax:       features.DebounceMax,
			enableEDSDebounce: features.EnableEDSDebounce,
		},
		adsClients:         map[string]*Connection{},
		Cache:              env.Cache,
		DiscoveryStartTime: processStartTime,
	}

	out.ClusterAliases = make(map[cluster.ID]cluster.ID)
	for alias := range clusterAliases {
		out.ClusterAliases[cluster.ID(alias)] = cluster.ID(clusterAliases[alias])
	}

	return out
}

func (s *DiscoveryServer) Register(rpcs *grpc.Server) {
	// Register v3 server
	discovery.RegisterAggregatedDiscoveryServiceServer(rpcs, s)
}

func (s *DiscoveryServer) Start(stopCh <-chan struct{}) {
	go s.handleUpdates(stopCh)
	go s.sendPushes(stopCh)
	go s.Cache.Run(stopCh)
}

func (s *DiscoveryServer) CachesSynced() {
	log.Infof("All caches have been synced up in %v, marking server ready", time.Since(s.DiscoveryStartTime))
	s.serverReady.Store(true)
}

func (s *DiscoveryServer) Shutdown() {
	s.pushQueue.ShutDown()
}

func (s *DiscoveryServer) initPushContext(req *model.PushRequest, oldPushContext *model.PushContext, version string) *model.PushContext {
	push := model.NewPushContext()
	push.PushVersion = version
	push.InitContext(s.Env, oldPushContext, req)
	s.dropCacheForRequest(req)
	s.Env.SetPushContext(push)
	return push
}

func (s *DiscoveryServer) globalPushContext() *model.PushContext {
	return s.Env.PushContext()
}

func (s *DiscoveryServer) NextVersion() string {
	return time.Now().Format(time.RFC3339) + "/" + strconv.FormatUint(s.pushVersion.Inc(), 10)
}

func (s *DiscoveryServer) Push(req *model.PushRequest) {
	if !req.Full {
		req.Push = s.globalPushContext()
		s.dropCacheForRequest(req)
		s.AdsPushAll(req)
		return
	}

	// Reset the status during the push.
	oldPushContext := s.globalPushContext()
	if oldPushContext != nil {
		oldPushContext.OnConfigChange()
	}
	// PushContext is reset after a config change. Previous status is
	// saved.
	versionLocal := s.NextVersion()
	push := s.initPushContext(req, oldPushContext, versionLocal)
	req.Push = push
	s.AdsPushAll(req)
}

func debounce(ch chan *model.PushRequest, stopCh <-chan struct{}, opts DebounceOptions, pushFn func(req *model.PushRequest), updateSent *atomic.Int64) {
	var timeChan <-chan time.Time
	var startDebounce time.Time
	var lastConfigUpdateTime time.Time

	pushCounter := 0
	debouncedEvents := 0

	var req *model.PushRequest
	free := true
	freeCh := make(chan struct{}, 1)

	push := func(req *model.PushRequest, debouncedEvents int, startDebounce time.Time) {
		pushFn(req)
		updateSent.Add(int64(debouncedEvents))
		freeCh <- struct{}{}
	}

	pushWorker := func() {
		eventDelay := time.Since(startDebounce)
		quietTime := time.Since(lastConfigUpdateTime)
		// it has been too long or quiet enough
		if eventDelay >= opts.debounceMax || quietTime >= opts.DebounceAfter {
			if req != nil {
				pushCounter++
				if req.ConfigsUpdated == nil {
					log.Infof("Push debounce stable[%d] %d for reason %s: %v since last change, %v since last push, full=%v",
						pushCounter, debouncedEvents, reasonsUpdated(req),
						quietTime, eventDelay, req.Full)
				} else {
					log.Infof("Push debounce stable[%d] %d for config %s: %v since last change, %v since last push, full=%v",
						pushCounter, debouncedEvents, configsUpdated(req),
						quietTime, eventDelay, req.Full)
				}
				free = false
				go push(req, debouncedEvents, startDebounce)
				req = nil
				debouncedEvents = 0
			}
		} else {
			timeChan = time.After(opts.DebounceAfter - quietTime)
		}
	}

	for {
		select {
		case <-freeCh:
			free = true
			pushWorker()
		case r := <-ch:
			// If reason is not set, record it as an unknown reason
			if len(r.Reason) == 0 {
				r.Reason = model.NewReasonStats(model.UnknownTrigger)
			}
			if !opts.enableEDSDebounce && !r.Full {
				// trigger push now, just for EDS
				go func(req *model.PushRequest) {
					pushFn(req)
					updateSent.Inc()
				}(r)
				continue
			}

			lastConfigUpdateTime = time.Now()
			wasNewDebounceWindow := debouncedEvents == 0
			if wasNewDebounceWindow {
				timeChan = time.After(opts.DebounceAfter)
				startDebounce = lastConfigUpdateTime
			}
			debouncedEvents++

			// Log each event that arrives, showing if it's the start of a new debounce window or merged
			if wasNewDebounceWindow {
				// First event in a new debounce window
				if len(r.ConfigsUpdated) > 0 {
					log.Debugf("Push debounce: new window started, event[1] for config %s (reason: %s)",
						configsUpdated(r), reasonsUpdated(r))
				} else {
					log.Debugf("Push debounce: new window started, event[1] for reason %s", reasonsUpdated(r))
				}
			} else {
				// Event merged into existing debounce window
				if len(r.ConfigsUpdated) > 0 {
					log.Debugf("Push debounce: event[%d] merged into window for config %s (reason: %s, total events: %d)",
						debouncedEvents, configsUpdated(r), reasonsUpdated(r), debouncedEvents)
				} else {
					log.Debugf("Push debounce: event[%d] merged into window for reason %s (total events: %d)",
						debouncedEvents, reasonsUpdated(r), debouncedEvents)
				}
			}

			req = req.Merge(r)
		case <-timeChan:
			if free {
				pushWorker()
			}
		case <-stopCh:
			return
		}
	}
}

func (s *DiscoveryServer) handleUpdates(stopCh <-chan struct{}) {
	debounce(s.pushChannel, stopCh, s.DebounceOptions, s.Push, s.CommittedUpdates)
}

func (s *DiscoveryServer) sendPushes(stopCh <-chan struct{}) {
	doSendPushes(stopCh, s.concurrentPushLimit, s.pushQueue)
}

func doSendPushes(stopCh <-chan struct{}, semaphore chan struct{}, queue *PushQueue) {
	for {
		select {
		case <-stopCh:
			return
		default:
			semaphore <- struct{}{}

			// Get the next proxy to push. This will block if there are no updates required.
			client, push, shuttingdown := queue.Dequeue()
			if shuttingdown {
				return
			}
			doneFunc := func() {
				queue.MarkDone(client)
				<-semaphore
			}

			var closed <-chan struct{}
			if client.deltaStream != nil {
				closed = client.deltaStream.Context().Done()
			} else {
				closed = client.StreamDone()
			}
			go func() {
				pushEv := &Event{
					pushRequest: push,
					done:        doneFunc,
				}

				select {
				case client.PushCh() <- pushEv:
					return
				case <-closed: // grpc stream was closed
					doneFunc()
					log.Infof("Client closed connection %v", client.ID())
				}
			}()
		}
	}
}

func (s *DiscoveryServer) dropCacheForRequest(req *model.PushRequest) {
	// If we don't know what updated, cannot safely cache. Clear the whole cache
	if req.Forced {
		s.Cache.ClearAll()
		log.Debugf("dropCacheForRequest: cleared all cache (Forced=true)")
	} else {
		// Otherwise, just clear the updated configs
		if len(req.ConfigsUpdated) > 0 {
			configs := make([]string, 0, len(req.ConfigsUpdated))
			for ckey := range req.ConfigsUpdated {
				configs = append(configs, ckey.String())
			}
			log.Debugf("dropCacheForRequest: clearing cache for configs: %v", configs)
		}
		s.Cache.Clear(req.ConfigsUpdated)
	}
}

func (s *DiscoveryServer) ConfigUpdate(req *model.PushRequest) {
	if model.HasConfigsOfKind(req.ConfigsUpdated, kind.Address) {
		s.Cache.ClearAll()
	}
	s.InboundUpdates.Inc()

	s.pushChannel <- req
}

func (s *DiscoveryServer) ServiceUpdate(shard model.ShardKey, hostname string, namespace string, event model.Event) {
	if event == model.EventDelete {
		s.Env.EndpointIndex.DeleteServiceShard(shard, hostname, namespace, false)
	} else {
	}
}

func (s *DiscoveryServer) Clients() []*Connection {
	s.adsClientsMutex.RLock()
	defer s.adsClientsMutex.RUnlock()
	clients := make([]*Connection, 0, len(s.adsClients))
	for _, con := range s.adsClients {
		// Check if connection is initialized by checking if the channel is closed
		// A closed channel returns immediately when read, indicating initialization is complete
		select {
		case <-con.InitializedCh():
			// Channel is closed (initialized), include this connection
			clients = append(clients, con)
		default:
			// Channel is still open (not initialized yet), skip
			continue
		}
	}
	return clients
}

func (s *DiscoveryServer) ProxyUpdate(clusterID cluster.ID, ip string) {
	var connection *Connection

	for _, v := range s.Clients() {
		// Check IPAddresses is not empty before accessing index 0
		if len(v.proxy.IPAddresses) > 0 && v.proxy.Metadata.ClusterID == clusterID && v.proxy.IPAddresses[0] == ip {
			connection = v
			break
		}
	}

	// It is possible that the envoy has not connected to this planet, maybe connected to another planet
	if connection == nil {
		return
	}

	s.pushQueue.Enqueue(connection, &model.PushRequest{
		Full:   true,
		Push:   s.globalPushContext(),
		Start:  time.Now(),
		Reason: model.NewReasonStats(model.ProxyUpdate),
		Forced: true,
	})
}

func (s *DiscoveryServer) EDSUpdate(shard model.ShardKey, serviceName string, namespace string, dubboEndpoints []*model.DubboEndpoint) {
	pushType := s.Env.EndpointIndex.UpdateServiceEndpoints(shard, serviceName, namespace, dubboEndpoints, true)
	// Always push EDS updates when endpoints change state
	// This ensures clients receive updates when:
	// 1. Endpoints become available (from empty to non-empty)
	// 2. Endpoints become unavailable (from non-empty to empty)
	// 3. Endpoint health status changes
	if pushType == model.IncrementalPush || pushType == model.FullPush {
		log.Infof("EDSUpdate: service %s/%s triggering %v push (endpoints=%d)", namespace, serviceName, pushType, len(dubboEndpoints))
		s.ConfigUpdate(&model.PushRequest{
			Full:           pushType == model.FullPush,
			ConfigsUpdated: sets.New(model.ConfigKey{Kind: kind.ServiceEntry, Name: serviceName, Namespace: namespace}),
			Reason:         model.NewReasonStats(model.EndpointUpdate),
		})
	} else if pushType == model.NoPush {
		// Even when UpdateServiceEndpoints returns NoPush, we may still need to push
		// This happens when:
		// 1. All old endpoints were unhealthy and new endpoints are also unhealthy (health status didn't change)
		// 2. But we still need to notify clients about the current state
		// For proxyless gRPC, we should push even if endpoints are empty to ensure clients know the state
		if len(dubboEndpoints) == 0 {
			log.Infof("EDSUpdate: service %s/%s endpoints became empty (NoPush), forcing push to clear client cache", namespace, serviceName)
			s.ConfigUpdate(&model.PushRequest{
				Full:           false, // Incremental push
				ConfigsUpdated: sets.New(model.ConfigKey{Kind: kind.ServiceEntry, Name: serviceName, Namespace: namespace}),
				Reason:         model.NewReasonStats(model.EndpointUpdate),
			})
		} else {
			// Endpoints exist but NoPush was returned - this means health status didn't change
			// For proxyless gRPC, we should still push to ensure clients have the latest endpoint info
			// This is especially important when endpoints become available after being empty
			log.Debugf("EDSUpdate: service %s/%s has %d endpoints but NoPush returned (health status unchanged), skipping push", namespace, serviceName, len(dubboEndpoints))
		}
	}
}

func (s *DiscoveryServer) EDSCacheUpdate(shard model.ShardKey, serviceName string, namespace string, dubboEndpoints []*model.DubboEndpoint) {
	s.Env.EndpointIndex.UpdateServiceEndpoints(shard, serviceName, namespace, dubboEndpoints, false)
}

func (s *DiscoveryServer) WaitForRequestLimit(ctx context.Context) error {
	if s.RequestRateLimit.Limit() == 0 {
		// Allow opt out when rate limiting is set to 0qps
		return nil
	}
	// Give a bit of time for queue to clear out, but if not fail fast. Client will connect to another
	// instance in best case, or retry with backoff.
	wait, cancel := context.WithTimeout(ctx, time.Second)
	defer cancel()
	return s.RequestRateLimit.Wait(wait)
}

func (s *DiscoveryServer) IsServerReady() bool {
	return s.serverReady.Load()
}

func configsUpdated(req *model.PushRequest) string {
	configs := ""
	for key := range req.ConfigsUpdated {
		configs += key.String()
		break
	}
	if len(req.ConfigsUpdated) > 1 {
		more := " and " + strconv.Itoa(len(req.ConfigsUpdated)-1) + " more configs"
		configs += more
	}

	return configs
}

func reasonsUpdated(req *model.PushRequest) string {
	var (
		reason0, reason1            model.TriggerReason
		reason0Cnt, reason1Cnt, idx int
	)
	for r, cnt := range req.Reason {
		if idx == 0 {
			reason0, reason0Cnt = r, cnt
		} else if idx == 1 {
			reason1, reason1Cnt = r, cnt
		} else {
			break
		}
		idx++
	}

	switch len(req.Reason) {
	case 0:
		return "unknown"
	case 1:
		return fmt.Sprintf("%s:%d", reason0, reason0Cnt)
	case 2:
		return fmt.Sprintf("%s:%d and %s:%d", reason0, reason0Cnt, reason1, reason1Cnt)
	default:
		return fmt.Sprintf("%s:%d and %d(%d) more reasons", reason0, reason0Cnt, len(req.Reason)-1,
			req.Reason.Count()-reason0Cnt)
	}
}
