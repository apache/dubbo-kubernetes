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
	"github.com/apache/dubbo-kubernetes/navigator/pkg/model"
	"github.com/apache/dubbo-kubernetes/pkg/xds"
	core "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	discovery "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v3"
)

type DeltaDiscoveryStream = discovery.AggregatedDiscoveryService_DeltaAggregatedResourcesServer

type Connection struct {
	xds.Connection
	node         *core.Node
	proxy        *model.Proxy
	deltaStream  DeltaDiscoveryStream
	deltaReqChan chan *discovery.DeltaDiscoveryRequest
	s            *DiscoveryServer
	ids          []string
}

func (conn *Connection) XdsConnection() *xds.Connection {
	return &conn.Connection
}

func (conn *Connection) Proxy() *model.Proxy {
	return conn.proxy
}
