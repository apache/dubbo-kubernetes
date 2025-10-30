package dubboagent

import (
	"context"
	"fmt"
	"math"
	"net"
	"sync"
	"time"

	"github.com/apache/dubbo-kubernetes/pkg/channels"
	dubbokeepalive "github.com/apache/dubbo-kubernetes/pkg/keepalive"
	"github.com/apache/dubbo-kubernetes/pkg/model"
	"github.com/apache/dubbo-kubernetes/pkg/uds"
	dubbogrpc "github.com/apache/dubbo-kubernetes/sail/pkg/grpc"
	"github.com/apache/dubbo-kubernetes/security/pkg/nodeagent/caclient"
	"github.com/apache/dubbo-kubernetes/security/pkg/pki/util"
	discovery "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v3"
	"go.uber.org/atomic"
	google_rpc "google.golang.org/genproto/googleapis/rpc/status"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/reflection"
	"google.golang.org/protobuf/types/known/anypb"
	meshconfig "istio.io/api/mesh/v1alpha1"
	"k8s.io/klog/v2"
)

type (
	DiscoveryStream      = discovery.AggregatedDiscoveryService_StreamAggregatedResourcesServer
	DeltaDiscoveryStream = discovery.AggregatedDiscoveryService_DeltaAggregatedResourcesServer
	DiscoveryClient      = discovery.AggregatedDiscoveryService_StreamAggregatedResourcesClient
	DeltaDiscoveryClient = discovery.AggregatedDiscoveryService_DeltaAggregatedResourcesClient
)

type ResponseHandler func(resp *anypb.Any) error

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
}

func (p *XdsProxy) StreamAggregatedResources(downstream DiscoveryStream) error {
	return p.handleStream(downstream)
}

type adsStream interface {
	Send(*discovery.DiscoveryResponse) error
	Recv() (*discovery.DiscoveryRequest, error)
	Context() context.Context
}

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
}

var connectionNumber = atomic.NewUint32(0)

func (p *XdsProxy) handleStream(downstream adsStream) error {
	con := &ProxyConnection{
		conID:           connectionNumber.Inc(),
		upstreamError:   make(chan error), // can be produced by recv and send
		downstreamError: make(chan error), // can be produced by recv and send
		responsesChan:   make(chan *discovery.DiscoveryResponse, 1),
		stopChan:        make(chan struct{}),
		downstream:      downstream,
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
	return p.handleUpstream(ctx, con, xds)
}

const (
	defaultClientMaxReceiveMessageSize = math.MaxInt32
)

func (p *XdsProxy) handleUpstream(ctx context.Context, con *ProxyConnection, xds discovery.AggregatedDiscoveryServiceClient) error {
	upstream, err := xds.StreamAggregatedResources(ctx,
		grpc.MaxCallRecvMsgSize(defaultClientMaxReceiveMessageSize))
	if err != nil {
		return err
	}
	klog.Infof("connected to upstream XDS server: %s", p.dubbodAddress)

	con.upstream = upstream

	// Handle upstream xds recv
	go func() {
		for {
			resp, err := con.upstream.Recv()
			if err != nil {
				upstreamErr(con, err)
				return
			}
			select {
			case con.responsesChan <- resp:
			case <-con.stopChan:
			}
		}
	}()

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
	go func() {
		for {
			// recv xds requests from envoy
			req, err := con.downstream.Recv()
			if err != nil {
				downstreamErr(con, err)
				return
			}

			// forward to istiod
			con.sendRequest(req)
			if !initialRequestsSent.Load() && req.TypeUrl == model.ListenerType {
				// fire off an initial NDS request
				if _, f := p.handlers[model.NameTableType]; f {
					con.sendRequest(&discovery.DiscoveryRequest{
						TypeUrl: model.NameTableType,
					})
				}
				// fire off an initial PCDS request
				if _, f := p.handlers[model.ProxyConfigType]; f {
					con.sendRequest(&discovery.DiscoveryRequest{
						TypeUrl: model.ProxyConfigType,
					})
				}
				// set flag before sending the initial request to prevent race.
				initialRequestsSent.Store(true)
				// Fire of a configured initial request, if there is one
				p.connectedMutex.RLock()
				initialRequest := p.initialHealthRequest
				if initialRequest != nil {
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
			klog.Infof("request for type url %s", req.TypeUrl)
			if req.TypeUrl == model.ExtensionConfigurationType {
				if req.VersionInfo != "" {
					p.ecdsLastAckVersion.Store(req.VersionInfo)
				}
				p.ecdsLastNonce.Store(req.ResponseNonce)
			}
			if err := con.upstream.Send(req); err != nil {
				err = fmt.Errorf("send error for type url %s: %v", req.TypeUrl, err)
				upstreamErr(con, err)
				return
			}
		case <-con.stopChan:
			return
		}
	}
}

func downstreamErr(con *ProxyConnection, err error) {
	switch dubbogrpc.GRPCErrorType(err) {
	case dubbogrpc.GracefulTermination:
		err = nil
		fallthrough
	case dubbogrpc.ExpectedError:
		klog.Errorf("downstream terminated with status %v", err)
	default:
		klog.Errorf("downstream terminated with unexpected error %v", err)
	}
	select {
	case con.downstreamError <- err:
	case <-con.stopChan:
	}
}

func (p *XdsProxy) handleUpstreamResponse(con *ProxyConnection) {
	for {
		select {
		case resp := <-con.responsesChan:
			// TODO: separate upstream response handling from requests sending, which are both time costly
			klog.V(2).InfoS(
				"Upstream response",
				"id", con.conID,
				"type", model.GetShortType(resp.TypeUrl),
				"resources", len(resp.Resources),
			)
			if h, f := p.handlers[resp.TypeUrl]; f {
				if len(resp.Resources) == 0 {
					// Empty response, nothing to do
					// This assumes internal types are always singleton
					break
				}
				err := h(resp.Resources[0])
				var errorResp *google_rpc.Status
				if err != nil {
					errorResp = &google_rpc.Status{
						Code:    int32(codes.Internal),
						Message: err.Error(),
					}
				}
				// Send ACK/NACK
				con.sendRequest(&discovery.DiscoveryRequest{
					VersionInfo:   resp.VersionInfo,
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

func (con *ProxyConnection) sendRequest(req *discovery.DiscoveryRequest) {
	con.requestsChan.Put(req)
}

func upstreamErr(con *ProxyConnection, err error) {
	switch dubbogrpc.GRPCErrorType(err) {
	case dubbogrpc.GracefulTermination:
		err = nil
		fallthrough
	case dubbogrpc.ExpectedError:
		klog.Errorf("upstream terminated with status %v", err)
	default:
		klog.Errorf("upstream terminated with unexpected error %v", err)
	}
	select {
	case con.upstreamError <- err:
	case <-con.stopChan:
	}
}

func (p *XdsProxy) buildUpstreamConn(ctx context.Context) (*grpc.ClientConn, error) {
	p.optsMutex.RLock()
	opts := p.dialOptions
	p.optsMutex.RUnlock()
	return grpc.DialContext(ctx, p.dubbodAddress, opts...)
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
	}

	if ia.cfg.EnableDynamicProxyConfig && ia.secretCache != nil {
		proxy.handlers[model.ProxyConfigType] = func(resp *anypb.Any) error {
			pc := &meshconfig.ProxyConfig{}
			if err := resp.UnmarshalTo(pc); err != nil {
				klog.Errorf("failed to unmarshal proxy config: %v", err)
				return err
			}
			caCerts := pc.GetCaCertificatesPem()
			klog.Infof("received new certificates to add to mesh trust domain: %v", caCerts)
			trustBundle := []byte{}
			for _, cert := range caCerts {
				trustBundle = util.AppendCertByte(trustBundle, []byte(cert))
			}
			return ia.secretCache.UpdateConfigTrustBundle(trustBundle)
		}
	}

	klog.Infof("Initializing with upstream address %q and cluster %q", proxy.dubbodAddress, proxy.clusterID)

	if err = proxy.initDownstreamServer(); err != nil {
		return nil, err
	}

	// 启动 gRPC 下游服务器，并用 WaitGroup 保证主进程活跃
	ia.wg.Add(1)
	go func() {
		defer ia.wg.Done()
		if err := proxy.downstreamGrpcServer.Serve(proxy.downstreamListener); err != nil {
			klog.Errorf("failed to accept downstream gRPC connection %v", err)
		}
	}()

	return proxy, nil
}

func (p *XdsProxy) initDownstreamServer() error {
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
	if agent.proxyConfig.ControlPlaneAuthPolicy == meshconfig.AuthenticationPolicy_NONE {
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

func (p *XdsProxy) close() {
	close(p.stopChan)
	if p.downstreamGrpcServer != nil {
		p.downstreamGrpcServer.Stop()
	}
	if p.downstreamListener != nil {
		_ = p.downstreamListener.Close()
	}
}
