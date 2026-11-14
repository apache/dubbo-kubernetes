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
	"encoding/json"
	"fmt"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/apache/dubbo-kubernetes/dubbod/discovery/pkg/config/memory"
	"github.com/apache/dubbo-kubernetes/dubbod/discovery/pkg/model"
	"github.com/apache/dubbo-kubernetes/dubbod/discovery/pkg/networking/triplegen"
	v1 "github.com/apache/dubbo-kubernetes/dubbod/discovery/pkg/xds/v1"
	"github.com/apache/dubbo-kubernetes/pkg/config"
	"github.com/apache/dubbo-kubernetes/pkg/config/host"
	"github.com/apache/dubbo-kubernetes/pkg/config/mesh"
	"github.com/apache/dubbo-kubernetes/pkg/config/mesh/meshwatcher"
	"github.com/apache/dubbo-kubernetes/pkg/config/protocol"
	"github.com/apache/dubbo-kubernetes/pkg/config/schema/collections"
	"github.com/apache/dubbo-kubernetes/pkg/config/schema/gvk"
	"github.com/apache/dubbo-kubernetes/pkg/config/schema/kind"
	"github.com/apache/dubbo-kubernetes/pkg/kube/krt"
	"github.com/apache/dubbo-kubernetes/pkg/util/sets"
	securityv1alpha3 "github.com/dubml/api/security/v1alpha3"
	cluster "github.com/dubml/xds-api/cluster/v1"
	core "github.com/dubml/xds-api/core/v1"
	tlsv1 "github.com/dubml/xds-api/extensions/transport_sockets/tls/v1"
	discovery "github.com/dubml/xds-api/service/discovery/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/structpb"
)

func TestInitProxyMetadataGeneratorSupport(t *testing.T) {
	server := &DiscoveryServer{}
	makeNode := func(meta map[string]any) *core.Node {
		t.Helper()
		st, err := structpb.NewStruct(meta)
		if err != nil {
			t.Fatalf("NewStruct() failed: %v", err)
		}
		return &core.Node{
			Id:       "inherent~10.0.0.1~pod-1.app~app.svc.cluster.local",
			Metadata: st,
		}
	}

	tripleProxy, err := server.initProxyMetadata(makeNode(map[string]any{
		"GENERATOR": "triple",
		"NAMESPACE": "app",
	}))
	if err != nil {
		t.Fatalf("initProxyMetadata(triple) failed: %v", err)
	}
	if tripleProxy.Metadata == nil || tripleProxy.Metadata.Generator != "triple" || !tripleProxy.IsInherentTriple() {
		t.Fatalf("unexpected triple proxy metadata: %+v", tripleProxy.Metadata)
	}

	_, err = server.initProxyMetadata(makeNode(map[string]any{
		"GENERATOR": "grpc",
		"NAMESPACE": "app",
	}))
	if err == nil {
		t.Fatal("initProxyMetadata(grpc) succeeded, want unsupported error")
	}
	if status.Code(err) != codes.InvalidArgument || !strings.Contains(err.Error(), "unsupported xDS generator") {
		t.Fatalf("initProxyMetadata(grpc) err = %v, want InvalidArgument unsupported xDS generator", err)
	}
}

func TestConnectionWatchesAnyServiceFiltersInherentEDS(t *testing.T) {
	con := &Connection{proxy: &model.Proxy{
		Type: model.Inherent,
		Metadata: &model.NodeMetadata{
			Generator: "triple",
		},
		WatchedResources: map[string]*model.WatchedResource{
			v1.EndpointType: {
				TypeUrl: v1.EndpointType,
				ResourceNames: sets.New(
					"outbound|80||nginx.app.svc.cluster.local",
					"outbound|80|v1|reviews.app.svc.cluster.local",
				),
			},
		},
	}}

	if !connectionWatchesAnyService(con, sets.New("nginx.app.svc.cluster.local")) {
		t.Fatalf("expected Inherent connection to watch nginx")
	}
	if connectionWatchesAnyService(con, sets.New("api.app.svc.cluster.local")) {
		t.Fatalf("expected Inherent connection not to watch api")
	}
}

func TestConnectionWatchesAnyServiceKeepsBroadPushForUnknownWatch(t *testing.T) {
	con := &Connection{proxy: &model.Proxy{
		Type: model.Inherent,
		Metadata: &model.NodeMetadata{
			Generator: "triple",
		},
		WatchedResources: map[string]*model.WatchedResource{},
	}}

	if !connectionWatchesAnyService(con, sets.New("api.app.svc.cluster.local")) {
		t.Fatalf("connection without EDS watch should keep broad push")
	}
}

func TestClientsForPushInherentLargeScaleTargetsWatchedService(t *testing.T) {
	server, req := newInherentTargetedPushServer(10000, 100, "svc-7.app.svc.cluster.local")

	clients := server.clientsForPush(req)
	if len(clients) != 100 {
		t.Fatalf("clientsForPush() returned %d clients, want 100", len(clients))
	}
	for _, con := range clients {
		if !connectionWatchesAnyService(con, sets.New("svc-7.app.svc.cluster.local")) {
			t.Fatalf("returned client %s does not watch targeted service", con.ID())
		}
	}
}

func TestPushConnectionExercisesInherentPushPath(t *testing.T) {
	server, con, stream := newInherentXDSTestServer(t)
	req := &model.PushRequest{
		Full:   true,
		Push:   con.proxy.LastPushContext,
		Reason: model.NewReasonStats(model.ProxyRequest),
		Start:  time.Now(),
	}
	proxyNeedsPushCalled := false
	server.ProxyNeedsPush = func(gotProxy *model.Proxy, gotReq *model.PushRequest) (*model.PushRequest, bool) {
		proxyNeedsPushCalled = true
		if gotProxy != con.proxy || gotReq != req {
			t.Fatalf("ProxyNeedsPush received proxy=%p request=%p, want proxy=%p request=%p", gotProxy, gotReq, con.proxy, req)
		}
		return gotReq, true
	}

	if err := server.pushConnection(con, &Event{pushRequest: req}); err != nil {
		t.Fatalf("pushConnection() failed: %v", err)
	}
	if !proxyNeedsPushCalled {
		t.Fatal("ProxyNeedsPush was not called")
	}
	responses := stream.takeAllResponses(t, 3)
	assertResponseTypeOrder(t, responses, v1.ListenerType, v1.ClusterType, v1.RouteType)
	if !con.proxy.LastPushTime.Equal(req.Start) {
		t.Fatalf("LastPushTime = %v, want %v", con.proxy.LastPushTime, req.Start)
	}
}

func TestPushConnectionSkipsUnreadyAndUnneededPushes(t *testing.T) {
	t.Run("uninitialized connection", func(t *testing.T) {
		server, _, stream := newInherentXDSTestServer(t)
		con := &Connection{}
		if err := server.pushConnection(con, &Event{pushRequest: &model.PushRequest{Full: true}}); err != nil {
			t.Fatalf("pushConnection() failed: %v", err)
		}
		stream.assertNoResponse(t)
	})

	t.Run("nil request", func(t *testing.T) {
		server, con, stream := newInherentXDSTestServer(t)
		if err := server.pushConnection(con, &Event{}); err != nil {
			t.Fatalf("pushConnection() failed: %v", err)
		}
		stream.assertNoResponse(t)
	})

	t.Run("proxy does not need push", func(t *testing.T) {
		server, con, stream := newInherentXDSTestServer(t)
		server.ProxyNeedsPush = func(_ *model.Proxy, req *model.PushRequest) (*model.PushRequest, bool) {
			return req, false
		}
		if err := server.pushConnection(con, &Event{pushRequest: &model.PushRequest{Push: con.proxy.LastPushContext}}); err != nil {
			t.Fatalf("pushConnection() failed: %v", err)
		}
		stream.assertNoResponse(t)
	})
}

func TestDiscoveryPushQueuesAndNotifies(t *testing.T) {
	for _, full := range []bool{false, true} {
		t.Run(fmt.Sprintf("full=%v", full), func(t *testing.T) {
			server, con, _ := newInherentXDSTestServer(t)
			server.addCon("inherent-1", con)

			var notified *model.ConfigChange
			server.Env.AddConfigHandler(func(change *model.ConfigChange) {
				notified = change
			})
			req := &model.PushRequest{
				Full:   full,
				Reason: model.NewReasonStats(model.ConfigUpdate),
				ConfigsUpdated: sets.New(model.ConfigKey{
					Kind:      kind.Service,
					Name:      "nginx.app.svc.cluster.local",
					Namespace: "app",
				}),
			}

			server.Push(req)

			queuedCon, queuedReq, shutdown := server.pushQueue.Dequeue()
			if shutdown {
				t.Fatal("push queue unexpectedly shut down")
			}
			if queuedCon != con || queuedReq != req {
				t.Fatalf("queued connection/request = %p/%p, want %p/%p", queuedCon, queuedReq, con, req)
			}
			if req.Push == nil || req.Start.IsZero() {
				t.Fatalf("Push() left request without context or start time: %+v", req)
			}
			if notified == nil || notified.Full != full || !notified.ConfigsUpdated.Equals(req.ConfigsUpdated) {
				t.Fatalf("configuration notification = %+v, want change for %+v", notified, req)
			}
			if full && req.Push.PushVersion == "test-version" {
				t.Fatalf("full Push() reused old push version %q", req.Push.PushVersion)
			}
			if !full && req.Push != server.globalPushContext() {
				t.Fatal("incremental Push() did not reuse the global push context")
			}
		})
	}
}

func TestPushStatusJSONIncludesOperationalSummary(t *testing.T) {
	env := model.NewEnvironment()
	push := model.NewPushContext()
	push.PushVersion = "v1"
	env.SetPushContext(push)

	server := NewDiscoveryServer(env, nil)
	con := &Connection{proxy: inherentConnectionProxy("svc-7.app.svc.cluster.local")}
	server.adsClients["inherent-1"] = con
	server.InboundUpdates.Store(3)
	server.CommittedUpdates.Store(2)
	server.pushQueue.Enqueue(con, &model.PushRequest{Push: push})

	data, err := server.pushStatusJSON(push)
	if err != nil {
		t.Fatalf("pushStatusJSON() failed: %v", err)
	}
	if string(data) == "null" || string(data) == "{}" {
		t.Fatalf("pushStatusJSON() = %s, want populated object", string(data))
	}

	var got pushStatusReport
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("pushStatusJSON() produced invalid JSON: %v", err)
	}
	if got.PushVersion != "v1" {
		t.Fatalf("PushVersion = %q, want v1", got.PushVersion)
	}
	if got.Clients.ConnectedEndpoints != 1 || got.Clients.InherentTriple != 1 || got.Clients.InherentTripleEDSWatchers != 1 {
		t.Fatalf("Clients = %+v, want one connected inherent EDS watcher", got.Clients)
	}
	if got.Queue.Pending != 1 || got.Queue.Queued != 1 {
		t.Fatalf("Queue = %+v, want one pending queued push", got.Queue)
	}
	if got.Updates.Inbound != 3 || got.Updates.Committed != 2 {
		t.Fatalf("Updates = %+v, want inbound=3 committed=2", got.Updates)
	}
	if string(got.ProxyStatus) != "{}" {
		t.Fatalf("ProxyStatus = %s, want {}", string(got.ProxyStatus))
	}
}

func TestEDSUpdateServiceAccountOrderingRegeneratesCDS(t *testing.T) {
	const (
		hostName    = "payment.app.svc.cluster.local"
		clusterName = "outbound|8080||" + hostName
		primarySAN  = "spiffe://cluster.local/ns/app/sa/payment"
		canarySAN   = "spiffe://cluster.local/ns/app/sa/payment-canary"
	)

	store := memory.Make(collections.DubboGatewayAPI())
	if _, err := store.Create(config.Config{
		Meta: config.Meta{
			GroupVersionKind: gvk.PeerAuthentication,
			Name:             "strict",
			Namespace:        "app",
			Domain:           "cluster.local",
		},
		Spec: &securityv1alpha3.PeerAuthentication{
			Mtls: &securityv1alpha3.PeerAuthentication_MutualTLS{
				Mode: securityv1alpha3.PeerAuthentication_MutualTLS_STRICT,
			},
		},
	}); err != nil {
		t.Fatalf("create PeerAuthentication: %v", err)
	}

	svc := &model.Service{
		Hostname: host.Name(hostName),
		Ports: model.PortList{{
			Name:     "http",
			Port:     8080,
			Protocol: protocol.HTTP2,
		}},
		Attributes: model.ServiceAttributes{
			Name:      "payment",
			Namespace: "app",
		},
	}

	env := model.NewEnvironment()
	env.ConfigStore = store
	env.ServiceDiscovery = staticADSServiceDiscovery{services: []*model.Service{svc}}
	env.Watcher = meshwatcher.ConfigAdapter(krt.NewStatic(&meshwatcher.MeshConfigResource{
		MeshConfig: mesh.DefaultMeshConfig(),
	}, true))
	env.Init()

	initialPush := model.NewPushContext()
	initialPush.InitContext(env, nil, nil)
	env.SetPushContext(initialPush)

	server := NewDiscoveryServer(env, nil)
	server.Authorize = allowTestWorkload
	server.Generators = map[string]model.XdsResourceGenerator{
		"triple/" + v1.ClusterType: &triplegen.TripleConfigGenerator{},
	}

	stream := newFakeADSStream()
	con := newConnection("127.0.0.1:26010", stream)
	con.s = server
	con.SetID("inherent-payment-caller")
	con.MarkInitialized()
	con.proxy = &model.Proxy{
		ID:              "caller.app",
		Type:            model.Inherent,
		DNSDomain:       "app.svc.cluster.local",
		ConfigNamespace: "app",
		Metadata:        &model.NodeMetadata{Generator: "triple", Namespace: "app"},
		WatchedResources: map[string]*model.WatchedResource{
			v1.ClusterType: {
				TypeUrl:       v1.ClusterType,
				ResourceNames: sets.New(clusterName),
			},
		},
		LastPushContext: initialPush,
		LastPushTime:    time.Now(),
	}
	server.addCon(con.ID(), con)

	decodeSingleClusterSANs := func(resp *discovery.DiscoveryResponse) (bool, []string) {
		t.Helper()
		if resp.TypeUrl != v1.ClusterType || len(resp.Resources) != 1 {
			t.Fatalf("unexpected CDS response type=%s resources=%d", resp.TypeUrl, len(resp.Resources))
		}
		generated := &cluster.Cluster{}
		if err := resp.Resources[0].UnmarshalTo(generated); err != nil {
			t.Fatalf("unmarshal cluster: %v", err)
		}
		if generated.GetTransportSocket() == nil {
			return false, nil
		}
		tlsContext := &tlsv1.UpstreamTlsContext{}
		if err := generated.GetTransportSocket().GetTypedConfig().UnmarshalTo(tlsContext); err != nil {
			t.Fatalf("unmarshal upstream TLS context: %v", err)
		}
		return true, tlsContext.GetCommonTlsContext().
			GetCombinedValidationContext().
			GetDefaultValidationContext().
			GetMatchSubjectAltNames()
	}

	applyNextFullEDSUpdate := func(endpoints []*model.DubboEndpoint) (bool, []string) {
		t.Helper()
		server.EDSUpdate(model.ShardKey{}, hostName, "app", endpoints)
		var req *model.PushRequest
		select {
		case req = <-server.pushChannel:
		case <-time.After(time.Second):
			t.Fatal("timed out waiting for PushRequest from EDSUpdate")
		}
		if !req.Full {
			t.Fatalf("EDSUpdate triggered req.Full = false, want true")
		}
		if !req.Reason.Has(model.EndpointUpdate) {
			t.Fatalf("EDSUpdate reason = %v, want EndpointUpdate", req.Reason)
		}
		server.Push(req)
		queuedCon, queuedReq, shutdown := server.pushQueue.Dequeue()
		if shutdown {
			t.Fatal("push queue unexpectedly shut down")
		}
		if err := server.pushConnection(queuedCon, &Event{pushRequest: queuedReq}); err != nil {
			t.Fatalf("pushConnection() failed: %v", err)
		}
		server.pushQueue.MarkDone(queuedCon)
		responses := stream.takeAllResponses(t, 1)
		return decodeSingleClusterSANs(responses[0])
	}

	// 1. Service arrived before endpoints: initial CDS omits mTLS because no SA exists yet.
	if err := server.pushConnection(con, &Event{pushRequest: &model.PushRequest{Full: true, Push: initialPush}}); err != nil {
		t.Fatalf("initial pushConnection() failed: %v", err)
	}
	hasMTLS, sans := decodeSingleClusterSANs(stream.takeAllResponses(t, 1)[0])
	if hasMTLS || len(sans) != 0 {
		t.Fatalf("initial CDS before endpoints had hasMTLS=%v sans=%v, want false/empty", hasMTLS, sans)
	}

	// 2. Endpoints arrive later: full push triggers and regenerated CDS carries mTLS + both SANs.
	hasMTLS, sans = applyNextFullEDSUpdate([]*model.DubboEndpoint{
		{
			Addresses:       []string{"10.0.0.10"},
			EndpointPort:    8080,
			ServicePortName: "http",
			ServiceAccount:  primarySAN,
			HealthStatus:    model.Healthy,
		},
		{
			Addresses:       []string{"10.0.0.11"},
			EndpointPort:    8080,
			ServicePortName: "http",
			ServiceAccount:  canarySAN,
			HealthStatus:    model.Healthy,
		},
	})
	if !hasMTLS || !slices.Equal(sans, []string{primarySAN, canarySAN}) {
		t.Fatalf("CDS after endpoints arrived: hasMTLS=%v sans=%v, want true [%s %s]", hasMTLS, sans, primarySAN, canarySAN)
	}

	// 3. Remove last endpoint using canarySAN: full push triggers and canarySAN disappears from CDS.
	hasMTLS, sans = applyNextFullEDSUpdate([]*model.DubboEndpoint{{
		Addresses:       []string{"10.0.0.10"},
		EndpointPort:    8080,
		ServicePortName: "http",
		ServiceAccount:  primarySAN,
		HealthStatus:    model.Healthy,
	}})
	if !hasMTLS || !slices.Equal(sans, []string{primarySAN}) {
		t.Fatalf("CDS after removing last canary endpoint: hasMTLS=%v sans=%v, want true [%s]", hasMTLS, sans, primarySAN)
	}

	// 4. Delete remaining endpoint via nil slice: full push clears SANs and omits mTLS.
	hasMTLS, sans = applyNextFullEDSUpdate(nil)
	if hasMTLS || len(sans) != 0 {
		t.Fatalf("CDS after deleting final endpoint: hasMTLS=%v sans=%v, want false/empty", hasMTLS, sans)
	}
}

func BenchmarkClientsForPushInherentTargeted10k(b *testing.B) {
	server, req := newInherentTargetedPushServer(10000, 100, "svc-7.app.svc.cluster.local")
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		clients := server.clientsForPush(req)
		if len(clients) != 100 {
			b.Fatalf("clientsForPush() returned %d clients, want 100", len(clients))
		}
	}
}

func BenchmarkClientsForPushInherentTargetedScale(b *testing.B) {
	for _, clientCount := range []int{10000, 50000, 100000} {
		b.Run(fmt.Sprintf("%d-clients", clientCount), func(b *testing.B) {
			server, req := newInherentTargetedPushServer(clientCount, 100, "svc-7.app.svc.cluster.local")
			want := clientCount / 100
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				clients := server.clientsForPush(req)
				if len(clients) != want {
					b.Fatalf("clientsForPush() returned %d clients, want %d", len(clients), want)
				}
			}
		})
	}
}

func newInherentTargetedPushServer(clientCount, serviceCount int, targetService string) (*DiscoveryServer, *model.PushRequest) {
	server := &DiscoveryServer{adsClients: map[string]*Connection{}}
	for i := 0; i < clientCount; i++ {
		service := fmt.Sprintf("svc-%d.app.svc.cluster.local", i%serviceCount)
		con := &Connection{proxy: inherentConnectionProxy(service)}
		con.SetID(fmt.Sprintf("inherent-%d", i))
		server.adsClients[con.ID()] = con
	}
	req := &model.PushRequest{
		ConfigsUpdated: sets.New(model.ConfigKey{
			Kind:      kind.Service,
			Name:      targetService,
			Namespace: "app",
		}),
	}
	return server, req
}

func inherentConnectionProxy(service string) *model.Proxy {
	return &model.Proxy{
		Type: model.Inherent,
		Metadata: &model.NodeMetadata{
			Generator: "triple",
		},
		WatchedResources: map[string]*model.WatchedResource{
			v1.EndpointType: {
				TypeUrl:       v1.EndpointType,
				ResourceNames: sets.New("outbound|80||" + service),
			},
		},
	}
}

type staticADSServiceDiscovery struct {
	services []*model.Service
}

func (s staticADSServiceDiscovery) Services() []*model.Service {
	return s.services
}

func (s staticADSServiceDiscovery) GetService(hostname host.Name) *model.Service {
	for _, svc := range s.services {
		if svc.Hostname == hostname {
			return svc
		}
	}
	return nil
}

func (s staticADSServiceDiscovery) GetProxyServiceTargets(*model.Proxy) []model.ServiceTarget {
	return nil
}
