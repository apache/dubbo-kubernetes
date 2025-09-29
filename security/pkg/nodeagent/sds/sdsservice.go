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

package sds

import (
	"context"
	"github.com/apache/dubbo-kubernetes/pkg/backoff"
	"github.com/apache/dubbo-kubernetes/pkg/security"
	"github.com/apache/dubbo-kubernetes/pkg/util/sets"
	"github.com/apache/dubbo-kubernetes/pkg/xds"
	core "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	discovery "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v3"
	sds "github.com/envoyproxy/go-control-plane/envoy/service/secret/v3"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	mesh "istio.io/api/mesh/v1alpha1"
	"k8s.io/klog/v2"
	"strconv"
	"sync"
	"sync/atomic"
)

type sdsservice struct {
	st security.SecretManager

	stop       chan struct{}
	rootCaPath string
	pkpConf    *mesh.PrivateKeyProvider

	sync.Mutex
	clients map[string]*Context
}

type Context struct {
	BaseConnection xds.Connection
	s              *sdsservice
	w              *Watch
}

type Watch struct {
	sync.Mutex
	watch *xds.WatchedResource
}

func newSDSService(st security.SecretManager, options *security.Options, pkpConf *mesh.PrivateKeyProvider) *sdsservice {
	ret := &sdsservice{
		st:      st,
		stop:    make(chan struct{}),
		pkpConf: pkpConf,
		clients: make(map[string]*Context),
	}

	ret.rootCaPath = options.CARootPath

	if options.FileMountedCerts || options.ServeOnlyFiles {
		return ret
	}

	// Pre-generate workload certificates to improve startup latency and ensure that for OUTPUT_CERTS
	// case we always write a certificate. A workload can technically run without any mTLS/CA
	// configured, in which case this will fail; if it becomes noisy we should disable the entire SDS
	// server in these cases.
	go func() {
		// TODO: do we need max timeout for retry, seems meaningless to retry forever if it never succeed
		b := backoff.NewExponentialBackOff(backoff.DefaultOption())
		// context for both timeout and channel, whichever stops first, the context will be done
		ctx, cancel := context.WithCancel(context.Background())
		go func() {
			select {
			case <-ret.stop:
				cancel()
			case <-ctx.Done():
			}
		}()
		defer cancel()
		_ = b.RetryWithContext(ctx, func() error {
			_, err := st.GenerateSecret(security.WorkloadKeyCertResourceName)
			if err != nil {
				klog.Warningf("failed to warm certificate: %v", err)
				return err
			}

			_, err = st.GenerateSecret(security.RootCertReqResourceName)
			if err != nil {
				klog.Warningf("failed to warm root certificate: %v", err)
				return err
			}

			return nil
		})
	}()

	return ret
}

// StreamSecrets serves SDS discovery requests and SDS push requests
func (s *sdsservice) StreamSecrets(stream sds.SecretDiscoveryService_StreamSecretsServer) error {
	return xds.Stream(&Context{
		BaseConnection: xds.NewConnection("", stream),
		s:              s,
		w:              &Watch{},
	})
}

func (s *sdsservice) DeltaSecrets(stream sds.SecretDiscoveryService_DeltaSecretsServer) error {
	return status.Error(codes.Unimplemented, "DeltaSecrets not implemented")
}

func (s *sdsservice) FetchSecrets(ctx context.Context, discReq *discovery.DiscoveryRequest) (*discovery.DiscoveryResponse, error) {
	return nil, status.Error(codes.Unimplemented, "FetchSecrets not implemented")
}

// register adds the SDS handle to the grpc server
func (s *sdsservice) register(rpcs *grpc.Server) {
	sds.RegisterSecretDiscoveryServiceServer(rpcs, s)
}

func (s *sdsservice) Close() {
	close(s.stop)
}

func (c *Context) XdsConnection() *xds.Connection {
	return &c.BaseConnection
}

var connectionNumber = int64(0)

func (c *Context) Initialize(_ *core.Node) error {
	id := atomic.AddInt64(&connectionNumber, 1)
	con := c.XdsConnection()
	con.SetID(strconv.FormatInt(id, 10))

	c.s.Lock()
	c.s.clients[con.ID()] = c
	c.s.Unlock()

	con.MarkInitialized()
	return nil
}

func (c *Context) Close() {
	c.s.Lock()
	defer c.s.Unlock()
	delete(c.s.clients, c.XdsConnection().ID())
}

func (c *Context) Watcher() xds.Watcher {
	return c.w
}

func (c *Context) Process(req *discovery.DiscoveryRequest) error {
	shouldRespond, delta := xds.ShouldRespond(c.Watcher(), c.XdsConnection().ID(), req)
	if !shouldRespond {
		return nil
	}
	resources := req.ResourceNames
	if !delta.IsEmpty() {
		resources = delta.Subscribed.UnsortedList()
	}
	res, err := c.s.generate(resources)
	if err != nil {
		return err
	}
	return xds.Send(c, res)
}

func (c *Context) Push(ev any) error {
	secretName := ev.(string)
	if !c.w.requested(secretName) {
		return nil
	}
	res, err := c.s.generate([]string{secretName})
	if err != nil {
		return err
	}
	return xds.Send(c, res)
}

func (w *Watch) requested(secretName string) bool {
	w.Lock()
	defer w.Unlock()
	if w.watch != nil {
		return w.watch.ResourceNames.Contains(secretName)
	}
	return false
}

func (w *Watch) GetWatchedResource(string) *xds.WatchedResource {
	w.Lock()
	defer w.Unlock()
	return w.watch
}

func (w *Watch) NewWatchedResource(typeURL string, names []string) {
	w.Lock()
	defer w.Unlock()
	w.watch = &xds.WatchedResource{TypeUrl: typeURL, ResourceNames: sets.New(names...)}
}

func (w *Watch) UpdateWatchedResource(_ string, f func(*xds.WatchedResource) *xds.WatchedResource) {
	w.Lock()
	defer w.Unlock()
	w.watch = f(w.watch)
}

func (w *Watch) DeleteWatchedResource(string) {
	w.Lock()
	defer w.Unlock()
	w.watch = nil
}

func (w *Watch) GetID() string {
	// This always maps to the same local Envoy instance.
	return ""
}

func (s *sdsservice) generate(resourceNames []string) (*discovery.DiscoveryResponse, error) {
	return &discovery.DiscoveryResponse{}, nil
}

func (s *sdsservice) push(secretName string) {
	s.Lock()
	defer s.Unlock()
	for _, client := range s.clients {
		go func(client *Context) {
			select {
			case client.XdsConnection().PushCh() <- secretName:
			case <-client.XdsConnection().StreamDone():
			}
		}(client)
	}
}
