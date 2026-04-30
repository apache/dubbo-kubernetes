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
	"context"
	"io"
	"testing"
	"time"

	"github.com/apache/dubbo-kubernetes/dubbod/discovery/pkg/model"
	"github.com/apache/dubbo-kubernetes/dubbod/discovery/pkg/util/protoconv"
	v1 "github.com/apache/dubbo-kubernetes/dubbod/discovery/pkg/xds/v1"
	"github.com/apache/dubbo-kubernetes/pkg/config"
	"github.com/apache/dubbo-kubernetes/pkg/config/host"
	"github.com/apache/dubbo-kubernetes/pkg/config/mesh/meshwatcher"
	"github.com/apache/dubbo-kubernetes/pkg/config/schema/collection"
	"github.com/apache/dubbo-kubernetes/pkg/kube/krt"
	"github.com/apache/dubbo-kubernetes/pkg/util/sets"
	meshv1alpha1 "github.com/kdubbo/api/mesh/v1alpha1"
	cluster "github.com/kdubbo/xds-api/cluster/v1"
	core "github.com/kdubbo/xds-api/core/v1"
	hcmv1 "github.com/kdubbo/xds-api/extensions/filters/v1/network/http_connection_manager"
	listener "github.com/kdubbo/xds-api/listener/v1"
	route "github.com/kdubbo/xds-api/route/v1"
	discovery "github.com/kdubbo/xds-api/service/discovery/v1"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/types/known/structpb"
)

const (
	testListenerName = "nginx.app.svc.cluster.local:8080"
	testClusterName  = "outbound|8080||nginx.app.svc.cluster.local"
	testNodeID       = "proxyless~10.0.0.1~pod-1~app.svc.cluster.local"
)

func TestPushXdsProxylessLDSCreatesDependentWatches(t *testing.T) {
	server, con, stream := newProxylessXDSTestServer(t)

	req := &model.PushRequest{
		Full:   true,
		Push:   con.proxy.LastPushContext,
		Reason: model.NewReasonStats(model.ProxyRequest),
		Start:  time.Now(),
	}

	if err := server.pushXds(con, con.proxy.GetWatchedResource(v1.ListenerType), req); err != nil {
		t.Fatalf("pushXds() failed: %v", err)
	}

	responses := stream.takeAllResponses(t, 3)
	assertResponseTypeOrder(t, responses, v1.ListenerType, v1.ClusterType, v1.RouteType)
	assertWatchedNames(t, con.proxy.GetWatchedResource(v1.ClusterType), testClusterName)
	assertWatchedNames(t, con.proxy.GetWatchedResource(v1.RouteType), testClusterName)
	assertWatchedNames(t, con.proxy.GetWatchedResource(v1.EndpointType), testClusterName)
	stream.assertNoResponse(t)
}

func TestStreamProxylessWildcardAckDoesNotRepushOrPrematurelySendEDS(t *testing.T) {
	server, _, stream := newProxylessXDSTestServer(t)
	server.serverReady.Store(true)

	errCh := make(chan error, 1)
	go func() {
		errCh <- server.Stream(stream)
	}()

	stream.recvCh <- &discovery.DiscoveryRequest{
		TypeUrl:       v1.ListenerType,
		ResourceNames: []string{testListenerName},
		Node:          testProxylessNode(t),
	}

	responses := stream.takeAllResponses(t, 3)
	assertResponseTypeOrder(t, responses, v1.ListenerType, v1.ClusterType, v1.RouteType)
	stream.assertNoResponse(t)

	con := waitForSingleADSClient(t, server)
	assertWatchedNames(t, con.proxy.GetWatchedResource(v1.ClusterType), testClusterName)
	assertWatchedNames(t, con.proxy.GetWatchedResource(v1.RouteType), testClusterName)
	assertWatchedNames(t, con.proxy.GetWatchedResource(v1.EndpointType), testClusterName)

	stream.recvCh <- &discovery.DiscoveryRequest{
		TypeUrl:       v1.ListenerType,
		VersionInfo:   responses[0].VersionInfo,
		ResponseNonce: responses[0].Nonce,
	}
	stream.assertNoResponse(t)

	close(stream.recvCh)
	select {
	case err := <-errCh:
		if err != nil {
			t.Fatalf("Stream() failed: %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("Stream() did not exit after client close")
	}
}

func newProxylessXDSTestServer(t *testing.T) (*DiscoveryServer, *Connection, *fakeADSStream) {
	t.Helper()

	env := model.NewEnvironment()
	env.ServiceDiscovery = testServiceDiscovery{}
	env.ConfigStore = testConfigStore{}
	env.Watcher = meshwatcher.ConfigAdapter(krt.NewStatic(&meshwatcher.MeshGlobalConfigResource{
		MeshGlobalConfig: &meshv1alpha1.MeshGlobalConfig{
			DefaultConfig: &meshv1alpha1.ProxyConfig{
				DiscoveryAddress: "127.0.0.1:15010",
			},
			RootNamespace: "dubbo-system",
		},
	}, true))
	env.Init()

	push := model.NewPushContext()
	push.PushVersion = "test-version"
	push.InitContext(env, nil, nil)
	env.SetPushContext(push)

	server := NewDiscoveryServer(env, nil, nil)
	server.Generators = map[string]model.XdsResourceGenerator{
		v1.ListenerType: staticResourceGenerator{resources: proxylessListenerResources()},
		v1.ClusterType:  staticResourceGenerator{resources: proxylessClusterResources()},
		v1.RouteType:    staticResourceGenerator{resources: proxylessRouteResources()},
	}

	stream := newFakeADSStream()
	con := newConnection("127.0.0.1:15010", stream)
	con.s = server
	con.proxy = &model.Proxy{
		ID:        "pod-1",
		Type:      model.Proxyless,
		DNSDomain: "app.svc.cluster.local",
		Metadata:  &model.NodeMetadata{Generator: "grpc", Namespace: "app"},
		WatchedResources: map[string]*model.WatchedResource{
			v1.ListenerType: {
				TypeUrl:       v1.ListenerType,
				ResourceNames: sets.New(testListenerName),
			},
		},
		LastPushContext: push,
		LastPushTime:    time.Now(),
	}

	return server, con, stream
}

type staticResourceGenerator struct {
	resources model.Resources
}

func (g staticResourceGenerator) Generate(*model.Proxy, *model.WatchedResource, *model.PushRequest) (model.Resources, model.XdsLogDetails, error) {
	return g.resources, model.DefaultXdsLogDetails, nil
}

type fakeADSStream struct {
	ctx      context.Context
	recvCh   chan *discovery.DiscoveryRequest
	sendCh   chan *discovery.DiscoveryResponse
	trailers metadata.MD
}

func newFakeADSStream() *fakeADSStream {
	return &fakeADSStream{
		ctx:    context.Background(),
		recvCh: make(chan *discovery.DiscoveryRequest, 8),
		sendCh: make(chan *discovery.DiscoveryResponse, 8),
	}
}

func (f *fakeADSStream) Send(resp *discovery.DiscoveryResponse) error {
	f.sendCh <- resp
	return nil
}

func (f *fakeADSStream) Recv() (*discovery.DiscoveryRequest, error) {
	req, ok := <-f.recvCh
	if !ok {
		return nil, io.EOF
	}
	return req, nil
}

func (f *fakeADSStream) SetHeader(metadata.MD) error {
	return nil
}

func (f *fakeADSStream) SendHeader(metadata.MD) error {
	return nil
}

func (f *fakeADSStream) SetTrailer(md metadata.MD) {
	f.trailers = md
}

func (f *fakeADSStream) Context() context.Context {
	return f.ctx
}

func (f *fakeADSStream) SendMsg(any) error {
	return nil
}

func (f *fakeADSStream) RecvMsg(any) error {
	return nil
}

func (f *fakeADSStream) takeAllResponses(t *testing.T, want int) []*discovery.DiscoveryResponse {
	t.Helper()
	out := make([]*discovery.DiscoveryResponse, 0, want)
	for len(out) < want {
		select {
		case resp := <-f.sendCh:
			out = append(out, resp)
		case <-time.After(time.Second):
			t.Fatalf("timed out waiting for response %d/%d", len(out)+1, want)
		}
	}
	return out
}

func (f *fakeADSStream) assertNoResponse(t *testing.T) {
	t.Helper()
	select {
	case resp := <-f.sendCh:
		t.Fatalf("unexpected response type %s", resp.TypeUrl)
	case <-time.After(150 * time.Millisecond):
	}
}

type testServiceDiscovery struct{}

func (testServiceDiscovery) Services() []*model.Service {
	return nil
}

func (testServiceDiscovery) GetService(host.Name) *model.Service {
	return nil
}

func (testServiceDiscovery) GetProxyServiceTargets(*model.Proxy) []model.ServiceTarget {
	return nil
}

type testConfigStore struct{}

func (testConfigStore) Schemas() collection.Schemas {
	return collection.Schemas{}
}

func (testConfigStore) Get(config.GroupVersionKind, string, string) *config.Config {
	return nil
}

func (testConfigStore) List(config.GroupVersionKind, string) []config.Config {
	return nil
}

func (testConfigStore) Create(config.Config) (string, error) {
	return "", nil
}

func (testConfigStore) Update(config.Config) (string, error) {
	return "", nil
}

func (testConfigStore) UpdateStatus(config.Config) (string, error) {
	return "", nil
}

func (testConfigStore) Patch(config.Config, config.PatchFunc) (string, error) {
	return "", nil
}

func (testConfigStore) Delete(config.GroupVersionKind, string, string, *string) error {
	return nil
}

func testProxylessNode(t *testing.T) *core.Node {
	t.Helper()
	meta, err := structpb.NewStruct(map[string]any{
		"GENERATOR": "grpc",
		"NAMESPACE": "app",
	})
	if err != nil {
		t.Fatalf("NewStruct() failed: %v", err)
	}
	return &core.Node{
		Id:       testNodeID,
		Metadata: meta,
	}
}

func proxylessListenerResources() model.Resources {
	hcm := &hcmv1.HttpConnectionManager{
		RouteSpecifier: &hcmv1.HttpConnectionManager_Rds{
			Rds: &hcmv1.Rds{
				RouteConfigName: testClusterName,
			},
		},
	}
	return model.Resources{
		{
			Name: testListenerName,
			Resource: protoconv.MessageToAny(&listener.Listener{
				Name: testListenerName,
				ApiListener: &listener.ApiListener{
					ApiListener: protoconv.MessageToAny(hcm),
				},
			}),
		},
	}
}

func proxylessClusterResources() model.Resources {
	return model.Resources{
		{
			Name: testClusterName,
			Resource: protoconv.MessageToAny(&cluster.Cluster{
				Name: testClusterName,
				ClusterDiscoveryType: &cluster.Cluster_Type{
					Type: cluster.Cluster_EDS,
				},
				EdsClusterConfig: &cluster.Cluster_EdsClusterConfig{
					ServiceName: testClusterName,
				},
			}),
		},
	}
}

func proxylessRouteResources() model.Resources {
	return model.Resources{
		{
			Name:     testClusterName,
			Resource: protoconv.MessageToAny(&route.RouteConfiguration{Name: testClusterName}),
		},
	}
}

func waitForSingleADSClient(t *testing.T, server *DiscoveryServer) *Connection {
	t.Helper()
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		clients := server.AllClients()
		if len(clients) == 1 {
			return clients[0]
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("expected 1 ADS client, got %d", len(server.AllClients()))
	return nil
}

func assertResponseTypeOrder(t *testing.T, got []*discovery.DiscoveryResponse, want ...string) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("response count = %d, want %d", len(got), len(want))
	}
	for i, expected := range want {
		if got[i].TypeUrl != expected {
			t.Fatalf("response[%d].TypeUrl = %s, want %s", i, got[i].TypeUrl, expected)
		}
	}
}

func assertWatchedNames(t *testing.T, wr *model.WatchedResource, names ...string) {
	t.Helper()
	if wr == nil {
		t.Fatal("watched resource is nil")
	}
	want := sets.New(names...)
	if wr.ResourceNames == nil || wr.ResourceNames.Len() != want.Len() {
		t.Fatalf("watched names = %v, want %v", resourceNamesOrNil(wr), want.UnsortedList())
	}
	for name := range want {
		if !wr.ResourceNames.Contains(name) {
			t.Fatalf("watched names = %v, want %v", resourceNamesOrNil(wr), want.UnsortedList())
		}
	}
}

func resourceNamesOrNil(wr *model.WatchedResource) []string {
	if wr == nil || wr.ResourceNames == nil {
		return nil
	}
	return wr.ResourceNames.UnsortedList()
}
