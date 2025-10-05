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
	"strings"
	"time"
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

		// Make sure the ConfigsUpdated map exists
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

func (s *DiscoveryServer) initProxyMetadata(node *core.Node) (*model.Proxy, error) {
	return nil, nil
}

func (s *DiscoveryServer) initConnection(node *core.Node, con *Connection, identities []string) error {
	return nil
}

func (s *DiscoveryServer) closeConnection(con *Connection) {
	if con.ID() == "" {
		return
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
		// Allow opt out when rate limiting is set to 0qps
		return nil
	}
	// Give a bit of time for queue to clear out, but if not fail fast. Client will connect to another
	// instance in best case, or retry with backoff.
	wait, cancel := context.WithTimeout(ctx, time.Second)
	defer cancel()
	return s.RequestRateLimit.Wait(wait)
}

func newConnection(peerAddr string, stream DiscoveryStream) *Connection {
	return &Connection{
		Connection: xds.NewConnection(peerAddr, stream),
	}
}

func (conn *Connection) watchedResourcesByOrder() []*model.WatchedResource {
	return nil
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

	// For now, don't let xDS piggyback debug requests start watchers.
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

		// The usage of LastPushTime (rather than time.Now()), is critical here for correctness; This time
		// is used by the XDS cache to determine if a entry is stale. If we use Now() with an old push context,
		// we may end up overriding active cache entries with stale ones.
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
