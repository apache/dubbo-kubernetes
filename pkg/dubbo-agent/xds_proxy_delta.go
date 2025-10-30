package dubboagent

import (
	"context"
	"fmt"
	"github.com/apache/dubbo-kubernetes/pkg/channels"
	"github.com/apache/dubbo-kubernetes/pkg/model"
	discovery "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v3"
	"go.uber.org/atomic"
	google_rpc "google.golang.org/genproto/googleapis/rpc/status"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"k8s.io/klog/v2"
	"time"
)

func (p *XdsProxy) DeltaAggregatedResources(downstream DeltaDiscoveryStream) error {
	con := &ProxyConnection{
		conID:             connectionNumber.Inc(),
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

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	upstreamConn, err := p.buildUpstreamConn(ctx)
	if err != nil {
		klog.Errorf("failed to connect to upstream %s: %v", p.dubbodAddress, err)
		return err
	}
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
	deltaUpstream, err := xds.DeltaAggregatedResources(ctx,
		grpc.MaxCallRecvMsgSize(defaultClientMaxReceiveMessageSize))
	if err != nil {
		// Envoy logs errors again, so no need to log beyond debug level
		klog.Errorf("failed to create delta upstream grpc client: %v", err)
		return err
	}
	klog.Infof("connected to delta upstream XDS server: %s", p.dubbodAddress)
	defer klog.Infof("disconnected from delta XDS server: %s", p.dubbodAddress)

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

func (con *ProxyConnection) sendDeltaRequest(req *discovery.DeltaDiscoveryRequest) {
	con.deltaRequestsChan.Put(req)
}

func (p *XdsProxy) handleUpstreamDeltaRequest(con *ProxyConnection) {
	initialRequestsSent := atomic.NewBool(false)
	go func() {
		for {
			// recv delta xds requests from envoy
			req, err := con.downstreamDeltas.Recv()
			if err != nil {
				downstreamErr(con, err)
				return
			}

			// forward to istiod
			con.sendDeltaRequest(req)
			if !initialRequestsSent.Load() && req.TypeUrl == model.ListenerType {
				// fire off an initial NDS request
				if _, f := p.handlers[model.NameTableType]; f {
					con.sendDeltaRequest(&discovery.DeltaDiscoveryRequest{
						TypeUrl: model.NameTableType,
					})
				}
				// fire off an initial PCDS request
				if _, f := p.handlers[model.ProxyConfigType]; f {
					con.sendDeltaRequest(&discovery.DeltaDiscoveryRequest{
						TypeUrl: model.ProxyConfigType,
					})
				}
				// set flag before sending the initial request to prevent race.
				initialRequestsSent.Store(true)
				// Fire of a configured initial request, if there is one
				p.connectedMutex.RLock()
				initialRequest := p.initialDeltaHealthRequest
				if initialRequest != nil {
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
			klog.V(2).InfoS(
				"Delta request",
				"type", model.GetShortType(req.TypeUrl),
				"sub", len(req.ResourceNamesSubscribe),
				"unsub", len(req.ResourceNamesUnsubscribe),
				"nonce", req.ResponseNonce,
				"initial", len(req.InitialResourceVersions),
			)
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
			klog.V(2).Infof(
				"Upstream response id=%s type=%s nonce=%s resources=%d removes=%d",
				con.conID,
				model.GetShortType(resp.TypeUrl),
				resp.Nonce,
				len(resp.Resources),
				len(resp.RemovedResources),
			)
			if h, f := p.handlers[resp.TypeUrl]; f {
				if len(resp.Resources) == 0 {
					// Empty response, nothing to do
					// This assumes internal types are always singleton
					break
				}
				err := h(resp.Resources[0].Resource)
				var errorResp *google_rpc.Status
				if err != nil {
					errorResp = &google_rpc.Status{
						Code:    int32(codes.Internal),
						Message: err.Error(),
					}
				}
				// Send ACK/NACK
				con.sendDeltaRequest(&discovery.DeltaDiscoveryRequest{
					TypeUrl:       resp.TypeUrl,
					ResponseNonce: resp.Nonce,
					ErrorDetail:   errorResp,
				})
				continue
			}
		case <-con.stopChan:
			return
		}
	}
}
