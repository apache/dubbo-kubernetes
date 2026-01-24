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

package adsc

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"math"
	"net"
	"os"
	"sync"
	"time"

	"github.com/apache/dubbo-kubernetes/api/mesh/v1alpha1"
	"github.com/apache/dubbo-kubernetes/dubbod/discovery/pkg/model"
	"github.com/apache/dubbo-kubernetes/dubbod/discovery/pkg/networking/util"
	v3 "github.com/apache/dubbo-kubernetes/dubbod/discovery/pkg/xds/v3"
	"github.com/apache/dubbo-kubernetes/pkg/backoff"
	"github.com/apache/dubbo-kubernetes/pkg/config/schema/collections"
	"github.com/apache/dubbo-kubernetes/pkg/config/schema/gvk"
	dubbolog "github.com/apache/dubbo-kubernetes/pkg/log"
	"github.com/apache/dubbo-kubernetes/pkg/security"
	"github.com/apache/dubbo-kubernetes/pkg/util/protomarshal"
	"github.com/apache/dubbo-kubernetes/pkg/wellknown"
	cluster "github.com/envoyproxy/go-control-plane/envoy/config/cluster/v3"
	endpoint "github.com/envoyproxy/go-control-plane/envoy/config/endpoint/v3"
	listener "github.com/envoyproxy/go-control-plane/envoy/config/listener/v3"
	route "github.com/envoyproxy/go-control-plane/envoy/config/route/v3"
	discovery "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v3"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/proto"
)

var log = dubbolog.RegisterScope("adsc", "adsc debugging")

type ResponseHandler interface {
	HandleResponse(con *ADSC, response *discovery.DiscoveryResponse)
}

const (
	defaultClientMaxReceiveMessageSize = math.MaxInt32
	defaultInitialConnWindowSize       = 1024 * 1024 // default gRPC InitialWindowSize
	defaultInitialWindowSize           = 1024 * 1024 // default gRPC ConnWindowSize
)

func defaultGrpcDialOptions() []grpc.DialOption {
	return []grpc.DialOption{
		grpc.WithInitialWindowSize(int32(defaultInitialWindowSize)),
		grpc.WithInitialConnWindowSize(int32(defaultInitialConnWindowSize)),
		grpc.WithDefaultCallOptions(grpc.MaxCallRecvMsgSize(defaultClientMaxReceiveMessageSize)),
	}
}

type Config struct {
	// Address of the xDS server
	Address string
	// Namespace defaults to 'default'
	Namespace string
	// CertDir is the directory where mTLS certs are configured.
	// If CertDir and Secret are empty, an insecure connection will be used.
	// TODO: implement SecretManager for cert dir
	CertDir string
	// Secrets is the interface used for getting keys and rootCA.
	SecretManager security.SecretManager
	GrpcOpts      []grpc.DialOption
	// XDSRootCAFile explicitly set the root CA to be used for the XDS connection.
	// Mirrors Envoy file.
	XDSRootCAFile string
	// XDSSAN is the expected SAN of the XDS server. If not set, the ProxyConfig.DiscoveryAddress is used.
	XDSSAN string
	// Revision for this control plane instance. We will only read configs that match this revision.
	Revision string
	// BackoffPolicy determines the reconnect policy.
	BackoffPolicy backoff.BackOff
}

type ADSC struct {
	// Stream is the GRPC connection stream, allowing direct GRPC send operations.
	// Set after Dial is called.
	stream discovery.AggregatedDiscoveryService_StreamAggregatedResourcesClient
	// xds client used to create a stream
	client discovery.AggregatedDiscoveryServiceClient
	conn   *grpc.ClientConn
	// Indicates if the ADSC client is closed
	closed    bool
	watchTime time.Time
	// Updates includes the type of the last update received from the server.
	Updates     chan string
	errChan     chan error
	XDSUpdates  chan *discovery.DiscoveryResponse
	VersionInfo map[string]string
	// Last received message, by type
	Received map[string]*discovery.DiscoveryResponse
	mutex    sync.RWMutex
	Mesh     *v1alpha1.MeshGlobalConfig
	Store    model.ConfigStore
	cfg      *ADSConfig
	// initialLoad tracks the time to receive the initial configuration.
	initialLoad time.Duration
	// indicates if the initial LDS request is sent
	initialLds bool
	// httpListeners contains received listeners with a http_connection_manager filter.
	httpListeners map[string]*listener.Listener
	// tcpListeners contains all listeners of type TCP (not-HTTP)
	tcpListeners map[string]*listener.Listener
	// All received clusters of type eds, keyed by name
	edsClusters map[string]*cluster.Cluster
	// All received clusters of no-eds type, keyed by name
	clusters map[string]*cluster.Cluster
	// All received routes, keyed by route name
	routes map[string]*route.RouteConfiguration
	// All received endpoints, keyed by cluster name
	eds  map[string]*endpoint.ClusterLoadAssignment
	sync map[string]time.Time
}

// ADSConfig for the ADS connection.
type ADSConfig struct {
	Config
	// InitialDiscoveryRequests is a list of resources to watch at first, represented as URLs (for new XDS resource naming)
	// or type URLs.
	InitialDiscoveryRequests []*discovery.DiscoveryRequest
	// ResponseHandler will be called on each DiscoveryResponse.
	// TODO: mirror Generator, allow adding handler per type
	ResponseHandler ResponseHandler
}

// New creates a new ADSC, maintaining a connection to an XDS server.
// Will:
// - get certificate using the Secret provider, if CertRequired
// - connect to the XDS server specified in ProxyConfig
// - send initial request for watched resources
// - wait for response from XDS server
// - on success, start a background thread to maintain the connection, with exp. backoff.
func New(discoveryAddr string, opts *ADSConfig) (*ADSC, error) {
	if opts == nil {
		opts = &ADSConfig{}
	}
	opts.Config = setDefaultConfig(&opts.Config)
	opts.Address = discoveryAddr
	adsc := &ADSC{
		Updates:     make(chan string, 100),
		XDSUpdates:  make(chan *discovery.DiscoveryResponse, 100),
		VersionInfo: map[string]string{},
		Received:    map[string]*discovery.DiscoveryResponse{},
		cfg:         opts,
		sync:        map[string]time.Time{},
		errChan:     make(chan error, 10),
	}
	if err := adsc.Dial(); err != nil {
		return nil, err
	}

	return adsc, nil
}

func setDefaultConfig(config *Config) Config {
	if config == nil {
		config = &Config{}
	}
	if config.Namespace == "" {
		config.Namespace = "default"
	}
	return *config
}

func ConfigInitialRequests() []*discovery.DiscoveryRequest {
	out := make([]*discovery.DiscoveryRequest, 0, len(collections.Dubbo.All())+1)
	out = append(out, &discovery.DiscoveryRequest{
		TypeUrl: gvk.MeshGlobalConfig.String(),
	})
	for _, sch := range collections.Dubbo.All() {
		out = append(out, &discovery.DiscoveryRequest{
			TypeUrl: sch.GroupVersionKind().String(),
		})
	}

	return out
}

func dialWithConfig(ctx context.Context, config *Config) (*grpc.ClientConn, error) {
	defaultGrpcDialOptions := defaultGrpcDialOptions()
	var grpcDialOptions []grpc.DialOption
	grpcDialOptions = append(grpcDialOptions, defaultGrpcDialOptions...)
	grpcDialOptions = append(grpcDialOptions, config.GrpcOpts...)

	var err error
	// If we need MTLS - CertDir or Secrets provider is set.
	if len(config.CertDir) > 0 || config.SecretManager != nil {
		tlsCfg, err := tlsConfig(config)
		if err != nil {
			return nil, err
		}
		creds := credentials.NewTLS(tlsCfg)
		grpcDialOptions = append(grpcDialOptions, grpc.WithTransportCredentials(creds))
	}

	if len(grpcDialOptions) == len(defaultGrpcDialOptions) {
		// Only disable transport security if the user didn't supply custom dial options
		grpcDialOptions = append(grpcDialOptions, grpc.WithTransportCredentials(insecure.NewCredentials()))
	}

	conn, err := grpc.DialContext(ctx, config.Address, grpcDialOptions...)
	if err != nil {
		return nil, err
	}
	return conn, nil
}

func tlsConfig(config *Config) (*tls.Config, error) {
	var serverCABytes []byte
	var err error

	getClientCertificate := getClientCertFn(config)

	// Load the root CAs
	if config.XDSRootCAFile != "" {
		serverCABytes, err = os.ReadFile(config.XDSRootCAFile)
		if err != nil {
			return nil, err
		}
	} else if config.SecretManager != nil {
		// This is a bit crazy - we could just use the file
		rootCA, err := config.SecretManager.GenerateSecret(security.RootCertReqResourceName)
		if err != nil {
			return nil, err
		}

		serverCABytes = rootCA.RootCert
	} else if config.CertDir != "" {
		serverCABytes, err = os.ReadFile(config.CertDir + "/root-cert.pem")
		if err != nil {
			return nil, err
		}
	}

	serverCAs := x509.NewCertPool()
	if ok := serverCAs.AppendCertsFromPEM(serverCABytes); !ok {
		return nil, err
	}

	shost, _, _ := net.SplitHostPort(config.Address)
	if config.XDSSAN != "" {
		shost = config.XDSSAN
	}

	// it's insecure only when a user explicitly enable insecure mode.
	return &tls.Config{
		GetClientCertificate: getClientCertificate,
		RootCAs:              serverCAs,
		ServerName:           shost,
	}, nil
}

// Dial connects to a ADS server, with optional MTLS authentication if a cert dir is specified.
func (a *ADSC) Dial() error {
	conn, err := dialWithConfig(context.Background(), &a.cfg.Config)
	if err != nil {
		return err
	}
	a.conn = conn
	return nil
}

// Raw send of a request.
func (a *ADSC) Send(req *discovery.DiscoveryRequest) error {
	req.ResponseNonce = time.Now().String()
	return a.stream.Send(req)
}

// WaitClear will clear the waiting events, so next call to Wait will get
// the next push type.
func (a *ADSC) WaitClear() {
	for {
		select {
		case <-a.Updates:
		default:
			return
		}
	}
}

// HasSynced returns true if configs have synced
func (a *ADSC) HasSynced() bool {
	return true
}

// Run will create a new stream using the existing grpc client connection and send the initial xds requests.
// And then it will run a go routine receiving and handling xds response.
// Note: it is non blocking
func (a *ADSC) Run() error {
	var err error
	a.client = discovery.NewAggregatedDiscoveryServiceClient(a.conn)
	a.stream, err = a.client.StreamAggregatedResources(context.Background())
	if err != nil {
		return err
	}

	a.initialLoad = 0
	a.initialLds = false

	// Send the initial requests
	for _, r := range a.cfg.InitialDiscoveryRequests {
		if r.TypeUrl == v3.ClusterType {
			a.watchTime = time.Now()
		}
		_ = a.Send(r)
	}

	go a.handleReceive()
	return nil
}

// Close the stream.
func (a *ADSC) Close() {
	a.mutex.Lock()
	_ = a.conn.Close()
	a.closed = true
	a.mutex.Unlock()
}

// reconnect will create a new stream
func (a *ADSC) reconnect() {
	a.mutex.RLock()
	if a.closed {
		a.mutex.RUnlock()
		return
	}
	a.mutex.RUnlock()

	err := a.Run()
	if err != nil {
		time.AfterFunc(a.cfg.BackoffPolicy.NextBackOff(), a.reconnect)
	}
}

func (a *ADSC) ack(msg *discovery.DiscoveryResponse) {
	var resources []string

	if msg.TypeUrl == v3.EndpointType {
		for c := range a.edsClusters {
			resources = append(resources, c)
		}
	}
	if msg.TypeUrl == v3.RouteType {
		for r := range a.routes {
			resources = append(resources, r)
		}
	}

	_ = a.stream.Send(&discovery.DiscoveryRequest{
		ResponseNonce: msg.Nonce,
		TypeUrl:       msg.TypeUrl,
		VersionInfo:   msg.VersionInfo,
		ResourceNames: resources,
	})
}

func (a *ADSC) handleReceive() {
	// We connected, so reset the backoff
	if a.cfg.BackoffPolicy != nil {
		a.cfg.BackoffPolicy.Reset()
	}
	for {
		var err error
		msg, err := a.stream.Recv()
		if err != nil {
			log.Errorf("connection closed with err: %v", err)
			select {
			case a.errChan <- err:
			default:
			}
			// if 'reconnect' enabled - schedule a new Run
			if a.cfg.BackoffPolicy != nil {
				time.AfterFunc(a.cfg.BackoffPolicy.NextBackOff(), a.reconnect)
			} else {
				a.Close()
				a.WaitClear()
				a.Updates <- ""
				a.XDSUpdates <- nil
				close(a.errChan)
			}
			return
		}

		// Group-value-kind - used for high level api generator.
		// MCP support removed - it's a legacy protocol replaced by APIGenerator

		// TODO WithLabels
		if a.cfg.ResponseHandler != nil {
			a.cfg.ResponseHandler.HandleResponse(a, msg)
		}

		if msg.TypeUrl == gvk.MeshGlobalConfig.String() &&
			len(msg.Resources) > 0 {
			rsc := msg.Resources[0]
			m := &v1alpha1.MeshGlobalConfig{}
			err = proto.Unmarshal(rsc.Value, m)
			if err != nil {
				log.Errorf("Failed to unmarshal mesh config: %v", err)
			}
			a.Mesh = m
			continue
		}

		// Process the resources.
		a.VersionInfo[msg.TypeUrl] = msg.VersionInfo
		switch msg.TypeUrl {
		case v3.ListenerType:
			listeners := make([]*listener.Listener, 0, len(msg.Resources))
			for _, rsc := range msg.Resources {
				valBytes := rsc.Value
				ll := &listener.Listener{}
				_ = proto.Unmarshal(valBytes, ll)
				listeners = append(listeners, ll)
			}
			a.handleLDS(listeners)
		case v3.ClusterType:
			clusters := make([]*cluster.Cluster, 0, len(msg.Resources))
			for _, rsc := range msg.Resources {
				valBytes := rsc.Value
				cl := &cluster.Cluster{}
				_ = proto.Unmarshal(valBytes, cl)
				clusters = append(clusters, cl)
			}
			a.handleCDS(clusters)
		case v3.EndpointType:
			eds := make([]*endpoint.ClusterLoadAssignment, 0, len(msg.Resources))
			for _, rsc := range msg.Resources {
				valBytes := rsc.Value
				el := &endpoint.ClusterLoadAssignment{}
				_ = proto.Unmarshal(valBytes, el)
				eds = append(eds, el)
			}
			a.handleEDS(eds)
		case v3.RouteType:
			routes := make([]*route.RouteConfiguration, 0, len(msg.Resources))
			for _, rsc := range msg.Resources {
				valBytes := rsc.Value
				rl := &route.RouteConfiguration{}
				_ = proto.Unmarshal(valBytes, rl)
				routes = append(routes, rl)
			}
			a.handleRDS(routes)
		default:
			// MCP support removed - it's a legacy protocol replaced by APIGenerator
		}

		// If we got no resource - still save to the store with empty name/namespace, to notify sync
		// This scheme also allows us to chunk large responses !

		// TODO: add hook to inject nacks

		a.mutex.Lock()
		a.Received[msg.TypeUrl] = msg
		a.ack(msg)
		a.mutex.Unlock()

		select {
		case a.XDSUpdates <- msg:
		default:
		}
	}
}

func (a *ADSC) handleLDS(ll []*listener.Listener) {
	lh := map[string]*listener.Listener{}

	routes := []string{}
	ldsSize := 0

	for _, l := range ll {
		ldsSize += proto.Size(l)

		// The last filter is the actual destination for inbound listener
		if l.ApiListener != nil {
			// This is an API Listener
			// TODO: extract VIP and RDS or cluster
			continue
		}
		fc := l.FilterChains[len(l.FilterChains)-1]
		// Find the terminal filter
		filter := fc.Filters[len(fc.Filters)-1]

		// The actual destination will be the next to the last if the last filter is a passthrough filter
		if fc.GetName() == util.PassthroughFilterChain {
			fc = l.FilterChains[len(l.FilterChains)-2]
			filter = fc.Filters[len(fc.Filters)-1]
		}

		switch filter.Name {
		case wellknown.HTTPConnectionManager:
			lh[l.Name] = l

			// Getting from config is too painful..
			port := l.Address.GetSocketAddress().GetPortValue()
			if port == 15002 {
				routes = append(routes, "http_proxy")
			} else {
				routes = append(routes, fmt.Sprintf("%d", port))
			}
		default:
			log.Info(protomarshal.ToJSONWithIndent(l, "  "))
		}
	}

	log.Infof("LDS: http=%d size=%d", len(lh), ldsSize)

	a.mutex.Lock()
	defer a.mutex.Unlock()
	if len(routes) > 0 {
		a.sendResources(v3.RouteType, routes)
	}
	a.httpListeners = lh

	select {
	case a.Updates <- v3.ListenerType:
	default:
	}
}

func (a *ADSC) handleRDS(configurations []*route.RouteConfiguration) {
	vh := 0
	rcount := 0
	size := 0

	rds := map[string]*route.RouteConfiguration{}

	for _, r := range configurations {
		for _, h := range r.VirtualHosts {
			vh++
			for _, rt := range h.Routes {
				rcount++
				// Example: match:<prefix:"/" > route:<cluster:"outbound|9154||load-se-154.local" ...
				log.Debugf("Handle route %v, path %v, cluster %v", h.Name, rt.Match.PathSpecifier, rt.GetRoute().GetCluster())
			}
		}
		rds[r.Name] = r
		size += proto.Size(r)
	}
	if a.initialLoad == 0 {
		a.initialLoad = time.Since(a.watchTime)
		log.Infof("RDS: %d size=%d vhosts=%d routes=%d time=%d", len(configurations), size, vh, rcount, a.initialLoad)
	} else {
		log.Infof("RDS: %d size=%d vhosts=%d routes=%d", len(configurations), size, vh, rcount)
	}

	a.mutex.Lock()
	a.routes = rds
	a.mutex.Unlock()

	select {
	case a.Updates <- v3.RouteType:
	default:
	}
}

func (a *ADSC) handleCDS(ll []*cluster.Cluster) {
	cn := make([]string, 0, len(ll))
	cdsSize := 0
	edscds := map[string]*cluster.Cluster{}
	cds := map[string]*cluster.Cluster{}
	for _, c := range ll {
		cdsSize += proto.Size(c)
		switch v := c.ClusterDiscoveryType.(type) {
		case *cluster.Cluster_Type:
			if v.Type != cluster.Cluster_EDS {
				cds[c.Name] = c
				continue
			}
		}
		cn = append(cn, c.Name)
		edscds[c.Name] = c
	}

	log.Infof("CDS: %d size=%d", len(cn), cdsSize)

	if len(cn) > 0 {
		a.sendResources(v3.EndpointType, cn)
	}

	a.mutex.Lock()
	defer a.mutex.Unlock()
	a.edsClusters = edscds
	a.clusters = cds

	select {
	case a.Updates <- v3.ClusterType:
	default:
	}
}

func (a *ADSC) handleEDS(eds []*endpoint.ClusterLoadAssignment) {
	la := map[string]*endpoint.ClusterLoadAssignment{}
	edsSize := 0
	ep := 0
	for _, cla := range eds {
		edsSize += proto.Size(cla)
		la[cla.ClusterName] = cla
		ep += len(cla.Endpoints)
	}

	log.Infof("eds: %d size=%d ep=%d", len(eds), edsSize, ep)
	if a.initialLoad == 0 && !a.initialLds {
		// first load - Envoy loads listeners after endpoints
		_ = a.stream.Send(&discovery.DiscoveryRequest{
			TypeUrl: v3.ListenerType,
		})
		a.initialLds = true
	}

	a.mutex.Lock()
	defer a.mutex.Unlock()
	a.eds = la

	select {
	case a.Updates <- v3.EndpointType:
	default:
	}
}

func (a *ADSC) sendResources(typeurl string, resource []string) {
	ex := a.Received[typeurl]
	version := ""
	nonce := ""
	if ex != nil {
		version = ex.VersionInfo
		nonce = ex.Nonce
	}
	_ = a.stream.Send(&discovery.DiscoveryRequest{
		ResponseNonce: nonce,
		VersionInfo:   version,
		TypeUrl:       typeurl,
		ResourceNames: resource,
	})
}
