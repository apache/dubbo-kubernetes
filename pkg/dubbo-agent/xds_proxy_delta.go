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

package dubboagent

import (
	"context"
	"fmt"
	"time"

	"github.com/apache/dubbo-kubernetes/pkg/channels"
	"github.com/apache/dubbo-kubernetes/pkg/model"
	core "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	discovery "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v3"
	"go.uber.org/atomic"
	google_rpc "google.golang.org/genproto/googleapis/rpc/status"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
)

func (p *XdsProxy) DeltaAggregatedResources(downstream DeltaDiscoveryStream) error {
	conID := connectionNumber.Inc()
	proxyLog.Infof("new delta downstream connection #%d", conID)
	con := &ProxyConnection{
		conID:             conID,
		upstreamError:     make(chan error), // can be produced by recv and send
		downstreamError:   make(chan error), // can be produced by recv and send
		deltaRequestsChan: channels.NewUnbounded[*discovery.DeltaDiscoveryRequest](),
		// Allow a buffer of 1. This ensures we queue up at most 2 (one in process, 1 pending) responses before forwarding.
		deltaResponsesChan: make(chan *discovery.DeltaDiscoveryResponse, 1),
		stopChan:           make(chan struct{}),
		downstreamDeltas:   downstream,
	}
	p.registerStream(con)
	defer p.unregisterStream(con)
	defer proxyLog.Infof("delta downstream connection #%d closed", conID)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*30)
	defer cancel()

	proxyLog.Infof("delta connection #%d building upstream connection to %s", conID, p.dubbodAddress)
	upstreamConn, err := p.buildUpstreamConn(ctx)
	if err != nil {
		proxyLog.Errorf("delta connection #%d failed to connect to upstream %s: %v", conID, p.dubbodAddress, err)
		return err
	}
	proxyLog.Infof("delta connection #%d successfully built upstream connection", conID)
	defer upstreamConn.Close()

	xds := discovery.NewAggregatedDiscoveryServiceClient(upstreamConn)
	ctx = metadata.AppendToOutgoingContext(context.Background(), "ClusterID", p.clusterID)
	for k, v := range p.xdsHeaders {
		ctx = metadata.AppendToOutgoingContext(ctx, k, v)
	}
	// We must propagate upstream termination to Envoy. This ensures that we resume the full XDS sequence on new connection
	return p.handleDeltaUpstream(ctx, con, xds)
}

func (p *XdsProxy) handleDeltaUpstream(ctx context.Context, con *ProxyConnection, xds discovery.AggregatedDiscoveryServiceClient) error {
	proxyLog.Infof("delta connection #%d connecting to upstream: %s", con.conID, p.dubbodAddress)
	deltaUpstream, err := xds.DeltaAggregatedResources(ctx,
		grpc.MaxCallRecvMsgSize(defaultClientMaxReceiveMessageSize))
	if err != nil {
		proxyLog.Errorf("delta connection #%d failed to create stream to upstream %s: %v", con.conID, p.dubbodAddress, err)
		return err
	}
	proxyLog.Infof("xdsproxy connected to delta upstream XDS server: %s id=%d", p.dubbodAddress, con.conID)
	defer proxyLog.Infof("xdsproxy disconnected from delta XDS server: %s id=%d", p.dubbodAddress, con.conID)

	con.upstreamDeltas = deltaUpstream

	go func() {
		for {
			resp, err := con.upstreamDeltas.Recv()
			if err != nil {
				upstreamErr(con, err)
				return
			}
			select {
			case con.deltaResponsesChan <- resp:
			case <-con.stopChan:
			}
		}
	}()

	go p.handleUpstreamDeltaRequest(con)
	go p.handleUpstreamDeltaResponse(con)

	for {
		select {
		case err := <-con.upstreamError:
			return err
		case err := <-con.downstreamError:
			// On downstream error, we will return. This propagates the error to downstream envoy which will trigger reconnect
			return err
		case <-con.stopChan:
			return nil
		}
	}
}

func (p *XdsProxy) handleUpstreamDeltaRequest(con *ProxyConnection) {
	initialRequestsSent := atomic.NewBool(false)
	nodeReceived := atomic.NewBool(false)
	go func() {
		for {
			// recv delta xds requests from envoy
			req, err := con.downstreamDeltas.Recv()
			if err != nil {
				downstreamErr(con, err)
				return
			}

			// Save Node from first request that contains it
			if req.Node != nil && req.Node.Id != "" {
				con.nodeMutex.Lock()
				if con.node == nil {
					// Deep copy to preserve the Node information
					con.node = &core.Node{
						Id:       req.Node.Id,
						Cluster:  req.Node.Cluster,
						Locality: req.Node.Locality,
						Metadata: req.Node.Metadata,
					}
					proxyLog.Debugf("delta connection #%d saved Node: %s", con.conID, req.Node.Id)
					nodeReceived.Store(true)
				}
				con.nodeMutex.Unlock()
			}

			// Skip health check probes that don't have Node information (before we've received any Node)
			if req.TypeUrl == model.HealthInfoType && !nodeReceived.Load() {
				proxyLog.Debugf("delta connection #%d skipping health check probe without Node", con.conID)
				continue
			}

			// For LDS requests (typically the first request), we must have Node for connection initialization
			// For other requests, if we already have Node saved, we can inject it
			con.nodeMutex.RLock()
			hasNode := con.node != nil
			con.nodeMutex.RUnlock()

			// For Delta XDS, any first request (not just LDS) without Node should wait
			// because planet-discovery needs Node in the first request to initialize connection
			if !nodeReceived.Load() && req.Node == nil && req.TypeUrl != model.HealthInfoType {
				proxyLog.Debugf("delta connection #%d received first request without Node (TypeUrl=%s), waiting for Node information", con.conID, model.GetShortType(req.TypeUrl))
				continue
			}

			// Ensure Node is set in request before forwarding
			con.nodeMutex.RLock()
			if con.node != nil && req.Node == nil {
				req.Node = con.node
			}
			con.nodeMutex.RUnlock()

			// Final check: for initialization, we must have Node
			if !hasNode && req.Node == nil && req.TypeUrl != model.HealthInfoType {
				proxyLog.Debugf("delta connection #%d cannot forward request without Node: TypeUrl=%s", con.conID, model.GetShortType(req.TypeUrl))
				continue
			}

			// forward to istiod
			con.sendDeltaRequest(req)
			if !initialRequestsSent.Load() && req.TypeUrl == model.ListenerType {
				// fire off an initial NDS request
				if _, f := p.handlers[model.NameTableType]; f {
					ndsReq := &discovery.DeltaDiscoveryRequest{
						TypeUrl: model.NameTableType,
					}
					// Include Node in internal requests
					con.nodeMutex.RLock()
					if con.node != nil {
						ndsReq.Node = con.node
					}
					con.nodeMutex.RUnlock()
					con.sendDeltaRequest(ndsReq)
				}
				// fire off an initial PCDS request
				if _, f := p.handlers[model.ProxyConfigType]; f {
					pcdsReq := &discovery.DeltaDiscoveryRequest{
						TypeUrl: model.ProxyConfigType,
					}
					// Include Node in internal requests
					con.nodeMutex.RLock()
					if con.node != nil {
						pcdsReq.Node = con.node
					}
					con.nodeMutex.RUnlock()
					con.sendDeltaRequest(pcdsReq)
				}
				// set flag before sending the initial request to prevent race.
				initialRequestsSent.Store(true)
				// Fire of a configured initial request, if there is one
				p.connectedMutex.RLock()
				initialRequest := p.initialDeltaHealthRequest
				if initialRequest != nil {
					// Ensure Node is set in initial health request
					con.nodeMutex.RLock()
					if con.node != nil && initialRequest.Node == nil {
						initialRequest.Node = con.node
					}
					con.nodeMutex.RUnlock()
					con.sendDeltaRequest(initialRequest)
				}
				p.connectedMutex.RUnlock()
			}
		}
	}()

	defer func() {
		_ = con.upstreamDeltas.CloseSend()
	}()
	for {
		select {
		case req := <-con.deltaRequestsChan.Get():
			con.deltaRequestsChan.Load()
			if req.TypeUrl == model.HealthInfoType && !initialRequestsSent.Load() {
				// only send healthcheck probe after LDS request has been sent
				continue
			}

			// Ensure Node is set before sending to upstream
			con.nodeMutex.RLock()
			if con.node != nil && req.Node == nil {
				req.Node = con.node
			}
			con.nodeMutex.RUnlock()

			// Final safety check: don't send if still no Node
			if req.Node == nil || req.Node.Id == "" {
				proxyLog.Warnf("delta connection #%d cannot send request without Node: TypeUrl=%s", con.conID, model.GetShortType(req.TypeUrl))
				continue
			}

			proxyLog.Debugf("Delta request type=%s sub=%d unsub=%d nonce=%s initial=%d",
				model.GetShortType(req.TypeUrl),
				len(req.ResourceNamesSubscribe),
				len(req.ResourceNamesUnsubscribe),
				req.ResponseNonce,
				len(req.InitialResourceVersions))
			if req.TypeUrl == model.ExtensionConfigurationType {
				p.ecdsLastNonce.Store(req.ResponseNonce)
			}

			if err := con.upstreamDeltas.Send(req); err != nil {
				err = fmt.Errorf("send error for type url %s: %v", req.TypeUrl, err)
				upstreamErr(con, err)
				return
			}
		case <-con.stopChan:
			return
		}
	}
}

func (p *XdsProxy) handleUpstreamDeltaResponse(con *ProxyConnection) {
	for {
		select {
		case resp := <-con.deltaResponsesChan:
			// TODO: separate upstream response handling from requests sending, which are both time costly
			proxyLog.Debugf(
				"Upstream delta response id=%d type=%s nonce=%s resources=%d removes=%d",
				con.conID,
				model.GetShortType(resp.TypeUrl),
				resp.Nonce,
				len(resp.Resources),
				len(resp.RemovedResources),
			)
			// Handle internal types (e.g., ProxyConfig) that need special processing
			if h, f := p.handlers[resp.TypeUrl]; f {
				if len(resp.Resources) > 0 {
					// Process the resource with the handler
					err := h(resp.Resources[0].Resource)
					var errorResp *google_rpc.Status
					if err != nil {
						errorResp = &google_rpc.Status{
							Code:    int32(codes.Internal),
							Message: err.Error(),
						}
					}
					// Send ACK/NACK
					ackReq := &discovery.DeltaDiscoveryRequest{
						TypeUrl:       resp.TypeUrl,
						ResponseNonce: resp.Nonce,
						ErrorDetail:   errorResp,
					}
					// Ensure Node is set in ACK requests
					con.nodeMutex.RLock()
					if con.node != nil {
						ackReq.Node = con.node
					}
					con.nodeMutex.RUnlock()
					con.sendDeltaRequest(ackReq)
				}
				// Continue to forward to downstream for transparency
			}

			// Forward all delta responses to downstream (gRPC client)
			if err := con.downstreamDeltas.Send(resp); err != nil {
				proxyLog.Errorf("delta connection #%d failed to send response to downstream: %v", con.conID, err)
				downstreamErr(con, err)
				return
			}

			// If there's no handler, we still need to send ACK for normal XDS types
			if _, hasHandler := p.handlers[resp.TypeUrl]; !hasHandler {
				// Send ACK for normal XDS delta responses
				ackReq := &discovery.DeltaDiscoveryRequest{
					TypeUrl:       resp.TypeUrl,
					ResponseNonce: resp.Nonce,
				}
				// Ensure Node is set in ACK requests
				con.nodeMutex.RLock()
				if con.node != nil {
					ackReq.Node = con.node
				}
				con.nodeMutex.RUnlock()
				con.sendDeltaRequest(ackReq)
			}
		case <-con.stopChan:
			return
		}
	}
}

func (con *ProxyConnection) sendDeltaRequest(req *discovery.DeltaDiscoveryRequest) {
	con.deltaRequestsChan.Put(req)
}
