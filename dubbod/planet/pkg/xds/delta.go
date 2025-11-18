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
	"errors"
	"strings"
	"time"

	dubbolog "github.com/apache/dubbo-kubernetes/pkg/log"

	dubbogrpc "github.com/apache/dubbo-kubernetes/dubbod/planet/pkg/grpc"
	"github.com/apache/dubbo-kubernetes/dubbod/planet/pkg/model"
	v3 "github.com/apache/dubbo-kubernetes/dubbod/planet/pkg/xds/v3"
	"github.com/apache/dubbo-kubernetes/pkg/util/sets"
	discovery "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v3"
	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"
)

var deltaLog = dubbolog.RegisterScope("delta", "delta xds debugging")

func (s *DiscoveryServer) forceEDSPush(con *Connection) error {
	if dwr := con.proxy.GetWatchedResource(v3.EndpointType); dwr != nil {
		request := &model.PushRequest{
			Full:   true,
			Push:   con.proxy.LastPushContext,
			Reason: model.NewReasonStats(model.DependentResource),
			Start:  con.proxy.LastPushTime,
			Forced: true,
		}
		deltaLog.Infof("%s: FORCE %s PUSH for warming.", v3.GetShortType(v3.EndpointType), con.ID())
		return s.pushDeltaXds(con, dwr, request)
	}
	return nil
}

func (s *DiscoveryServer) StreamDeltas(stream DeltaDiscoveryStream) error {
	if !s.IsServerReady() {
		return errors.New("server is not ready to serve discovery information")
	}

	ctx := stream.Context()
	peerAddr := "0.0.0.0"
	if peerInfo, ok := peer.FromContext(ctx); ok {
		peerAddr = peerInfo.Addr.String()
	}

	if err := s.WaitForRequestLimit(stream.Context()); err != nil {
		deltaLog.Warnf("%q exceeded rate limit: %v", peerAddr, err)
		return status.Errorf(codes.ResourceExhausted, "request rate limit exceeded: %v", err)
	}

	ids, err := s.authenticate(ctx)
	if err != nil {
		return status.Error(codes.Unauthenticated, err.Error())
	}

	s.globalPushContext().InitContext(s.Env, nil, nil)
	con := newDeltaConnection(peerAddr, stream)
	con.s = s

	go s.receiveDelta(con, ids)

	<-con.InitializedCh()

	for {
		select {
		case req, ok := <-con.deltaReqChan:
			if ok {
				if err := s.processDeltaRequest(req, con); err != nil {
					return err
				}
			} else {
				// Remote side closed connection or error processing the request.
				return <-con.ErrorCh()
			}
		case <-con.StopCh():
			return nil
		default:
		}
		select {
		case req, ok := <-con.deltaReqChan:
			if ok {
				if err := s.processDeltaRequest(req, con); err != nil {
					return err
				}
			} else {
				return <-con.ErrorCh()
			}
		case ev := <-con.PushCh():
			pushEv := ev.(*Event)
			err := s.pushConnectionDelta(con, pushEv)
			pushEv.done()
			if err != nil {
				return err
			}
		case <-con.StopCh():
			return nil
		}
	}
}

func (s *DiscoveryServer) receiveDelta(con *Connection, identities []string) {
	defer func() {
		close(con.deltaReqChan)
		close(con.ErrorCh())
		select {
		case <-con.InitializedCh():
		default:
			close(con.InitializedCh())
		}
	}()
	firstRequest := true
	for {
		req, err := con.deltaStream.Recv()
		if err != nil {
			if dubbogrpc.GRPCErrorType(err) != dubbogrpc.UnexpectedError {
				deltaLog.Infof("%q %s terminated", con.Peer(), con.ID())
				return
			}
			con.ErrorCh() <- err
			deltaLog.Errorf("%q %s terminated with error: %v", con.Peer(), con.ID(), err)
			return
		}
		if firstRequest {
			// probe happens before envoy sends first xDS request
			if req.TypeUrl == v3.HealthInfoType {
				deltaLog.Warnf("%q %s send health check probe before normal xDS request", con.Peer(), con.ID())
				continue
			}
			firstRequest = false
			if req.Node == nil || req.Node.Id == "" {
				con.ErrorCh() <- status.New(codes.InvalidArgument, "missing node information").Err()
				return
			}
			if err := s.initConnection(req.Node, con, identities); err != nil {
				con.ErrorCh() <- err
				return
			}
			defer s.closeConnection(con)
			deltaLog.Infof("new delta connection for node:%s", con.ID())
		}

		subscribeStr := " [wildcard]"
		if len(req.ResourceNamesSubscribe) > 0 {
			subscribeStr = " [" + strings.Join(req.ResourceNamesSubscribe, ", ") + "]"
		}
		unsubscribeStr := ""
		if len(req.ResourceNamesUnsubscribe) > 0 {
			unsubscribeStr = " unsubscribe:[" + strings.Join(req.ResourceNamesUnsubscribe, ", ") + "]"
		}
		deltaLog.Infof("%s: RAW DELTA REQ %s sub:%d%s nonce:%s%s",
			v3.GetShortType(req.TypeUrl), con.ID(), len(req.ResourceNamesSubscribe), subscribeStr,
			req.ResponseNonce, unsubscribeStr)

		select {
		case con.deltaReqChan <- req:
		case <-con.deltaStream.Context().Done():
			deltaLog.Infof("%q %s terminated with stream closed", con.Peer(), con.ID())
			return
		}
	}
}

func (s *DiscoveryServer) pushConnectionDelta(con *Connection, pushEv *Event) error {
	// Check if connection is initialized - proxy must be set before we can push
	if con.proxy == nil {
		// Connection is not yet initialized, skip this push
		// The push will happen after initialization completes
		deltaLog.Debugf("Skipping push to %v, connection not yet initialized", con.ID())
		return nil
	}

	pushRequest := pushEv.pushRequest
	if pushRequest == nil {
		deltaLog.Warnf("Skipping push to %v, pushRequest is nil", con.ID())
		return nil
	}

	if pushRequest.Full {
		// Update Proxy with current information.
		s.computeProxyState(con.proxy, pushRequest)
	}

	// Check if ProxyNeedsPush function is set, otherwise default to always push
	var needsPush bool
	if s.ProxyNeedsPush != nil {
		pushRequest, needsPush = s.ProxyNeedsPush(con.proxy, pushRequest)
	} else {
		// Default behavior: always push if ProxyNeedsPush is not set
		needsPush = true
	}
	if !needsPush {
		deltaLog.Debugf("Skipping push to %v, no updates required", con.ID())
		return nil
	}

	// Send pushes to all generators
	// Each Generator is responsible for determining if the push event requires a push
	wrl := con.watchedResourcesByOrder()
	for _, w := range wrl {
		if err := s.pushDeltaXds(con, w, pushRequest); err != nil {
			return err
		}
	}

	return nil
}

func deltaWatchedResources(existing sets.String, request *discovery.DeltaDiscoveryRequest) (sets.String, bool, bool) {
	res := existing
	if res == nil {
		res = sets.New[string]()
	}
	changed := false
	for _, r := range request.ResourceNamesSubscribe {
		if !res.InsertContains(r) {
			changed = true
		}
	}
	for r := range request.InitialResourceVersions {
		if !res.InsertContains(r) {
			changed = true
		}
	}
	for _, r := range request.ResourceNamesUnsubscribe {
		if res.DeleteContains(r) {
			changed = true
		}
	}
	wildcard := false
	if res.Contains("*") {
		wildcard = true
		res.Delete("*")
	}
	if len(request.ResourceNamesSubscribe) == 0 {
		wildcard = true
	}
	return res, wildcard, changed
}

func shouldRespondDelta(con *Connection, request *discovery.DeltaDiscoveryRequest) bool {
	stype := v3.GetShortType(request.TypeUrl)

	if request.ErrorDetail != nil {
		errCode := codes.Code(request.ErrorDetail.Code)
		deltaLog.Warnf("%s: ACK ERROR %s %s:%s", stype, con.ID(), errCode.String(), request.ErrorDetail.GetMessage())
		con.proxy.UpdateWatchedResource(request.TypeUrl, func(wr *model.WatchedResource) *model.WatchedResource {
			wr.LastError = request.ErrorDetail.GetMessage()
			return wr
		})
		return false
	}

	deltaLog.Infof("%s REQUEST %v: sub:%v unsub:%v initial:%v", stype, con.ID(),
		request.ResourceNamesSubscribe, request.ResourceNamesUnsubscribe, request.InitialResourceVersions)
	previousInfo := con.proxy.GetWatchedResource(request.TypeUrl)

	if previousInfo == nil {
		con.proxy.Lock()
		defer con.proxy.Unlock()

		if len(request.InitialResourceVersions) > 0 {
			deltaLog.Infof("%s: RECONNECT %s %s resources:%v", stype, con.ID(), request.ResponseNonce, len(request.InitialResourceVersions))
		} else {
			deltaLog.Infof("%s: INIT %s %s", stype, con.ID(), request.ResponseNonce)
		}

		res, wildcard, _ := deltaWatchedResources(nil, request)
		skip := request.TypeUrl == v3.AddressType && wildcard
		if skip {
			res = nil
		}
		con.proxy.WatchedResources[request.TypeUrl] = &model.WatchedResource{
			TypeUrl:       request.TypeUrl,
			ResourceNames: res,
			Wildcard:      wildcard,
		}
		return true
	}

	if request.ResponseNonce != "" && request.ResponseNonce != previousInfo.NonceSent {
		deltaLog.Infof("%s: REQ %s Expired nonce received %s, sent %s", stype,
			con.ID(), request.ResponseNonce, previousInfo.NonceSent)
		return false
	}

	spontaneousReq := request.ResponseNonce == ""

	var alwaysRespond bool
	var subChanged bool

	con.proxy.UpdateWatchedResource(request.TypeUrl, func(wr *model.WatchedResource) *model.WatchedResource {
		wr.ResourceNames, _, subChanged = deltaWatchedResources(wr.ResourceNames, request)
		if !spontaneousReq {
			wr.LastError = ""
			wr.NonceAcked = request.ResponseNonce
		}
		alwaysRespond = wr.AlwaysRespond
		wr.AlwaysRespond = false
		return wr
	})

	if spontaneousReq && !subChanged || !spontaneousReq && subChanged {
		deltaLog.Infof("%s: Subscribed resources check mismatch: %v vs %v", stype, spontaneousReq, subChanged)
	}

	if !subChanged {
		if alwaysRespond {
			deltaLog.Infof("%s: FORCE RESPONSE %s for warming.", stype, con.ID())
			return true
		}

		deltaLog.Infof("%s: ACK %s %s", stype, con.ID(), request.ResponseNonce)
		return false
	}
	deltaLog.Infof("%s: RESOURCE CHANGE %s %s", stype, con.ID(), request.ResponseNonce)

	return true
}

func (conn *Connection) sendDelta(res *discovery.DeltaDiscoveryResponse, newResourceNames sets.String) error {
	sendResonse := func() error {
		defer func() {}()
		return conn.deltaStream.Send(res)
	}
	err := sendResonse()
	if err == nil {
		if !strings.HasPrefix(res.TypeUrl, v3.DebugType) {
			conn.proxy.UpdateWatchedResource(res.TypeUrl, func(wr *model.WatchedResource) *model.WatchedResource {
				if wr == nil {
					wr = &model.WatchedResource{TypeUrl: res.TypeUrl}
				}
				// some resources dynamically update ResourceNames. Most don't though
				if newResourceNames != nil {
					wr.ResourceNames = newResourceNames
				}
				wr.NonceSent = res.Nonce
				wr.LastSendTime = time.Now()
				return wr
			})
		}
	} else if status.Convert(err).Code() == codes.DeadlineExceeded {
		deltaLog.Infof("Timeout writing %s: %v", conn.ID(), v3.GetShortType(res.TypeUrl))
	}
	return err
}

func nonce(noncePrefix string) string {
	return noncePrefix + uuid.New().String()
}
