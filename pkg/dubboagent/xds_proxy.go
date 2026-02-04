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

package dubboagent

import (
	"context"
	"fmt"
	"math"
	"net"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/apache/dubbo-kubernetes/pkg/log"

	meshv1alpha1 "github.com/apache/dubbo-kubernetes/api/mesh/v1alpha1"
	dubbogrpc "github.com/apache/dubbo-kubernetes/dubbod/discovery/pkg/grpc"
	"github.com/apache/dubbo-kubernetes/dubbod/security/pkg/nodeagent/caclient"
	"github.com/apache/dubbo-kubernetes/pkg/channels"
	dubbokeepalive "github.com/apache/dubbo-kubernetes/pkg/keepalive"
	"github.com/apache/dubbo-kubernetes/pkg/model"
	"github.com/apache/dubbo-kubernetes/pkg/pixiu"
	"github.com/apache/dubbo-kubernetes/pkg/uds"
	cluster "github.com/envoyproxy/go-control-plane/envoy/config/cluster/v3"
	core "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	endpoint "github.com/envoyproxy/go-control-plane/envoy/config/endpoint/v3"
	listener "github.com/envoyproxy/go-control-plane/envoy/config/listener/v3"
	route "github.com/envoyproxy/go-control-plane/envoy/config/route/v3"
	discovery "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v3"
	"go.uber.org/atomic"
	google_rpc "google.golang.org/genproto/googleapis/rpc/status"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/reflection"
	"google.golang.org/protobuf/types/known/anypb"
)

const (
	defaultClientMaxReceiveMessageSize = math.MaxInt32
)

var proxyLog = log.RegisterScope("xdsproxy", "xDS Proxy in Dubbo Agent")

var connectionNumber = atomic.NewUint32(0)

type adsStream interface {
	Send(*discovery.DiscoveryResponse) error
	Recv() (*discovery.DiscoveryRequest, error)
	Context() context.Context
}

type (
	DiscoveryStream      = discovery.AggregatedDiscoveryService_StreamAggregatedResourcesServer
	DeltaDiscoveryStream = discovery.AggregatedDiscoveryService_DeltaAggregatedResourcesServer
	DiscoveryClient      = discovery.AggregatedDiscoveryService_StreamAggregatedResourcesClient
	DeltaDiscoveryClient = discovery.AggregatedDiscoveryService_DeltaAggregatedResourcesClient
)

type ResponseHandler func(resp *anypb.Any) error

type ProxyConnection struct {
	conID              uint32
	upstreamError      chan error
	downstreamError    chan error
	requestsChan       *channels.Unbounded[*discovery.DiscoveryRequest]
	responsesChan      chan *discovery.DiscoveryResponse
	deltaRequestsChan  *channels.Unbounded[*discovery.DeltaDiscoveryRequest]
	deltaResponsesChan chan *discovery.DeltaDiscoveryResponse
	stopChan           chan struct{}
	downstream         adsStream
	upstream           DiscoveryClient
	downstreamDeltas   DeltaDiscoveryStream
	upstreamDeltas     DeltaDiscoveryClient
	node               *core.Node // Preserve Node from first request
	nodeMutex          sync.RWMutex
}

type XdsProxy struct {
	stopChan                  chan struct{}
	downstreamGrpcServer      *grpc.Server
	downstreamListener        net.Listener
	optsMutex                 sync.RWMutex
	dialOptions               []grpc.DialOption
	dubbodSAN                 string
	dubbodAddress             string
	xdsHeaders                map[string]string
	xdsUdsPath                string
	proxyAddresses            []string
	ia                        *Agent
	clusterID                 string
	handlers                  map[string]ResponseHandler
	downstreamGrpcOptions     []grpc.ServerOption
	connectedMutex            sync.RWMutex
	connected                 *ProxyConnection
	ecdsLastAckVersion        atomic.String
	ecdsLastNonce             atomic.String
	initialHealthRequest      *discovery.DiscoveryRequest
	initialDeltaHealthRequest *discovery.DeltaDiscoveryRequest
	// Upstream connection for proxyless mode
	bootstrapNode *core.Node
	ConnMutex     sync.RWMutex
	Conn          *ProxyConnection

	pixiuConverter      *pixiu.ConfigConverter
	pixiuConfigPath     string
	pixiuConverterMutex sync.RWMutex
}

func initXdsProxy(ia *Agent) (*XdsProxy, error) {
	var err error

	proxy := &XdsProxy{
		dubbodAddress:         ia.proxyConfig.DiscoveryAddress,
		dubbodSAN:             ia.cfg.DubbodSAN,
		clusterID:             ia.secOpts.ClusterID,
		stopChan:              make(chan struct{}),
		xdsHeaders:            ia.cfg.XDSHeaders,
		xdsUdsPath:            ia.cfg.XdsUdsPath,
		proxyAddresses:        ia.cfg.ProxyIPAddresses,
		ia:                    ia,
		downstreamGrpcOptions: ia.cfg.DownstreamGrpcOptions,
		handlers:              make(map[string]ResponseHandler),
	}

	// Initialize dial options immediately, required for connecting to upstream
	if err = proxy.initDubbodDialOptions(ia); err != nil {
		return nil, fmt.Errorf("failed to init dubbod dial options: %v", err)
	}

	// Initialize Pixiu converter for router mode (Gateway Pods)
	if ia.cfg.ProxyType == model.Router {
		proxy.pixiuConverter = pixiu.NewConfigConverter()
		// Get Pixiu config path from environment variable or use default
		proxy.pixiuConfigPath = os.Getenv("PROXY_CONFIG_PATH")
		if proxy.pixiuConfigPath == "" {
			proxy.pixiuConfigPath = "/etc/pixiu/config/pixiu.yaml"
		}
		proxyLog.Infof("Initialized Pixiu converter for router mode, config path: %s", proxy.pixiuConfigPath)
	}

	proxyLog.Infof("Initializing with upstream address %q and cluster %q", proxy.dubbodAddress, proxy.clusterID)

	if err = proxy.initDownstreamServer(); err != nil {
		return nil, err
	}

	ia.wg.Add(1)
	go func() {
		defer ia.wg.Done()
		proxyLog.Infof("Starting XDS proxy server listening on UDS: %s", proxy.xdsUdsPath)
		proxyLog.Infof("XDS proxy server ready to accept connections on UDS: %s", proxy.xdsUdsPath)
		if err := proxy.downstreamGrpcServer.Serve(proxy.downstreamListener); err != nil {
			proxyLog.Errorf("XDS proxy server stopped accepting connections: %v", err)
		}
		proxyLog.Warnf("XDS proxy server stopped serving on UDS: %s", proxy.xdsUdsPath)
	}()

	// For proxyless mode, establish an upstream connection using bootstrap Node
	// This ensures the proxy is ready even before downstream clients connect
	// Use a retry loop with exponential backoff to automatically reconnect on failures
	ia.wg.Add(1)
	go func() {
		defer ia.wg.Done()
		// Wait for bootstrap Node to be set (with timeout)
		// The Node is set synchronously after bootstrap file generation in agent.Run()
		for i := 0; i < 50; i++ {
			proxy.ConnMutex.RLock()
			nodeReady := proxy.bootstrapNode != nil
			proxy.ConnMutex.RUnlock()
			if nodeReady {
				break
			}
			select {
			case <-proxy.stopChan:
				return
			case <-time.After(100 * time.Millisecond):
			}
		}
		proxy.ConnMutex.RLock()
		nodeReady := proxy.bootstrapNode != nil
		proxy.ConnMutex.RUnlock()
		if !nodeReady {
			proxyLog.Warnf("Bootstrap Node not set after 5 seconds, proceeding anyway")
		}

		maxBackoff := 30 * time.Second
		backoff := time.Second

		for {
			select {
			case <-proxy.stopChan:
				return
			default:
			}

			// Establish connection
			connDone, err := proxy.establishConnection(ia)
			if err != nil {
				// Connection failed, log and retry with exponential backoff
				proxyLog.Warnf("Failed to establish upstream connection: %v, retrying in %v", err, backoff)

				select {
				case <-proxy.stopChan:
					return
				case <-time.After(backoff):
					backoff *= 2
					if backoff > maxBackoff {
						backoff = maxBackoff
					}
				}
				continue
			}

			// Connection successful, reset backoff
			backoff = time.Second
			proxyLog.Infof("Upstream Connected Successfully")
			// Wait for connection to terminate (connDone will be closed when connection dies)
			select {
			case <-proxy.stopChan:
				return
			case <-connDone:
				proxyLog.Warnf("connection terminated, will retry")
			}
		}
	}()

	return proxy, nil
}

func (p *XdsProxy) StreamAggregatedResources(downstream DiscoveryStream) error {
	return p.handleStream(downstream)
}

func (p *XdsProxy) handleStream(downstream adsStream) error {
	conID := connectionNumber.Inc()
	proxyLog.Infof("new downstream connection #%d", conID)
	con := &ProxyConnection{
		conID:           conID,
		upstreamError:   make(chan error), // can be produced by recv and send
		downstreamError: make(chan error), // can be produced by recv and send
		requestsChan:    channels.NewUnbounded[*discovery.DiscoveryRequest](),
		responsesChan:   make(chan *discovery.DiscoveryResponse, 1),
		stopChan:        make(chan struct{}),
		downstream:      downstream,
	}

	p.registerStream(con)
	defer p.unregisterStream(con)
	defer proxyLog.Infof("downstream connection #%d closed", conID)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*30)
	defer cancel()

	proxyLog.Infof("connection #%d building upstream connection to %s", conID, p.dubbodAddress)
	upstreamConn, err := p.buildUpstreamConn(ctx)
	if err != nil {
		proxyLog.Errorf("connection #%d failed to connect to upstream %s: %v", conID, p.dubbodAddress, err)
		return err
	}
	proxyLog.Infof("connection #%d successfully built upstream connection", conID)
	defer upstreamConn.Close()

	xds := discovery.NewAggregatedDiscoveryServiceClient(upstreamConn)
	ctx = metadata.AppendToOutgoingContext(context.Background(), "ClusterID", p.clusterID)
	for k, v := range p.xdsHeaders {
		ctx = metadata.AppendToOutgoingContext(ctx, k, v)
	}
	// We must propagate upstream termination to Envoy. This ensures that we resume the full XDS sequence on new connection
	return p.handleUpstream(ctx, con, xds)
}

func (p *XdsProxy) handleUpstream(ctx context.Context, con *ProxyConnection, xds discovery.AggregatedDiscoveryServiceClient) error {
	proxyLog.Infof("connection #%d connecting to upstream: %s", con.conID, p.dubbodAddress)
	upstream, err := xds.StreamAggregatedResources(ctx,
		grpc.MaxCallRecvMsgSize(defaultClientMaxReceiveMessageSize))
	if err != nil {
		proxyLog.Errorf("connection #%d failed to create stream to upstream %s: %v", con.conID, p.dubbodAddress, err)
		return err
	}
	proxyLog.Infof("Connected to upstream XDS server: %s id=%d", p.dubbodAddress, con.conID)

	// Log when we start receiving responses
	go func() {
		firstResponse := true
		for {
			resp, err := upstream.Recv()
			if err != nil {
				if firstResponse {
					proxyLog.Warnf("connection #%d upstream Recv failed before first response: %v", con.conID, err)
				}
				upstreamErr(con, err)
				return
			}
			if firstResponse {
				proxyLog.Infof("connection #%d received first response from upstream: TypeUrl=%s, Resources=%d",
					con.conID, model.GetShortType(resp.TypeUrl), len(resp.Resources))
				firstResponse = false
			} else {
				proxyLog.Debugf("connection #%d received response from upstream: TypeUrl=%s, Resources=%d, VersionInfo=%s",
					con.conID, model.GetShortType(resp.TypeUrl), len(resp.Resources), resp.VersionInfo)
			}
			select {
			case con.responsesChan <- resp:
			case <-con.stopChan:
				return
			}
		}
	}()

	con.upstream = upstream

	go p.handleUpstreamRequest(con)
	go p.handleUpstreamResponse(con)

	for {
		select {
		case err := <-con.upstreamError:
			return err
		case err := <-con.downstreamError:
			return err
		case <-con.stopChan:
			return nil
		}
	}
}

func (p *XdsProxy) handleUpstreamRequest(con *ProxyConnection) {
	initialRequestsSent := atomic.NewBool(false)
	nodeReceived := atomic.NewBool(false)
	go func() {
		for {
			// recv xds requests from envoy
			req, err := con.downstream.Recv()
			if err != nil {
				proxyLog.Warnf("connection #%d downstream Recv error: %v", con.conID, err)
				downstreamErr(con, err)
				return
			}

			proxyLog.Debugf("connection #%d received request: TypeUrl=%s, Node=%v, ResourceNames=%d",
				con.conID, model.GetShortType(req.TypeUrl), req.Node != nil, len(req.ResourceNames))

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
					proxyLog.Infof("connection #%d saved Node: %s", con.conID, req.Node.Id)
					nodeReceived.Store(true)
				}
				con.nodeMutex.Unlock()
			}

			// Skip health check probes that don't have Node information (before we've received any Node)
			if req.TypeUrl == model.HealthInfoType && !nodeReceived.Load() {
				proxyLog.Debugf("connection #%d skipping health check probe without Node", con.conID)
				continue
			}

			// For LDS requests (typically the first request), we must have Node for connection initialization
			// For other requests, if we already have Node saved, we can inject it
			con.nodeMutex.RLock()
			hasNode := con.node != nil
			con.nodeMutex.RUnlock()

			// For XDS, any first request (not just LDS) without Node should wait
			// because dubbo-discovery needs Node in the first request to initialize connection
			if !nodeReceived.Load() && req.Node == nil && req.TypeUrl != model.HealthInfoType {
				proxyLog.Debugf("connection #%d received first request without Node (TypeUrl=%s), waiting for Node information", con.conID, model.GetShortType(req.TypeUrl))
				continue
			}

			// Ensure Node is set in request before forwarding
			// For proxyless gRPC, we must ensure Node is always set in every request
			con.nodeMutex.RLock()
			if con.node != nil {
				if req.Node == nil {
					// Deep copy Node to avoid race conditions
					req.Node = &core.Node{
						Id:       con.node.Id,
						Cluster:  con.node.Cluster,
						Locality: con.node.Locality,
						Metadata: con.node.Metadata,
					}
					proxyLog.Debugf("connection #%d added saved Node to request", con.conID)
				} else if req.Node.Id == "" {
					// If Node exists but Id is empty, copy from saved node
					req.Node.Id = con.node.Id
					if req.Node.Metadata == nil {
						req.Node.Metadata = con.node.Metadata
					}
					proxyLog.Debugf("connection #%d filled empty Node.Id in request", con.conID)
				}
			}
			con.nodeMutex.RUnlock()

			// Final check: for initialization, we must have Node
			if !hasNode && req.Node == nil && req.TypeUrl != model.HealthInfoType {
				proxyLog.Warnf("connection #%d cannot forward request without Node: TypeUrl=%s", con.conID, model.GetShortType(req.TypeUrl))
				continue
			}

			// forward to dubbod
			con.sendRequest(req)
			if !initialRequestsSent.Load() && req.TypeUrl == model.ListenerType {
				// fire off an initial PCDS request
				if _, f := p.handlers[model.ProxyConfigType]; f {
					pcdsReq := &discovery.DiscoveryRequest{
						TypeUrl: model.ProxyConfigType,
					}
					// Include Node in internal requests
					con.nodeMutex.RLock()
					if con.node != nil {
						pcdsReq.Node = con.node
					}
					con.nodeMutex.RUnlock()
					con.sendRequest(pcdsReq)
				}
				// set flag before sending the initial request to prevent race.
				initialRequestsSent.Store(true)
				// Fire of a configured initial request, if there is one
				p.connectedMutex.RLock()
				initialRequest := p.initialHealthRequest
				if initialRequest != nil {
					// Ensure Node is set in initial health request
					con.nodeMutex.RLock()
					if con.node != nil && initialRequest.Node == nil {
						initialRequest.Node = con.node
					}
					con.nodeMutex.RUnlock()
					con.sendRequest(initialRequest)
				}
				p.connectedMutex.RUnlock()
			}
		}
	}()

	defer con.upstream.CloseSend() // nolint
	for {
		select {
		case req := <-con.requestsChan.Get():
			con.requestsChan.Load()
			if req.TypeUrl == model.HealthInfoType && !initialRequestsSent.Load() {
				// only send healthcheck probe after LDS request has been sent
				continue
			}

			// Ensure Node is set before sending to upstream
			// For proxyless gRPC, we must ensure Node is always set in every request
			con.nodeMutex.RLock()
			if con.node != nil {
				if req.Node == nil {
					// Deep copy Node to avoid race conditions
					req.Node = &core.Node{
						Id:       con.node.Id,
						Cluster:  con.node.Cluster,
						Locality: con.node.Locality,
						Metadata: con.node.Metadata,
					}
				} else if req.Node.Id == "" {
					// If Node exists but Id is empty, copy from saved node
					req.Node.Id = con.node.Id
					if req.Node.Metadata == nil {
						req.Node.Metadata = con.node.Metadata
					}
				}
			}
			con.nodeMutex.RUnlock()

			// Final safety check: don't send if still no Node
			if req.Node == nil || req.Node.Id == "" {
				proxyLog.Warnf("connection #%d cannot send request without Node: TypeUrl=%s", con.conID, model.GetShortType(req.TypeUrl))
				continue
			}

			// Only log at debug level to reduce noise - these are normal operations
			proxyLog.Debugf("connection #%d forwarding request: TypeUrl=%s, Node=%v",
				con.conID, model.GetShortType(req.TypeUrl), req.Node != nil && req.Node.Id != "")
			if err := con.upstream.Send(req); err != nil {
				proxyLog.Errorf("connection #%d failed to send request upstream: TypeUrl=%s, error=%v",
					con.conID, model.GetShortType(req.TypeUrl), err)
				err = fmt.Errorf("send error for type url %s: %v", req.TypeUrl, err)
				upstreamErr(con, err)
				return
			}
			// Only log at debug level to reduce noise - successful sends are normal
			proxyLog.Debugf("connection #%d successfully sent request upstream: TypeUrl=%s",
				con.conID, model.GetShortType(req.TypeUrl))
		case <-con.stopChan:
			return
		}
	}
}

func (p *XdsProxy) handleUpstreamResponse(con *ProxyConnection) {
	for {
		select {
		case resp := <-con.responsesChan:
			// TODO: separate upstream response handling from requests sending, which are both time costly
			proxyLog.Debugf("Upstream response id=%d type=%s resources=%d",
				con.conID,
				model.GetShortType(resp.TypeUrl),
				len(resp.Resources))
			// Handle internal types (e.g., ProxyConfig) that need special processing
			hasHandler := false
			if h, f := p.handlers[resp.TypeUrl]; f {
				hasHandler = true
				if len(resp.Resources) > 0 {
					// Process the resource with the handler
					err := h(resp.Resources[0])
					var errorResp *google_rpc.Status
					if err != nil {
						errorResp = &google_rpc.Status{
							Code:    int32(codes.Internal),
							Message: err.Error(),
						}
					}
					// Send ACK/NACK for internal types
					ackReq := &discovery.DiscoveryRequest{
						VersionInfo:   resp.VersionInfo,
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
					con.sendRequest(ackReq)
					// For ProxyConfig type, we don't forward to downstream as it's only for agent
					if resp.TypeUrl == model.ProxyConfigType {
						continue
					}
				}
			}

			// Update Pixiu config for router mode (Gateway Pods)
			if p.pixiuConverter != nil {
				p.updatePixiuConfig(resp)
			}

			// Forward all non-internal responses to downstream (gRPC client)
			proxyLog.Debugf("connection #%d forwarding response to downstream: TypeUrl=%s, Resources=%d, VersionInfo=%s",
				con.conID, model.GetShortType(resp.TypeUrl), len(resp.Resources), resp.VersionInfo)
			if err := con.downstream.Send(resp); err != nil {
				proxyLog.Errorf("connection #%d failed to send response to downstream: %v", con.conID, err)
				downstreamErr(con, err)
				return
			}
			proxyLog.Debugf("connection #%d successfully forwarded response to downstream: TypeUrl=%s, Resources=%d",
				con.conID, model.GetShortType(resp.TypeUrl), len(resp.Resources))

			// Send ACK for normal XDS responses (if not already sent by handler)
			if !hasHandler {
				ackReq := &discovery.DiscoveryRequest{
					VersionInfo:   resp.VersionInfo,
					TypeUrl:       resp.TypeUrl,
					ResponseNonce: resp.Nonce,
				}
				// Ensure Node is set in ACK requests
				con.nodeMutex.RLock()
				if con.node != nil {
					ackReq.Node = con.node
				}
				con.nodeMutex.RUnlock()
				con.sendRequest(ackReq)
			}
		case <-con.stopChan:
			return
		}
	}
}

func (p *XdsProxy) buildUpstreamConn(ctx context.Context) (*grpc.ClientConn, error) {
	p.optsMutex.RLock()
	opts := p.dialOptions
	p.optsMutex.RUnlock()

	if len(opts) == 0 {
		proxyLog.Warnf("dialOptions is empty, connection may fail. Address: %s", p.dubbodAddress)
	}

	proxyLog.Debugf("dialing %s with %d options", p.dubbodAddress, len(opts))
	conn, err := grpc.DialContext(ctx, p.dubbodAddress, opts...)
	if err != nil {
		return nil, fmt.Errorf("grpc.DialContext failed for %s: %w", p.dubbodAddress, err)
	}
	return conn, nil
}

func (p *XdsProxy) unregisterStream(c *ProxyConnection) {
	p.connectedMutex.Lock()
	defer p.connectedMutex.Unlock()
	if p.connected != nil && p.connected == c {
		close(p.connected.stopChan)
		p.connected = nil
	}
}

func (p *XdsProxy) registerStream(c *ProxyConnection) {
	p.connectedMutex.Lock()
	defer p.connectedMutex.Unlock()
	if p.connected != nil {
		close(p.connected.stopChan)
	}
	p.connected = c
}

func (p *XdsProxy) initDownstreamServer() error {
	// Convert relative path to absolute path for UDS socket
	absPath, err := filepath.Abs(p.xdsUdsPath)
	if err != nil {
		return fmt.Errorf("failed to resolve absolute path for UDS: %v", err)
	}
	p.xdsUdsPath = absPath

	l, err := uds.NewListener(p.xdsUdsPath)
	if err != nil {
		return err
	}
	// TODO: Expose keepalive options to agent cmd line flags.
	opts := p.downstreamGrpcOptions
	opts = append(opts, dubbogrpc.ServerOptions(dubbokeepalive.DefaultOption())...)
	grpcs := grpc.NewServer(opts...)
	discovery.RegisterAggregatedDiscoveryServiceServer(grpcs, p)
	reflection.Register(grpcs)
	p.downstreamGrpcServer = grpcs
	p.downstreamListener = l
	proxyLog.Infof("XDS proxy server initialized, listening on UDS socket: %s", p.xdsUdsPath)
	return nil
}

func (p *XdsProxy) initDubbodDialOptions(agent *Agent) error {
	opts, err := p.buildUpstreamClientDialOpts(agent)
	if err != nil {
		return err
	}

	p.optsMutex.Lock()
	p.dialOptions = opts
	p.optsMutex.Unlock()
	return nil
}

func (p *XdsProxy) buildUpstreamClientDialOpts(sa *Agent) ([]grpc.DialOption, error) {
	// For NONE auth policy, use insecure credentials
	if sa.proxyConfig.ControlPlaneAuthPolicy == meshv1alpha1.AuthenticationPolicy_NONE {
		options := []grpc.DialOption{
			grpc.WithTransportCredentials(insecure.NewCredentials()),
		}
		if sa.secOpts.CredFetcher != nil {
			options = append(options, grpc.WithPerRPCCredentials(caclient.NewDefaultTokenProvider(sa.secOpts)))
		}
		return options, nil
	}

	tlsOpts, err := p.getTLSOptions(sa)
	if err != nil {
		return nil, fmt.Errorf("failed to get TLS options to talk to upstream: %v", err)
	}
	options, err := dubbogrpc.ClientOptions(nil, tlsOpts)
	if err != nil {
		return nil, err
	}
	if sa.secOpts.CredFetcher != nil {
		options = append(options, grpc.WithPerRPCCredentials(caclient.NewDefaultTokenProvider(sa.secOpts)))
	}
	return options, nil
}

func (p *XdsProxy) getTLSOptions(agent *Agent) (*dubbogrpc.TLSOptions, error) {
	if agent.proxyConfig.ControlPlaneAuthPolicy == meshv1alpha1.AuthenticationPolicy_NONE {
		return nil, nil
	}
	xdsCACertPath, err := agent.FindRootCAForXDS()
	if err != nil {
		return nil, fmt.Errorf("failed to find root CA cert for XDS: %v", err)
	}
	key, cert := agent.GetKeyCertsForXDS()
	return &dubbogrpc.TLSOptions{
		RootCert:      xdsCACertPath,
		Key:           key,
		Cert:          cert,
		ServerAddress: agent.proxyConfig.DiscoveryAddress,
		SAN:           p.dubbodSAN,
	}, nil
}

func (p *XdsProxy) SetBootstrapNode(node *core.Node) {
	p.ConnMutex.Lock()
	p.bootstrapNode = node
	p.ConnMutex.Unlock()
}

func (p *XdsProxy) establishConnection(ia *Agent) (<-chan struct{}, error) {
	p.ConnMutex.Lock()
	node := p.bootstrapNode
	// Clean up old connection if it exists
	if p.Conn != nil {
		close(p.Conn.stopChan)
		p.Conn = nil
	}
	p.ConnMutex.Unlock()

	if node == nil {
		return nil, fmt.Errorf("bootstrap node not available")
	}

	if ia.cfg.ProxyType == model.Proxyless {
		proxyLog.Infof("Connecting proxyless upstream connection with Node: %s", node.Id)
	} else if ia.cfg.ProxyType == model.Router {
		proxyLog.Infof("Connecting router upstream connection with Node: %s", node.Id)
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*30)
	defer cancel()

	upstreamConn, err := p.buildUpstreamConn(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to build upstream connection: %w", err)
	}
	// Don't defer close here - connection will be closed when connection terminates

	// Create a channel that will be closed when connection terminates
	connDone := make(chan struct{})

	xds := discovery.NewAggregatedDiscoveryServiceClient(upstreamConn)
	ctx = metadata.AppendToOutgoingContext(context.Background(), "ClusterID", p.clusterID)
	for k, v := range p.xdsHeaders {
		ctx = metadata.AppendToOutgoingContext(ctx, k, v)
	}

	upstream, err := xds.StreamAggregatedResources(ctx,
		grpc.MaxCallRecvMsgSize(defaultClientMaxReceiveMessageSize))
	if err != nil {
		_ = upstreamConn.Close()
		return nil, fmt.Errorf("failed to create upstream stream: %w", err)
	}
	proxyLog.Infof("Connected to upstream XDS server: %s", p.dubbodAddress)

	conID := connectionNumber.Inc()
	con := &ProxyConnection{
		conID:         conID,
		upstreamError: make(chan error, 1),
		requestsChan:  channels.NewUnbounded[*discovery.DiscoveryRequest](),
		responsesChan: make(chan *discovery.DiscoveryResponse, 1),
		stopChan:      make(chan struct{}),
		upstream:      upstream,
		node:          node,
	}

	p.ConnMutex.Lock()
	p.Conn = con
	p.ConnMutex.Unlock()

	// Close upstream connection when connection terminates and signal done channel
	go func() {
		select {
		case <-con.stopChan:
		case <-con.upstreamError:
		case <-p.stopChan:
		}
		_ = upstreamConn.Close()
		p.ConnMutex.Lock()
		if p.Conn == con {
			p.Conn = nil
		}
		p.ConnMutex.Unlock()
		close(connDone)
	}()

	// Send initial LDS request with bootstrap Node to initialize connection
	ldsReq := &discovery.DiscoveryRequest{
		TypeUrl: model.ListenerType,
		Node:    node,
	}
	proxyLog.Debugf("connection sending initial LDS request with Node: %s", node.Id)
	if err := upstream.Send(ldsReq); err != nil {
		_ = upstreamConn.Close()
		p.ConnMutex.Lock()
		if p.Conn == con {
			p.Conn = nil
		}
		p.ConnMutex.Unlock()
		close(connDone)
		return nil, fmt.Errorf("failed to send initial LDS request: %w", err)
	}

	// Handle responses (discard or queue for future downstream clients)
	go func() {
		defer func() {
			// Signal connection termination
			select {
			case con.upstreamError <- fmt.Errorf("response handler terminated"):
			case <-con.stopChan:
			}
		}()
		for {
			select {
			case <-con.stopChan:
				return
			case <-p.stopChan:
				return
			default:
			}

			resp, err := upstream.Recv()
			if err != nil {
				upstreamErr(con, err)
				return
			}
			proxyLog.Debugf("connection received response: TypeUrl=%s, Resources=%d",
				model.GetShortType(resp.TypeUrl), len(resp.Resources))
			// Send ACK
			ackReq := &discovery.DiscoveryRequest{
				VersionInfo:   resp.VersionInfo,
				TypeUrl:       resp.TypeUrl,
				ResponseNonce: resp.Nonce,
				Node:          node,
			}
			con.sendRequest(ackReq)
		}
	}()

	// Send requests
	go func() {
		defer func() {
			// Signal connection termination if send handler exits
			select {
			case con.upstreamError <- fmt.Errorf("send handler terminated"):
			case <-con.stopChan:
			}
		}()
		for {
			select {
			case req := <-con.requestsChan.Get():
				con.requestsChan.Load()
				if req.Node == nil {
					req.Node = node
				}
				if err := upstream.Send(req); err != nil {
					upstreamErr(con, err)
					return
				}
			case <-con.stopChan:
				return
			case <-p.stopChan:
				return
			}
		}
	}()

	// Return immediately - connection is connected and running in background
	// The connDone channel will be closed when connection terminates
	return connDone, nil
}

// updatePixiuConfig updates Pixiu configuration from xDS responses
func (p *XdsProxy) updatePixiuConfig(resp *discovery.DiscoveryResponse) {
	if p.pixiuConverter == nil {
		return
	}

	p.pixiuConverterMutex.Lock()
	defer p.pixiuConverterMutex.Unlock()

	// Process different xDS resource types
	switch resp.TypeUrl {
	case model.ListenerType:
		for _, resource := range resp.Resources {
			var l listener.Listener
			if err := resource.UnmarshalTo(&l); err != nil {
				proxyLog.Warnf("Failed to unmarshal listener: %v", err)
				continue
			}
			p.pixiuConverter.UpdateListener(l.Name, &l)
			proxyLog.Debugf("Updated Pixiu listener: %s", l.Name)
		}
	case model.RouteType:
		for _, resource := range resp.Resources {
			var r route.RouteConfiguration
			if err := resource.UnmarshalTo(&r); err != nil {
				proxyLog.Warnf("Failed to unmarshal route: %v", err)
				continue
			}
			p.pixiuConverter.UpdateRoute(r.Name, &r)
			proxyLog.Debugf("Updated Pixiu route: %s", r.Name)
		}
	case model.ClusterType:
		for _, resource := range resp.Resources {
			var c cluster.Cluster
			if err := resource.UnmarshalTo(&c); err != nil {
				proxyLog.Warnf("Failed to unmarshal cluster: %v", err)
				continue
			}
			p.pixiuConverter.UpdateCluster(c.Name, &c)
			proxyLog.Debugf("Updated Pixiu cluster: %s", c.Name)
		}
	case model.EndpointType:
		for _, resource := range resp.Resources {
			var e endpoint.ClusterLoadAssignment
			if err := resource.UnmarshalTo(&e); err != nil {
				proxyLog.Warnf("Failed to unmarshal endpoint: %v", err)
				continue
			}
			p.pixiuConverter.UpdateEndpoint(e.ClusterName, &e)
			proxyLog.Debugf("Updated Pixiu endpoint: %s", e.ClusterName)
		}
	}

	// Convert to Pixiu config and write to file
	configData, err := p.pixiuConverter.ConvertToPixiuConfig()
	if err != nil {
		proxyLog.Errorf("Failed to convert to Pixiu config: %v", err)
		return
	}

	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(p.pixiuConfigPath), 0755); err != nil {
		proxyLog.Errorf("Failed to create Pixiu config directory: %v", err)
		return
	}

	// Write config file
	if err := os.WriteFile(p.pixiuConfigPath, configData, 0644); err != nil {
		proxyLog.Errorf("Failed to write Pixiu config: %v", err)
		return
	}

	proxyLog.Infof("Updated Pixiu config at %s", p.pixiuConfigPath)
}

func (p *XdsProxy) close() {
	close(p.stopChan)
	p.ConnMutex.Lock()
	if p.Conn != nil {
		close(p.Conn.stopChan)
		p.Conn = nil
	}
	p.ConnMutex.Unlock()
	if p.downstreamGrpcServer != nil {
		p.downstreamGrpcServer.Stop()
	}
	if p.downstreamListener != nil {
		_ = p.downstreamListener.Close()
	}
}

func (con *ProxyConnection) sendRequest(req *discovery.DiscoveryRequest) {
	con.requestsChan.Put(req)
}

func upstreamErr(con *ProxyConnection, err error) {
	switch dubbogrpc.GRPCErrorType(err) {
	case dubbogrpc.GracefulTermination:
		err = nil
		fallthrough
	case dubbogrpc.ExpectedError:
		proxyLog.Errorf("upstream terminated with status %v", err)
	default:
		proxyLog.Errorf("upstream terminated with unexpected error %v", err)
	}
	select {
	case con.upstreamError <- err:
	case <-con.stopChan:
	}
}

func downstreamErr(con *ProxyConnection, err error) {
	switch dubbogrpc.GRPCErrorType(err) {
	case dubbogrpc.GracefulTermination:
		err = nil
		fallthrough
	case dubbogrpc.ExpectedError:
		proxyLog.Errorf("downstream terminated with status %v", err)
	default:
		proxyLog.Errorf("downstream terminated with unexpected error %v", err)
	}
	select {
	case con.downstreamError <- err:
	case <-con.stopChan:
	}
}
