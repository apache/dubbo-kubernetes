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
	"github.com/apache/dubbo-kubernetes/pkg/xds"
	"github.com/apache/dubbo-kubernetes/sail/pkg/model"
	core "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	discovery "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v3"
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
	return nil
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
