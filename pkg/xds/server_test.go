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
	"testing"

	"github.com/apache/dubbo-kubernetes/pkg/model"
	"github.com/apache/dubbo-kubernetes/pkg/util/sets"
	discovery "github.com/kdubbo/xds-api/service/discovery/v1"
)

type testWatcher struct {
	resources map[string]*WatchedResource
}

func newTestWatcher() *testWatcher {
	return &testWatcher{resources: map[string]*WatchedResource{}}
}

func (w *testWatcher) DeleteWatchedResource(url string) {
	delete(w.resources, url)
}

func (w *testWatcher) GetWatchedResource(url string) *WatchedResource {
	return w.resources[url]
}

func (w *testWatcher) NewWatchedResource(url string, names []string) {
	w.resources[url] = &WatchedResource{TypeUrl: url, ResourceNames: sets.New(names...)}
}

func (w *testWatcher) UpdateWatchedResource(url string, updateFn func(*WatchedResource) *WatchedResource) {
	wr := updateFn(w.resources[url])
	if wr == nil {
		delete(w.resources, url)
		return
	}
	w.resources[url] = wr
}

func (w *testWatcher) GetID() string {
	return "test"
}

func TestShouldRespondPreservesNonceAckWithoutResourceNames(t *testing.T) {
	watcher := newTestWatcher()
	routeName := "outbound|80||nginx.app.svc.cluster.local"

	shouldRespond, _ := ShouldRespond(watcher, "proxyless", &discovery.DiscoveryRequest{
		TypeUrl:       model.RouteType,
		ResourceNames: []string{routeName},
	})
	if !shouldRespond {
		t.Fatalf("initial RDS request should respond")
	}

	watcher.UpdateWatchedResource(model.RouteType, func(wr *WatchedResource) *WatchedResource {
		wr.NonceSent = "nonce-1"
		return wr
	})

	shouldRespond, _ = ShouldRespond(watcher, "proxyless", &discovery.DiscoveryRequest{
		TypeUrl:       model.RouteType,
		VersionInfo:   "version-1",
		ResponseNonce: "nonce-1",
	})
	if shouldRespond {
		t.Fatalf("nonce-matching ACK without resource names should not trigger a response")
	}

	wr := watcher.GetWatchedResource(model.RouteType)
	if wr == nil {
		t.Fatalf("RDS watch was deleted")
	}
	if !wr.ResourceNames.Contains(routeName) {
		t.Fatalf("RDS watch lost route %q: %v", routeName, wr.ResourceNames.UnsortedList())
	}
	if wr.NonceAcked != "nonce-1" {
		t.Fatalf("NonceAcked = %q, want nonce-1", wr.NonceAcked)
	}
}

func TestShouldRespondStillDeletesExplicitUnsubscribe(t *testing.T) {
	watcher := newTestWatcher()
	watcher.NewWatchedResource(model.RouteType, []string{"outbound|80||nginx.app.svc.cluster.local"})

	shouldRespond, _ := ShouldRespond(watcher, "proxyless", &discovery.DiscoveryRequest{
		TypeUrl: model.RouteType,
	})
	if shouldRespond {
		t.Fatalf("unsubscribe should not trigger a response")
	}
	if wr := watcher.GetWatchedResource(model.RouteType); wr != nil {
		t.Fatalf("RDS watch still present after unsubscribe: %v", wr.ResourceNames.UnsortedList())
	}
}
