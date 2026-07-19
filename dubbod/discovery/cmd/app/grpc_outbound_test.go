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
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	neturl "net/url"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	tlsv1 "github.com/kdubbo/xds-api/extensions/transport_sockets/tls/v1"
	routev1 "github.com/kdubbo/xds-api/route/v1"
	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

func TestAutoDiscoverServiceTargetSingleService(t *testing.T) {
	host, port, err := autoDiscoverServiceTarget([]string{
		"KUBERNETES_SERVICE_HOST=10.96.0.1",
		"KUBERNETES_SERVICE_PORT=443",
		"NGINX_SERVICE_HOST=10.96.12.34",
		"NGINX_SERVICE_PORT=80",
	}, "app", "cluster.local")
	if err != nil {
		t.Fatalf("autoDiscoverServiceTarget() error = %v", err)
	}
	if host != "nginx.app.svc.cluster.local" {
		t.Fatalf("host = %q, want %q", host, "nginx.app.svc.cluster.local")
	}
	if port != 80 {
		t.Fatalf("port = %d, want 80", port)
	}
}

func TestAutoDiscoverServiceTargetMultipleServices(t *testing.T) {
	_, _, err := autoDiscoverServiceTarget([]string{
		"NGINX_SERVICE_HOST=10.96.12.34",
		"NGINX_SERVICE_PORT=80",
		"REDIS_SERVICE_HOST=10.96.22.44",
		"REDIS_SERVICE_PORT=6379",
	}, "app", "cluster.local")
	if err == nil {
		t.Fatalf("autoDiscoverServiceTarget() error = nil, want multiple services error")
	}
	if !strings.Contains(err.Error(), "multiple Services detected") {
		t.Fatalf("error = %q, want multiple services message", err.Error())
	}
}

func TestSmoothWeightedPickerDistributesRequestsPerPick(t *testing.T) {
	picker, err := newSmoothWeightedPicker(xdsRouteSnapshot{
		Host: "nginx.app.svc.cluster.local",
		Port: 80,
		Destinations: []xdsDestination{
			{
				Cluster:   "outbound|80|v1|nginx.app.svc.cluster.local",
				Subset:    "v1",
				Weight:    23,
				Endpoints: []xdsEndpoint{{Address: "10.0.0.1", Port: 80}},
			},
			{
				Cluster:   "outbound|80|v2|nginx.app.svc.cluster.local",
				Subset:    "v2",
				Weight:    77,
				Endpoints: []xdsEndpoint{{Address: "10.0.0.2", Port: 80}},
			},
		},
	})
	if err != nil {
		t.Fatalf("newSmoothWeightedPicker() error = %v", err)
	}

	counts := map[string]int{}
	sequence := make([]string, 0, 100)
	for i := 0; i < 100; i++ {
		destination, _, err := picker.Next()
		if err != nil {
			t.Fatalf("picker.Next() error = %v", err)
		}
		counts[destination.Subset]++
		sequence = append(sequence, destination.Subset)
	}

	if counts["v1"] != 23 || counts["v2"] != 77 {
		t.Fatalf("counts = %#v, want v1=23 and v2=77", counts)
	}
	if strings.Count(strings.Join(sequence[:10], ","), "v1") == 0 {
		t.Fatalf("first 10 picks = %v, want interleaved per-request selection", sequence[:10])
	}
}

func TestSmoothWeightedPickerRoundRobinsEndpointsWithinCluster(t *testing.T) {
	picker, err := newSmoothWeightedPicker(xdsRouteSnapshot{
		Host: "nginx.app.svc.cluster.local",
		Port: 80,
		Destinations: []xdsDestination{
			{
				Cluster: "outbound|80|v1|nginx.app.svc.cluster.local",
				Subset:  "v1",
				Weight:  1,
				Endpoints: []xdsEndpoint{
					{Address: "10.0.0.1", Port: 80},
					{Address: "10.0.0.2", Port: 80},
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("newSmoothWeightedPicker() error = %v", err)
	}

	addresses := make([]string, 0, 4)
	for i := 0; i < 4; i++ {
		_, endpoint, err := picker.Next()
		if err != nil {
			t.Fatalf("picker.Next() error = %v", err)
		}
		addresses = append(addresses, endpoint.Address)
	}

	want := []string{"10.0.0.1", "10.0.0.2", "10.0.0.1", "10.0.0.2"}
	for i := range want {
		if addresses[i] != want[i] {
			t.Fatalf("addresses = %v, want %v", addresses, want)
		}
	}
}

func TestRunSampleRequestsUsesUpdatedSnapshotOnSameStream(t *testing.T) {
	serverV1 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = fmt.Fprintln(w, "nginx v1")
	}))
	defer serverV1.Close()
	serverV2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = fmt.Fprintln(w, "nginx v2")
	}))
	defer serverV2.Close()

	v1Endpoint := endpointForServer(t, serverV1)
	v2Endpoint := endpointForServer(t, serverV2)
	initial := xdsRouteSnapshot{
		Host: "nginx.app.svc.cluster.local",
		Port: 80,
		Destinations: []xdsDestination{{
			Cluster:   "outbound|80|v1|nginx.app.svc.cluster.local",
			Subset:    "v1",
			Weight:    1,
			Endpoints: []xdsEndpoint{v1Endpoint},
		}},
	}
	updated := xdsRouteSnapshot{
		Host: "nginx.app.svc.cluster.local",
		Port: 80,
		Destinations: []xdsDestination{{
			Cluster:   "outbound|80|v2|nginx.app.svc.cluster.local",
			Subset:    "v2",
			Weight:    1,
			Endpoints: []xdsEndpoint{v2Endpoint},
		}},
	}

	client := &sampleADSClient{
		host:      initial.Host,
		port:      initial.Port,
		route:     map[string]uint32{initial.Destinations[0].Cluster: 1},
		endpoints: map[string][]xdsEndpoint{initial.Destinations[0].Cluster: {v1Endpoint}},
	}

	oldStdout := os.Stdout
	readPipe, writePipe, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe() error = %v", err)
	}
	os.Stdout = writePipe
	defer func() {
		os.Stdout = oldStdout
	}()

	errCh := make(chan error, 1)
	go func() {
		errCh <- runSampleRequests(context.Background(), client, initial, 6, 20*time.Millisecond, time.Second)
		_ = writePipe.Close()
	}()

	time.Sleep(50 * time.Millisecond)
	client.mu.Lock()
	client.route = map[string]uint32{updated.Destinations[0].Cluster: 1}
	client.endpoints = map[string][]xdsEndpoint{updated.Destinations[0].Cluster: {v2Endpoint}}
	client.mu.Unlock()

	output, readErr := io.ReadAll(readPipe)
	if readErr != nil {
		t.Fatalf("io.ReadAll() error = %v", readErr)
	}
	if err := <-errCh; err != nil {
		t.Fatalf("runSampleRequests() error = %v", err)
	}

	lines := strings.Fields(string(output))
	if !contains(lines, "v1") || !contains(lines, "v2") {
		t.Fatalf("output = %q, want both v1 and v2 responses after live route update", string(output))
	}
}

func TestRunSampleRequestsUsesRequestPathAndHeaders(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/reviews" {
			t.Fatalf("path = %q, want /reviews", r.URL.Path)
		}
		if got := r.Header.Get("end-user"); got != "terminal-user" {
			t.Fatalf("end-user header = %q, want terminal-user", got)
		}
		_, _ = fmt.Fprintln(w, "reviews v1")
	}))
	defer server.Close()

	endpoint := endpointForServer(t, server)
	snapshot := xdsRouteSnapshot{
		Host: "reviews.moviereview.svc.cluster.local",
		Port: 9080,
		Destinations: []xdsDestination{{
			Cluster:   "outbound|9080|v1|reviews.moviereview.svc.cluster.local",
			Subset:    "v1",
			Weight:    1,
			Endpoints: []xdsEndpoint{endpoint},
		}},
	}
	client := &sampleADSClient{
		host:           snapshot.Host,
		port:           snapshot.Port,
		path:           "/reviews",
		requestHeaders: http.Header{"End-User": []string{"terminal-user"}},
		route:          map[string]uint32{snapshot.Destinations[0].Cluster: 1},
		endpoints:      map[string][]xdsEndpoint{snapshot.Destinations[0].Cluster: {endpoint}},
	}

	if err := runSampleRequests(context.Background(), client, snapshot, 1, 0, time.Second); err != nil {
		t.Fatalf("runSampleRequests() error = %v", err)
	}
}

func TestRunSampleRequestsUsesRouteTimeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(50 * time.Millisecond)
		_, _ = w.Write([]byte("late"))
	}))
	defer server.Close()

	endpoint := endpointForServer(t, server)
	snapshot := xdsRouteSnapshot{
		Host:    "reviews.moviereview.svc.cluster.local",
		Port:    9080,
		Timeout: "1ms",
		Destinations: []xdsDestination{{
			Cluster:   "outbound|9080|v1|reviews.moviereview.svc.cluster.local",
			Host:      "reviews.moviereview.svc.cluster.local",
			Subset:    "v1",
			Weight:    100,
			Endpoints: []xdsEndpoint{endpoint},
		}},
	}
	client := &sampleADSClient{
		host:         snapshot.Host,
		port:         snapshot.Port,
		path:         "/",
		route:        map[string]uint32{snapshot.Destinations[0].Cluster: 100},
		routeTimeout: time.Millisecond,
		endpoints:    map[string][]xdsEndpoint{snapshot.Destinations[0].Cluster: {endpoint}},
	}

	err := runSampleRequests(context.Background(), client, snapshot, 1, 0, time.Second)
	if err == nil || !strings.Contains(err.Error(), "context deadline exceeded") {
		t.Fatalf("runSampleRequests() error = %v, want context deadline exceeded", err)
	}
}

func TestRouteWeightsFromRoutesFiltersHeaderMatchedRoute(t *testing.T) {
	resources := []*anypb.Any{mustAnyRouteConfig(t, &routev1.RouteConfiguration{
		VirtualHosts: []*routev1.VirtualHost{{
			Routes: []*routev1.Route{
				{
					Match: &routev1.RouteMatch{
						PathSpecifier: &routev1.RouteMatch_Prefix{Prefix: "/"},
						Headers: []*routev1.HeaderMatcher{{
							Name: "end-user",
							HeaderMatchSpecifier: &routev1.HeaderMatcher_ExactMatch{
								ExactMatch: "terminal-user",
							},
						}},
					},
					Action: weightedRouteAction(map[string]uint32{
						"outbound|9080|v1|reviews.moviereview.svc.cluster.local": 100,
					}),
				},
				{
					Match: &routev1.RouteMatch{PathSpecifier: &routev1.RouteMatch_Prefix{Prefix: "/"}},
					Action: weightedRouteAction(map[string]uint32{
						"outbound|9080|v2|reviews.moviereview.svc.cluster.local": 20,
						"outbound|9080|v3|reviews.moviereview.svc.cluster.local": 80,
					}),
				},
			},
		}},
	})}

	weights, _, _, err := routeWeightsFromRoutes(resources, "/", http.Header{"End-User": []string{"terminal-user"}})
	if err != nil {
		t.Fatalf("routeWeightsFromRoutes() error = %v", err)
	}
	if got := weights["outbound|9080|v1|reviews.moviereview.svc.cluster.local"]; got != 100 || len(weights) != 1 {
		t.Fatalf("terminal-user weights = %v, want v1=100 only", weights)
	}

	weights, _, _, err = routeWeightsFromRoutes(resources, "/", nil)
	if err != nil {
		t.Fatalf("routeWeightsFromRoutes() fallback error = %v", err)
	}
	if len(weights) != 2 ||
		weights["outbound|9080|v2|reviews.moviereview.svc.cluster.local"] != 20 ||
		weights["outbound|9080|v3|reviews.moviereview.svc.cluster.local"] != 80 {
		t.Fatalf("fallback weights = %v, want v2=20 and v3=80", weights)
	}
}

func TestRouteWeightsFromRoutesReadsRequestTimeout(t *testing.T) {
	resources := []*anypb.Any{mustAnyRouteConfig(t, &routev1.RouteConfiguration{
		VirtualHosts: []*routev1.VirtualHost{{
			Routes: []*routev1.Route{{
				Match: &routev1.RouteMatch{PathSpecifier: &routev1.RouteMatch_Prefix{Prefix: "/reviews"}},
				Action: &routev1.Route_Route{
					Route: &routev1.RouteAction{
						ClusterSpecifier: &routev1.RouteAction_WeightedClusters{
							WeightedClusters: &routev1.WeightedCluster{Clusters: []*routev1.WeightedCluster_ClusterWeight{{
								Name:   "outbound|9080|v1|reviews.moviereview.svc.cluster.local",
								Weight: wrapperspb.UInt32(100),
							}}},
						},
						Timeout: durationpb.New(500 * time.Millisecond),
					},
				},
			}},
		}},
	})}

	weights, _, timeout, err := routeWeightsFromRoutes(resources, "/reviews", nil)
	if err != nil {
		t.Fatalf("routeWeightsFromRoutes() error = %v", err)
	}
	if got := weights["outbound|9080|v1|reviews.moviereview.svc.cluster.local"]; got != 100 {
		t.Fatalf("weights = %v, want v1=100", weights)
	}
	if timeout != 500*time.Millisecond {
		t.Fatalf("timeout = %v, want 500ms", timeout)
	}
}

func TestSampleRequestClientsDoNotBypassMTLS(t *testing.T) {
	clients := newSampleRequestClients(&sampleADSClient{}, time.Second)
	destination := xdsDestination{
		Cluster: "outbound|80|v1|nginx.app.svc.cluster.local",
		Host:    "nginx.app.svc.cluster.local",
		TLSMode: "DUBBO_MUTUAL",
		SNI:     "nginx.app.svc.cluster.local",
		tlsContext: &tlsv1.UpstreamTlsContext{
			Sni: "nginx.app.svc.cluster.local",
		},
	}

	_, _, err := clients.clientForDestination(destination)
	if err == nil {
		t.Fatalf("clientForDestination() error = nil, want bootstrap error for mTLS route")
	}
	if !strings.Contains(err.Error(), "data-plane mTLS requires GRPC_XDS_BOOTSTRAP or --bootstrap") {
		t.Fatalf("error = %q, want bootstrap requirement", err.Error())
	}
}

func TestDestinationFromClusterExposesTLSContext(t *testing.T) {
	tlsContext := &tlsv1.UpstreamTlsContext{Sni: "nginx.app.svc.cluster.local"}
	destination := destinationFromCluster(
		"outbound|80|v1|nginx.app.svc.cluster.local",
		50,
		[]xdsEndpoint{{Address: "10.0.0.1", Port: 80}},
		tlsContext,
	)

	if destination.TLSMode != "DUBBO_MUTUAL" {
		t.Fatalf("TLSMode = %q, want DUBBO_MUTUAL", destination.TLSMode)
	}
	if destination.SNI != "nginx.app.svc.cluster.local" {
		t.Fatalf("SNI = %q, want nginx host", destination.SNI)
	}
	if destination.tlsContext != tlsContext {
		t.Fatalf("tlsContext was not preserved")
	}
}

func TestSubsetWeightsUsesServiceNameWhenSubsetIsEmpty(t *testing.T) {
	weights := subsetWeights(xdsRouteSnapshot{
		Destinations: []xdsDestination{
			{Host: "reviews-1.default.svc.cluster.local", Weight: 20},
			{Host: "reviews-2.default.svc.cluster.local", Weight: 80},
		},
	})
	if len(weights) != 2 || weights["reviews-1"] != 20 || weights["reviews-2"] != 80 {
		t.Fatalf("weights = %v, want reviews-1=20 and reviews-2=80", weights)
	}
}

func TestSubsetWeightsKeepsSubsetName(t *testing.T) {
	weights := subsetWeights(xdsRouteSnapshot{
		Destinations: []xdsDestination{
			{Host: "reviews.default.svc.cluster.local", Subset: "v1", Weight: 20},
			{Host: "reviews.default.svc.cluster.local", Subset: "v2", Weight: 80},
		},
	})
	if len(weights) != 2 || weights["v1"] != 20 || weights["v2"] != 80 {
		t.Fatalf("weights = %v, want v1=20 and v2=80", weights)
	}
}

func endpointForServer(t *testing.T, server *httptest.Server) xdsEndpoint {
	t.Helper()
	parsed, err := neturl.Parse(server.URL)
	if err != nil {
		t.Fatalf("url.Parse(%q) error = %v", server.URL, err)
	}
	host, portValue, err := net.SplitHostPort(parsed.Host)
	if err != nil {
		t.Fatalf("net.SplitHostPort(%q) error = %v", parsed.Host, err)
	}
	port, err := strconv.Atoi(portValue)
	if err != nil {
		t.Fatalf("strconv.Atoi(%q) error = %v", portValue, err)
	}
	return xdsEndpoint{Address: host, Port: uint32(port)}
}

func contains(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

func mustAnyRouteConfig(t *testing.T, rc *routev1.RouteConfiguration) *anypb.Any {
	t.Helper()
	out, err := anypb.New(rc)
	if err != nil {
		t.Fatalf("anypb.New() error = %v", err)
	}
	return out
}

func weightedRouteAction(weights map[string]uint32) *routev1.Route_Route {
	clusters := make([]*routev1.WeightedCluster_ClusterWeight, 0, len(weights))
	for name, weight := range weights {
		clusters = append(clusters, &routev1.WeightedCluster_ClusterWeight{
			Name:   name,
			Weight: wrapperspb.UInt32(weight),
		})
	}
	return &routev1.Route_Route{
		Route: &routev1.RouteAction{
			ClusterSpecifier: &routev1.RouteAction_WeightedClusters{
				WeightedClusters: &routev1.WeightedCluster{Clusters: clusters},
			},
		},
	}
}
