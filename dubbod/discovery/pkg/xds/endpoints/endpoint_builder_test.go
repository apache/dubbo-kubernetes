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
	"testing"

	"github.com/apache/dubbo-kubernetes/dubbod/discovery/pkg/config/memory"
	"github.com/apache/dubbo-kubernetes/dubbod/discovery/pkg/model"
	"github.com/apache/dubbo-kubernetes/pkg/cluster"
	"github.com/apache/dubbo-kubernetes/pkg/config"
	"github.com/apache/dubbo-kubernetes/pkg/config/host"
	"github.com/apache/dubbo-kubernetes/pkg/config/mesh"
	"github.com/apache/dubbo-kubernetes/pkg/config/mesh/meshwatcher"
	"github.com/apache/dubbo-kubernetes/pkg/config/protocol"
	"github.com/apache/dubbo-kubernetes/pkg/config/schema/collections"
	"github.com/apache/dubbo-kubernetes/pkg/kube/krt"
	"github.com/apache/dubbo-kubernetes/pkg/kube/multicluster"
	endpoint "github.com/kdubbo/xds-api/endpoint/v1"
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

func TestBuildClusterLoadAssignmentUsesEastWestGatewayForRemoteShard(t *testing.T) {
	hostname := host.Name("nginx.app.svc.cluster.local")
	svc := newEndpointTestService("nginx", "app", string(hostname), 80)
	push := newEndpointTestPushContext(t, nil, []*model.Service{svc})
	index := model.NewEndpointIndex(model.DisabledCache{})
	index.UpdateServiceEndpoints(model.ShardKey{Cluster: cluster.ID("remote")}, string(hostname), "app", []*model.DubboEndpoint{{
		Addresses:       []string{"192.168.219.71"},
		EndpointPort:    80,
		ServicePortName: "http",
		HealthStatus:    model.Healthy,
	}}, false)

	clusterName := model.BuildSubsetKey(model.TrafficDirectionOutbound, "", hostname, 80)
	proxy := newEndpointTestProxy()
	proxy.Metadata.ClusterID = cluster.ID("primary")
	builder := NewEndpointBuilder(clusterName, proxy, push)
	cla := builder.BuildClusterLoadAssignmentWithGateways(index, map[cluster.ID]multicluster.EastWestGateway{
		cluster.ID("remote"): {Cluster: cluster.ID("remote"), Address: "192.168.15.155", Port: 15443},
	})

	if got := firstEndpointAddress(t, cla); got != "192.168.15.155" {
		t.Fatalf("endpoint address = %q, want east-west gateway", got)
	}
	if got := firstEndpointPort(t, cla); got != 15443 {
		t.Fatalf("endpoint port = %d, want east-west gateway port", got)
	}
}

func TestBuildClusterLoadAssignmentKeepsLocalShardPodIPWhenGatewayConfigured(t *testing.T) {
	hostname := host.Name("nginx.app.svc.cluster.local")
	svc := newEndpointTestService("nginx", "app", string(hostname), 80)
	push := newEndpointTestPushContext(t, nil, []*model.Service{svc})
	index := model.NewEndpointIndex(model.DisabledCache{})
	index.UpdateServiceEndpoints(model.ShardKey{Cluster: cluster.ID("primary")}, string(hostname), "app", []*model.DubboEndpoint{{
		Addresses:       []string{"10.0.0.1"},
		EndpointPort:    80,
		ServicePortName: "http",
		HealthStatus:    model.Healthy,
	}}, false)

	clusterName := model.BuildSubsetKey(model.TrafficDirectionOutbound, "", hostname, 80)
	proxy := newEndpointTestProxy()
	proxy.Metadata.ClusterID = cluster.ID("primary")
	builder := NewEndpointBuilder(clusterName, proxy, push)
	cla := builder.BuildClusterLoadAssignmentWithGateways(index, map[cluster.ID]multicluster.EastWestGateway{
		cluster.ID("primary"): {Cluster: cluster.ID("primary"), Address: "192.168.15.164", Port: 15443},
	})

	if got := firstEndpointAddress(t, cla); got != "10.0.0.1" {
		t.Fatalf("endpoint address = %q, want local pod IP", got)
	}
	if got := firstEndpointPort(t, cla); got != 80 {
		t.Fatalf("endpoint port = %d, want local endpoint port", got)
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
		ID:              "proxyless~10.0.0.2~nginx-consumer.app~app.svc.cluster.local",
		Type:            model.Proxyless,
		ConfigNamespace: "app",
		Metadata: &model.NodeMetadata{
			Generator: "grpc",
			Namespace: "app",
			ClusterID: cluster.ID("primary"),
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
