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

package client

import (
	"context"
	"github.com/apache/dubbo-kubernetes/pkg/core"
	"github.com/pkg/errors"
	"google.golang.org/grpc/connectivity"

	"sync"
	"time"
)

var poolLog = core.Log.WithName("intercp").WithName("client").WithName("pool")

type accessedConn struct {
	conn           Conn
	url            string
	lastAccessTime time.Time
}

// Pool keeps the list of clients to inter-cp servers.
// Because the list of inter-cp servers changes in runtime, we need to properly manage the connections to them (initialize, share, close etc.)
// Pool helps us to not reimplement this for every inter-cp service (catalog, envoyadmin, etc.)
type Pool struct {
	newConn      func(string, *TLSConfig) (Conn, error)
	idleDeadline time.Duration // the time after which we close the connection if it was not fetched from the pool
	now          func() time.Time
	connections  map[string]*accessedConn
	mut          sync.Mutex

	tlsCfg *TLSConfig
}

var TLSNotConfigured = errors.New("tls config is not yet set")

func NewPool(
	newConn func(string, *TLSConfig) (Conn, error),
	idleDeadline time.Duration,
	now func() time.Time,
) *Pool {
	return &Pool{
		newConn:      newConn,
		idleDeadline: idleDeadline,
		now:          now,
		connections:  map[string]*accessedConn{},
		mut:          sync.Mutex{},
	}
}

func (c *Pool) Client(serverURL string) (Conn, error) {
	c.mut.Lock()
	defer c.mut.Unlock()
	if c.tlsCfg == nil {
		return nil, TLSNotConfigured
	}
	ac, ok := c.connections[serverURL]
	createNewConnection := !ok
	if ok && ac.conn.GetState() == connectivity.TransientFailure {
		createNewConnection = true
		poolLog.Info("closing broken connection", "url", serverURL)
		if err := ac.conn.Close(); err != nil {
			poolLog.Error(err, "cannot close the connection", "url", serverURL)
		}
	}
	if createNewConnection {
		poolLog.Info("creating new connection", "url", serverURL)
		conn, err := c.newConn(serverURL, c.tlsCfg)
		if err != nil {
			return nil, err
		}
		ac = &accessedConn{
			conn: conn,
			url:  serverURL,
		}
	}
	ac.lastAccessTime = c.now()
	c.connections[serverURL] = ac
	return ac.conn, nil
}

// SetTLSConfig can configure TLS in runtime.
// Because CA of the inter-cp server is managed by the CP in the runtime we cannot configure it when we create the pool.
func (c *Pool) SetTLSConfig(tlsCfg *TLSConfig) {
	c.mut.Lock()
	c.tlsCfg = tlsCfg
	c.mut.Unlock()
}

func (c *Pool) StartCleanup(ctx context.Context, ticker *time.Ticker) {
	for {
		select {
		case now := <-ticker.C:
			c.cleanup(now)
		case <-ctx.Done():
			return
		}
	}
}

func (c *Pool) cleanup(now time.Time) {
	c.mut.Lock()
	defer c.mut.Unlock()
	for url, accessedConn := range c.connections {
		if now.Sub(accessedConn.lastAccessTime) > c.idleDeadline {
			poolLog.Info("closing connection due to lack of activity", "url", accessedConn.url)
			if err := accessedConn.conn.Close(); err != nil {
				poolLog.Error(err, "cannot close the connection", "url", accessedConn.url)
			}
			delete(c.connections, url)
		}
	}
}
