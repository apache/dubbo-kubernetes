package adsc

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"github.com/apache/dubbo-kubernetes/pkg/backoff"
	"github.com/apache/dubbo-kubernetes/pkg/config"
	"github.com/apache/dubbo-kubernetes/pkg/config/schema/collections"
	"github.com/apache/dubbo-kubernetes/pkg/config/schema/gvk"
	"github.com/apache/dubbo-kubernetes/pkg/security"
	"github.com/apache/dubbo-kubernetes/pkg/util/protomarshal"
	"github.com/apache/dubbo-kubernetes/pkg/wellknown"
	"github.com/apache/dubbo-kubernetes/sail/pkg/model"
	"github.com/apache/dubbo-kubernetes/sail/pkg/networking/util"
	v3 "github.com/apache/dubbo-kubernetes/sail/pkg/xds/v3"
	cluster "github.com/envoyproxy/go-control-plane/envoy/config/cluster/v3"
	core "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	endpoint "github.com/envoyproxy/go-control-plane/envoy/config/endpoint/v3"
	listener "github.com/envoyproxy/go-control-plane/envoy/config/listener/v3"
	route "github.com/envoyproxy/go-control-plane/envoy/config/route/v3"
	discovery "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v3"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/proto"
	anypb "google.golang.org/protobuf/types/known/anypb"
	pstruct "google.golang.org/protobuf/types/known/structpb"
	mcp "istio.io/api/mcp/v1alpha1"
	"istio.io/api/mesh/v1alpha1"
	"k8s.io/klog/v2"
	"math"
	"net"
	"os"
	"strings"
	"sync"
	"time"
)

const (
	defaultClientMaxReceiveMessageSize = math.MaxInt32
	defaultInitialConnWindowSize       = 1024 * 1024 // default gRPC InitialWindowSize
	defaultInitialWindowSize           = 1024 * 1024 // default gRPC ConnWindowSize
)

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
	// InsecureSkipVerify skips client verification the server's certificate chain and host name.
	InsecureSkipVerify bool
	// Workload defaults to 'test'
	Workload string
	// Revision for this control plane instance. We will only read configs that match this revision.
	Revision string
	// Meta includes additional metadata for the node
	Meta *pstruct.Struct
	// IP is currently the primary key used to locate inbound configs. It is sent by client,
	// must match a known endpoint IP. Tests can use a ServiceEntry to register fake IPs.
	IP       string
	Locality *core.Locality
	// BackoffPolicy determines the reconnect policy. Based on MCP client.
	BackoffPolicy backoff.BackOff
}

// ADSC implements a basic client for ADS, for use in stress tests and tools
// or libraries that need to connect to Istio pilot or other ADS servers.
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

	mutex sync.RWMutex

	Mesh *v1alpha1.MeshConfig

	// Retrieved configurations can be stored using the common istio model interface.
	Store model.ConfigStore
	cfg   *ADSConfig
	// sendNodeMeta is set to true if the connection is new - and we need to send node meta.,
	sendNodeMeta bool
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

type ResponseHandler interface {
	HandleResponse(con *ADSC, response *discovery.DiscoveryResponse)
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

func defaultGrpcDialOptions() []grpc.DialOption {
	return []grpc.DialOption{
		// TODO(SpecialYang) maybe need to make it configurable.
		grpc.WithInitialWindowSize(int32(defaultInitialWindowSize)),
		grpc.WithInitialConnWindowSize(int32(defaultInitialConnWindowSize)),
		grpc.WithDefaultCallOptions(grpc.MaxCallRecvMsgSize(defaultClientMaxReceiveMessageSize)),
	}
}

func (a *ADSC) node() *core.Node {
	return buildNode(&a.cfg.Config)
}

func buildNode(config *Config) *core.Node {
	n := &core.Node{
		// Id:       nodeID(config),
		Locality: config.Locality,
	}
	if config.Meta == nil {
		n.Metadata = &pstruct.Struct{
			Fields: map[string]*pstruct.Value{
				"ISTIO_VERSION": {Kind: &pstruct.Value_StringValue{StringValue: "65536.65536.65536"}},
			},
		}
	} else {
		n.Metadata = config.Meta
		if config.Meta.Fields["ISTIO_VERSION"] == nil {
			config.Meta.Fields["ISTIO_VERSION"] = &pstruct.Value{Kind: &pstruct.Value_StringValue{StringValue: "65536.65536.65536"}}
		}
	}
	return n
}

func (a *ADSC) handleRecv() {
	// We connected, so reset the backoff
	if a.cfg.BackoffPolicy != nil {
		a.cfg.BackoffPolicy.Reset()
	}
	for {
		var err error
		msg, err := a.stream.Recv()
		if err != nil {
			klog.Errorf("connection closed with err: %v", err)
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
		resourceGvk, isMCP := convertTypeURLToMCPGVK(msg.TypeUrl)

		// TODO WithLabels
		if a.cfg.ResponseHandler != nil {
			a.cfg.ResponseHandler.HandleResponse(a, msg)
		}

		if msg.TypeUrl == gvk.MeshConfig.String() &&
			len(msg.Resources) > 0 {
			rsc := msg.Resources[0]
			m := &v1alpha1.MeshConfig{}
			err = proto.Unmarshal(rsc.Value, m)
			if err != nil {
				klog.Errorf("Failed to unmarshal mesh config: %v", err)
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
			if isMCP {
				a.handleMCP(resourceGvk, msg.Resources)
			}
		}

		// If we got no resource - still save to the store with empty name/namespace, to notify sync
		// This scheme also allows us to chunk large responses !

		// TODO: add hook to inject nacks

		a.mutex.Lock()
		if isMCP {
			if _, exist := a.sync[resourceGvk.String()]; !exist {
				a.sync[resourceGvk.String()] = time.Now()
			}
		}
		a.Received[msg.TypeUrl] = msg
		a.ack(msg)
		a.mutex.Unlock()

		select {
		case a.XDSUpdates <- msg:
		default:
		}
	}
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
	if a.sendNodeMeta {
		req.Node = a.node()
		a.sendNodeMeta = false
	}
	req.ResponseNonce = time.Now().String()
	// if adscLog.DebugEnabled() {
	// 	strReq, _ := protomarshal.ToJSONWithIndent(req, "  ")
	// 	adscLog.Debugf("Sending Discovery Request to istiod: %s", strReq)
	// }
	return a.stream.Send(req)
}

// HasSynced returns true if MCP configs have synced
func (a *ADSC) HasSynced() bool {
	if a.cfg == nil || len(a.cfg.InitialDiscoveryRequests) == 0 {
		return true
	}

	a.mutex.RLock()
	defer a.mutex.RUnlock()

	for _, req := range a.cfg.InitialDiscoveryRequests {
		_, isMCP := convertTypeURLToMCPGVK(req.TypeUrl)
		if !isMCP {
			continue
		}

		if _, ok := a.sync[req.TypeUrl]; !ok {
			return false
		}
	}

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
	a.sendNodeMeta = true
	a.initialLoad = 0
	a.initialLds = false
	// Send the initial requests
	for _, r := range a.cfg.InitialDiscoveryRequests {
		if r.TypeUrl == v3.ClusterType {
			a.watchTime = time.Now()
		}
		_ = a.Send(r)
	}

	go a.handleRecv()
	return nil
}

// Close the stream.
func (a *ADSC) Close() {
	a.mutex.Lock()
	_ = a.conn.Close()
	a.closed = true
	a.mutex.Unlock()
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

	// nolint: gosec
	// it's insecure only when a user explicitly enable insecure mode.
	return &tls.Config{
		GetClientCertificate: getClientCertificate,
		RootCAs:              serverCAs,
		ServerName:           shost,
		InsecureSkipVerify:   config.InsecureSkipVerify,
	}, nil
}

func ConfigInitialRequests() []*discovery.DiscoveryRequest {
	out := make([]*discovery.DiscoveryRequest, 0, len(collections.Sail.All())+1)
	out = append(out, &discovery.DiscoveryRequest{
		TypeUrl: gvk.MeshConfig.String(),
	})
	for _, sch := range collections.Sail.All() {
		out = append(out, &discovery.DiscoveryRequest{
			TypeUrl: sch.GroupVersionKind().String(),
		})
	}

	return out
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

	if strings.HasPrefix(msg.TypeUrl, v3.DebugType) {
		// If the response is for istio.io/debug or istio.io/debug/*,
		// skip to send ACK.
		return
	}

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
		Node:          a.node(),
		VersionInfo:   msg.VersionInfo,
		ResourceNames: resources,
	})
}

func (a *ADSC) handleMCP(groupVersionKind config.GroupVersionKind, resources []*anypb.Any) {
	// Generic - fill up the store
	if a.Store == nil {
		return
	}

	existingConfigs := a.Store.List(groupVersionKind, "")

	received := make(map[string]*config.Config)
	for _, rsc := range resources {
		m := &mcp.Resource{}
		err := rsc.UnmarshalTo(m)
		if err != nil {
			klog.Errorf("Error unmarshalling received MCP config %v", err)
			continue
		}
		newCfg, err := a.mcpToSail(m)
		if err != nil {
			klog.Errorf("Invalid data: %v (%v)", err, string(rsc.Value))
			continue
		}
		if newCfg == nil {
			continue
		}
		received[newCfg.Namespace+"/"+newCfg.Name] = newCfg

		newCfg.GroupVersionKind = groupVersionKind
		oldCfg := a.Store.Get(newCfg.GroupVersionKind, newCfg.Name, newCfg.Namespace)

		if oldCfg == nil {
			if _, err = a.Store.Create(*newCfg); err != nil {
				klog.Errorf("Error adding a new resource to the store %v", err)
				continue
			}
		} else if oldCfg.ResourceVersion != newCfg.ResourceVersion || newCfg.ResourceVersion == "" {
			// update the store only when resource version differs or unset.
			// newCfg.Annotations[mem.ResourceVersion] = newCfg.ResourceVersion
			newCfg.ResourceVersion = oldCfg.ResourceVersion
			if _, err = a.Store.Update(*newCfg); err != nil {
				klog.Errorf("Error updating an existing resource in the store %v", err)
				continue
			}
		}
	}

	// remove deleted resources from cache
	for _, config := range existingConfigs {
		if _, ok := received[config.Namespace+"/"+config.Name]; !ok {
			if err := a.Store.Delete(config.GroupVersionKind, config.Name, config.Namespace, nil); err != nil {
				klog.Errorf("Error deleting an outdated resource from the store %v", err)
				continue
			}
		}
	}
}

func (a *ADSC) mcpToSail(m *mcp.Resource) (*config.Config, error) {
	if m == nil || m.Metadata == nil {
		return &config.Config{}, nil
	}
	c := &config.Config{
		Meta: config.Meta{
			ResourceVersion: m.Metadata.Version,
			Labels:          m.Metadata.Labels,
			Annotations:     m.Metadata.Annotations,
		},
	}

	if !config.ObjectInRevision(c, a.cfg.Revision) { // In case upstream does not support rev in node meta.
		return nil, nil
	}

	if c.Meta.Annotations == nil {
		c.Meta.Annotations = make(map[string]string)
	}
	nsn := strings.Split(m.Metadata.Name, "/")
	if len(nsn) != 2 {
		return nil, fmt.Errorf("invalid name %s", m.Metadata.Name)
	}
	c.Namespace = nsn[0]
	c.Name = nsn[1]
	var err error
	c.CreationTimestamp = m.Metadata.CreateTime.AsTime()

	pb, err := m.Body.UnmarshalNew()
	if err != nil {
		return nil, err
	}
	c.Spec = pb
	return c, nil
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

	klog.Infof("CDS: %d size=%d", len(cn), cdsSize)

	if len(cn) > 0 {
		a.sendRsc(v3.EndpointType, cn)
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

	klog.Infof("eds: %d size=%d ep=%d", len(eds), edsSize, ep)
	if a.initialLoad == 0 && !a.initialLds {
		// first load - Envoy loads listeners after endpoints
		_ = a.stream.Send(&discovery.DiscoveryRequest{
			Node:    a.node(),
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

// nolint: staticcheck
func (a *ADSC) handleLDS(ll []*listener.Listener) {
	lh := map[string]*listener.Listener{}
	lt := map[string]*listener.Listener{}

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
		case wellknown.TCPProxy:
			lt[l.Name] = l
			config, _ := protomarshal.MessageToStructSlow(filter.GetTypedConfig())
			c := config.Fields["cluster"].GetStringValue()
			klog.V(2).Infof("TCP: %s -> %s", l.Name, c)
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
			klog.Infof(protomarshal.ToJSONWithIndent(l, "  "))
		}
	}

	klog.Infof("LDS: http=%d tcp=%d size=%d", len(lh), len(lt), ldsSize)

	a.mutex.Lock()
	defer a.mutex.Unlock()
	if len(routes) > 0 {
		a.sendRsc(v3.RouteType, routes)
	}
	a.httpListeners = lh
	a.tcpListeners = lt

	select {
	case a.Updates <- v3.ListenerType:
	default:
	}
}

func (a *ADSC) sendRsc(typeurl string, rsc []string) {
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
		Node:          a.node(),
		TypeUrl:       typeurl,
		ResourceNames: rsc,
	})
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
				klog.V(2).Infof("Handle route %v, path %v, cluster %v", h.Name, rt.Match.PathSpecifier, rt.GetRoute().GetCluster())
			}
		}
		rds[r.Name] = r
		size += proto.Size(r)
	}
	if a.initialLoad == 0 {
		a.initialLoad = time.Since(a.watchTime)
		klog.Infof("RDS: %d size=%d vhosts=%d routes=%d time=%d", len(configurations), size, vh, rcount, a.initialLoad)
	} else {
		klog.Infof("RDS: %d size=%d vhosts=%d routes=%d", len(configurations), size, vh, rcount)
	}

	a.mutex.Lock()
	a.routes = rds
	a.mutex.Unlock()

	select {
	case a.Updates <- v3.RouteType:
	default:
	}
}
