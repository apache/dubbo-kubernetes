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
	"github.com/apache/dubbo-kubernetes/pkg/maps"
	"github.com/apache/dubbo-kubernetes/pkg/util/sets"
	"github.com/apache/dubbo-kubernetes/pkg/xds"
	"github.com/apache/dubbo-kubernetes/sail/pkg/model"
	v3 "github.com/apache/dubbo-kubernetes/sail/pkg/xds/v3"
	core "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	discovery "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v3"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"
	"k8s.io/klog/v2"
	"strconv"
	"strings"
	"sync/atomic"
	"time"
)

var connectionNumber = int64(0)

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

func (conn *Connection) XdsConnection() *xds.Connection {
	return &conn.Connection
}

func (conn *Connection) Proxy() *model.Proxy {
	return conn.proxy
}

func (s *DiscoveryServer) DeltaAggregatedResources(stream discovery.AggregatedDiscoveryService_DeltaAggregatedResourcesServer) error {
	return s.StreamDeltas(stream)
}

func (s *DiscoveryServer) StreamAggregatedResources(stream DiscoveryStream) error {
	return s.Stream(stream)
}

func (s *DiscoveryServer) StartPush(req *model.PushRequest) {
	req.Start = time.Now()
	for _, p := range s.AllClients() {
		s.pushQueue.Enqueue(p, req)
	}
}

func (s *DiscoveryServer) AllClients() []*Connection {
	s.adsClientsMutex.RLock()
	defer s.adsClientsMutex.RUnlock()
	return maps.Values(s.adsClients)
}

func (s *DiscoveryServer) AdsPushAll(req *model.PushRequest) {
	if !req.Full {
		klog.Infof("XDS: Incremental Pushing ConnectedEndpoints:%d", s.adsClientCount())
	} else {
		totalService := len(req.Push.GetAllServices())
		klog.Infof("XDS: Pushing Services:%d ConnectedEndpoints:%d", totalService, s.adsClientCount())

		if req.ConfigsUpdated == nil {
			req.ConfigsUpdated = make(sets.Set[model.ConfigKey])
		}
	}
	s.StartPush(req)
}

func (s *DiscoveryServer) adsClientCount() int {
	s.adsClientsMutex.RLock()
	defer s.adsClientsMutex.RUnlock()
	return len(s.adsClients)
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

func connectionID(node string) string {
	id := atomic.AddInt64(&connectionNumber, 1)
	return node + "-" + strconv.FormatInt(id, 10)
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

	if err := s.authorize(con, identities); err != nil {
		return err
	}
	s.addCon(con.ID(), con)

	defer con.MarkInitialized()
	if err := s.initializeProxy(con); err != nil {
		s.closeConnection(con)
		return err
	}

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

func (s *DiscoveryServer) addCon(conID string, con *Connection) {
	s.adsClientsMutex.Lock()
	defer s.adsClientsMutex.Unlock()
	s.adsClients[conID] = con
}

func (s *DiscoveryServer) removeCon(conID string) {
	s.adsClientsMutex.Lock()
	defer s.adsClientsMutex.Unlock()

	if _, exist := s.adsClients[conID]; !exist {
		klog.Errorf("ADS: Removing connection for non-existing node:%v.", conID)
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

	// TODO WaitForRequestLimit?

	if err := s.WaitForRequestLimit(stream.Context()); err != nil {
		klog.Warningf("ADS: %q exceeded rate limit: %v", peerAddr, err)
		return status.Errorf(codes.ResourceExhausted, "request rate limit exceeded: %v", err)
	}

	ids, err := s.authenticate(ctx)
	if err != nil {
		return status.Error(codes.Unauthenticated, err.Error())
	}

	if ids != nil {
		klog.V(2).Infof("Authenticated XDS: %v with identity %v", peerAddr, ids)
	} else {
		klog.V(2).Infof("Unauthenticated XDS: %s", peerAddr)
	}
	s.globalPushContext().InitContext(s.Env, nil, nil)
	con := newConnection(peerAddr, stream)
	con.ids = ids
	con.s = s
	return xds.Stream(con)
}

func (s *DiscoveryServer) WaitForRequestLimit(ctx context.Context) error {
	if s.RequestRateLimit.Limit() == 0 {
		return nil
	}
	wait, cancel := context.WithTimeout(ctx, time.Second)
	defer cancel()
	return s.RequestRateLimit.Wait(wait)
}

func newConnection(peerAddr string, stream DiscoveryStream) *Connection {
	return &Connection{
		Connection: xds.NewConnection(peerAddr, stream),
	}
}

var PushOrder = []string{
	v3.ClusterType,
	v3.EndpointType,
	v3.ListenerType,
	v3.RouteType,
	v3.AddressType,
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

func (s *DiscoveryServer) pushConnection(con *Connection, pushEv *Event) error {
	pushRequest := pushEv.pushRequest

	if pushRequest.Full {
		// Update Proxy with current information.
		s.computeProxyState(con.proxy, pushRequest)
	}

	pushRequest, needsPush := s.ProxyNeedsPush(con.proxy, pushRequest)
	if !needsPush {
		klog.V(2).Infof("Skipping push to %v, no updates required", con.ID())
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
	klog.V(2).Infof("ADS:%s: REQ %s resources:%d nonce:%s ", stype,
		con.ID(), len(req.ResourceNames), req.ResponseNonce)
	if req.TypeUrl == v3.HealthInfoType {
		return nil
	}

	if strings.HasPrefix(req.TypeUrl, v3.DebugType) {
		return s.pushXds(con,
			&model.WatchedResource{TypeUrl: req.TypeUrl, ResourceNames: sets.New(req.ResourceNames...)},
			&model.PushRequest{Full: true, Push: con.proxy.LastPushContext, Forced: true})
	}

	shouldRespond, delta := xds.ShouldRespond(con.proxy, con.ID(), req)
	if !shouldRespond {
		return nil
	}

	request := &model.PushRequest{
		Full:   true,
		Push:   con.proxy.LastPushContext,
		Reason: model.NewReasonStats(model.ProxyRequest),
		Start:  con.proxy.LastPushTime,
		Delta:  delta,
		Forced: true,
	}
	return s.pushXds(con, con.proxy.GetWatchedResource(req.TypeUrl), request)
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
