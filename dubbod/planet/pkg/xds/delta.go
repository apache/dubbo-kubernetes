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

	dubbogrpc "github.com/apache/dubbo-kubernetes/dubbod/planet/pkg/grpc"
	"github.com/apache/dubbo-kubernetes/dubbod/planet/pkg/model"
	v3 "github.com/apache/dubbo-kubernetes/dubbod/planet/pkg/xds/v3"
	"github.com/apache/dubbo-kubernetes/pkg/util/sets"
	"github.com/apache/dubbo-kubernetes/pkg/xds"
	discovery "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v3"
	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"
)

func (s *DiscoveryServer) processDeltaRequest(req *discovery.DeltaDiscoveryRequest, con *Connection) error {
	stype := v3.GetShortType(req.TypeUrl)
	log.Infof("%s: REQ %s resources sub:%d unsub:%d nonce:%s", stype,
		con.ID(), len(req.ResourceNamesSubscribe), len(req.ResourceNamesUnsubscribe), req.ResponseNonce)

	if req.TypeUrl == v3.HealthInfoType {
		return nil
	}
	if strings.HasPrefix(req.TypeUrl, v3.DebugType) {
		return s.pushDeltaXds(con,
			&model.WatchedResource{TypeUrl: req.TypeUrl, ResourceNames: sets.New(req.ResourceNamesSubscribe...)},
			&model.PushRequest{Full: true, Push: con.proxy.LastPushContext, Forced: true})
	}

	shouldRespond := shouldRespondDelta(con, req)
	if !shouldRespond {
		return nil
	}

	subs, _, _ := deltaWatchedResources(nil, req)
	request := &model.PushRequest{
		Full:   true,
		Push:   con.proxy.LastPushContext,
		Reason: model.NewReasonStats(model.ProxyRequest),
		Start:  con.proxy.LastPushTime,
		Delta: model.ResourceDelta{
			Subscribed:   subs,
			Unsubscribed: sets.New(req.ResourceNamesUnsubscribe...).Delete("*"),
		},
		Forced: true,
	}

	err := s.pushDeltaXds(con, con.proxy.GetWatchedResource(req.TypeUrl), request)
	if err != nil {
		return err
	}
	if req.TypeUrl != v3.ClusterType {
		return nil
	}
	return s.forceEDSPush(con)
}

func (s *DiscoveryServer) forceEDSPush(con *Connection) error {
	if dwr := con.proxy.GetWatchedResource(v3.EndpointType); dwr != nil {
		request := &model.PushRequest{
			Full:   true,
			Push:   con.proxy.LastPushContext,
			Reason: model.NewReasonStats(model.DependentResource),
			Start:  con.proxy.LastPushTime,
			Forced: true,
		}
		log.Infof("%s: FORCE %s PUSH for warming.", v3.GetShortType(v3.EndpointType), con.ID())
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
		log.Warnf("%q exceeded rate limit: %v", peerAddr, err)
		return status.Errorf(codes.ResourceExhausted, "request rate limit exceeded: %v", err)
	}

	ids, err := s.authenticate(ctx)
	if err != nil {
		return status.Error(codes.Unauthenticated, err.Error())
	}
	if ids != nil {
		log.Debugf("Authenticated XDS: %v with identity %v", peerAddr, ids)
	} else {
		log.Debugf("Unauthenticated XDS: %v", peerAddr)
	}

	s.globalPushContext().InitContext(s.Env, nil, nil)
	con := newDeltaConnection(peerAddr, stream)

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
				log.Infof("%q %s terminated", con.Peer(), con.ID())
				return
			}
			con.ErrorCh() <- err
			log.Errorf("%q %s terminated with error: %v", con.Peer(), con.ID(), err)
			return
		}
		if firstRequest {
			// probe happens before envoy sends first xDS request
			if req.TypeUrl == v3.HealthInfoType {
				log.Warnf("%q %s send health check probe before normal xDS request", con.Peer(), con.ID())
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
			log.Infof("new delta connection for node:%s", con.ID())
		}

		select {
		case con.deltaReqChan <- req:
		case <-con.deltaStream.Context().Done():
			log.Infof("%q %s terminated with stream closed", con.Peer(), con.ID())
			return
		}
	}
}

func (s *DiscoveryServer) pushConnectionDelta(con *Connection, pushEv *Event) error {
	// Check if connection is initialized - proxy must be set before we can push
	if con.proxy == nil {
		// Connection is not yet initialized, skip this push
		// The push will happen after initialization completes
		log.Debugf("Skipping push to %v, connection not yet initialized", con.ID())
		return nil
	}

	pushRequest := pushEv.pushRequest
	if pushRequest == nil {
		log.Warnf("Skipping push to %v, pushRequest is nil", con.ID())
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
		log.Debugf("Skipping push to %v, no updates required", con.ID())
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

func (s *DiscoveryServer) pushDeltaXds(con *Connection, w *model.WatchedResource, req *model.PushRequest) error {
	if w == nil {
		return nil
	}
	gen := s.findGenerator(w.TypeUrl, con)
	if gen == nil {
		return nil
	}

	var logFiltered string
	var res model.Resources
	var deletedRes model.DeletedResources
	var logdata model.XdsLogDetails
	var usedDelta bool
	var err error
	switch g := gen.(type) {
	case model.XdsDeltaResourceGenerator:
		res, deletedRes, logdata, usedDelta, err = g.GenerateDeltas(con.proxy, req, w)
	case model.XdsResourceGenerator:
		res, logdata, err = g.Generate(con.proxy, w, req)
	}
	if err != nil || (res == nil && deletedRes == nil) {
		return err
	}
	defer func() {}()
	resp := &discovery.DeltaDiscoveryResponse{
		ControlPlane: ControlPlane(w.TypeUrl),
		TypeUrl:      w.TypeUrl,
		// TODO: send different version for incremental eds
		SystemVersionInfo: req.Push.PushVersion,
		Nonce:             nonce(req.Push.PushVersion),
		Resources:         res,
	}
	if usedDelta {
		resp.RemovedResources = deletedRes
	} else if req.Full {
		// similar to sotw
		removed := w.ResourceNames.Copy()
		for _, r := range res {
			removed.Delete(r.Name)
		}
		resp.RemovedResources = sets.SortedList(removed)
	}
	var newResourceNames sets.String
	if len(resp.RemovedResources) > 0 {
		log.Infof("%v REMOVE for node:%s %v", v3.GetShortType(w.TypeUrl), con.ID(), resp.RemovedResources)
	}

	ptype := "PUSH"
	info := ""
	if logdata.Incremental {
		ptype = "PUSH INC"
	}
	if len(logdata.AdditionalInfo) > 0 {
		info = " " + logdata.AdditionalInfo
	}
	if len(logFiltered) > 0 {
		info += logFiltered
	}

	if err := con.sendDelta(resp, newResourceNames); err != nil {
		return err
	}

	switch {
	case !req.Full:
	default:
		log.Infof("%s: %s%s for node:%s resources:%d removed:%d%s",
			v3.GetShortType(w.TypeUrl), ptype, req.PushReason(), con.proxy.ID, len(res), len(resp.RemovedResources), info)
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
		log.Warnf("%s: ACK ERROR %s %s:%s", stype, con.ID(), errCode.String(), request.ErrorDetail.GetMessage())
		con.proxy.UpdateWatchedResource(request.TypeUrl, func(wr *model.WatchedResource) *model.WatchedResource {
			wr.LastError = request.ErrorDetail.GetMessage()
			return wr
		})
		return false
	}

	log.Infof("%s REQUEST %v: sub:%v unsub:%v initial:%v", stype, con.ID(),
		request.ResourceNamesSubscribe, request.ResourceNamesUnsubscribe, request.InitialResourceVersions)
	previousInfo := con.proxy.GetWatchedResource(request.TypeUrl)

	if previousInfo == nil {
		con.proxy.Lock()
		defer con.proxy.Unlock()

		if len(request.InitialResourceVersions) > 0 {
			log.Infof("%s: RECONNECT %s %s resources:%v", stype, con.ID(), request.ResponseNonce, len(request.InitialResourceVersions))
		} else {
			log.Infof("%s: INIT %s %s", stype, con.ID(), request.ResponseNonce)
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
		log.Infof("%s: REQ %s Expired nonce received %s, sent %s", stype,
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
		log.Infof("%s: Subscribed resources check mismatch: %v vs %v", stype, spontaneousReq, subChanged)
	}

	if !subChanged {
		if alwaysRespond {
			log.Infof("%s: FORCE RESPONSE %s for warming.", stype, con.ID())
			return true
		}

		log.Infof("%s: ACK %s %s", stype, con.ID(), request.ResponseNonce)
		return false
	}
	log.Infof("%s: RESOURCE CHANGE %s %s", stype, con.ID(), request.ResponseNonce)

	return true
}

func newDeltaConnection(peerAddr string, stream DeltaDiscoveryStream) *Connection {
	return &Connection{
		Connection:   xds.NewConnection(peerAddr, nil),
		deltaStream:  stream,
		deltaReqChan: make(chan *discovery.DeltaDiscoveryRequest, 1),
	}
}

func nonce(noncePrefix string) string {
	return noncePrefix + uuid.New().String()
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
		log.Infof("Timeout writing %s: %v", conn.ID(), v3.GetShortType(res.TypeUrl))
	}
	return err
}
