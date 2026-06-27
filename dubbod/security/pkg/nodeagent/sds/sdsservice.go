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

package sds

import (
	"context"
	"fmt"
	"io"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/apache/dubbo-kubernetes/pkg/backoff"
	"github.com/apache/dubbo-kubernetes/pkg/log"
	xdsmodel "github.com/apache/dubbo-kubernetes/pkg/model"
	"github.com/apache/dubbo-kubernetes/pkg/security"
	"github.com/apache/dubbo-kubernetes/pkg/util/sets"
	"github.com/apache/dubbo-kubernetes/pkg/xds"
	core "github.com/kdubbo/xds-api/core/v1"
	tlsv1 "github.com/kdubbo/xds-api/extensions/transport_sockets/tls/v1"
	discovery "github.com/kdubbo/xds-api/service/discovery/v1"
	sds "github.com/kdubbo/xds-api/service/secret/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/anypb"
)

var sdsServiceLog = log.RegisterScope("sds", "SDS service debugging")

var connectionNumber = int64(0)
var deltaNonceNumber = int64(0)
var versionNumber = int64(0)

type sdsservice struct {
	sds.UnimplementedSecretDiscoveryServiceServer

	st security.SecretManager

	stop       chan struct{}
	rootCaPath string

	sync.Mutex
	clients map[string]*Context
}

type Watch struct {
	sync.Mutex
	watch *xds.WatchedResource
}

type Context struct {
	BaseConnection xds.Connection
	s              *sdsservice
	w              *Watch
}

func newSDSService(st security.SecretManager, options *security.Options) *sdsservice {
	ret := &sdsservice{
		st:      st,
		stop:    make(chan struct{}),
		clients: make(map[string]*Context),
	}

	ret.rootCaPath = options.CARootPath

	if options.FileMountedCerts || options.ServeOnlyFiles {
		return ret
	}

	go func() {
		// TODO: do we need max timeout for retry, seems meaningless to retry forever if it never succeed
		b := backoff.NewExponentialBackOff(backoff.DefaultOption())
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
				sdsServiceLog.Warnf("failed to warm certificate: %v", err)
				return err
			}

			_, err = st.GenerateSecret(security.RootCertReqResourceName)
			if err != nil {
				sdsServiceLog.Warnf("failed to warm root certificate: %v", err)
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
	ctx := &Context{
		BaseConnection: xds.NewConnection("", nil),
		s:              s,
		w:              &Watch{},
	}
	reqCh := make(chan *discovery.DeltaDiscoveryRequest, 1)
	errCh := make(chan error, 1)
	go func() {
		defer close(reqCh)
		for {
			req, err := stream.Recv()
			if err != nil {
				errCh <- err
				return
			}
			select {
			case reqCh <- req:
			case <-stream.Context().Done():
				errCh <- stream.Context().Err()
				return
			}
		}
	}()

	initialized := false
	for {
		select {
		case req, ok := <-reqCh:
			if !ok {
				err := <-errCh
				if err == io.EOF {
					return nil
				}
				return err
			}
			if !initialized {
				if req.Node == nil || req.Node.Id == "" {
					return status.New(codes.InvalidArgument, "missing node information").Err()
				}
				if err := ctx.Initialize(req.Node); err != nil {
					return err
				}
				defer ctx.Close()
				initialized = true
			}
			shouldRespond, delta := ctx.shouldRespondDelta(req)
			if !shouldRespond {
				continue
			}
			resourceNames := ctx.deltaResourceNames(req.TypeUrl, delta)
			if err := ctx.sendDelta(stream, req.TypeUrl, resourceNames, delta.Unsubscribed.UnsortedList()); err != nil {
				return err
			}
		case ev := <-ctx.XdsConnection().PushCh():
			secretName := ev.(string)
			if !ctx.w.requested(secretName) {
				continue
			}
			if err := ctx.sendDelta(stream, xdsmodel.SecretType, []string{secretName}, nil); err != nil {
				return err
			}
		case <-stream.Context().Done():
			if initialized {
				ctx.Close()
			}
			return stream.Context().Err()
		}
	}
}

func (s *sdsservice) FetchSecrets(_ context.Context, discReq *discovery.DiscoveryRequest) (*discovery.DiscoveryResponse, error) {
	if discReq == nil {
		return nil, status.New(codes.InvalidArgument, "missing discovery request").Err()
	}
	return s.generate(discReq.ResourceNames)
}

// register adds the SDS handle to the grpc server
func (s *sdsservice) register(rpcs *grpc.Server) {
	sds.RegisterSecretDiscoveryServiceServer(rpcs, s)
}

func (s *sdsservice) generate(resourceNames []string) (*discovery.DiscoveryResponse, error) {
	resources := make([]*anypb.Any, 0, len(resourceNames))
	for _, resourceName := range resourceNames {
		secret, err := s.st.GenerateSecret(resourceName)
		if err != nil {
			return nil, fmt.Errorf("failed to generate secret for %s: %v", resourceName, err)
		}
		if secret == nil {
			return nil, fmt.Errorf("failed to generate secret for %s: secret is nil", resourceName)
		}
		res, err := anypb.New(toXdsSecret(secret, resourceName))
		if err != nil {
			return nil, fmt.Errorf("failed to marshal secret for %s: %v", resourceName, err)
		}
		resources = append(resources, res)
	}
	version := strconv.FormatInt(atomic.AddInt64(&versionNumber, 1), 10)
	return &discovery.DiscoveryResponse{
		TypeUrl:     xdsmodel.SecretType,
		VersionInfo: version,
		Nonce:       version,
		Resources:   resources,
	}, nil
}

func toXdsSecret(item *security.SecretItem, requestedName string) *tlsv1.Secret {
	name := item.ResourceName
	if name == "" {
		name = requestedName
	}
	secret := &tlsv1.Secret{Name: name}
	if name == security.RootCertReqResourceName {
		secret.Type = &tlsv1.Secret_ValidationContext{
			ValidationContext: &tlsv1.CertificateValidationContext{
				TrustedCa: inlineBytes(item.RootCert),
			},
		}
		return secret
	}
	if cfg, ok := security.SdsCertificateConfigFromResourceName(name); ok && cfg.IsRootCertificate() {
		secret.Type = &tlsv1.Secret_ValidationContext{
			ValidationContext: &tlsv1.CertificateValidationContext{
				TrustedCa: inlineBytes(item.RootCert),
			},
		}
		return secret
	}
	secret.Type = &tlsv1.Secret_TlsCertificate{
		TlsCertificate: &tlsv1.TlsCertificate{
			CertificateChain: inlineBytes(item.CertificateChain),
			PrivateKey:       inlineBytes(item.PrivateKey),
		},
	}
	return secret
}

func inlineBytes(value []byte) *core.DataSource {
	return &core.DataSource{
		Specifier: &core.DataSource_InlineBytes{
			InlineBytes: value,
		},
	}
}

func (c *Context) shouldRespondDelta(req *discovery.DeltaDiscoveryRequest) (bool, xds.ResourceDelta) {
	if req.ErrorDetail != nil {
		errCode := codes.Code(req.ErrorDetail.Code)
		sdsServiceLog.Warnf("%s: ACK ERROR %s %s:%s", xdsmodel.GetShortType(req.TypeUrl), c.XdsConnection().ID(), errCode.String(), req.ErrorDetail.GetMessage())
		c.w.UpdateWatchedResource(req.TypeUrl, func(wr *xds.WatchedResource) *xds.WatchedResource {
			if wr == nil {
				wr = &xds.WatchedResource{TypeUrl: req.TypeUrl}
			}
			wr.LastError = req.ErrorDetail.GetMessage()
			return wr
		})
		return false, xds.ResourceDelta{}
	}

	previous := c.w.GetWatchedResource(req.TypeUrl)
	if previous == nil {
		names, _ := deltaWatchedSecrets(nil, req)
		c.w.NewWatchedResource(req.TypeUrl, names.UnsortedList())
		return true, xds.ResourceDelta{Subscribed: names, Unsubscribed: sets.New(req.ResourceNamesUnsubscribe...)}
	}

	if req.ResponseNonce != "" && req.ResponseNonce != previous.NonceSent {
		sdsServiceLog.Debugf("%s: REQ %s expired nonce received %s, sent %s",
			xdsmodel.GetShortType(req.TypeUrl), c.XdsConnection().ID(), req.ResponseNonce, previous.NonceSent)
		return false, xds.ResourceDelta{}
	}

	spontaneousReq := req.ResponseNonce == ""
	var subChanged bool
	var alwaysRespond bool
	var subscribed sets.String
	c.w.UpdateWatchedResource(req.TypeUrl, func(wr *xds.WatchedResource) *xds.WatchedResource {
		if wr == nil {
			wr = &xds.WatchedResource{TypeUrl: req.TypeUrl}
		}
		before := wr.ResourceNames
		wr.ResourceNames, subChanged = deltaWatchedSecrets(wr.ResourceNames, req)
		subscribed = wr.ResourceNames.Difference(before)
		if !spontaneousReq {
			wr.LastError = ""
			wr.NonceAcked = req.ResponseNonce
		}
		alwaysRespond = wr.AlwaysRespond
		wr.AlwaysRespond = false
		return wr
	})

	if !subChanged {
		if alwaysRespond {
			return true, xds.ResourceDelta{}
		}
		return false, xds.ResourceDelta{}
	}
	return true, xds.ResourceDelta{
		Subscribed:   subscribed,
		Unsubscribed: sets.New(req.ResourceNamesUnsubscribe...),
	}
}

func deltaWatchedSecrets(existing sets.String, req *discovery.DeltaDiscoveryRequest) (sets.String, bool) {
	out := sets.New[string]()
	if existing != nil {
		out = existing.Copy()
	}
	before := out.Copy()
	out.InsertAll(req.ResourceNamesSubscribe...)
	for name := range req.InitialResourceVersions {
		out.Insert(name)
	}
	out.DeleteAll(req.ResourceNamesUnsubscribe...)
	return out, !out.Equals(before)
}

func (c *Context) deltaResourceNames(typeURL string, delta xds.ResourceDelta) []string {
	if len(delta.Subscribed) > 0 {
		return delta.Subscribed.UnsortedList()
	}
	if len(delta.Unsubscribed) > 0 {
		return nil
	}
	wr := c.w.GetWatchedResource(typeURL)
	if wr == nil || wr.ResourceNames == nil {
		return nil
	}
	return wr.ResourceNames.UnsortedList()
}

func (c *Context) sendDelta(stream sds.SecretDiscoveryService_DeltaSecretsServer, typeURL string, resourceNames []string, removed []string) error {
	res, err := c.s.generate(resourceNames)
	if err != nil {
		return err
	}
	nonce := strconv.FormatInt(atomic.AddInt64(&deltaNonceNumber, 1), 10)
	resp := &discovery.DeltaDiscoveryResponse{
		SystemVersionInfo: res.VersionInfo,
		TypeUrl:           typeURL,
		RemovedResources:  removed,
		Nonce:             nonce,
		ControlPlane:      res.ControlPlane,
	}
	if resp.SystemVersionInfo == "" {
		resp.SystemVersionInfo = nonce
	}
	if res.TypeUrl != "" {
		resp.TypeUrl = res.TypeUrl
	}
	for i, resource := range res.Resources {
		name := ""
		if i < len(resourceNames) {
			name = resourceNames[i]
		}
		resp.Resources = append(resp.Resources, &discovery.Resource{
			Name:     name,
			Resource: resource,
		})
	}
	if err := stream.Send(resp); err != nil {
		return err
	}
	c.w.UpdateWatchedResource(resp.TypeUrl, func(wr *xds.WatchedResource) *xds.WatchedResource {
		if wr == nil {
			wr = &xds.WatchedResource{TypeUrl: resp.TypeUrl}
		}
		wr.NonceSent = resp.Nonce
		wr.LastSendTime = time.Now()
		return wr
	})
	return nil
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

func (s *sdsservice) Close() {
	close(s.stop)
}

func (w *Watch) requested(secretName string) bool {
	w.Lock()
	defer w.Unlock()
	if w.watch != nil {
		return w.watch.ResourceNames.Contains(secretName)
	}
	return false
}

func (w *Watch) NewWatchedResource(typeURL string, names []string) {
	w.Lock()
	defer w.Unlock()
	w.watch = &xds.WatchedResource{TypeUrl: typeURL, ResourceNames: sets.New(names...)}
}

func (w *Watch) GetWatchedResource(string) *xds.WatchedResource {
	w.Lock()
	defer w.Unlock()
	return w.watch
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
	return ""
}

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

func (c *Context) Watcher() xds.Watcher {
	return c.w
}

func (c *Context) XdsConnection() *xds.Connection {
	return &c.BaseConnection
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

func (c *Context) Close() {
	c.s.Lock()
	defer c.s.Unlock()
	delete(c.s.clients, c.XdsConnection().ID())
}
