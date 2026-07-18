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

package xds

import (
	"fmt"
	"io"
	"testing"
	"time"

	"github.com/apache/dubbo-kubernetes/dubbod/discovery/pkg/model"
	v1 "github.com/apache/dubbo-kubernetes/dubbod/discovery/pkg/xds/v1"
	"github.com/apache/dubbo-kubernetes/pkg/config/host"
	"github.com/apache/dubbo-kubernetes/pkg/config/mesh"
	"github.com/apache/dubbo-kubernetes/pkg/config/mesh/meshwatcher"
	"github.com/apache/dubbo-kubernetes/pkg/config/protocol"
	"github.com/apache/dubbo-kubernetes/pkg/config/schema/kind"
	"github.com/apache/dubbo-kubernetes/pkg/kube/krt"
	dubbolog "github.com/apache/dubbo-kubernetes/pkg/log"
	"github.com/apache/dubbo-kubernetes/pkg/util/sets"
	endpoint "github.com/kdubbo/xds-api/endpoint/v1"
	discovery "github.com/kdubbo/xds-api/service/discovery/v1"
)

type proxylessPushScale struct {
	services            int
	endpointsPerService int
	connections         int
}

func (s proxylessPushScale) name() string {
	return fmt.Sprintf(
		"services=%d/endpoints_per_service=%d/connections=%d",
		s.services,
		s.endpointsPerService,
		s.connections,
	)
}

var proxylessPushScaleMatrix = []proxylessPushScale{
	{services: 10, endpointsPerService: 10, connections: 100},
	{services: 100, endpointsPerService: 10, connections: 100},
	{services: 1000, endpointsPerService: 10, connections: 100},
	{services: 100, endpointsPerService: 1, connections: 100},
	{services: 100, endpointsPerService: 100, connections: 100},
	{services: 100, endpointsPerService: 10, connections: 10},
	{services: 100, endpointsPerService: 10, connections: 1000},
}

var proxylessFullPushScaleMatrix = []proxylessPushScale{
	{services: 10, endpointsPerService: 10, connections: 10},
	{services: 100, endpointsPerService: 10, connections: 10},
	{services: 1000, endpointsPerService: 10, connections: 10},
	{services: 100, endpointsPerService: 1, connections: 10},
	{services: 100, endpointsPerService: 100, connections: 10},
	{services: 100, endpointsPerService: 10, connections: 1},
	{services: 100, endpointsPerService: 10, connections: 100},
}

var benchmarkPushResponse *discovery.DiscoveryResponse

func BenchmarkProxylessPushScale(b *testing.B) {
	for _, scale := range proxylessPushScaleMatrix {
		b.Run(scale.name(), func(b *testing.B) {
			fixture := newProxylessPushBenchmarkFixture(b, scale)
			b.ReportAllocs()
			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				responses, resources, lastResponse := fixture.pushOnce(b)
				if responses != scale.connections || resources != scale.connections {
					b.Fatalf(
						"push produced responses/resources = %d/%d, want %d/%d",
						responses,
						resources,
						scale.connections,
						scale.connections,
					)
				}
				benchmarkPushResponse = lastResponse
			}
			b.StopTimer()
			b.ReportMetric(float64(scale.services), "services")
			b.ReportMetric(float64(scale.endpointsPerService), "endpoints/service")
			b.ReportMetric(float64(scale.connections), "connections")
			b.ReportMetric(float64(scale.endpointsPerService*scale.connections), "endpoint-copies/op")
		})
	}
}

func BenchmarkProxylessFullPushScale(b *testing.B) {
	for _, scale := range proxylessFullPushScaleMatrix {
		b.Run(scale.name(), func(b *testing.B) {
			fixture := newProxylessPushBenchmarkFixture(b, scale)
			fixture.request.Full = true
			fixture.request.Forced = true
			fixture.request.ConfigsUpdated = nil
			wantResources := scale.services * scale.connections
			b.ReportAllocs()
			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				responses, resources, lastResponse := fixture.pushOnce(b)
				if responses != scale.connections || resources != wantResources {
					b.Fatalf(
						"full push produced responses/resources = %d/%d, want %d/%d",
						responses,
						resources,
						scale.connections,
						wantResources,
					)
				}
				benchmarkPushResponse = lastResponse
			}
			b.StopTimer()
			b.ReportMetric(float64(scale.services), "services")
			b.ReportMetric(float64(scale.endpointsPerService), "endpoints/service")
			b.ReportMetric(float64(scale.connections), "connections")
			b.ReportMetric(float64(scale.services*scale.endpointsPerService*scale.connections), "endpoint-copies/op")
		})
	}
}

func TestProxylessPushBenchmarkFixture(t *testing.T) {
	scale := proxylessPushScale{services: 3, endpointsPerService: 2, connections: 4}
	fixture := newProxylessPushBenchmarkFixture(t, scale)

	responses, resources, lastResponse := fixture.pushOnce(t)
	if responses != scale.connections || resources != scale.connections {
		t.Fatalf(
			"push produced responses/resources = %d/%d, want %d/%d",
			responses,
			resources,
			scale.connections,
			scale.connections,
		)
	}
	if lastResponse == nil || len(lastResponse.Resources) != 1 {
		t.Fatalf("last response has %d resources, want 1", len(lastResponse.GetResources()))
	}
	assignment := &endpoint.ClusterLoadAssignment{}
	if err := lastResponse.Resources[0].UnmarshalTo(assignment); err != nil {
		t.Fatalf("unmarshal endpoint assignment: %v", err)
	}
	gotEndpoints := 0
	for _, locality := range assignment.Endpoints {
		gotEndpoints += len(locality.LbEndpoints)
	}
	if gotEndpoints != scale.endpointsPerService {
		t.Fatalf("endpoint assignment has %d endpoints, want %d", gotEndpoints, scale.endpointsPerService)
	}
}

type proxylessPushBenchmarkFixture struct {
	server      *DiscoveryServer
	request     *model.PushRequest
	stream      *fakeADSStream
	connections int
}

func newProxylessPushBenchmarkFixture(tb testing.TB, scale proxylessPushScale) *proxylessPushBenchmarkFixture {
	tb.Helper()
	if scale.services <= 0 || scale.endpointsPerService <= 0 || scale.connections <= 0 {
		tb.Fatalf("benchmark scale values must all be positive: %+v", scale)
	}
	silenceProxylessPushBenchmarkLogs(tb)

	services := make([]*model.Service, 0, scale.services)
	serviceByHostname := make(map[host.Name]*model.Service, scale.services)
	clusterNames := make([]string, 0, scale.services)
	for serviceIndex := 0; serviceIndex < scale.services; serviceIndex++ {
		serviceName := fmt.Sprintf("svc-%d", serviceIndex)
		hostname := host.Name(serviceName + ".app.svc.cluster.local")
		service := &model.Service{
			Hostname: hostname,
			Ports: model.PortList{{
				Name:     "grpc",
				Port:     80,
				Protocol: protocol.GRPC,
			}},
			Attributes: model.ServiceAttributes{
				Name:      serviceName,
				Namespace: "app",
			},
		}
		services = append(services, service)
		serviceByHostname[hostname] = service
		clusterNames = append(clusterNames, model.BuildSubsetKey(model.TrafficDirectionOutbound, "", hostname, 80))
	}

	env := model.NewEnvironment()
	env.ConfigStore = testConfigStore{}
	env.ServiceDiscovery = benchmarkServiceDiscovery{services: services, byHostname: serviceByHostname}
	env.Watcher = meshwatcher.ConfigAdapter(krt.NewStatic(&meshwatcher.MeshConfigResource{
		MeshConfig: mesh.DefaultMeshConfig(),
	}, true))
	env.Init()

	for serviceIndex, service := range services {
		endpoints := make([]*model.DubboEndpoint, 0, scale.endpointsPerService)
		for endpointIndex := 0; endpointIndex < scale.endpointsPerService; endpointIndex++ {
			endpoints = append(endpoints, &model.DubboEndpoint{
				Addresses:       []string{benchmarkEndpointAddress(serviceIndex, endpointIndex)},
				EndpointPort:    80,
				ServicePortName: "grpc",
				HealthStatus:    model.Healthy,
				Namespace:       "app",
				WorkloadName:    fmt.Sprintf("%s-%d", service.Attributes.Name, endpointIndex),
			})
		}
		env.EndpointIndex.UpdateServiceEndpoints(model.ShardKey{}, string(service.Hostname), "app", endpoints, false)
	}

	push := model.NewPushContext()
	push.PushVersion = "benchmark"
	push.InitContext(env, nil, nil)
	env.SetPushContext(push)

	server := NewDiscoveryServer(env, nil, nil)
	server.Generators[v1.EndpointType] = &EdsGenerator{
		Cache:         model.DisabledCache{},
		EndpointIndex: env.EndpointIndex,
	}
	stream := newFakeADSStream()
	for connectionIndex := 0; connectionIndex < scale.connections; connectionIndex++ {
		connectionID := fmt.Sprintf("proxyless-%d", connectionIndex)
		connection := newConnection("127.0.0.1:26010", stream)
		connection.SetID(connectionID)
		connection.s = server
		connection.proxy = &model.Proxy{
			ID:              connectionID,
			Type:            model.Proxyless,
			ConfigNamespace: "app",
			Metadata: &model.NodeMetadata{
				Generator: "grpc",
				Namespace: "app",
			},
			WatchedResources: map[string]*model.WatchedResource{
				v1.EndpointType: {
					TypeUrl:       v1.EndpointType,
					ResourceNames: sets.New(clusterNames...),
				},
			},
			LastPushContext: push,
			LastPushTime:    time.Now(),
		}
		server.adsClients[connectionID] = connection
	}

	targetService := services[0]
	return &proxylessPushBenchmarkFixture{
		server: server,
		request: &model.PushRequest{
			Push:   push,
			Start:  time.Now(),
			Reason: model.NewReasonStats(model.EndpointUpdate),
			ConfigsUpdated: sets.New(model.ConfigKey{
				Kind:      kind.Service,
				Name:      string(targetService.Hostname),
				Namespace: targetService.Attributes.Namespace,
			}),
		},
		stream:      stream,
		connections: scale.connections,
	}
}

func silenceProxylessPushBenchmarkLogs(tb testing.TB) {
	tb.Helper()
	for _, scopeName := range []string{"ads", "model"} {
		scope := dubbolog.FindScope(scopeName)
		if scope == nil {
			continue
		}
		previousOutput := scope.GetOutput()
		scope.SetOutput(io.Discard)
		tb.Cleanup(func() {
			scope.SetOutput(previousOutput)
		})
	}
}

func (f *proxylessPushBenchmarkFixture) pushOnce(tb testing.TB) (int, int, *discovery.DiscoveryResponse) {
	tb.Helper()
	clients := f.server.clientsForPush(f.request)
	if len(clients) != f.connections {
		tb.Fatalf("clientsForPush() returned %d clients, want %d", len(clients), f.connections)
	}

	responses := 0
	resources := 0
	var lastResponse *discovery.DiscoveryResponse
	for _, connection := range clients {
		if err := f.server.pushConnection(connection, &Event{pushRequest: f.request}); err != nil {
			tb.Fatalf("pushConnection(%s): %v", connection.ID(), err)
		}
		select {
		case response := <-f.stream.sendCh:
			responses++
			resources += len(response.Resources)
			lastResponse = response
		default:
			tb.Fatalf("pushConnection(%s) did not send a response", connection.ID())
		}
	}
	return responses, resources, lastResponse
}

func benchmarkEndpointAddress(serviceIndex, endpointIndex int) string {
	return fmt.Sprintf(
		"10.%d.%d.%d",
		(serviceIndex/250)%250+1,
		serviceIndex%250+1,
		endpointIndex%250+1,
	)
}

type benchmarkServiceDiscovery struct {
	services   []*model.Service
	byHostname map[host.Name]*model.Service
}

func (d benchmarkServiceDiscovery) Services() []*model.Service {
	return d.services
}

func (d benchmarkServiceDiscovery) GetService(hostname host.Name) *model.Service {
	return d.byHostname[hostname]
}

func (benchmarkServiceDiscovery) GetProxyServiceTargets(*model.Proxy) []model.ServiceTarget {
	return nil
}
