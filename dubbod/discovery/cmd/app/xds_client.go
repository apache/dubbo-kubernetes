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
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	pkgbootstrap "github.com/apache/dubbo-kubernetes/pkg/bootstrap"
	"github.com/apache/dubbo-kubernetes/pkg/config/constants"
	meshconfig "github.com/apache/dubbo-kubernetes/pkg/config/mesh"
	v1 "github.com/apache/dubbo-kubernetes/pkg/model"
	clusterv1 "github.com/kdubbo/xds-api/cluster/v1"
	corev1 "github.com/kdubbo/xds-api/core/v1"
	endpointv1 "github.com/kdubbo/xds-api/endpoint/v1"
	hcmv1 "github.com/kdubbo/xds-api/extensions/filters/v1/network/http_connection_manager"
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
	Endpoints []xdsEndpoint `json:"endpoints,omitempty"`
}

type xdsRouteSnapshot struct {
	Host         string           `json:"host"`
	Port         int              `json:"port"`
	Destinations []xdsDestination `json:"destinations"`
}

type sampleADSClient struct {
	conn   *grpc.ClientConn
	stream discovery.AggregatedDiscoveryService_StreamAggregatedResourcesClient
	node   *corev1.Node
	target string
	host   string
	port   int

	mu        sync.RWMutex
	subs      map[string][]string
	route     map[string]uint32
	endpoints map[string][]xdsEndpoint
	updates   chan struct{}
	errs      chan error
}

func newXDSClientCommand() *cobra.Command {
	namespace := firstNonEmpty(os.Getenv("POD_NAMESPACE"), "default")
	trustDomain := firstNonEmpty(os.Getenv("TRUST_DOMAIN"), constants.DefaultClusterLocalDomain)
	domainSuffix := firstNonEmpty(os.Getenv("DOMAIN_SUFFIX"), trustDomain, constants.DefaultClusterLocalDomain)
	host, port, hostErr := autoDiscoverServiceTarget(os.Environ(), namespace, domainSuffix)
	opts := &xdsClientOptions{
		host:           firstNonEmpty(os.Getenv("DUBBO_SERVICE_HOST"), host),
		port:           firstIntFromEnv(int(port), "DUBBO_SERVICE_PORT"),
		xdsAddress:     firstNonEmpty(os.Getenv("XDS_ADDRESS"), "dubbod.dubbo-system.svc:26010"),
		bootstrapPath:  os.Getenv("GRPC_XDS_BOOTSTRAP"),
		namespace:      namespace,
		podName:        firstNonEmpty(os.Getenv("POD_NAME"), os.Getenv("HOSTNAME"), "xds-client"),
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
		Use:   "xds-client [count]",
		Short: "run a no-proxy ADS stream client for service-to-service sample traffic",
		Args:  cobra.MaximumNArgs(1),
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
		conn:      conn,
		stream:    stream,
		node:      node,
		target:    opts.target,
		host:      opts.host,
		port:      opts.port,
		subs:      map[string][]string{},
		route:     map[string]uint32{},
		endpoints: map[string][]xdsEndpoint{},
		updates:   make(chan struct{}, 1),
		errs:      make(chan error, 1),
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
		weights, clusters, err := routeWeightsFromRoutes(resp.Resources)
		if err != nil {
			return err
		}
		c.mu.Lock()
		c.route = weights
		c.mu.Unlock()
		c.notify()
		if len(clusters) > 0 {
			return c.subscribe(v1.ClusterType, clusters)
		}
	case v1.ClusterType:
		edsNames, err := edsNamesFromClusters(resp.Resources)
		if err != nil {
			return err
		}
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
	for clusterName, weight := range c.route {
		if weight == 0 {
			continue
		}
		eps := append([]xdsEndpoint(nil), c.endpoints[clusterName]...)
		dest := destinationFromCluster(clusterName, weight, eps)
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

func routeWeightsFromRoutes(resources []*anypb.Any) (map[string]uint32, []string, error) {
	weights := map[string]uint32{}
	for _, resource := range resources {
		rc := &routev1.RouteConfiguration{}
		if err := proto.Unmarshal(resource.Value, rc); err != nil {
			return nil, nil, err
		}
		for _, vh := range rc.GetVirtualHosts() {
			for _, rt := range vh.GetRoutes() {
				action := rt.GetRoute()
				if action == nil {
					continue
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
		}
	}
	clusters := make([]string, 0, len(weights))
	for clusterName := range weights {
		clusters = append(clusters, clusterName)
	}
	return weights, sortedUnique(clusters), nil
}

func edsNamesFromClusters(resources []*anypb.Any) ([]string, error) {
	out := make([]string, 0, len(resources))
	for _, resource := range resources {
		c := &clusterv1.Cluster{}
		if err := proto.Unmarshal(resource.Value, c); err != nil {
			return nil, err
		}
		serviceName := c.GetName()
		if eds := c.GetEdsClusterConfig(); eds != nil && eds.GetServiceName() != "" {
			serviceName = eds.GetServiceName()
		}
		if serviceName != "" {
			out = append(out, serviceName)
		}
	}
	return sortedUnique(out), nil
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

func destinationFromCluster(clusterName string, weight uint32, endpoints []xdsEndpoint) xdsDestination {
	parts := strings.Split(clusterName, "|")
	dest := xdsDestination{Cluster: clusterName, Weight: weight, Endpoints: endpoints}
	if len(parts) == 4 {
		dest.Host = parts[3]
		dest.Subset = parts[2]
	}
	return dest
}

func runSampleRequests(ctx context.Context, adsClient *sampleADSClient, snapshot xdsRouteSnapshot, count int, requestInterval, requestTimeout time.Duration) error {
	picker, err := newSmoothWeightedPicker(snapshot)
	if err != nil {
		return err
	}
	currentSignature := snapshotSignature(snapshot)
	client := &http.Client{Timeout: requestTimeout}
	for i := 0; i < count; i++ {
		if updated, ok := adsClient.readySnapshot(nil); ok {
			updatedSignature := snapshotSignature(updated)
			if updatedSignature != currentSignature {
				snapshot = updated
				currentSignature = updatedSignature
				picker, err = newSmoothWeightedPicker(snapshot)
				if err != nil {
					return err
				}
			}
		}
		_, endpoint, err := picker.Next()
		if err != nil {
			return err
		}
		req, err := http.NewRequestWithContext(ctx, http.MethodGet,
			fmt.Sprintf("http://%s/", net.JoinHostPort(endpoint.Address, strconv.Itoa(int(endpoint.Port)))), nil)
		if err != nil {
			return err
		}
		req.Host = snapshot.Host
		resp, err := client.Do(req)
		if err != nil {
			return err
		}
		body, readErr := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		if readErr != nil {
			return readErr
		}
		fmt.Println(strings.TrimSpace(string(body)))
		if requestInterval > 0 && i+1 < count {
			timer := time.NewTimer(requestInterval)
			select {
			case <-ctx.Done():
				if !timer.Stop() {
					<-timer.C
				}
				return ctx.Err()
			case <-timer.C:
			}
		}
	}
	return nil
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

func subsetWeights(snapshot xdsRouteSnapshot) map[string]uint32 {
	out := map[string]uint32{}
	for _, dest := range snapshot.Destinations {
		out[dest.Subset] = dest.Weight
	}
	return out
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
		label := dest.Subset
		if label == "" {
			label = dest.Cluster
		}
		parts = append(parts, fmt.Sprintf("%s=%d endpoints=%d", label, dest.Weight, len(dest.Endpoints)))
	}
	return fmt.Sprintf("%s:%d %s", snapshot.Host, snapshot.Port, strings.Join(parts, ","))
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
