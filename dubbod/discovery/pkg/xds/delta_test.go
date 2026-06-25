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

package xds

import (
	"context"
	"io"
	"testing"

	"github.com/apache/dubbo-kubernetes/dubbod/discovery/pkg/model"
	"github.com/apache/dubbo-kubernetes/dubbod/discovery/pkg/util/protoconv"
	v1 "github.com/apache/dubbo-kubernetes/dubbod/discovery/pkg/xds/v1"
	"github.com/apache/dubbo-kubernetes/pkg/util/sets"
	cluster "github.com/kdubbo/xds-api/cluster/v1"
	discovery "github.com/kdubbo/xds-api/service/discovery/v1"
	"google.golang.org/grpc/metadata"
)

func TestPushDeltaXdsFiltersSubscribedResources(t *testing.T) {
	oldName := "outbound|80||old.app.svc.cluster.local"
	newName := "outbound|80||new.app.svc.cluster.local"
	server, con, stream := newDeltaXDSTestServer(filteredDeltaGenerator{})
	con.proxy.NewWatchedResource(v1.ClusterType, []string{oldName, newName})

	err := server.pushDeltaXds(con, con.proxy.GetWatchedResource(v1.ClusterType), &model.PushRequest{
		Push: con.proxy.LastPushContext,
		Delta: model.ResourceDelta{
			Subscribed: sets.New(newName),
		},
	})
	if err != nil {
		t.Fatalf("pushDeltaXds() failed: %v", err)
	}

	resp := stream.takeDeltaResponse(t)
	if got := resourceNames(resp.Resources); len(got) != 1 || got[0] != newName {
		t.Fatalf("resources = %v, want [%s]", got, newName)
	}
	assertWatchedNames(t, con.proxy.GetWatchedResource(v1.ClusterType), newName)
	if con.proxy.GetWatchedResource(v1.ClusterType).NonceSent == "" {
		t.Fatal("NonceSent was not recorded")
	}
}

func TestPushDeltaXdsAppliesRemovedResourcesToWatchedState(t *testing.T) {
	removedName := "outbound|80||removed.app.svc.cluster.local"
	keptName := "outbound|80||kept.app.svc.cluster.local"
	addedName := "outbound|80||added.app.svc.cluster.local"
	server, con, stream := newDeltaXDSTestServer(staticDeltaGenerator{
		resources: model.Resources{clusterResource(addedName)},
		removed:   []string{removedName},
	})
	con.proxy.NewWatchedResource(v1.ClusterType, []string{removedName, keptName})

	err := server.pushDeltaXds(con, con.proxy.GetWatchedResource(v1.ClusterType), &model.PushRequest{
		Push: con.proxy.LastPushContext,
	})
	if err != nil {
		t.Fatalf("pushDeltaXds() failed: %v", err)
	}

	resp := stream.takeDeltaResponse(t)
	if got := resp.RemovedResources; len(got) != 1 || got[0] != removedName {
		t.Fatalf("removed = %v, want [%s]", got, removedName)
	}
	assertWatchedNames(t, con.proxy.GetWatchedResource(v1.ClusterType), keptName, addedName)
}

func newDeltaXDSTestServer(generator model.XdsResourceGenerator) (*DiscoveryServer, *Connection, *fakeDeltaADSStream) {
	push := model.NewPushContext()
	push.PushVersion = "test-version"
	server := NewDiscoveryServer(model.NewEnvironment(), nil, nil)
	server.Generators = map[string]model.XdsResourceGenerator{
		v1.ClusterType: generator,
	}
	stream := newFakeDeltaADSStream()
	con := newDeltaConnection("127.0.0.1:26010", stream)
	con.s = server
	con.proxy = &model.Proxy{
		ID:               "pod-1",
		Type:             model.Proxyless,
		Metadata:         &model.NodeMetadata{Generator: "grpc", Namespace: "app"},
		WatchedResources: map[string]*model.WatchedResource{},
		LastPushContext:  push,
	}
	return server, con, stream
}

type filteredDeltaGenerator struct{}

func (filteredDeltaGenerator) Generate(_ *model.Proxy, w *model.WatchedResource, _ *model.PushRequest) (model.Resources, model.XdsLogDetails, error) {
	resources := make(model.Resources, 0, len(w.ResourceNames))
	for name := range w.ResourceNames {
		resources = append(resources, clusterResource(name))
	}
	return resources, model.DefaultXdsLogDetails, nil
}

type staticDeltaGenerator struct {
	resources model.Resources
	removed   []string
}

func (g staticDeltaGenerator) Generate(*model.Proxy, *model.WatchedResource, *model.PushRequest) (model.Resources, model.XdsLogDetails, error) {
	return g.resources, model.DefaultXdsLogDetails, nil
}

func (g staticDeltaGenerator) GenerateDeltas(*model.Proxy, *model.PushRequest, *model.WatchedResource) (model.Resources, model.DeletedResources, model.XdsLogDetails, bool, error) {
	return g.resources, g.removed, model.DefaultXdsLogDetails, true, nil
}

func clusterResource(name string) *discovery.Resource {
	return &discovery.Resource{
		Name:     name,
		Resource: protoconv.MessageToAny(&cluster.Cluster{Name: name}),
	}
}

func resourceNames(resources model.Resources) []string {
	out := make([]string, 0, len(resources))
	for _, resource := range resources {
		out = append(out, resource.Name)
	}
	return out
}

type fakeDeltaADSStream struct {
	ctx    context.Context
	recvCh chan *discovery.DeltaDiscoveryRequest
	sendCh chan *discovery.DeltaDiscoveryResponse
}

func newFakeDeltaADSStream() *fakeDeltaADSStream {
	return &fakeDeltaADSStream{
		ctx:    context.Background(),
		recvCh: make(chan *discovery.DeltaDiscoveryRequest, 8),
		sendCh: make(chan *discovery.DeltaDiscoveryResponse, 8),
	}
}

func (f *fakeDeltaADSStream) Send(resp *discovery.DeltaDiscoveryResponse) error {
	f.sendCh <- resp
	return nil
}

func (f *fakeDeltaADSStream) Recv() (*discovery.DeltaDiscoveryRequest, error) {
	req, ok := <-f.recvCh
	if !ok {
		return nil, io.EOF
	}
	return req, nil
}

func (f *fakeDeltaADSStream) SetHeader(metadata.MD) error {
	return nil
}

func (f *fakeDeltaADSStream) SendHeader(metadata.MD) error {
	return nil
}

func (f *fakeDeltaADSStream) SetTrailer(metadata.MD) {}

func (f *fakeDeltaADSStream) Context() context.Context {
	return f.ctx
}

func (f *fakeDeltaADSStream) SendMsg(any) error {
	return nil
}

func (f *fakeDeltaADSStream) RecvMsg(any) error {
	return nil
}

func (f *fakeDeltaADSStream) takeDeltaResponse(t *testing.T) *discovery.DeltaDiscoveryResponse {
	t.Helper()
	select {
	case resp := <-f.sendCh:
		return resp
	default:
		t.Fatal("expected delta response")
		return nil
	}
}
