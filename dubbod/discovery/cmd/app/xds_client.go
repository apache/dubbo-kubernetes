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

package app

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	pkgbootstrap "github.com/apache/dubbo-kubernetes/pkg/bootstrap"
	"github.com/apache/dubbo-kubernetes/pkg/config/constants"
	meshconfig "github.com/apache/dubbo-kubernetes/pkg/config/mesh"
	"github.com/apache/dubbo-kubernetes/pkg/log"
	v1 "github.com/apache/dubbo-kubernetes/pkg/model"
	clusterv1 "github.com/kdubbo/xds-api/cluster/v1"
	corev1 "github.com/kdubbo/xds-api/core/v1"
	endpointv1 "github.com/kdubbo/xds-api/endpoint/v1"
	hcmv1 "github.com/kdubbo/xds-api/extensions/filters/v1/network/http_connection_manager"
	tlsv1 "github.com/kdubbo/xds-api/extensions/transport_sockets/tls/v1"
	xdsresolver "github.com/kdubbo/xds-api/grpc/resolver"
	listenerv1 "github.com/kdubbo/xds-api/listener/v1"
	routev1 "github.com/kdubbo/xds-api/route/v1"
	discovery "github.com/kdubbo/xds-api/service/discovery/v1"
	"github.com/spf13/cobra"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/structpb"
)

type xdsClientOptions struct {
	host            string
	port            int
	path            string
	headers         []string
	requestHeaders  http.Header
	target          string
	xdsAddress      string
	bootstrapPath   string
	namespace       string
	podName         string
	podIP           string
	serviceAccount  string
	trustDomain     string
	domainSuffix    string
	clusterID       string
	count           int
	expect          string
	waitTimeout     time.Duration
	requestInterval time.Duration
	requestTimeout  time.Duration
	printRoute      bool
	watch           bool
	insecure        bool
}

type xdsEndpoint struct {
	Address string `json:"address"`
	Port    uint32 `json:"port"`
}

type xdsDestination struct {
	Cluster   string        `json:"cluster"`
	Host      string        `json:"host"`
	Subset    string        `json:"subset,omitempty"`
	Weight    uint32        `json:"weight"`
	TLSMode   string        `json:"tlsMode,omitempty"`
	SNI       string        `json:"sni,omitempty"`
	Endpoints []xdsEndpoint `json:"endpoints,omitempty"`

	tlsContext *tlsv1.UpstreamTlsContext
}

type xdsRouteSnapshot struct {
	Host         string           `json:"host"`
	Port         int              `json:"port"`
	Timeout      string           `json:"timeout,omitempty"`
	Destinations []xdsDestination `json:"destinations"`
}

type sampleADSClient struct {
	conn           *grpc.ClientConn
	stream         discovery.AggregatedDiscoveryService_StreamAggregatedResourcesClient
	node           *corev1.Node
	target         string
	host           string
	port           int
	path           string
	requestHeaders http.Header

	mu            sync.RWMutex
	subs          map[string][]string
	route         map[string]uint32
	routeTimeout  time.Duration
	endpoints     map[string][]xdsEndpoint
	clusterTLS    map[string]*tlsv1.UpstreamTlsContext
	updates       chan struct{}
	errs          chan error
	bootstrapPath string
	bootstrap     *xdsresolver.BootstrapConfig
}

func newXClientCommand() *cobra.Command {
	namespace := firstNonEmpty(os.Getenv("POD_NAMESPACE"), "default")
	trustDomain := firstNonEmpty(os.Getenv("TRUST_DOMAIN"), constants.DefaultClusterLocalDomain)
	domainSuffix := firstNonEmpty(os.Getenv("DOMAIN_SUFFIX"), trustDomain, constants.DefaultClusterLocalDomain)
	host, port, hostErr := autoDiscoverServiceTarget(os.Environ(), namespace, domainSuffix)
	opts := &xdsClientOptions{
		host:           firstNonEmpty(os.Getenv("DUBBO_SERVICE_HOST"), host),
		port:           firstIntFromEnv(int(port), "DUBBO_SERVICE_PORT"),
		path:           firstNonEmpty(os.Getenv("REQUEST_PATH"), "/"),
		xdsAddress:     firstNonEmpty(os.Getenv("XDS_ADDRESS"), "dubbod.dubbo-system.svc:26010"),
		bootstrapPath:  os.Getenv("GRPC_XDS_BOOTSTRAP"),
		namespace:      namespace,
		podName:        firstNonEmpty(os.Getenv("POD_NAME"), os.Getenv("HOSTNAME"), "xclient"),
		podIP:          firstNonEmpty(os.Getenv("INSTANCE_IP"), os.Getenv("POD_IP"), "127.0.0.1"),
		serviceAccount: firstNonEmpty(os.Getenv("SERVICE_ACCOUNT"), "default"),
		trustDomain:    trustDomain,
		domainSuffix:   domainSuffix,
		clusterID:      firstNonEmpty(os.Getenv("DUBBO_META_CLUSTER_ID"), string(constants.DefaultClusterName)),
		count:          intFromEnv("REQUEST_COUNT", 20),
		expect:         os.Getenv("EXPECTED_WEIGHTS"),
		waitTimeout:    durationSecondsFromEnv("RUNTIME_WAIT_TIMEOUT", 30*time.Second),
	}

	c := &cobra.Command{
		Use:   "xclient [count]",
		Short: "run a no-proxy ADS stream client for service-to-service sample traffic",
		Args:  cobra.MaximumNArgs(1),
		PreRunE: func(cmd *cobra.Command, args []string) error {
			log.SetDefaultScope(xclientLogScope)
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 1 {
				count, err := strconv.Atoi(args[0])
				if err != nil {
					return fmt.Errorf("invalid count %q: %w", args[0], err)
				}
				opts.count = count
			}
			if opts.host == "" && hostErr != nil {
				return hostErr
			}
			return opts.run(cmd.Context())
		},
	}
	c.Flags().StringVar(&opts.host, "host", opts.host, "service DNS name to call")
	c.Flags().IntVar(&opts.port, "port", opts.port, "service port to call")
	c.Flags().StringVar(&opts.path, "path", opts.path, "HTTP request path")
	c.Flags().StringArrayVar(&opts.headers, "header", nil, "HTTP request header in key=value form; may be repeated")
	c.Flags().StringVar(&opts.target, "target", opts.target, "xDS LDS target; defaults to host:port")
	c.Flags().StringVar(&opts.xdsAddress, "xds-address", opts.xdsAddress, "ADS server address used when no bootstrap file is present")
	c.Flags().StringVar(&opts.bootstrapPath, "bootstrap", opts.bootstrapPath, "gRPC xDS bootstrap file")
	c.Flags().StringVar(&opts.namespace, "namespace", opts.namespace, "client config namespace")
	c.Flags().StringVar(&opts.podName, "pod-name", opts.podName, "client pod name for xDS node identity")
	c.Flags().StringVar(&opts.podIP, "pod-ip", opts.podIP, "client pod IP for xDS node identity")
	c.Flags().StringVar(&opts.serviceAccount, "service-account", opts.serviceAccount, "client service account for xDS node identity")
	c.Flags().StringVar(&opts.trustDomain, "trust-domain", opts.trustDomain, "trust domain for xDS node metadata")
	c.Flags().StringVar(&opts.domainSuffix, "domain-suffix", opts.domainSuffix, "Kubernetes service DNS suffix")
	c.Flags().StringVar(&opts.clusterID, "cluster-id", opts.clusterID, "xDS cluster ID")
	c.Flags().StringVar(&opts.expect, "expect", opts.expect, "expected subset weights, for example v1=20,v2=80")
	c.Flags().DurationVar(&opts.waitTimeout, "wait-timeout", opts.waitTimeout, "time to wait for the streamed route and endpoints")
	c.Flags().DurationVar(&opts.requestInterval, "request-interval", 0, "pause between requests so the same xDS stream can pick up live route updates")
	c.Flags().DurationVar(&opts.requestTimeout, "request-timeout", 5*time.Second, "timeout for each upstream HTTP request")
	c.Flags().BoolVar(&opts.printRoute, "print-route", false, "print the current streamed route and exit")
	c.Flags().BoolVar(&opts.watch, "watch", false, "keep the ADS stream open and print route changes")
	c.Flags().BoolVar(&opts.insecure, "insecure", false, "force insecure ADS transport")
	return c
}

func (o *xdsClientOptions) run(ctx context.Context) error {
	if o.host == "" {
		return fmt.Errorf("target host is required; set --host or DUBBO_SERVICE_HOST, or expose exactly one Service in the namespace before starting the pod")
	}
	if o.target == "" {
		o.target = net.JoinHostPort(o.host, strconv.Itoa(o.port))
	}
	o.path = normalizeRequestPath(o.path)
	requestHeaders, err := parseRequestHeaders(o.headers)
	if err != nil {
		return err
	}
	o.requestHeaders = requestHeaders
	expected, err := parseExpectedWeights(o.expect)
	if err != nil {
		return err
	}

	client, err := newSampleADSClient(ctx, o)
	if err != nil {
		return err
	}
	defer client.close()
	if err := client.start(); err != nil {
		return err
	}

	if o.watch {
		return client.watch(ctx)
	}

	snapshot, err := client.waitForRoute(ctx, expected, o.waitTimeout)
	if err != nil {
		return err
	}
	if o.printRoute {
		out, err := json.MarshalIndent(snapshot, "", "  ")
		if err != nil {
			return err
		}
		fmt.Println(string(out))
		return nil
	}
	return runSampleRequests(ctx, client, snapshot, o.count, o.requestInterval, o.requestTimeout)
}

func newSampleADSClient(ctx context.Context, opts *xdsClientOptions) (*sampleADSClient, error) {
	node, addr, dialOpts, err := adsDialConfig(opts)
	if err != nil {
		return nil, err
	}
	conn, err := grpc.DialContext(ctx, addr, dialOpts...)
	if err != nil {
		return nil, fmt.Errorf("dial ADS server %s: %w", addr, err)
	}
	xds := discovery.NewAggregatedDiscoveryServiceClient(conn)
	stream, err := xds.StreamAggregatedResources(ctx)
	if err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("open ADS stream: %w", err)
	}
	return &sampleADSClient{
		conn:           conn,
		stream:         stream,
		node:           node,
		target:         opts.target,
		host:           opts.host,
		port:           opts.port,
		path:           opts.path,
		requestHeaders: opts.requestHeaders,
		subs:           map[string][]string{},
		route:          map[string]uint32{},
		endpoints:      map[string][]xdsEndpoint{},
		clusterTLS:     map[string]*tlsv1.UpstreamTlsContext{},
		updates:        make(chan struct{}, 1),
		errs:           make(chan error, 1),
		bootstrapPath:  opts.bootstrapPath,
	}, nil
}

func adsDialConfig(opts *xdsClientOptions) (*corev1.Node, string, []grpc.DialOption, error) {
	if opts.bootstrapPath != "" {
		if bootstrap, err := xdsresolver.ParseBootstrap(opts.bootstrapPath); err == nil {
			creds, err := xdsresolver.TransportCredentialsFromBootstrap(bootstrap)
			if err != nil {
				return nil, "", nil, err
			}
			if opts.insecure {
				creds = insecure.NewCredentials()
			}
			return bootstrap.Node, xdsresolver.DialAddress(bootstrap.ServerURI), []grpc.DialOption{grpc.WithTransportCredentials(creds)}, nil
		} else if !os.IsNotExist(err) {
			return nil, "", nil, err
		}
	}

	node, err := buildADSNode(opts)
	if err != nil {
		return nil, "", nil, err
	}
	creds := grpc.WithTransportCredentials(insecure.NewCredentials())
	if !opts.insecure && strings.HasSuffix(opts.xdsAddress, ":26012") {
		creds = grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{MinVersion: tls.VersionTLS12}))
	}
	return node, opts.xdsAddress, []grpc.DialOption{creds}, nil
}

func buildADSNode(opts *xdsClientOptions) (*corev1.Node, error) {
	proxyConfig := meshconfig.DefaultProxyConfig()
	proxyConfig.DiscoveryAddress = opts.xdsAddress
	nodeID := strings.Join([]string{
		string(v1.Proxyless),
		opts.podIP,
		opts.podName + "." + opts.namespace,
		fmt.Sprintf("%s.svc.%s", opts.namespace, opts.domainSuffix),
	}, "~")
	node, err := pkgbootstrap.GetNodeMetaData(pkgbootstrap.MetadataOptions{
		ID:          nodeID,
		InstanceIPs: []string{opts.podIP},
		ProxyConfig: proxyConfig,
		Envs: []string{
			"DUBBO_META_GENERATOR=grpc",
			"DUBBO_META_CLUSTER_ID=" + opts.clusterID,
			"DUBBO_META_NAMESPACE=" + opts.namespace,
			"DUBBO_META_MESH_ID=" + opts.trustDomain,
			"TRUST_DOMAIN=" + opts.trustDomain,
			"POD_NAME=" + opts.podName,
			"POD_NAMESPACE=" + opts.namespace,
			"INSTANCE_IP=" + opts.podIP,
			"SERVICE_ACCOUNT=" + opts.serviceAccount,
		},
	})
	if err != nil {
		return nil, err
	}
	meta, err := structFromMetadata(node.Metadata)
	if err != nil {
		return nil, err
	}
	return &corev1.Node{Id: node.ID, Locality: node.Locality, Metadata: meta}, nil
}

func structFromMetadata(metadata any) (*structpb.Struct, error) {
	data, err := json.Marshal(metadata)
	if err != nil {
		return nil, err
	}
	raw := map[string]any{}
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, err
	}
	if proxyConfig, ok := raw["PROXY_CONFIG"].(map[string]any); ok {
		if clusterName, ok := proxyConfig["ClusterName"].(map[string]any); ok {
			if serviceCluster, ok := clusterName["ServiceCluster"].(string); ok {
				proxyConfig["ClusterName"] = serviceCluster
			}
		}
	}
	return structpb.NewStruct(raw)
}

func (c *sampleADSClient) start() error {
	if err := c.subscribe(v1.ListenerType, []string{c.target}); err != nil {
		return err
	}
	go c.receive()
	return nil
}

func (c *sampleADSClient) receive() {
	for {
		resp, err := c.stream.Recv()
		if err != nil {
			select {
			case c.errs <- err:
			default:
			}
			return
		}
		if err := c.ack(resp); err != nil {
			select {
			case c.errs <- err:
			default:
			}
			return
		}
		if err := c.handleResponse(resp); err != nil {
			select {
			case c.errs <- err:
			default:
			}
			return
		}
	}
}

func (c *sampleADSClient) subscribe(typeURL string, resourceNames []string) error {
	resourceNames = sortedUnique(resourceNames)
	c.mu.Lock()
	c.subs[typeURL] = resourceNames
	c.mu.Unlock()
	return c.stream.Send(&discovery.DiscoveryRequest{
		Node:          c.node,
		TypeUrl:       typeURL,
		ResourceNames: resourceNames,
	})
}

func (c *sampleADSClient) ack(resp *discovery.DiscoveryResponse) error {
	c.mu.RLock()
	resourceNames := append([]string(nil), c.subs[resp.TypeUrl]...)
	c.mu.RUnlock()
	return c.stream.Send(&discovery.DiscoveryRequest{
		Node:          c.node,
		TypeUrl:       resp.TypeUrl,
		VersionInfo:   resp.VersionInfo,
		ResponseNonce: resp.Nonce,
		ResourceNames: resourceNames,
	})
}

func (c *sampleADSClient) handleResponse(resp *discovery.DiscoveryResponse) error {
	switch resp.TypeUrl {
	case v1.ListenerType:
		routeNames, err := routeNamesFromListeners(resp.Resources)
		if err != nil {
			return err
		}
		if len(routeNames) > 0 {
			return c.subscribe(v1.RouteType, routeNames)
		}
	case v1.RouteType:
		weights, clusters, timeout, err := routeWeightsFromRoutes(resp.Resources, c.path, c.requestHeaders)
		if err != nil {
			return err
		}
		c.mu.Lock()
		c.route = weights
		c.routeTimeout = timeout
		c.mu.Unlock()
		c.notify()
		if len(clusters) > 0 {
			return c.subscribe(v1.ClusterType, clusters)
		}
	case v1.ClusterType:
		edsNames, clusterTLS, err := edsNamesAndTLSFromClusters(resp.Resources)
		if err != nil {
			return err
		}
		c.mu.Lock()
		for clusterName, tlsContext := range clusterTLS {
			c.clusterTLS[clusterName] = tlsContext
		}
		c.mu.Unlock()
		c.notify()
		if len(edsNames) > 0 {
			return c.subscribe(v1.EndpointType, edsNames)
		}
	case v1.EndpointType:
		endpoints, err := endpointsFromAssignments(resp.Resources)
		if err != nil {
			return err
		}
		c.mu.Lock()
		for clusterName, eps := range endpoints {
			c.endpoints[clusterName] = eps
		}
		c.mu.Unlock()
		c.notify()
	}
	return nil
}

func (c *sampleADSClient) waitForRoute(ctx context.Context, expected map[string]uint32, timeout time.Duration) (xdsRouteSnapshot, error) {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	for {
		if snapshot, ok := c.readySnapshot(expected); ok {
			return snapshot, nil
		}
		select {
		case <-c.updates:
		case err := <-c.errs:
			return xdsRouteSnapshot{}, err
		case <-ctx.Done():
			snapshot, _ := c.readySnapshot(nil)
			if len(snapshot.Destinations) == 0 {
				return xdsRouteSnapshot{}, fmt.Errorf("route not found for %s:%d from ADS stream", c.host, c.port)
			}
			return xdsRouteSnapshot{}, fmt.Errorf("route weights %v did not match expected %v", subsetWeights(snapshot), expected)
		}
	}
}

func (c *sampleADSClient) watch(ctx context.Context) error {
	for {
		select {
		case <-c.updates:
			snapshot, _ := c.readySnapshot(nil)
			if len(snapshot.Destinations) > 0 {
				fmt.Println(routeSummary(snapshot))
			}
		case err := <-c.errs:
			return err
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

func (c *sampleADSClient) readySnapshot(expected map[string]uint32) (xdsRouteSnapshot, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	snapshot := xdsRouteSnapshot{Host: c.host, Port: c.port}
	if c.routeTimeout > 0 {
		snapshot.Timeout = c.routeTimeout.String()
	}
	for clusterName, weight := range c.route {
		if weight == 0 {
			continue
		}
		eps := append([]xdsEndpoint(nil), c.endpoints[clusterName]...)
		dest := destinationFromCluster(clusterName, weight, eps, c.clusterTLS[clusterName])
		snapshot.Destinations = append(snapshot.Destinations, dest)
	}
	sort.Slice(snapshot.Destinations, func(i, j int) bool {
		return snapshot.Destinations[i].Cluster < snapshot.Destinations[j].Cluster
	})
	if len(snapshot.Destinations) == 0 {
		return snapshot, false
	}
	for _, dest := range snapshot.Destinations {
		if len(dest.Endpoints) == 0 {
			return snapshot, false
		}
	}
	if len(expected) > 0 && !weightsMatch(subsetWeights(snapshot), expected) {
		return snapshot, false
	}
	return snapshot, true
}

func (c *sampleADSClient) notify() {
	select {
	case c.updates <- struct{}{}:
	default:
	}
}

func (c *sampleADSClient) close() {
	if c.conn != nil {
		_ = c.conn.Close()
	}
}

func routeNamesFromListeners(resources []*anypb.Any) ([]string, error) {
	out := make([]string, 0, len(resources))
	for _, resource := range resources {
		l := &listenerv1.Listener{}
		if err := proto.Unmarshal(resource.Value, l); err != nil {
			return nil, err
		}
		if apiListener := l.GetApiListener(); apiListener != nil && apiListener.ApiListener != nil {
			hcm := &hcmv1.HttpConnectionManager{}
			if err := apiListener.ApiListener.UnmarshalTo(hcm); err != nil {
				return nil, err
			}
			if rds := hcm.GetRds(); rds != nil && rds.GetRouteConfigName() != "" {
				out = append(out, rds.GetRouteConfigName())
			}
		}
	}
	return sortedUnique(out), nil
}

func routeWeightsFromRoutes(resources []*anypb.Any, requestPath string, requestHeaders http.Header) (map[string]uint32, []string, time.Duration, error) {
	weights := map[string]uint32{}
	for _, resource := range resources {
		rc := &routev1.RouteConfiguration{}
		if err := proto.Unmarshal(resource.Value, rc); err != nil {
			return nil, nil, 0, err
		}
		for _, vh := range rc.GetVirtualHosts() {
			for _, rt := range vh.GetRoutes() {
				if !routeMatchesRequest(rt.GetMatch(), requestPath, requestHeaders) {
					continue
				}
				action := rt.GetRoute()
				addRouteActionWeights(weights, action)
				return weights, sortedWeightClusterNames(weights), routeActionTimeout(action), nil
			}
		}
	}
	return weights, sortedWeightClusterNames(weights), 0, nil
}

func routeActionTimeout(action *routev1.RouteAction) time.Duration {
	if action == nil || action.GetTimeout() == nil {
		return 0
	}
	return action.GetTimeout().AsDuration()
}

func addRouteActionWeights(weights map[string]uint32, action *routev1.RouteAction) {
	if action == nil {
		return
	}
	if clusterName := action.GetCluster(); clusterName != "" {
		weights[clusterName] += 1
	}
	if weighted := action.GetWeightedClusters(); weighted != nil {
		for _, clusterWeight := range weighted.GetClusters() {
			if clusterWeight.GetName() == "" {
				continue
			}
			weight := uint32(1)
			if clusterWeight.GetWeight() != nil {
				weight = clusterWeight.GetWeight().GetValue()
			}
			weights[clusterWeight.GetName()] += weight
		}
	}
}

func sortedWeightClusterNames(weights map[string]uint32) []string {
	clusters := make([]string, 0, len(weights))
	for clusterName := range weights {
		clusters = append(clusters, clusterName)
	}
	return sortedUnique(clusters)
}

func routeMatchesRequest(match *routev1.RouteMatch, requestPath string, requestHeaders http.Header) bool {
	if match == nil {
		return true
	}
	switch spec := match.PathSpecifier.(type) {
	case *routev1.RouteMatch_Path:
		if requestPath != spec.Path {
			return false
		}
	case *routev1.RouteMatch_Prefix:
		if !strings.HasPrefix(requestPath, spec.Prefix) {
			return false
		}
	case *routev1.RouteMatch_SafeRegex:
		if spec.SafeRegex == nil {
			return false
		}
		ok, err := regexp.MatchString(spec.SafeRegex.Regex, requestPath)
		if err != nil || !ok {
			return false
		}
	}
	for _, header := range match.GetHeaders() {
		if !headerMatchesRequest(header, requestHeaders) {
			return false
		}
	}
	return true
}

func headerMatchesRequest(match *routev1.HeaderMatcher, requestHeaders http.Header) bool {
	if match == nil {
		return true
	}
	value := requestHeaders.Get(match.GetName())
	switch spec := match.HeaderMatchSpecifier.(type) {
	case *routev1.HeaderMatcher_ExactMatch:
		return value == spec.ExactMatch
	case *routev1.HeaderMatcher_PrefixMatch:
		return strings.HasPrefix(value, spec.PrefixMatch)
	case *routev1.HeaderMatcher_SafeRegexMatch:
		if spec.SafeRegexMatch == nil {
			return false
		}
		ok, err := regexp.MatchString(spec.SafeRegexMatch.Regex, value)
		return err == nil && ok
	default:
		return value != ""
	}
}

func edsNamesAndTLSFromClusters(resources []*anypb.Any) ([]string, map[string]*tlsv1.UpstreamTlsContext, error) {
	out := make([]string, 0, len(resources))
	clusterTLS := map[string]*tlsv1.UpstreamTlsContext{}
	for _, resource := range resources {
		c := &clusterv1.Cluster{}
		if err := proto.Unmarshal(resource.Value, c); err != nil {
			return nil, nil, err
		}
		clusterTLS[c.Name] = upstreamTLSContextFromCluster(c)
		serviceName := c.GetName()
		if eds := c.GetEdsClusterConfig(); eds != nil && eds.GetServiceName() != "" {
			serviceName = eds.GetServiceName()
		}
		if serviceName != "" {
			out = append(out, serviceName)
		}
	}
	return sortedUnique(out), clusterTLS, nil
}

func upstreamTLSContextFromCluster(c *clusterv1.Cluster) *tlsv1.UpstreamTlsContext {
	if c == nil || c.TransportSocket == nil {
		return nil
	}
	typedConfig := c.TransportSocket.GetTypedConfig()
	if typedConfig == nil {
		return nil
	}
	upstreamTLS := &tlsv1.UpstreamTlsContext{}
	if err := anypb.UnmarshalTo(typedConfig, upstreamTLS, proto.UnmarshalOptions{}); err != nil {
		return nil
	}
	return upstreamTLS
}

func endpointsFromAssignments(resources []*anypb.Any) (map[string][]xdsEndpoint, error) {
	out := map[string][]xdsEndpoint{}
	for _, resource := range resources {
		cla := &endpointv1.ClusterLoadAssignment{}
		if err := proto.Unmarshal(resource.Value, cla); err != nil {
			return nil, err
		}
		for _, locality := range cla.GetEndpoints() {
			for _, lbEndpoint := range locality.GetLbEndpoints() {
				endpoint := lbEndpoint.GetEndpoint()
				if endpoint == nil || endpoint.GetAddress() == nil || endpoint.GetAddress().GetSocketAddress() == nil {
					continue
				}
				socket := endpoint.GetAddress().GetSocketAddress()
				if socket.GetAddress() == "" || socket.GetPortValue() == 0 {
					continue
				}
				out[cla.GetClusterName()] = append(out[cla.GetClusterName()], xdsEndpoint{
					Address: socket.GetAddress(),
					Port:    socket.GetPortValue(),
				})
			}
		}
	}
	for clusterName := range out {
		sort.Slice(out[clusterName], func(i, j int) bool {
			if out[clusterName][i].Address != out[clusterName][j].Address {
				return out[clusterName][i].Address < out[clusterName][j].Address
			}
			return out[clusterName][i].Port < out[clusterName][j].Port
		})
	}
	return out, nil
}

func destinationFromCluster(clusterName string, weight uint32, endpoints []xdsEndpoint, tlsContext *tlsv1.UpstreamTlsContext) xdsDestination {
	parts := strings.Split(clusterName, "|")
	dest := xdsDestination{Cluster: clusterName, Weight: weight, Endpoints: endpoints, tlsContext: tlsContext}
	if len(parts) == 4 {
		dest.Host = parts[3]
		dest.Subset = parts[2]
	}
	if tlsContext != nil {
		dest.TLSMode = "DUBBO_MUTUAL"
		dest.SNI = tlsContext.Sni
	}
	return dest
}

func runSampleRequests(ctx context.Context, adsClient *sampleADSClient, snapshot xdsRouteSnapshot, count int, requestInterval, requestTimeout time.Duration) error {
	_, err := runSampleRequestsWithOutput(ctx, adsClient, snapshot, count, requestInterval, requestTimeout, os.Stdout)
	return err
}

func runSampleRequestsWithOutput(ctx context.Context, adsClient *sampleADSClient, snapshot xdsRouteSnapshot, count int, requestInterval, requestTimeout time.Duration, writer io.Writer) ([]string, error) {
	picker, err := newSmoothWeightedPicker(snapshot)
	if err != nil {
		return nil, err
	}
	currentSignature := snapshotSignature(snapshot)
	clients := newSampleRequestClients(adsClient, requestTimeout)
	output := make([]string, 0, count)
	for i := 0; i < count; i++ {
		if updated, ok := adsClient.readySnapshot(nil); ok {
			updatedSignature := snapshotSignature(updated)
			if updatedSignature != currentSignature {
				snapshot = updated
				currentSignature = updatedSignature
				picker, err = newSmoothWeightedPicker(snapshot)
				if err != nil {
					return nil, err
				}
			}
		}
		destination, endpoint, err := picker.Next()
		if err != nil {
			return nil, err
		}
		httpClient, scheme, err := clients.clientForDestination(destination)
		if err != nil {
			return nil, err
		}
		requestCtx := ctx
		cancel := func() {}
		if timeout, ok := routeTimeoutDuration(snapshot.Timeout); ok {
			requestCtx, cancel = context.WithTimeout(ctx, timeout)
		}
		req, err := http.NewRequestWithContext(requestCtx, http.MethodGet,
			fmt.Sprintf("%s://%s%s", scheme, net.JoinHostPort(endpoint.Address, strconv.Itoa(int(endpoint.Port))), adsClient.path), nil)
		if err != nil {
			cancel()
			return nil, err
		}
		req.Host = snapshot.Host
		for name, values := range adsClient.requestHeaders {
			for _, value := range values {
				req.Header.Add(name, value)
			}
		}
		resp, err := httpClient.Do(req)
		if err != nil {
			cancel()
			return nil, err
		}
		body, readErr := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		cancel()
		if readErr != nil {
			return nil, readErr
		}
		line := strings.TrimSpace(string(body))
		output = append(output, line+"\n")
		if writer != nil {
			fmt.Fprintln(writer, line)
		}
		if requestInterval > 0 && i+1 < count {
			timer := time.NewTimer(requestInterval)
			select {
			case <-ctx.Done():
				if !timer.Stop() {
					<-timer.C
				}
				return output, ctx.Err()
			case <-timer.C:
			}
		}
	}
	return output, nil
}

func routeTimeoutDuration(value string) (time.Duration, bool) {
	if value == "" {
		return 0, false
	}
	timeout, err := time.ParseDuration(value)
	if err != nil || timeout <= 0 {
		return 0, false
	}
	return timeout, true
}

type sampleRequestClients struct {
	adsClient      *sampleADSClient
	requestTimeout time.Duration
	plaintext      *http.Client
	tlsClients     map[string]*http.Client
}

func newSampleRequestClients(adsClient *sampleADSClient, requestTimeout time.Duration) *sampleRequestClients {
	return &sampleRequestClients{
		adsClient:      adsClient,
		requestTimeout: requestTimeout,
		plaintext:      &http.Client{Timeout: requestTimeout},
		tlsClients:     map[string]*http.Client{},
	}
}

func (c *sampleRequestClients) clientForDestination(destination xdsDestination) (*http.Client, string, error) {
	if destination.tlsContext == nil {
		return c.plaintext, "http", nil
	}
	bootstrap, err := c.adsClient.bootstrapConfig()
	if err != nil {
		return nil, "", err
	}
	tlsConfig, err := xdsresolver.DataPlaneTLSConfigFromBootstrap(bootstrap, destination.tlsContext, destination.Host)
	if err != nil {
		return nil, "", err
	}
	key := destination.TLSMode + "|" + destination.SNI
	if key == "|" {
		key = destination.Cluster
	}
	if cached := c.tlsClients[key]; cached != nil {
		return cached, "https", nil
	}
	client := &http.Client{
		Timeout: c.requestTimeout,
		Transport: &http.Transport{
			TLSClientConfig: tlsConfig,
		},
	}
	c.tlsClients[key] = client
	return client, "https", nil
}

func (c *sampleADSClient) bootstrapConfig() (*xdsresolver.BootstrapConfig, error) {
	if c.bootstrap != nil {
		return c.bootstrap, nil
	}
	path := c.bootstrapPath
	if path == "" {
		path = os.Getenv("GRPC_XDS_BOOTSTRAP")
	}
	if path == "" {
		return nil, fmt.Errorf("data-plane mTLS requires GRPC_XDS_BOOTSTRAP or --bootstrap")
	}
	bootstrap, err := xdsresolver.ParseBootstrap(path)
	if err != nil {
		return nil, err
	}
	c.bootstrap = bootstrap
	return c.bootstrap, nil
}

func activeDestinations(snapshot xdsRouteSnapshot) []xdsDestination {
	out := make([]xdsDestination, 0, len(snapshot.Destinations))
	for _, dest := range snapshot.Destinations {
		if dest.Weight > 0 && len(dest.Endpoints) > 0 {
			out = append(out, dest)
		}
	}
	return out
}

type smoothWeightedPicker struct {
	host          string
	port          int
	totalWeight   int64
	items         []smoothWeightedDestination
	endpointIndex map[string]int
}

type smoothWeightedDestination struct {
	destination xdsDestination
	weight      int64
	current     int64
}

func newSmoothWeightedPicker(snapshot xdsRouteSnapshot) (*smoothWeightedPicker, error) {
	destinations := activeDestinations(snapshot)
	if len(destinations) == 0 {
		return nil, fmt.Errorf("route %s:%d has no endpoints", snapshot.Host, snapshot.Port)
	}
	picker := &smoothWeightedPicker{
		host:          snapshot.Host,
		port:          snapshot.Port,
		items:         make([]smoothWeightedDestination, 0, len(destinations)),
		endpointIndex: map[string]int{},
	}
	for _, destination := range destinations {
		if destination.Weight == 0 || len(destination.Endpoints) == 0 {
			continue
		}
		picker.totalWeight += int64(destination.Weight)
		picker.items = append(picker.items, smoothWeightedDestination{
			destination: destination,
			weight:      int64(destination.Weight),
		})
	}
	if len(picker.items) == 0 || picker.totalWeight == 0 {
		return nil, fmt.Errorf("route %s:%d has no endpoints", snapshot.Host, snapshot.Port)
	}
	return picker, nil
}

func (p *smoothWeightedPicker) Next() (xdsDestination, xdsEndpoint, error) {
	if len(p.items) == 0 || p.totalWeight == 0 {
		return xdsDestination{}, xdsEndpoint{}, fmt.Errorf("route %s:%d has no endpoints", p.host, p.port)
	}
	best := 0
	for i := range p.items {
		p.items[i].current += p.items[i].weight
		if p.items[i].current > p.items[best].current {
			best = i
		}
	}
	p.items[best].current -= p.totalWeight
	destination := p.items[best].destination
	endpoint := destination.Endpoints[p.endpointIndex[destination.Cluster]%len(destination.Endpoints)]
	p.endpointIndex[destination.Cluster]++
	return destination, endpoint, nil
}

func snapshotSignature(snapshot xdsRouteSnapshot) string {
	data, err := json.Marshal(snapshot)
	if err != nil {
		return routeSummary(snapshot)
	}
	return string(data)
}

func parseExpectedWeights(value string) (map[string]uint32, error) {
	if strings.TrimSpace(value) == "" {
		return nil, nil
	}
	out := map[string]uint32{}
	for _, item := range strings.Split(value, ",") {
		parts := strings.SplitN(item, "=", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid --expect item %q", item)
		}
		weight, err := strconv.ParseUint(strings.TrimSpace(parts[1]), 10, 32)
		if err != nil {
			return nil, fmt.Errorf("invalid --expect weight %q: %w", parts[1], err)
		}
		out[strings.TrimSpace(parts[0])] = uint32(weight)
	}
	return out, nil
}

func normalizeRequestPath(path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return "/"
	}
	if !strings.HasPrefix(path, "/") {
		return "/" + path
	}
	return path
}

func parseRequestHeaders(values []string) (http.Header, error) {
	headers := http.Header{}
	for _, value := range values {
		key, headerValue, ok := strings.Cut(value, "=")
		key = strings.TrimSpace(key)
		if !ok || key == "" {
			return nil, fmt.Errorf("invalid --header item %q, want key=value", value)
		}
		headers.Add(key, strings.TrimSpace(headerValue))
	}
	return headers, nil
}

func subsetWeights(snapshot xdsRouteSnapshot) map[string]uint32 {
	out := map[string]uint32{}
	for _, dest := range snapshot.Destinations {
		out[destinationWeightKey(dest)] = dest.Weight
	}
	return out
}

func destinationWeightKey(dest xdsDestination) string {
	if dest.Subset != "" {
		return dest.Subset
	}
	if dest.Host != "" {
		if service, _, found := strings.Cut(dest.Host, "."); found && service != "" {
			return service
		}
		return dest.Host
	}
	return dest.Cluster
}

func weightsMatch(got, want map[string]uint32) bool {
	if len(got) != len(want) {
		return false
	}
	for key, value := range want {
		if got[key] != value {
			return false
		}
	}
	return true
}

func routeSummary(snapshot xdsRouteSnapshot) string {
	parts := make([]string, 0, len(snapshot.Destinations))
	for _, dest := range snapshot.Destinations {
		parts = append(parts, fmt.Sprintf("%s=%d endpoints=%d", destinationWeightKey(dest), dest.Weight, len(dest.Endpoints)))
	}
	timeout := ""
	if snapshot.Timeout != "" {
		timeout = " timeout=" + snapshot.Timeout
	}
	return fmt.Sprintf("%s:%d%s %s", snapshot.Host, snapshot.Port, timeout, strings.Join(parts, ","))
}

func sortedUnique(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	seen := map[string]struct{}{}
	out := make([]string, 0, len(values))
	for _, value := range values {
		if value == "" {
			continue
		}
		if _, found := seen[value]; found {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	sort.Strings(out)
	return out
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}

func firstIntFromEnv(fallback int, names ...string) int {
	for _, name := range names {
		if value := os.Getenv(name); value != "" {
			if parsed, err := strconv.Atoi(value); err == nil {
				return parsed
			}
		}
	}
	return fallback
}

func intFromEnv(name string, fallback int) int {
	if value := os.Getenv(name); value != "" {
		if parsed, err := strconv.Atoi(value); err == nil {
			return parsed
		}
	}
	return fallback
}

func durationSecondsFromEnv(name string, fallback time.Duration) time.Duration {
	if value := os.Getenv(name); value != "" {
		if parsed, err := strconv.Atoi(value); err == nil {
			return time.Duration(parsed) * time.Second
		}
		if parsed, err := time.ParseDuration(value); err == nil {
			return parsed
		}
	}
	return fallback
}

func autoDiscoverServiceTarget(env []string, namespace, domainSuffix string) (string, int, error) {
	type serviceTarget struct {
		name string
		port int
	}

	ports := map[string]int{}
	hosts := map[string]struct{}{}
	for _, raw := range env {
		key, value, found := strings.Cut(raw, "=")
		if !found {
			continue
		}
		switch {
		case strings.HasSuffix(key, "_SERVICE_HOST"):
			base := strings.TrimSuffix(key, "_SERVICE_HOST")
			hosts[base] = struct{}{}
		case strings.HasSuffix(key, "_SERVICE_PORT"):
			base := strings.TrimSuffix(key, "_SERVICE_PORT")
			port, err := strconv.Atoi(value)
			if err != nil {
				continue
			}
			ports[base] = port
		}
	}

	candidates := make([]serviceTarget, 0)
	for base := range hosts {
		if base == "KUBERNETES" {
			continue
		}
		port, found := ports[base]
		if !found {
			continue
		}
		candidates = append(candidates, serviceTarget{
			name: strings.ToLower(strings.ReplaceAll(base, "_", "-")),
			port: port,
		})
	}
	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].name < candidates[j].name
	})
	if len(candidates) == 0 {
		return "", 80, nil
	}
	if len(candidates) > 1 {
		names := make([]string, 0, len(candidates))
		for _, candidate := range candidates {
			names = append(names, candidate.name)
		}
		return "", 80, fmt.Errorf("multiple Services detected in namespace: %s; set --host or DUBBO_SERVICE_HOST", strings.Join(names, ", "))
	}
	host := fmt.Sprintf("%s.%s.svc.%s", candidates[0].name, namespace, domainSuffix)
	return host, candidates[0].port, nil
}
