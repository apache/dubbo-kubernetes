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

package endpoints

import (
	"strings"
	"testing"
	"unicode"

	"github.com/apache/dubbo-kubernetes/dubbod/discovery/pkg/config/memory"
	"github.com/apache/dubbo-kubernetes/dubbod/discovery/pkg/model"
	"github.com/apache/dubbo-kubernetes/pkg/config"
	"github.com/apache/dubbo-kubernetes/pkg/config/host"
	"github.com/apache/dubbo-kubernetes/pkg/config/mesh"
	"github.com/apache/dubbo-kubernetes/pkg/config/mesh/meshwatcher"
	"github.com/apache/dubbo-kubernetes/pkg/config/protocol"
	"github.com/apache/dubbo-kubernetes/pkg/config/schema/collections"
	"github.com/apache/dubbo-kubernetes/pkg/config/schema/gvk"
	"github.com/apache/dubbo-kubernetes/pkg/kube/krt"
	security "github.com/dubml/api/security/v1alpha3"
	endpoint "github.com/dubml/xds-api/endpoint/v1"
)

func TestBuildClusterLoadAssignmentKeepsAppPortWithoutDUBBOMutual(t *testing.T) {
	hostname := host.Name("nginx.app.svc.cluster.local")
	svc := newEndpointTestService("nginx", "app", string(hostname), 80)
	push := newEndpointTestPushContext(t, nil, []*model.Service{svc})
	index := model.NewEndpointIndex(model.DisabledCache{})
	index.UpdateServiceEndpoints(model.ShardKey{}, string(hostname), "app", []*model.DubboEndpoint{{
		Addresses:       []string{"10.0.0.1"},
		EndpointPort:    80,
		ServicePortName: "http",
		HealthStatus:    model.Healthy,
	}}, false)

	clusterName := model.BuildSubsetKey(model.TrafficDirectionOutbound, "", hostname, 80)
	builder := NewEndpointBuilder(clusterName, newEndpointTestProxy(), push)
	cla := builder.BuildClusterLoadAssignment(index)

	if got := firstEndpointPort(t, cla); got != 80 {
		t.Fatalf("endpoint port = %d, want app port 80", got)
	}
}

func TestBuildClusterLoadAssignmentKeepsNativeAppPortWithStrictMTLS(t *testing.T) {
	hostname := host.Name("payment.app.svc.cluster.local")
	svc := newEndpointTestService("payment", "app", string(hostname), 8080)
	push := newEndpointTestPushContext(t, []config.Config{{
		Meta: config.Meta{
			GroupVersionKind: gvk.PeerAuthentication,
			Name:             "strict",
			Namespace:        "app",
		},
		Spec: &security.PeerAuthentication{
			Mtls: &security.PeerAuthentication_MutualTLS{
				Mode: security.PeerAuthentication_MutualTLS_STRICT,
			},
		},
	}}, []*model.Service{svc})
	index := model.NewEndpointIndex(model.DisabledCache{})
	index.UpdateServiceEndpoints(model.ShardKey{}, string(hostname), "app", []*model.DubboEndpoint{{
		Addresses:       []string{"10.0.0.5"},
		EndpointPort:    8080,
		ServicePortName: "http",
		HealthStatus:    model.Healthy,
	}}, false)

	clusterName := model.BuildSubsetKey(model.TrafficDirectionOutbound, "", hostname, 8080)
	cla := NewEndpointBuilder(clusterName, newEndpointTestProxy(), push).BuildClusterLoadAssignment(index)
	if got := firstEndpointPort(t, cla); got != 8080 {
		t.Fatalf("endpoint port = %d, want native application port 8080", got)
	}
}

func TestBuildClusterLoadAssignmentUsesExternalNameDNS(t *testing.T) {
	hostname := host.Name("httpbin-egress.app.svc.cluster.local")
	svc := newEndpointTestService("httpbin-egress", "app", string(hostname), 443)
	svc.Resolution = model.Alias
	svc.Attributes.ExternalName = "httpbin.org"
	push := newEndpointTestPushContext(t, nil, []*model.Service{svc})
	index := model.NewEndpointIndex(model.DisabledCache{})

	clusterName := model.BuildSubsetKey(model.TrafficDirectionOutbound, "", hostname, 443)
	builder := NewEndpointBuilder(clusterName, newEndpointTestProxy(), push)
	cla := builder.BuildClusterLoadAssignment(index)

	if got := firstEndpointAddress(t, cla); got != "httpbin.org" {
		t.Fatalf("endpoint address = %q, want external DNS name", got)
	}
	if got := firstEndpointPort(t, cla); got != 443 {
		t.Fatalf("endpoint port = %d, want service port 443", got)
	}
}

func TestBuildClusterLoadAssignmentPreservesVMLocalityHealthAndWeight(t *testing.T) {
	hostname := host.Name("reviews.mesh.local")
	svc := newEndpointTestService("reviews", "bookinfo", string(hostname), 50051)
	push := newEndpointTestPushContext(t, nil, []*model.Service{svc})
	index := model.NewEndpointIndex(model.DisabledCache{})
	index.UpdateServiceEndpoints(model.ShardKey{}, string(hostname), "bookinfo", []*model.DubboEndpoint{
		{
			Addresses:       []string{"10.0.0.10"},
			EndpointPort:    50051,
			ServicePortName: "http",
			Locality:        "us-east-1/zone-a/rack-1",
			LbWeight:        7,
			HealthStatus:    model.Healthy,
		},
		{
			Addresses:       []string{"10.0.0.11"},
			EndpointPort:    50051,
			ServicePortName: "http",
			Locality:        "us-west-2/zone-b",
			LbWeight:        3,
			HealthStatus:    model.UnHealthy,
		},
	}, false)

	clusterName := model.BuildSubsetKey(model.TrafficDirectionOutbound, "", hostname, 50051)
	cla := NewEndpointBuilder(clusterName, newEndpointTestProxy(), push).BuildClusterLoadAssignment(index)
	localities := cla.GetEndpoints()
	if len(localities) != 2 {
		t.Fatalf("localities = %d, want 2", len(localities))
	}
	if got := localities[0].GetLocality(); got.GetRegion() != "us-east-1" || got.GetZone() != "zone-a" || got.GetSubZone() != "rack-1" {
		t.Fatalf("first locality = %v", got)
	}
	if got := localities[0].GetLoadBalancingWeight().GetValue(); got != 7 {
		t.Fatalf("first locality weight = %d, want 7", got)
	}
	first := localities[0].GetLbEndpoints()[0]
	metadata := first.GetMetadata().GetFilterMetadata()["networking.dubbo.apache.org"]
	if got := metadata.GetFields()["locality"].GetStringValue(); got != "us-east-1/zone-a/rack-1" {
		t.Fatalf("first endpoint locality metadata = %q", got)
	}
	if got := first.GetLoadBalancingWeight().GetValue(); got != 7 {
		t.Fatalf("first endpoint weight = %d, want 7", got)
	}
	if got := localities[1].GetLbEndpoints()[0].GetHealthStatus(); got.String() != "UNHEALTHY" {
		t.Fatalf("second endpoint health = %v, want UNHEALTHY", got)
	}
}

func firstEndpointAddress(t *testing.T, cla *endpoint.ClusterLoadAssignment) string {
	t.Helper()
	localities := cla.GetEndpoints()
	if len(localities) == 0 || len(localities[0].GetLbEndpoints()) == 0 {
		t.Fatalf("CLA has no endpoints")
	}
	return localities[0].GetLbEndpoints()[0].GetEndpoint().GetAddress().GetSocketAddress().GetAddress()
}

func firstEndpointPort(t *testing.T, cla *endpoint.ClusterLoadAssignment) uint32 {
	t.Helper()
	localities := cla.GetEndpoints()
	if len(localities) == 0 || len(localities[0].GetLbEndpoints()) == 0 {
		t.Fatalf("CLA has no endpoints")
	}
	return localities[0].GetLbEndpoints()[0].GetEndpoint().GetAddress().GetSocketAddress().GetPortValue()
}

func newEndpointTestPushContext(t *testing.T, configs []config.Config, services []*model.Service) *model.PushContext {
	t.Helper()
	store := memory.Make(collections.DubboGatewayAPI())
	for _, cfg := range configs {
		if _, err := store.Create(cfg); err != nil {
			t.Fatalf("create config %s/%s: %v", cfg.Namespace, cfg.Name, err)
		}
	}
	env := model.NewEnvironment()
	env.ConfigStore = store
	env.ServiceDiscovery = endpointTestServiceDiscovery{services: services}
	env.Watcher = meshwatcher.ConfigAdapter(krt.NewStatic(&meshwatcher.MeshConfigResource{
		MeshConfig: mesh.DefaultMeshConfig(),
	}, true))
	env.Init()
	push := model.NewPushContext()
	push.InitContext(env, nil, nil)
	return push
}

func newEndpointTestProxy() *model.Proxy {
	return &model.Proxy{
		ID:              "inherent~10.0.0.2~nginx-consumer.app~app.svc.cluster.local",
		Type:            model.Inherent,
		ConfigNamespace: "app",
		Metadata: &model.NodeMetadata{
			Generator: "triple",
			Namespace: "app",
			ClusterID: "Kubernetes",
		},
	}
}

func newEndpointTestService(name, namespace, hostname string, port int) *model.Service {
	return &model.Service{
		Hostname: host.Name(hostname),
		Ports: model.PortList{{
			Name:     "http",
			Port:     port,
			Protocol: protocol.HTTP2,
		}},
		Attributes: model.ServiceAttributes{
			Name:      name,
			Namespace: namespace,
		},
	}
}

type endpointTestServiceDiscovery struct {
	services []*model.Service
}

func (s endpointTestServiceDiscovery) Services() []*model.Service {
	return s.services
}

func (s endpointTestServiceDiscovery) GetService(hostname host.Name) *model.Service {
	for _, svc := range s.services {
		if svc.Hostname == hostname {
			return svc
		}
	}
	return nil
}

func (s endpointTestServiceDiscovery) GetProxyServiceTargets(*model.Proxy) []model.ServiceTarget {
	return nil
}

func TestBuildClusterLoadAssignmentSnapshotEndpointsContract(t *testing.T) {
	assertCLAContract := func(t *testing.T, cla *endpoint.ClusterLoadAssignment) {
		t.Helper()
		seen := map[string]struct{}{}
		for lIdx, loc := range cla.GetEndpoints() {
			if loc.GetProximity() != nil {
				t.Fatalf("locality[%d] proximity = %v, want nil", lIdx, loc.GetProximity())
			}
			for eIdx, lbEp := range loc.GetLbEndpoints() {
				if hs := lbEp.GetHealthStatus(); hs < 0 || hs > 5 {
					t.Fatalf("locality[%d].endpoint[%d] invalid health_status %d", lIdx, eIdx, hs)
				}
				epIdentifier, ok := lbEp.GetHostIdentifier().(*endpoint.LbEndpoint_Endpoint)
				if !ok || epIdentifier.Endpoint == nil {
					t.Fatalf("locality[%d].endpoint[%d] host_identifier = %T, want LbEndpoint_Endpoint", lIdx, eIdx, lbEp.GetHostIdentifier())
				}
				sa := epIdentifier.Endpoint.GetAddress().GetSocketAddress()
				if sa == nil {
					t.Fatalf("locality[%d].endpoint[%d] address = %v, want SocketAddress", lIdx, eIdx, epIdentifier.Endpoint.GetAddress())
				}
				port := sa.GetPortValue()
				if port == 0 || port > 65535 {
					t.Fatalf("locality[%d].endpoint[%d] port = %d, want 1..65535", lIdx, eIdx, port)
				}
				hostAddr := sa.GetAddress()
				if hostAddr == "" || strings.ContainsAny(hostAddr, "/@") {
					t.Fatalf("locality[%d].endpoint[%d] invalid host %q", lIdx, eIdx, hostAddr)
				}
				for _, r := range hostAddr {
					if unicode.IsSpace(r) {
						t.Fatalf("locality[%d].endpoint[%d] host %q contains whitespace", lIdx, eIdx, hostAddr)
					}
				}
				key := formatEndpointAddressKey(hostAddr, port)
				if _, dup := seen[key]; dup {
					t.Fatalf("duplicate endpoint address %q in CLA", key)
				}
				seen[key] = struct{}{}
			}
		}
	}

	t.Run("filters invalid hosts ports unix pipes and deduplicates across shards", func(t *testing.T) {
		hostname := host.Name("reviews.app.svc.cluster.local")
		svc := newEndpointTestService("reviews", "app", string(hostname), 8080)
		push := newEndpointTestPushContext(t, nil, []*model.Service{svc})
		index := model.NewEndpointIndex(model.DisabledCache{})

		shardA := model.ShardKey{Provider: "Kubernetes"}
		shardB := model.ShardKey{Provider: "Mock"}
		index.UpdateServiceEndpoints(shardA, string(hostname), "app", []*model.DubboEndpoint{
			{Addresses: []string{"10.0.0.1"}, EndpointPort: 8080, ServicePortName: "http", HealthStatus: model.Healthy},
			{Addresses: []string{"unix:///var/run/app.sock"}, EndpointPort: 0, ServicePortName: "http", HealthStatus: model.Healthy},
			{Addresses: []string{"10.0.0.2"}, EndpointPort: 0, ServicePortName: "http", HealthStatus: model.Healthy},
			{Addresses: []string{"10.0.0.3"}, EndpointPort: 70000, ServicePortName: "http", HealthStatus: model.Healthy},
			{Addresses: []string{"10.0.0.4/extra"}, EndpointPort: 8080, ServicePortName: "http", HealthStatus: model.Healthy},
			{Addresses: []string{"user@10.0.0.5"}, EndpointPort: 8080, ServicePortName: "http", HealthStatus: model.Healthy},
			{Addresses: []string{"10.0.0.6 "}, EndpointPort: 8080, ServicePortName: "http", HealthStatus: model.Healthy},
		}, false)
		index.UpdateServiceEndpoints(shardB, string(hostname), "app", []*model.DubboEndpoint{
			{Addresses: []string{"10.0.0.1"}, EndpointPort: 8080, ServicePortName: "http", HealthStatus: model.Healthy},
			{Addresses: []string{"fd00::1"}, EndpointPort: 8080, ServicePortName: "http", HealthStatus: model.Healthy},
			{Addresses: []string{"fd00::1"}, EndpointPort: 8080, ServicePortName: "http", HealthStatus: model.Healthy},
		}, false)

		clusterName := model.BuildSubsetKey(model.TrafficDirectionOutbound, "", hostname, 8080)
		cla := NewEndpointBuilder(clusterName, newEndpointTestProxy(), push).BuildClusterLoadAssignment(index)
		assertCLAContract(t, cla)

		if len(cla.GetEndpoints()) != 1 || len(cla.GetEndpoints()[0].GetLbEndpoints()) != 2 {
			t.Fatalf("got %d localities / endpoints %v, want exactly 2 unique valid endpoints", len(cla.GetEndpoints()), cla.GetEndpoints())
		}
	})

	t.Run("invalid external name returns empty CLA", func(t *testing.T) {
		hostname := host.Name("bad-egress.app.svc.cluster.local")
		svc := newEndpointTestService("bad-egress", "app", string(hostname), 443)
		svc.Resolution = model.Alias
		svc.Attributes.ExternalName = "bad/host@example.com"
		push := newEndpointTestPushContext(t, nil, []*model.Service{svc})
		index := model.NewEndpointIndex(model.DisabledCache{})

		clusterName := model.BuildSubsetKey(model.TrafficDirectionOutbound, "", hostname, 443)
		cla := NewEndpointBuilder(clusterName, newEndpointTestProxy(), push).BuildClusterLoadAssignment(index)
		assertCLAContract(t, cla)
		if len(cla.GetEndpoints()) != 0 {
			t.Fatalf("endpoints = %d, want 0 for invalid ExternalName", len(cla.GetEndpoints()))
		}
	})
}
