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
	"testing"
	"time"

	"github.com/apache/dubbo-kubernetes/dubbod/discovery/pkg/model"
	v1 "github.com/apache/dubbo-kubernetes/dubbod/discovery/pkg/xds/v1"
	"github.com/apache/dubbo-kubernetes/pkg/config/schema/kind"
	"github.com/apache/dubbo-kubernetes/pkg/util/sets"
)

func TestConnectionWatchesAnyServiceFiltersProxylessEDS(t *testing.T) {
	con := &Connection{proxy: &model.Proxy{
		Type: model.Proxyless,
		Metadata: &model.NodeMetadata{
			Generator: "grpc",
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
		t.Fatalf("expected proxyless connection to watch nginx")
	}
	if connectionWatchesAnyService(con, sets.New("api.app.svc.cluster.local")) {
		t.Fatalf("expected proxyless connection not to watch api")
	}
}

func TestConnectionWatchesAnyServiceKeepsBroadPushForUnknownWatch(t *testing.T) {
	con := &Connection{proxy: &model.Proxy{
		Type: model.Proxyless,
		Metadata: &model.NodeMetadata{
			Generator: "grpc",
		},
		WatchedResources: map[string]*model.WatchedResource{},
	}}

	if !connectionWatchesAnyService(con, sets.New("api.app.svc.cluster.local")) {
		t.Fatalf("connection without EDS watch should keep broad push")
	}
}

func TestClientsForPushProxylessLargeScaleTargetsWatchedService(t *testing.T) {
	server, req := newProxylessTargetedPushServer(10000, 100, "svc-7.app.svc.cluster.local")

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

func TestPushConnectionExercisesProxylessPushPath(t *testing.T) {
	server, con, stream := newProxylessXDSTestServer(t)
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
		server, _, stream := newProxylessXDSTestServer(t)
		con := &Connection{}
		if err := server.pushConnection(con, &Event{pushRequest: &model.PushRequest{Full: true}}); err != nil {
			t.Fatalf("pushConnection() failed: %v", err)
		}
		stream.assertNoResponse(t)
	})

	t.Run("nil request", func(t *testing.T) {
		server, con, stream := newProxylessXDSTestServer(t)
		if err := server.pushConnection(con, &Event{}); err != nil {
			t.Fatalf("pushConnection() failed: %v", err)
		}
		stream.assertNoResponse(t)
	})

	t.Run("proxy does not need push", func(t *testing.T) {
		server, con, stream := newProxylessXDSTestServer(t)
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
			server, con, _ := newProxylessXDSTestServer(t)
			server.addCon("proxyless-1", con)

			var notified *model.PushRequest
			server.RuntimeConfigUpdate = func(req *model.PushRequest) {
				notified = req
			}
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
			if notified != req {
				t.Fatalf("RuntimeConfigUpdate received %p, want %p", notified, req)
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

	server := NewDiscoveryServer(env, nil, nil)
	con := &Connection{proxy: proxylessConnectionProxy("svc-7.app.svc.cluster.local")}
	server.adsClients["proxyless-1"] = con
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
	if got.Clients.ConnectedEndpoints != 1 || got.Clients.ProxylessGRPC != 1 || got.Clients.ProxylessGRPCEDSWatchers != 1 {
		t.Fatalf("Clients = %+v, want one connected proxyless EDS watcher", got.Clients)
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

func BenchmarkClientsForPushProxylessTargeted10k(b *testing.B) {
	server, req := newProxylessTargetedPushServer(10000, 100, "svc-7.app.svc.cluster.local")
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		clients := server.clientsForPush(req)
		if len(clients) != 100 {
			b.Fatalf("clientsForPush() returned %d clients, want 100", len(clients))
		}
	}
}

func BenchmarkClientsForPushProxylessTargetedScale(b *testing.B) {
	for _, clientCount := range []int{10000, 50000, 100000} {
		b.Run(fmt.Sprintf("%d-clients", clientCount), func(b *testing.B) {
			server, req := newProxylessTargetedPushServer(clientCount, 100, "svc-7.app.svc.cluster.local")
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

func newProxylessTargetedPushServer(clientCount, serviceCount int, targetService string) (*DiscoveryServer, *model.PushRequest) {
	server := &DiscoveryServer{adsClients: map[string]*Connection{}}
	for i := 0; i < clientCount; i++ {
		service := fmt.Sprintf("svc-%d.app.svc.cluster.local", i%serviceCount)
		con := &Connection{proxy: proxylessConnectionProxy(service)}
		con.SetID(fmt.Sprintf("proxyless-%d", i))
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

func proxylessConnectionProxy(service string) *model.Proxy {
	return &model.Proxy{
		Type: model.Proxyless,
		Metadata: &model.NodeMetadata{
			Generator: "grpc",
		},
		WatchedResources: map[string]*model.WatchedResource{
			v1.EndpointType: {
				TypeUrl:       v1.EndpointType,
				ResourceNames: sets.New("outbound|80||" + service),
			},
		},
	}
}
