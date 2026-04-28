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
		endpoints: map[string][]xdsEndpoint{initial.Destinations[0].Cluster: []xdsEndpoint{v1Endpoint}},
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
	client.endpoints = map[string][]xdsEndpoint{updated.Destinations[0].Cluster: []xdsEndpoint{v2Endpoint}}
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
