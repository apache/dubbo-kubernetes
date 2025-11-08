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
	"strings"
	"time"

	dubbogrpc "github.com/apache/dubbo-kubernetes/dubbod/planet/pkg/grpc"
	"github.com/apache/dubbo-kubernetes/pkg/model"
	"github.com/apache/dubbo-kubernetes/pkg/util/sets"
	core "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	discovery "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v3"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/anypb"
	"k8s.io/klog/v2"
)

type DiscoveryStream = discovery.AggregatedDiscoveryService_StreamAggregatedResourcesServer

type Resources = []*discovery.Resource

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

type ConnectionContext interface {
	XdsConnection() *Connection
	Watcher() Watcher
	Initialize(node *core.Node) error
	Close()
	Process(req *discovery.DiscoveryRequest) error
	Push(ev any) error
}

type Watcher interface {
	DeleteWatchedResource(url string)
	GetWatchedResource(url string) *WatchedResource
	NewWatchedResource(url string, names []string)
	UpdateWatchedResource(string, func(*WatchedResource) *WatchedResource)
	GetID() string
}

type WatchedResource struct {
	TypeUrl       string
	ResourceNames sets.String
	Wildcard      bool
	NonceSent     string
	NonceAcked    string
	AlwaysRespond bool
	LastSendTime  time.Time
	LastError     string
	LastResources Resources
}

type ResourceDelta struct {
	// Subscribed indicates the client requested these additional resources
	Subscribed sets.String
	// Unsubscribed indicates the client no longer requires these resources
	Unsubscribed sets.String
}

var emptyResourceDelta = ResourceDelta{}

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

func Stream(ctx ConnectionContext) error {
	con := ctx.XdsConnection()
	go Receive(ctx)
	<-con.initialized

	for {
		// Go select{} statements are not ordered; the same channel can be chosen many times.
		// For requests, these are higher priority (client may be blocked on startup until these are done)
		// and often very cheap to handle (simple ACK), so we check it first.
		select {
		case req, ok := <-con.reqChan:
			if ok {
				if err := ctx.Process(req); err != nil {
					return err
				}
			} else {
				// Remote side closed connection or error processing the request.
				return <-con.errorChan
			}
		case <-con.stop:
			return nil
		default:
		}
		// If there wasn't already a request, poll for requests and pushes. Note: if we have a huge
		// amount of incoming requests, we may still send some pushes, as we do not `continue` above;
		// however, requests will be handled ~2x as much as pushes. This ensures a wave of requests
		// cannot completely starve pushes. However, this scenario is unlikely.
		select {
		case req, ok := <-con.reqChan:
			if ok {
				if err := ctx.Process(req); err != nil {
					return err
				}
			} else {
				return <-con.errorChan
			}
		case pushEv := <-con.pushChannel:
			err := ctx.Push(pushEv)
			if err != nil {
				return err
			}
		case <-con.stop:
			return nil
		}
	}
}

func Receive(ctx ConnectionContext) {
	con := ctx.XdsConnection()
	defer func() {
		close(con.errorChan)
		close(con.reqChan)
		select {
		case <-con.initialized:
		default:
			close(con.initialized)
		}
	}()

	firstRequest := true
	for {
		req, err := con.stream.Recv()
		if err != nil {
			if dubbogrpc.GRPCErrorType(err) != dubbogrpc.UnexpectedError {
				klog.Infof("%q %s terminated", con.peerAddr, con.conID)
				return
			}
			con.errorChan <- err
			klog.Errorf("%q %s terminated with error: %v", con.peerAddr, con.conID, err)
			return
		}
		if firstRequest {
			if req.TypeUrl == model.HealthInfoType {
				klog.Warningf("%q %s send health check probe before normal xDS request", con.peerAddr, con.conID)
				continue
			}
			firstRequest = false
			if req.Node == nil || req.Node.Id == "" {
				con.errorChan <- status.New(codes.InvalidArgument, "missing node information").Err()
				return
			}
			if err := ctx.Initialize(req.Node); err != nil {
				con.errorChan <- err
				return
			}
			defer ctx.Close()
			// Connection logged in initConnection() after addCon() to ensure accurate counting
		}

		select {
		case con.reqChan <- req:
		case <-con.stream.Context().Done():
			klog.Infof("%q %s terminated with stream closed", con.peerAddr, con.conID)
			return
		}
	}
}

func Send(ctx ConnectionContext, res *discovery.DiscoveryResponse) error {
	conn := ctx.XdsConnection()
	sendResponse := func() error {
		return conn.stream.Send(res)
	}
	err := sendResponse()
	if err == nil {
		if res.Nonce != "" && !strings.HasPrefix(res.TypeUrl, model.DebugType) {
			ctx.Watcher().UpdateWatchedResource(res.TypeUrl, func(wr *WatchedResource) *WatchedResource {
				if wr == nil {
					wr = &WatchedResource{TypeUrl: res.TypeUrl}
				}
				wr.NonceSent = res.Nonce
				wr.LastSendTime = time.Now()
				return wr
			})
		}
	} else if status.Convert(err).Code() == codes.DeadlineExceeded {
		klog.Infof("Timeout writing %s: %v", conn.conID, model.GetShortType(res.TypeUrl))
	}
	return err
}

func (conn *Connection) ID() string {
	return conn.conID
}

func (conn *Connection) Peer() string {
	return conn.peerAddr
}

func (conn *Connection) SetID(id string) {
	conn.conID = id
}

func (conn *Connection) MarkInitialized() {
	close(conn.initialized)
}

func (conn *Connection) PushCh() chan any {
	return conn.pushChannel
}

func (conn *Connection) StreamDone() <-chan struct{} {
	return conn.stream.Context().Done()
}

func (conn *Connection) InitializedCh() chan struct{} {
	return conn.initialized
}

func (conn *Connection) ErrorCh() chan error {
	return conn.errorChan
}

func (conn *Connection) StopCh() chan struct{} {
	return conn.stop
}

func ShouldRespond(w Watcher, id string, request *discovery.DiscoveryRequest) (bool, ResourceDelta) {
	stype := model.GetShortType(request.TypeUrl)

	if request.ErrorDetail != nil {
		errCode := codes.Code(request.ErrorDetail.Code)
		klog.Warningf("%s: ACK ERROR %s %s:%s", stype, id, errCode.String(), request.ErrorDetail.GetMessage())
		w.UpdateWatchedResource(request.TypeUrl, func(wr *WatchedResource) *WatchedResource {
			wr.LastError = request.ErrorDetail.GetMessage()
			return wr
		})
		return false, emptyResourceDelta
	}

	if shouldUnsubscribe(request) {
		log.Debugf("%s: UNSUBSCRIBE %s %s %s", stype, id, request.VersionInfo, request.ResponseNonce)
		w.DeleteWatchedResource(request.TypeUrl)
		return false, emptyResourceDelta
	}

	previousInfo := w.GetWatchedResource(request.TypeUrl)
	// This can happen in two cases:
	// 1. When Envoy starts for the first time, it sends an initial Discovery request to Istiod.
	// 2. When Envoy reconnects to a new Istiod that does not have information about this typeUrl
	// i.e. non empty response nonce.
	// We should always respond with the current resource names.
	if previousInfo == nil {
		// No previous info - this is a new request
		if request.ResponseNonce == "" {
			// Initial request with no nonce
			log.Debugf("%s: INIT %s %s", stype, id, request.VersionInfo)
			w.NewWatchedResource(request.TypeUrl, request.ResourceNames)
			return true, emptyResourceDelta
		}
		// Client sent a nonce but we have no previous info
		// This could be a reconnect, but if we're getting many requests with different nonces,
		// they're likely stale/expired requests that should be ignored
		// Only treat as reconnect if this is the first request we see with a nonce
		// For subsequent requests with nonces we don't recognize, treat as expired
		log.Debugf("%s: REQ %s Unknown nonce (no previous info): %s, treating as expired/stale", stype, id, request.ResponseNonce)
		// Create the watched resource but don't respond - let the client retry with empty nonce
		w.NewWatchedResource(request.TypeUrl, request.ResourceNames)
		return false, emptyResourceDelta
	}

	// We have previous info - check nonce match
	if request.ResponseNonce == "" {
		// Client sent empty nonce but we have previous info - this is a new request
		log.Debugf("%s: INIT (empty nonce) %s %s", stype, id, request.VersionInfo)
		w.NewWatchedResource(request.TypeUrl, request.ResourceNames)
		return true, emptyResourceDelta
	}

	// If there is mismatch in the nonce, that is a case of expired/stale nonce.
	// A nonce becomes stale following a newer nonce being sent to Envoy.
	// previousInfo.NonceSent can be empty if we previously had shouldRespond=true but didn't send any resources.
	if request.ResponseNonce != previousInfo.NonceSent {
		// Expired/stale nonce - don't respond, just log at debug level
		if previousInfo.NonceSent == "" {
			// We never sent a nonce, but client sent one - this is unusual but treat as expired
			log.Debugf("%s: REQ %s Expired nonce received %s, but we never sent any nonce", stype,
				id, request.ResponseNonce)
		} else {
			// Normal case: client sent stale nonce
			log.Debugf("%s: REQ %s Expired nonce received %s, sent %s", stype,
				id, request.ResponseNonce, previousInfo.NonceSent)
		}
		return false, emptyResourceDelta
	}

	// If it comes here, that means nonce match.
	var previousResources sets.String
	var cur sets.String
	var alwaysRespond bool
	w.UpdateWatchedResource(request.TypeUrl, func(wr *WatchedResource) *WatchedResource {
		// Clear last error, we got an ACK.
		wr.LastError = ""
		previousResources = wr.ResourceNames
		wr.NonceAcked = request.ResponseNonce
		wr.ResourceNames = sets.New(request.ResourceNames...)
		cur = wr.ResourceNames
		alwaysRespond = wr.AlwaysRespond
		wr.AlwaysRespond = false
		return wr
	})

	// Envoy can send two DiscoveryRequests with same version and nonce.
	// when it detects a new resource. We should respond if they change.
	removed := previousResources.Difference(cur)
	added := cur.Difference(previousResources)

	// CRITICAL FIX: For proxyless gRPC, if client sends wildcard (empty ResourceNames) after receiving specific resources,
	// this is likely an ACK and we should NOT push all resources again
	// Check if this is a wildcard request after specific resources were sent
	if len(request.ResourceNames) == 0 && len(previousResources) > 0 && previousInfo.NonceSent != "" {
		// This is a wildcard request after specific resources were sent
		// For proxyless gRPC clients, this should be treated as ACK, not a request for all resources
		// The client already has the resources from the previous push
		// Check if this is proxyless by attempting to get the proxy from watcher
		// For now, we'll check if this is a proxyless scenario by the context
		// If previous resources existed and client sends empty names with matching nonce, it's an ACK
		log.Debugf("%s: wildcard request after specific resources (prev: %d resources, nonce: %s), treating as ACK",
			stype, len(previousResources), previousInfo.NonceSent)
		// Update ResourceNames to keep previous resources (don't clear them)
		w.UpdateWatchedResource(request.TypeUrl, func(wr *WatchedResource) *WatchedResource {
			if wr == nil {
				return nil
			}
			// Keep the previous ResourceNames, don't clear them with empty set
			// The client is ACKing the previous push, not requesting all resources
			wr.ResourceNames = previousResources
			return wr
		})
		return false, emptyResourceDelta
	}

	// We should always respond "alwaysRespond" marked requests to let Envoy finish warming
	// even though Nonce match and it looks like an ACK.
	if alwaysRespond {
		klog.Infof("%s: FORCE RESPONSE %s for warming.", stype, id)
		return true, emptyResourceDelta
	}

	if len(removed) == 0 && len(added) == 0 {
		log.Debugf("%s: ACK %s %s %s", stype, id, request.VersionInfo, request.ResponseNonce)
		return false, emptyResourceDelta
	}
	log.Debugf("%s: RESOURCE CHANGE added %v removed %v %s %s %s", stype,
		added, removed, id, request.VersionInfo, request.ResponseNonce)

	// For non wildcard resource, if no new resources are subscribed, it means we do not need to push.
	if !IsWildcardTypeURL(request.TypeUrl) && len(added) == 0 {
		return false, emptyResourceDelta
	}

	return true, ResourceDelta{
		Subscribed: added,
		// we do not need to set unsubscribed for StoW
	}
}

func shouldUnsubscribe(request *discovery.DiscoveryRequest) bool {
	return len(request.ResourceNames) == 0 && !IsWildcardTypeURL(request.TypeUrl)
}

func IsWildcardTypeURL(typeURL string) bool {
	switch typeURL {
	case model.SecretType, model.EndpointType, model.RouteType, model.ExtensionConfigurationType:
		// By XDS spec, these are not wildcard
		return false
	case model.ClusterType, model.ListenerType:
		// By XDS spec, these are wildcard
		return true
	default:
		// All of our internal types use wildcard semantics
		return true
	}
}

func ResourcesToAny(r Resources) []*anypb.Any {
	a := make([]*anypb.Any, 0, len(r))
	for _, rr := range r {
		a = append(a, rr.Resource)
	}
	return a
}

func (rd ResourceDelta) IsEmpty() bool {
	return len(rd.Subscribed) == 0 && len(rd.Unsubscribed) == 0
}
