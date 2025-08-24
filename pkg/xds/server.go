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
	discovery "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v3"
	"time"
)

type DiscoveryStream = discovery.AggregatedDiscoveryService_StreamAggregatedResourcesServer

type Connection struct {
	peerAddr    string
	connectedAt time.Time
	conID       string
	pushChannel chan any
	stream      DiscoveryStream
	initialized chan struct{}
	stop        chan struct{}
	reqChan     chan *discovery.DiscoveryRequest
	errorChan   chan error
}

func NewConnection(peerAddr string, stream DiscoveryStream) Connection {
	return Connection{
		pushChannel: make(chan any),
		initialized: make(chan struct{}),
		stop:        make(chan struct{}),
		reqChan:     make(chan *discovery.DiscoveryRequest, 1),
		errorChan:   make(chan error, 1),
		peerAddr:    peerAddr,
		connectedAt: time.Now(),
		stream:      stream,
	}
}
