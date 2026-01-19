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
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"github.com/apache/dubbo-kubernetes/pkg/maps"

	"github.com/apache/dubbo-kubernetes/dubbod/planet/pkg/model"
	v3 "github.com/apache/dubbo-kubernetes/dubbod/planet/pkg/xds/v3"
	"github.com/apache/dubbo-kubernetes/pkg/util/sets"
	"github.com/apache/dubbo-kubernetes/pkg/xds"
	core "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	discovery "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v3"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"
)

var (
	connectionNumber = int64(0)
	log              = xds.Log
)

type (
	DiscoveryStream      = xds.DiscoveryStream
	DeltaDiscoveryStream = discovery.AggregatedDiscoveryService_DeltaAggregatedResourcesServer
	DeltaDiscoveryClient = discovery.AggregatedDiscoveryService_DeltaAggregatedResourcesClient
)

type Connection struct {
	xds.Connection
	node         *core.Node
	proxy        *model.Proxy
	deltaStream  DeltaDiscoveryStream
	deltaReqChan chan *discovery.DeltaDiscoveryRequest
	s            *DiscoveryServer
	ids          []string
}

type Event struct {
	pushRequest *model.PushRequest
	done        func()
}

func (s *DiscoveryServer) StreamAggregatedResources(stream DiscoveryStream) error {
	return s.Stream(stream)
}

func (s *DiscoveryServer) DeltaAggregatedResources(stream discovery.AggregatedDiscoveryService_DeltaAggregatedResourcesServer) error {
	return s.StreamDeltas(stream)
}

func (s *DiscoveryServer) AllClients() []*Connection {
	s.adsClientsMutex.RLock()
	defer s.adsClientsMutex.RUnlock()
	return maps.Values(s.adsClients)
}

func (s *DiscoveryServer) StartPush(req *model.PushRequest) {
	req.Start = time.Now()
	for _, p := range s.AllClients() {
		s.pushQueue.Enqueue(p, req)
	}
}

func (s *DiscoveryServer) adsClientCount() int {
	s.adsClientsMutex.RLock()
	defer s.adsClientsMutex.RUnlock()
	return len(s.adsClients)
}

func (s *DiscoveryServer) AdsPushAll(req *model.PushRequest) {
	connectedEndpoints := s.adsClientCount()
	if !req.Full {
		// Incremental push - only log connection count and version
		log.Infof("XDS: Incremental Push to %d Endpoints, Version:%s",
			connectedEndpoints, req.Push.PushVersion)
	} else {
		// Full push - log services, connections, and version
		totalService := len(req.Push.GetAllServices())
		log.Infof("XDS: Pushing Services:%d ConnectedEndpoints:%d Version:%s",
			totalService, connectedEndpoints, req.Push.PushVersion)

		if req.ConfigsUpdated == nil {
			req.ConfigsUpdated = make(sets.Set[model.ConfigKey])
		}
	}
	s.StartPush(req)
}

func (s *DiscoveryServer) initConnection(node *core.Node, con *Connection, identities []string) error {
	proxy, err := s.initProxyMetadata(node)
	if err != nil {
		return err
	}

	if alias, exists := s.ClusterAliases[proxy.Metadata.ClusterID]; exists {
		proxy.Metadata.ClusterID = alias
	}

	proxy.LastPushContext = s.globalPushContext()
	con.SetID(connectionID(proxy.ID))
	con.node = node
	con.proxy = proxy

	// TODO authorize

	s.addCon(con.ID(), con)
	currentCount := s.adsClientCount()
	log.Infof("new connection for node:%s (total connections: %d)", con.ID(), currentCount)
	defer con.MarkInitialized()

	if err := s.initializeProxy(con); err != nil {
		s.closeConnection(con)
		return err
	}

	// Trigger a ConfigUpdate to ensure ConnectedEndpoints count is updated in subsequent XDS: Pushing logs.
	// The push will be debounced, so this is safe to call.
	s.ConfigUpdate(&model.PushRequest{
		Full:   true,
		Forced: false,
		Reason: model.NewReasonStats(model.ProxyUpdate),
	})

	return nil
}

func (s *DiscoveryServer) closeConnection(con *Connection) {
	if con.ID() == "" {
		return
	}
	s.removeCon(con.ID())
}

func (s *DiscoveryServer) initializeProxy(con *Connection) error {
	proxy := con.proxy
	s.computeProxyState(proxy, nil)

	proxy.DiscoverIPMode()

	proxy.WatchedResources = map[string]*model.WatchedResource{}

	// Based on node metadata and version, we can associate a different generator.
	if proxy.Metadata.Generator != "" {
		proxy.XdsResourceGenerator = s.Generators[proxy.Metadata.Generator]
	}

	return nil
}

func (s *DiscoveryServer) initProxyMetadata(node *core.Node) (*model.Proxy, error) {
	meta, err := model.ParseMetadata(node.Metadata)
	if err != nil {
		return nil, status.New(codes.InvalidArgument, err.Error()).Err()
	}
	proxy, err := model.ParseServiceNodeWithMetadata(node.Id, meta)
	if err != nil {
		return nil, status.New(codes.InvalidArgument, err.Error()).Err()
	}
	// Update the config namespace associated with this proxy
	proxy.ConfigNamespace = model.GetProxyConfigNamespace(proxy)
	proxy.XdsNode = node
	return proxy, nil
}

func (s *DiscoveryServer) computeProxyState(proxy *model.Proxy, request *model.PushRequest) {
	proxy.Lock()
	defer proxy.Unlock()
	// 1. If request == nil(initiation phase) or request.ConfigsUpdated == nil(global push), set proxy serviceTargets.
	// 2. otherwise only set when svc update, this is for the case that a service may select the proxy
	if request == nil || request.Forced ||
		proxy.ShouldUpdateServiceTargets(request.ConfigsUpdated) {
		proxy.SetServiceTargets(s.Env.ServiceDiscovery)
	}

	recomputeLabels := request == nil || request.IsProxyUpdate()
	if recomputeLabels {
	}

	push := proxy.LastPushContext
	if request == nil {
	} else {
		push = request.Push
		if request.Forced {
		}
		for conf := range request.ConfigsUpdated {
			switch conf.Kind {
			}
		}
	}

	proxy.LastPushContext = push
	if request != nil {
		proxy.LastPushTime = request.Start
	}
}

func (s *DiscoveryServer) addCon(conID string, con *Connection) {
	s.adsClientsMutex.Lock()
	defer s.adsClientsMutex.Unlock()
	s.adsClients[conID] = con
}

func (s *DiscoveryServer) removeCon(conID string) {
	s.adsClientsMutex.Lock()
	defer s.adsClientsMutex.Unlock()

	if _, exist := s.adsClients[conID]; !exist {
		log.Errorf("Removing connection for non-existing node:%v.", conID)
	} else {
		delete(s.adsClients, conID)
	}
}

func (s *DiscoveryServer) Stream(stream DiscoveryStream) error {
	if !s.IsServerReady() {
		return status.Error(codes.Unavailable, "server is not ready to serve discovery information")
	}

	ctx := stream.Context()
	peerAddr := "0.0.0.0"
	if peerInfo, ok := peer.FromContext(ctx); ok {
		peerAddr = peerInfo.Addr.String()
	}

	if err := s.WaitForRequestLimit(stream.Context()); err != nil {
		log.Warnf("%q exceeded rate limit: %v", peerAddr, err)
		return status.Errorf(codes.ResourceExhausted, "request rate limit exceeded: %v", err)
	}

	// TODO authenticate

	s.globalPushContext().InitContext(s.Env, nil, nil)
	con := newConnection(peerAddr, stream)
	con.s = s
	return xds.Stream(con)
}

func (s *DiscoveryServer) pushConnection(con *Connection, pushEv *Event) error {
	// Check if connection is initialized - proxy must be set before we can push
	if con.proxy == nil {
		// Connection is not yet initialized, skip this push
		// The push will happen after initialization completes
		log.Debugf("Skipping push to %v, connection not yet initialized", con.ID())
		return nil
	}

	pushRequest := pushEv.pushRequest
	if pushRequest == nil {
		log.Warnf("Skipping push to %v, pushRequest is nil", con.ID())
		return nil
	}

	if pushRequest.Full {
		// Update Proxy with current information.
		s.computeProxyState(con.proxy, pushRequest)
	}

	// Check if ProxyNeedsPush function is set, otherwise default to always push
	var needsPush bool
	if s.ProxyNeedsPush != nil {
		pushRequest, needsPush = s.ProxyNeedsPush(con.proxy, pushRequest)
	} else {
		// Default behavior: always push if ProxyNeedsPush is not set
		needsPush = true
	}
	if !needsPush {
		log.Debugf("Skipping push to %v, no updates required", con.ID())
		return nil
	}

	// Send pushes to all generators
	// Each Generator is responsible for determining if the push event requires a push
	wrl := con.watchedResourcesByOrder()
	for _, w := range wrl {
		if err := s.pushXds(con, w, pushRequest); err != nil {
			return err
		}
	}
	return nil
}

func (s *DiscoveryServer) processRequest(req *discovery.DiscoveryRequest, con *Connection) error {
	stype := v3.GetShortType(req.TypeUrl)
	if req.TypeUrl == v3.HealthInfoType {
		return nil
	}

	shouldRespond, delta := xds.ShouldRespond(con.proxy, con.ID(), req)

	// Log NEW requests (will respond) at INFO level so every grpcurl request is visible
	// This ensures every grpcurl request triggers visible xDS logs in control plane
	// Format: "LDS: REQ node-xxx resources:1 nonce:abc123 [resource1, resource2] (will respond)"
	resourceNamesStr := ""
	if len(req.ResourceNames) > 0 {
		// Show resource names for better debugging
		if len(req.ResourceNames) <= 10 {
			resourceNamesStr = fmt.Sprintf(" [%s]", strings.Join(req.ResourceNames, ", "))
		} else {
			// If too many resources, show first 5 and count
			resourceNamesStr = fmt.Sprintf(" [%s, ... and %d more]", strings.Join(req.ResourceNames[:5], ", "), len(req.ResourceNames)-5)
		}
	} else {
		resourceNamesStr = " [wildcard]"
	}

	if shouldRespond {
		log.Infof("%s: REQ %s resources:%d nonce:%s%s (will respond)", stype,
			con.ID(), len(req.ResourceNames), req.ResponseNonce, resourceNamesStr)
	} else {
		log.Infof("%s: REQ %s resources:%d nonce:%s%s (ACK/ignored)", stype,
			con.ID(), len(req.ResourceNames), req.ResponseNonce, resourceNamesStr)
	}

	if !shouldRespond {
		// Don't process ACK/ignored/expired nonce requests
		return nil
	}

	// For proxyless gRPC, if client sends wildcard (empty ResourceNames) after receiving specific resources,
	// this is likely an ACK and we should NOT push all resources again
	// Check if this is a wildcard request after specific resources were sent
	watchedResource := con.proxy.GetWatchedResource(req.TypeUrl)
	if con.proxy.IsProxylessGrpc() && len(req.ResourceNames) == 0 && watchedResource != nil && len(watchedResource.ResourceNames) > 0 && watchedResource.NonceSent != "" {
		// This is a wildcard ACK after specific resources were sent
		// ShouldRespond should have returned false, but we check here as safety net
		// Update the WatchedResource to reflect the ACK, but don't push
		log.Debugf("%s: proxyless gRPC wildcard ACK after specific resources (prev: %d resources, nonce: %s), skipping push",
			stype, len(watchedResource.ResourceNames), watchedResource.NonceSent)
		// ShouldRespond should have already handled this, but we ensure no push happens
		return nil
	}

	// Log PUSH request BEFORE calling pushXds with detailed information
	// This helps track what resources are being requested for each push
	pushResourceNamesStr := ""
	if len(req.ResourceNames) > 0 {
		if len(req.ResourceNames) <= 10 {
			pushResourceNamesStr = fmt.Sprintf(" resources:%d [%s]", len(req.ResourceNames), strings.Join(req.ResourceNames, ", "))
		} else {
			pushResourceNamesStr = fmt.Sprintf(" resources:%d [%s, ... and %d more]", len(req.ResourceNames), strings.Join(req.ResourceNames[:5], ", "), len(req.ResourceNames)-5)
		}
	} else {
		pushResourceNamesStr = " resources:0 [wildcard]"
	}
	log.Infof("%s: PUSH request for node:%s%s", stype, con.ID(), pushResourceNamesStr)

	// Don't set Forced=true for regular proxy requests to avoid unnecessary ServiceTargets recomputation
	// Only set Forced for debug requests or when explicitly needed
	// This prevents ServiceTargets from being recomputed on every request, which can cause
	// inconsistent listener generation and push loops
	request := &model.PushRequest{
		Full:   true,
		Push:   con.proxy.LastPushContext,
		Reason: model.NewReasonStats(model.ProxyRequest),
		Start:  con.proxy.LastPushTime,
		Delta:  delta,
		Forced: false, // Only recompute ServiceTargets when ConfigsUpdated indicates service changes
	}

	// Get WatchedResource after ShouldRespond has created it
	// ShouldRespond may have created a new WatchedResource for first-time requests
	w := con.proxy.GetWatchedResource(req.TypeUrl)
	if w == nil {
		// This should not happen if ShouldRespond returned true, but handle it gracefully
		log.Warnf("%s: WatchedResource is nil for %s after ShouldRespond returned true, creating it", stype, con.ID())
		con.proxy.NewWatchedResource(req.TypeUrl, req.ResourceNames)
		w = con.proxy.GetWatchedResource(req.TypeUrl)
	}

	return s.pushXds(con, w, request)
}

func (s *DiscoveryServer) processDeltaRequest(req *discovery.DeltaDiscoveryRequest, con *Connection) error {
	stype := v3.GetShortType(req.TypeUrl)
	deltaLog.Infof("%s: REQ %s resources sub:%d unsub:%d nonce:%s", stype,
		con.ID(), len(req.ResourceNamesSubscribe), len(req.ResourceNamesUnsubscribe), req.ResponseNonce)

	if req.TypeUrl == v3.HealthInfoType {
		return nil
	}

	shouldRespond := shouldRespondDelta(con, req)
	if !shouldRespond {
		return nil
	}

	subs, _, _ := deltaWatchedResources(nil, req)
	request := &model.PushRequest{
		Full:   true,
		Push:   con.proxy.LastPushContext,
		Reason: model.NewReasonStats(model.ProxyRequest),
		Start:  con.proxy.LastPushTime,
		Delta: model.ResourceDelta{
			Subscribed:   subs,
			Unsubscribed: sets.New(req.ResourceNamesUnsubscribe...).Delete("*"),
		},
		Forced: true,
	}

	err := s.pushDeltaXds(con, con.proxy.GetWatchedResource(req.TypeUrl), request)
	if err != nil {
		return err
	}
	if req.TypeUrl != v3.ClusterType {
		return nil
	}
	return s.forceEDSPush(con)
}

func newConnection(peerAddr string, stream DiscoveryStream) *Connection {
	return &Connection{
		Connection: xds.NewConnection(peerAddr, stream),
	}
}

func newDeltaConnection(peerAddr string, stream DeltaDiscoveryStream) *Connection {
	return &Connection{
		Connection:   xds.NewConnection(peerAddr, nil),
		deltaStream:  stream,
		deltaReqChan: make(chan *discovery.DeltaDiscoveryRequest, 1),
	}
}

var PushOrder = []string{
	v3.ClusterType,
	v3.EndpointType,
	v3.ListenerType,
	v3.RouteType,
}

var KnownOrderedTypeUrls = sets.New(PushOrder...)

func (conn *Connection) watchedResourcesByOrder() []*model.WatchedResource {
	allWatched := conn.proxy.ShallowCloneWatchedResources()
	ordered := make([]*model.WatchedResource, 0, len(allWatched))
	// first add all known types, in order
	for _, tp := range PushOrder {
		if allWatched[tp] != nil {
			ordered = append(ordered, allWatched[tp])
		}
	}
	// Then add any undeclared types
	for tp, res := range allWatched {
		if !KnownOrderedTypeUrls.Contains(tp) {
			ordered = append(ordered, res)
		}
	}
	return ordered
}

func (conn *Connection) Initialize(node *core.Node) error {
	return conn.s.initConnection(node, conn, conn.ids)
}

func (conn *Connection) Close() {
	conn.s.closeConnection(conn)
}

func (conn *Connection) Push(ev any) error {
	pushEv := ev.(*Event)
	err := conn.s.pushConnection(conn, pushEv)
	pushEv.done()
	return err
}

func (conn *Connection) Process(req *discovery.DiscoveryRequest) error {
	return conn.s.processRequest(req, conn)
}

func (conn *Connection) Watcher() xds.Watcher {
	return conn.proxy
}

func (conn *Connection) Proxy() *model.Proxy {
	return conn.proxy
}

func (conn *Connection) XdsConnection() *xds.Connection {
	return &conn.Connection
}

func connectionID(node string) string {
	id := atomic.AddInt64(&connectionNumber, 1)
	return node + "-" + strconv.FormatInt(id, 10)
}
