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

package model

import (
	"slices"
	"testing"
	"time"

	"github.com/apache/dubbo-kubernetes/dubbod/discovery/pkg/serviceregistry/provider"
	"github.com/apache/dubbo-kubernetes/pkg/config"
	"github.com/apache/dubbo-kubernetes/pkg/config/host"
	"github.com/apache/dubbo-kubernetes/pkg/config/mesh"
	"github.com/apache/dubbo-kubernetes/pkg/config/mesh/meshwatcher"
	"github.com/apache/dubbo-kubernetes/pkg/config/protocol"
	"github.com/apache/dubbo-kubernetes/pkg/config/schema/collection"
	"github.com/apache/dubbo-kubernetes/pkg/config/schema/kind"
	"github.com/apache/dubbo-kubernetes/pkg/kube/krt"
	"github.com/apache/dubbo-kubernetes/pkg/util/sets"
)

type stubConfigStore struct{}

func (stubConfigStore) Schemas() collection.Schemas { return collection.Schemas{} }
func (stubConfigStore) Get(config.GroupVersionKind, string, string) *config.Config {
	return nil
}
func (stubConfigStore) List(config.GroupVersionKind, string) []config.Config {
	return nil
}
func (stubConfigStore) Create(config.Config) (string, error)               { return "", nil }
func (stubConfigStore) Update(config.Config) (string, error)               { return "", nil }
func (stubConfigStore) UpdateStatus(config.Config) (string, error)         { return "", nil }
func (stubConfigStore) Patch(config.Config, config.PatchFunc) (string, error) { return "", nil }
func (stubConfigStore) Delete(config.GroupVersionKind, string, string, *string) error {
	return nil
}

type stubServiceDiscovery struct {
	services []*Service
}

func (s stubServiceDiscovery) Services() []*Service {
	return s.services
}

func (s stubServiceDiscovery) GetService(hostname host.Name) *Service {
	for _, svc := range s.services {
		if svc.Hostname == hostname {
			return svc
		}
	}
	return nil
}

func (s stubServiceDiscovery) GetProxyServiceTargets(*Proxy) []ServiceTarget {
	return nil
}

func TestPushRequestCopyMergePreservesQueuedUpdates(t *testing.T) {
	oldPush := NewPushContext()
	newPush := NewPushContext()
	start := time.Unix(100, 0)

	existing := &PushRequest{
		Reason:           NewReasonStats(EndpointUpdate),
		ConfigsUpdated:   sets.New(ConfigKey{Kind: kind.Service, Name: "nginx.app.svc.cluster.local", Namespace: "app"}),
		AddressesUpdated: sets.New("10.0.0.1"),
		Push:             oldPush,
		Start:            start,
		Delta: ResourceDelta{
			Subscribed: sets.New("outbound|80||nginx.app.svc.cluster.local"),
		},
	}
	incoming := &PushRequest{
		Reason:           NewReasonStats(ConfigUpdate),
		ConfigsUpdated:   sets.New(ConfigKey{Kind: kind.PeerAuthentication, Name: "default", Namespace: "app"}),
		AddressesUpdated: sets.New("10.0.0.2"),
		Full:             true,
		Forced:           true,
		Push:             newPush,
		Start:            start.Add(time.Second),
		Delta: ResourceDelta{
			Unsubscribed: sets.New("outbound|80||old.app.svc.cluster.local"),
		},
	}

	merged := existing.CopyMerge(incoming)
	if merged == existing || merged == incoming {
		t.Fatalf("CopyMerge returned an input request")
	}
	if !merged.Full || !merged.Forced {
		t.Fatalf("Full/Forced = %v/%v, want true/true", merged.Full, merged.Forced)
	}
	if merged.Push != newPush {
		t.Fatalf("Push was not updated to latest context")
	}
	if !merged.Start.Equal(start) {
		t.Fatalf("Start = %v, want %v", merged.Start, start)
	}
	if got := merged.Reason[EndpointUpdate]; got != 1 {
		t.Fatalf("EndpointUpdate reason count = %d, want 1", got)
	}
	if got := merged.Reason[ConfigUpdate]; got != 1 {
		t.Fatalf("ConfigUpdate reason count = %d, want 1", got)
	}
	if len(merged.ConfigsUpdated) != 2 {
		t.Fatalf("ConfigsUpdated = %v, want 2 entries", merged.ConfigsUpdated)
	}
	if !merged.AddressesUpdated.Contains("10.0.0.1") || !merged.AddressesUpdated.Contains("10.0.0.2") {
		t.Fatalf("AddressesUpdated = %v, want both addresses", merged.AddressesUpdated)
	}
	if !merged.Delta.Subscribed.Contains("outbound|80||nginx.app.svc.cluster.local") {
		t.Fatalf("Subscribed delta = %v, want nginx cluster", merged.Delta.Subscribed)
	}
	if !merged.Delta.Unsubscribed.Contains("outbound|80||old.app.svc.cluster.local") {
		t.Fatalf("Unsubscribed delta = %v, want old cluster", merged.Delta.Unsubscribed)
	}

	merged.ConfigsUpdated.Insert(ConfigKey{Kind: kind.Service, Name: "api.app.svc.cluster.local", Namespace: "app"})
	if len(existing.ConfigsUpdated) != 1 {
		t.Fatalf("CopyMerge mutated existing ConfigsUpdated: %v", existing.ConfigsUpdated)
	}
}

func TestPushContextStatusJSONEmptyObject(t *testing.T) {
	data, err := NewPushContext().StatusJSON()
	if err != nil {
		t.Fatalf("StatusJSON() failed: %v", err)
	}
	if string(data) != "{}" {
		t.Fatalf("StatusJSON() = %s, want {}", string(data))
	}
}

func TestConfigSnapshotRecomputesServiceAccountsOnNonServicePush(t *testing.T) {
	const hostName = host.Name("payment.app.svc.cluster.local")
	const wantSAN = "spiffe://cluster.local/ns/app/sa/payment"

	svc := &Service{
		Hostname: hostName,
		Ports:    PortList{{Name: "http", Port: 8080, Protocol: protocol.HTTP2}},
		Attributes: ServiceAttributes{
			Name:      "payment",
			Namespace: "app",
		},
	}

	env := NewEnvironment()
	env.ConfigStore = stubConfigStore{}
	env.ServiceDiscovery = stubServiceDiscovery{services: []*Service{svc}}
	env.Watcher = meshwatcher.ConfigAdapter(krt.NewStatic(&meshwatcher.MeshConfigResource{
		MeshConfig: mesh.DefaultMeshConfig(),
	}, true))
	env.Init()

	initialPush := NewPushContext()
	initialPush.InitContext(env, nil, nil)
	if got := initialPush.ServiceAccounts(hostName, "app"); len(got) != 0 {
		t.Fatalf("initial ServiceAccounts = %v, want empty", got)
	}

	env.EndpointIndex.UpdateServiceEndpoints(ShardKey{}, string(hostName), "app", []*DubboEndpoint{{
		Addresses:       []string{"10.0.0.10"},
		EndpointPort:    8080,
		ServicePortName: "http",
		ServiceAccount:  wantSAN,
		HealthStatus:    Healthy,
	}}, false)

	nonServiceChange := &PushRequest{
		Full: true,
		ConfigsUpdated: sets.New(ConfigKey{
			Kind:      kind.PeerAuthentication,
			Name:      "strict",
			Namespace: "app",
		}),
	}
	updatedPush := NewPushContext()
	updatedPush.InitContext(env, initialPush, nonServiceChange)

	if got := updatedPush.ServiceAccounts(hostName, "app"); !slices.Equal(got, []string{wantSAN}) {
		t.Fatalf("updated ServiceAccounts = %v, want [%s]", got, wantSAN)
	}
	if got := initialPush.ServiceAccounts(hostName, "app"); len(got) != 0 {
		t.Fatalf("initial ServiceAccounts mutated to %v", got)
	}

	const rotatedSAN = "spiffe://cluster.local/ns/app/sa/payment-v2"
	env.EndpointIndex.UpdateServiceEndpoints(ShardKey{}, string(hostName), "app", []*DubboEndpoint{{
		Addresses:       []string{"10.0.0.11"},
		EndpointPort:    8080,
		ServicePortName: "http",
		ServiceAccount:  rotatedSAN,
		HealthStatus:    Healthy,
	}}, false)
	rotatedPush := NewPushContext()
	rotatedPush.InitContext(env, updatedPush, nonServiceChange)
	if got := rotatedPush.ServiceAccounts(hostName, "app"); !slices.Equal(got, []string{rotatedSAN}) {
		t.Fatalf("rotated ServiceAccounts = %v, want [%s]", got, rotatedSAN)
	}
	if got := updatedPush.ServiceAccounts(hostName, "app"); !slices.Equal(got, []string{wantSAN}) {
		t.Fatalf("updated ServiceAccounts mutated to %v", got)
	}

	if pushType := env.EndpointIndex.UpdateServiceEndpoints(ShardKey{}, string(hostName), "app", nil, false); pushType != FullPush {
		t.Fatalf("UpdateServiceEndpoints(..., nil, false) pushType = %v, want FullPush", pushType)
	}
	clearedPush := NewPushContext()
	clearedPush.InitContext(env, rotatedPush, nonServiceChange)
	if got := clearedPush.ServiceAccounts(hostName, "app"); len(got) != 0 {
		t.Fatalf("cleared ServiceAccounts = %v, want empty", got)
	}
}

func TestEndpointIndexServiceAccountPushTypeAndReset(t *testing.T) {
	const (
		svcName = "payment.app.svc.cluster.local"
		ns      = "app"
		sa1     = "spiffe://cluster.local/ns/app/sa/payment"
		sa2     = "spiffe://cluster.local/ns/app/sa/payment-v2"
	)

	index := NewEndpointIndex(DisabledCache{})
	shardA := ShardKey{Provider: provider.Kubernetes}
	shardB := ShardKey{Provider: provider.External}

	// Empty delete on unseen service stays incremental.
	if got := index.UpdateServiceEndpoints(shardA, svcName, ns, nil, false); got != IncrementalPush {
		t.Fatalf("unseen empty update pushType = %v, want IncrementalPush", got)
	}

	// Creating shard without SA still triggers FullPush for new endpoint shard.
	if got := index.UpdateServiceEndpoints(shardA, svcName, ns, []*DubboEndpoint{{
		Addresses:       []string{"10.0.0.10"},
		EndpointPort:    8080,
		ServicePortName: "http",
		HealthStatus:    Healthy,
	}}, false); got != FullPush {
		t.Fatalf("initial shard create pushType = %v, want FullPush", got)
	}

	// Adding an SA to an existing shard must trigger FullPush.
	if got := index.UpdateServiceEndpoints(shardA, svcName, ns, []*DubboEndpoint{{
		Addresses:       []string{"10.0.0.10"},
		EndpointPort:    8080,
		ServicePortName: "http",
		ServiceAccount:  sa1,
		HealthStatus:    Healthy,
	}}, false); got != FullPush {
		t.Fatalf("adding SA on existing shard pushType = %v, want FullPush", got)
	}

	// Endpoint churn with unchanged SA set stays IncrementalPush.
	if got := index.UpdateServiceEndpoints(shardA, svcName, ns, []*DubboEndpoint{{
		Addresses:       []string{"10.0.0.11"},
		EndpointPort:    8080,
		ServicePortName: "http",
		ServiceAccount:  sa1,
		HealthStatus:    Healthy,
	}}, false); got != IncrementalPush {
		t.Fatalf("same-SA endpoint update pushType = %v, want IncrementalPush", got)
	}

	// Adding a second shard with a new SA triggers FullPush.
	if got := index.UpdateServiceEndpoints(shardB, svcName, ns, []*DubboEndpoint{{
		Addresses:       []string{"10.0.0.12"},
		EndpointPort:    8080,
		ServicePortName: "http",
		ServiceAccount:  sa2,
		HealthStatus:    Healthy,
	}}, false); got != FullPush {
		t.Fatalf("second shard new SA pushType = %v, want FullPush", got)
	}

	// Deleting shardB via nil endpoints recomputes ServiceAccounts from shardA and triggers FullPush.
	if got := index.UpdateServiceEndpoints(shardB, svcName, ns, nil, false); got != FullPush {
		t.Fatalf("deleting shardB SA pushType = %v, want FullPush", got)
	}
	shards, ok := index.ShardsForService(svcName, ns)
	if !ok {
		t.Fatal("expected EndpointShards entry to remain preserved")
	}
	shards.RLock()
	remainingSAs := sets.SortedList(shards.ServiceAccounts)
	shards.RUnlock()
	if !slices.Equal(remainingSAs, []string{sa1}) {
		t.Fatalf("remaining ServiceAccounts = %v, want [%s]", remainingSAs, sa1)
	}

	// Deleting last shard via nil endpoints clears ServiceAccounts and triggers FullPush.
	if got := index.UpdateServiceEndpoints(shardA, svcName, ns, nil, false); got != FullPush {
		t.Fatalf("deleting last SA shard pushType = %v, want FullPush", got)
	}
	shards.RLock()
	clearedLen := shards.ServiceAccounts.Len()
	shards.RUnlock()
	if clearedLen != 0 {
		t.Fatalf("cleared ServiceAccounts len = %d, want 0", clearedLen)
	}

	// Repeated empty update after SAs are already empty stays IncrementalPush.
	if got := index.UpdateServiceEndpoints(shardA, svcName, ns, nil, false); got != IncrementalPush {
		t.Fatalf("repeated empty update pushType = %v, want IncrementalPush", got)
	}

	// Adding SA back to preserved empty shard triggers FullPush again.
	if got := index.UpdateServiceEndpoints(shardA, svcName, ns, []*DubboEndpoint{{
		Addresses:       []string{"10.0.0.13"},
		EndpointPort:    8080,
		ServicePortName: "http",
		ServiceAccount:  sa1,
		HealthStatus:    Healthy,
	}}, false); got != FullPush {
		t.Fatalf("re-adding SA after empty pushType = %v, want FullPush", got)
	}
}
