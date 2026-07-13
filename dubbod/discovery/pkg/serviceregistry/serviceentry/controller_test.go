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

package serviceentry

import (
	"testing"

	"github.com/apache/dubbo-kubernetes/dubbod/discovery/pkg/config/memory"
	"github.com/apache/dubbo-kubernetes/dubbod/discovery/pkg/model"
	"github.com/apache/dubbo-kubernetes/pkg/cluster"
	"github.com/apache/dubbo-kubernetes/pkg/config"
	"github.com/apache/dubbo-kubernetes/pkg/config/schema/collections"
	"github.com/apache/dubbo-kubernetes/pkg/config/schema/gvk"
	networking "github.com/kdubbo/api/networking/v1alpha3"
	typev1alpha3 "github.com/kdubbo/api/type/v1alpha3"
)

func TestControllerReconcilesServiceEntryAndWorkloadEntry(t *testing.T) {
	configs := newFakeConfigController()
	updater := &fakeXDSUpdater{}
	controller := NewController(Options{
		ConfigController: configs,
		XDSUpdater:       updater,
		ClusterID:        cluster.ID("cluster-1"),
	})

	workload := config.Config{Meta: config.Meta{
		GroupVersionKind: gvk.WorkloadEntry,
		Name:             "reviews-vm",
		Namespace:        "bookinfo",
	}, Spec: &networking.WorkloadEntry{
		Address:        "10.0.0.10",
		Ports:          map[string]uint32{"grpc": 16000},
		Labels:         map[string]string{"app": "reviews"},
		ServiceAccount: "reviews",
	}}
	configs.create(t, workload)
	entry := config.Config{Meta: config.Meta{
		GroupVersionKind: gvk.ServiceEntry,
		Name:             "reviews-external",
		Namespace:        "bookinfo",
	}, Spec: &networking.ServiceEntry{
		Hosts:      []string{"reviews.example.com"},
		Addresses:  []string{"240.0.0.1"},
		Location:   networking.ServiceEntry_MESH_INTERNAL,
		Resolution: networking.ServiceEntry_STATIC,
		Ports: []*networking.ServicePort{{
			Name: "grpc", Number: 50051, Protocol: "GRPC", TargetPort: 15000,
		}},
		WorkloadSelector: &typev1alpha3.WorkloadSelector{MatchLabels: map[string]string{"app": "reviews"}},
	}}
	configs.create(t, entry)

	service := controller.GetService("reviews.example.com")
	if service == nil {
		t.Fatal("service was not created")
	}
	if service.DefaultAddress != "240.0.0.1" || service.Resolution != model.ClientSideLB || service.MeshExternal {
		t.Fatalf("unexpected service: address=%s resolution=%v external=%v", service.DefaultAddress, service.Resolution, service.MeshExternal)
	}
	last := updater.lastEDS(t)
	if len(last.endpoints) != 1 {
		t.Fatalf("got %d endpoints, want 1", len(last.endpoints))
	}
	endpoint := last.endpoints[0]
	if endpoint.FirstAddressOrNil() != "10.0.0.10" || endpoint.EndpointPort != 16000 || endpoint.ServiceAccount != "reviews" {
		t.Fatalf("unexpected endpoint: address=%s port=%d serviceAccount=%s", endpoint.FirstAddressOrNil(), endpoint.EndpointPort, endpoint.ServiceAccount)
	}

	workload.Spec.(*networking.WorkloadEntry).Address = "10.0.0.11"
	configs.update(t, workload)
	if got := updater.lastEDS(t).endpoints[0].FirstAddressOrNil(); got != "10.0.0.11" {
		t.Fatalf("updated endpoint address = %s, want 10.0.0.11", got)
	}
	if got := len(updater.serviceEvents); got != 1 {
		t.Fatalf("endpoint-only update emitted %d service events, want 1 total", got)
	}
	edsUpdates := len(updater.edsEvents)
	configs.create(t, config.Config{Meta: config.Meta{
		GroupVersionKind: gvk.WorkloadEntry,
		Name:             "unrelated",
		Namespace:        "bookinfo",
	}, Spec: &networking.WorkloadEntry{Address: "10.0.0.20", Labels: map[string]string{"app": "other"}}})
	if got := len(updater.edsEvents); got != edsUpdates {
		t.Fatalf("unrelated workload emitted an EDS update: got %d updates, want %d", got, edsUpdates)
	}

	configs.delete(t, gvk.ServiceEntry, entry.Name, entry.Namespace)
	if controller.GetService("reviews.example.com") != nil {
		t.Fatal("service was not deleted")
	}
	if got := updater.serviceEvents[len(updater.serviceEvents)-1].event; got != model.EventDelete {
		t.Fatalf("last service event = %v, want delete", got)
	}
	if got := len(updater.lastEDS(t).endpoints); got != 0 {
		t.Fatalf("deleted service has %d endpoints, want 0", got)
	}
}

type fakeConfigController struct {
	model.ConfigStore
	handlers map[config.GroupVersionKind][]model.EventHandler
}

func newFakeConfigController() *fakeConfigController {
	return &fakeConfigController{
		ConfigStore: memory.Make(collections.Dubbo),
		handlers:    make(map[config.GroupVersionKind][]model.EventHandler),
	}
}

func (f *fakeConfigController) RegisterEventHandler(kind config.GroupVersionKind, handler model.EventHandler) {
	f.handlers[kind] = append(f.handlers[kind], handler)
}

func (f *fakeConfigController) Run(stop <-chan struct{}) { <-stop }
func (f *fakeConfigController) HasSynced() bool          { return true }

func (f *fakeConfigController) create(t *testing.T, cfg config.Config) {
	t.Helper()
	if _, err := f.ConfigStore.Create(cfg); err != nil {
		t.Fatal(err)
	}
	current := *f.Get(cfg.GroupVersionKind, cfg.Name, cfg.Namespace)
	for _, handler := range f.handlers[cfg.GroupVersionKind] {
		handler(config.Config{}, current, model.EventAdd)
	}
}

func (f *fakeConfigController) update(t *testing.T, cfg config.Config) {
	t.Helper()
	old := *f.Get(cfg.GroupVersionKind, cfg.Name, cfg.Namespace)
	cfg.ResourceVersion = ""
	if _, err := f.ConfigStore.Update(cfg); err != nil {
		t.Fatal(err)
	}
	current := *f.Get(cfg.GroupVersionKind, cfg.Name, cfg.Namespace)
	for _, handler := range f.handlers[cfg.GroupVersionKind] {
		handler(old, current, model.EventUpdate)
	}
}

func (f *fakeConfigController) delete(t *testing.T, kind config.GroupVersionKind, name, namespace string) {
	t.Helper()
	old := *f.Get(kind, name, namespace)
	if err := f.ConfigStore.Delete(kind, name, namespace, nil); err != nil {
		t.Fatal(err)
	}
	for _, handler := range f.handlers[kind] {
		handler(old, config.Config{}, model.EventDelete)
	}
}

type serviceEvent struct {
	event model.Event
}

type edsEvent struct {
	endpoints []*model.DubboEndpoint
}

type fakeXDSUpdater struct {
	serviceEvents []serviceEvent
	edsEvents     []edsEvent
}

func (*fakeXDSUpdater) ConfigUpdate(*model.PushRequest) {}
func (f *fakeXDSUpdater) ServiceUpdate(_ model.ShardKey, _, _ string, event model.Event) {
	f.serviceEvents = append(f.serviceEvents, serviceEvent{event: event})
}
func (f *fakeXDSUpdater) EDSUpdate(_ model.ShardKey, _, _ string, endpoints []*model.DubboEndpoint) {
	f.edsEvents = append(f.edsEvents, edsEvent{endpoints: endpoints})
}
func (*fakeXDSUpdater) EDSCacheUpdate(model.ShardKey, string, string, []*model.DubboEndpoint) {}
func (*fakeXDSUpdater) ProxyUpdate(cluster.ID, string)                                        {}

func (f *fakeXDSUpdater) lastEDS(t *testing.T) edsEvent {
	t.Helper()
	if len(f.edsEvents) == 0 {
		t.Fatal("no EDS update received")
	}
	return f.edsEvents[len(f.edsEvents)-1]
}
