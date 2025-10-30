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

	"github.com/apache/dubbo-kubernetes/pkg/model"
	"github.com/apache/dubbo-kubernetes/pkg/util/sets"
	dubbogrpc "github.com/apache/dubbo-kubernetes/sail/pkg/grpc"
	core "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	discovery "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v3"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/anypb"
	"k8s.io/klog/v2"
)

type DiscoveryStream = discovery.AggregatedDiscoveryService_StreamAggregatedResourcesServer

type Resources = []*discovery.Resource

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

type Watcher interface {
	DeleteWatchedResource(url string)
	GetWatchedResource(url string) *WatchedResource
	NewWatchedResource(url string, names []string)
	UpdateWatchedResource(string, func(*WatchedResource) *WatchedResource)
	GetID() string
}

type ConnectionContext interface {
	XdsConnection() *Connection
	Watcher() Watcher
	Initialize(node *core.Node) error
	Close()
	Process(req *discovery.DiscoveryRequest) error
	Push(ev any) error
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

func Stream(ctx ConnectionContext) error {
	con := ctx.XdsConnection()
	klog.Infof("ADS: Stream starting for connection: %v", con.ID())
	go Receive(ctx)
	<-con.initialized
	klog.Infof("ADS: Connection initialized: %v", con.ID())

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
				klog.Infof("ADS: %q %s terminated", con.peerAddr, con.conID)
				return
			}
			con.errorChan <- err
			klog.Errorf("ADS: %q %s terminated with error: %v", con.peerAddr, con.conID, err)
			return
		}
		if firstRequest {
			if req.TypeUrl == model.HealthInfoType {
				klog.Warningf("ADS: %q %s send health check probe before normal xDS request", con.peerAddr, con.conID)
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
			klog.Infof("ADS: new connection for node:%s", con.conID)
		}

		select {
		case con.reqChan <- req:
		case <-con.stream.Context().Done():
			klog.Infof("ADS: %q %s terminated with stream closed", con.peerAddr, con.conID)
			return
		}
	}
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

func ShouldRespond(w Watcher, id string, request *discovery.DiscoveryRequest) (bool, ResourceDelta) {
	stype := model.GetShortType(request.TypeUrl)

	if request.ErrorDetail != nil {
		errCode := codes.Code(request.ErrorDetail.Code)
		klog.Warningf("ADS:%s: ACK ERROR %s %s:%s", stype, id, errCode.String(), request.ErrorDetail.GetMessage())
		w.UpdateWatchedResource(request.TypeUrl, func(wr *WatchedResource) *WatchedResource {
			wr.LastError = request.ErrorDetail.GetMessage()
			return wr
		})
		return false, emptyResourceDelta
	}

	if shouldUnsubscribe(request) {
		klog.V(2).Infof("ADS:%s: UNSUBSCRIBE %s %s %s", stype, id, request.VersionInfo, request.ResponseNonce)
		w.DeleteWatchedResource(request.TypeUrl)
		return false, emptyResourceDelta
	}

	previousInfo := w.GetWatchedResource(request.TypeUrl)
	// This can happen in two cases:
	// 1. When Envoy starts for the first time, it sends an initial Discovery request to Istiod.
	// 2. When Envoy reconnects to a new Istiod that does not have information about this typeUrl
	// i.e. non empty response nonce.
	// We should always respond with the current resource names.
	if request.ResponseNonce == "" || previousInfo == nil {
		klog.V(2).Infof("ADS:%s: INIT/RECONNECT %s %s %s", stype, id, request.VersionInfo, request.ResponseNonce)
		w.NewWatchedResource(request.TypeUrl, request.ResourceNames)
		return true, emptyResourceDelta
	}

	// If there is mismatch in the nonce, that is a case of expired/stale nonce.
	// A nonce becomes stale following a newer nonce being sent to Envoy.
	// previousInfo.NonceSent can be empty if we previously had shouldRespond=true but didn't send any resources.
	if request.ResponseNonce != previousInfo.NonceSent {
		if previousInfo.NonceSent == "" {
			// Assert we do not end up in an invalid state
			klog.V(2).Infof("ADS:%s: REQ %s Expired nonce received %s, but we never sent any nonce", stype,
				id, request.ResponseNonce)
		}
		klog.V(2).Infof("ADS:%s: REQ %s Expired nonce received %s, sent %s", stype,
			id, request.ResponseNonce, previousInfo.NonceSent)
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

	// We should always respond "alwaysRespond" marked requests to let Envoy finish warming
	// even though Nonce match and it looks like an ACK.
	if alwaysRespond {
		klog.Infof("ADS:%s: FORCE RESPONSE %s for warming.", stype, id)
		return true, emptyResourceDelta
	}

	if len(removed) == 0 && len(added) == 0 {
		klog.V(2).Infof("ADS:%s: ACK %s %s %s", stype, id, request.VersionInfo, request.ResponseNonce)
		return false, emptyResourceDelta
	}
	klog.V(2).Infof("ADS:%s: RESOURCE CHANGE added %v removed %v %s %s %s", stype,
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

func ResourcesToAny(r Resources) []*anypb.Any {
	a := make([]*anypb.Any, 0, len(r))
	for _, rr := range r {
		a = append(a, rr.Resource)
	}
	return a
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

type ResourceDelta struct {
	// Subscribed indicates the client requested these additional resources
	Subscribed sets.String
	// Unsubscribed indicates the client no longer requires these resources
	Unsubscribed sets.String
}

var emptyResourceDelta = ResourceDelta{}

func (rd ResourceDelta) IsEmpty() bool {
	return len(rd.Subscribed) == 0 && len(rd.Unsubscribed) == 0
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
