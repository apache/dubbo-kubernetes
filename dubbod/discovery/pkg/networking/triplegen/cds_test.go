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

package triplegen

import (
	"slices"
	"testing"

	"github.com/apache/dubbo-kubernetes/dubbod/discovery/pkg/features"
	"github.com/apache/dubbo-kubernetes/dubbod/discovery/pkg/model"
	"github.com/apache/dubbo-kubernetes/dubbod/discovery/pkg/serviceregistry/provider"
	"github.com/apache/dubbo-kubernetes/pkg/config"
	"github.com/apache/dubbo-kubernetes/pkg/config/host"
	"github.com/apache/dubbo-kubernetes/pkg/config/schema/gvk"
	"github.com/apache/dubbo-kubernetes/pkg/config/schema/kind"
	"github.com/apache/dubbo-kubernetes/pkg/util/sets"
	security "github.com/dubml/api/security/v1alpha3"
	cluster "github.com/dubml/xds-api/cluster/v1"
	core "github.com/dubml/xds-api/core/v1"
	tlsv1 "github.com/dubml/xds-api/extensions/transport_sockets/tls/v1"
)

func TestStrictPeerAuthenticationPinsBackendIdentity(t *testing.T) {
	hostName := host.Name("payment.app.svc.cluster.local")
	service := newRDSTestService("payment", "app", string(hostName), 8080)
	backendSAN := "spiffe://cluster.local/ns/app/sa/payment"
	service.ServiceAccounts = []string{backendSAN}
	push := newRDSTestPushContext(t, []config.Config{
		{
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
		},
	}, []*model.Service{service})

	resources := (&TripleConfigGenerator{}).BuildClusters(&model.Proxy{
		ID:              "inherent~10.0.0.2~caller.app~app.svc.cluster.local",
		Type:            model.Inherent,
		ConfigNamespace: "app",
	}, push, []string{"outbound|8080||" + string(hostName)})
	if len(resources) != 1 {
		t.Fatalf("resources = %d, want 1", len(resources))
	}
	generated := &cluster.Cluster{}
	if err := resources[0].GetResource().UnmarshalTo(generated); err != nil {
		t.Fatalf("unmarshal cluster: %v", err)
	}
	if generated.GetLbPolicy() != cluster.Cluster_ROUND_ROBIN {
		t.Fatalf("lb_policy = %v, want ROUND_ROBIN (0)", generated.GetLbPolicy())
	}
	if generated.GetTransportSocket() == nil {
		t.Fatal("STRICT PeerAuthentication did not produce an mTLS transport socket")
	}
	tlsContext := &tlsv1.UpstreamTlsContext{}
	if err := generated.GetTransportSocket().GetTypedConfig().UnmarshalTo(tlsContext); err != nil {
		t.Fatalf("unmarshal upstream TLS context: %v", err)
	}
	got := tlsContext.GetCommonTlsContext().
		GetCombinedValidationContext().
		GetDefaultValidationContext().
		GetMatchSubjectAltNames()
	if len(got) != 1 || got[0] != backendSAN {
		t.Fatalf("SAN pins = %v, want only the backend identity", got)
	}
}

func TestBuildClustersAlwaysUsesRoundRobinRegardlessOfDefaultLBPolicy(t *testing.T) {
	hostName := host.Name("payment.app.svc.cluster.local")
	service := newRDSTestService("payment", "app", string(hostName), 8080)
	push := newRDSTestPushContext(t, nil, []*model.Service{service})

	orig := features.DefaultLoadBalancerPolicy
	t.Cleanup(func() {
		features.DefaultLoadBalancerPolicy = orig
	})

	for _, policy := range []string{"", "LEAST_REQUEST", "RING_HASH", "RANDOM"} {
		features.DefaultLoadBalancerPolicy = policy
		resources := (&TripleConfigGenerator{}).BuildClusters(&model.Proxy{
			ID:              "inherent~10.0.0.2~caller.app~app.svc.cluster.local",
			Type:            model.Inherent,
			ConfigNamespace: "app",
		}, push, []string{"outbound|8080||" + string(hostName)})
		if len(resources) != 1 {
			t.Fatalf("policy %q: resources = %d, want 1", policy, len(resources))
		}
		generated := &cluster.Cluster{}
		if err := resources[0].GetResource().UnmarshalTo(generated); err != nil {
			t.Fatalf("policy %q: unmarshal cluster: %v", policy, err)
		}
		if generated.GetLbPolicy() != cluster.Cluster_ROUND_ROBIN {
			t.Fatalf("policy %q: lb_policy = %v (%d), want ROUND_ROBIN (0)", policy, generated.GetLbPolicy(), int32(generated.GetLbPolicy()))
		}
	}
}

func TestBuildClustersSnapshotValidationContract(t *testing.T) {
	hostName := host.Name("payment.app.svc.cluster.local")
	backendSAN := "spiffe://cluster.local/ns/app/sa/payment"
	service := newRDSTestService("payment", "app", string(hostName), 8080)
	service.ServiceAccounts = []string{backendSAN}

	strictConfig := config.Config{
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
	}

	clusterName := "outbound|8080||" + string(hostName)
	proxy := &model.Proxy{
		ID:              "inherent~10.0.0.2~caller.app~app.svc.cluster.local",
		Type:            model.Inherent,
		ConfigNamespace: "app",
	}

	assertClusterContract := func(t *testing.T, c *cluster.Cluster, wantMTLS bool) {
		t.Helper()
		if c.GetLbPolicy() != cluster.Cluster_ROUND_ROBIN {
			t.Fatalf("lb_policy = %v, want ROUND_ROBIN (0)", c.GetLbPolicy())
		}
		if len(c.GetHealthChecks()) != 0 {
			t.Fatalf("health_checks = %d, want 0", len(c.GetHealthChecks()))
		}
		if c.GetCircuitBreakers() != nil {
			t.Fatalf("circuit_breakers = %v, want nil", c.GetCircuitBreakers())
		}
		if c.GetOutlierDetection() != nil {
			t.Fatalf("outlier_detection = %v, want nil", c.GetOutlierDetection())
		}
		if c.GetMaxRequestsPerConnection() != nil {
			t.Fatalf("max_requests_per_connection = %v, want nil", c.GetMaxRequestsPerConnection())
		}
		if c.GetPerConnectionBufferLimitBytes() != nil {
			t.Fatalf("per_connection_buffer_limit_bytes = %v, want nil", c.GetPerConnectionBufferLimitBytes())
		}
		for _, status := range c.GetCommonLbConfig().GetOverrideHostStatus().GetStatuses() {
			if status < 0 || status > 5 {
				t.Fatalf("override_host_status contains out-of-range status %d", status)
			}
		}
		if timeout := c.GetConnectTimeout(); timeout != nil {
			if timeout.GetSeconds() < 0 || timeout.GetNanos() < 0 || timeout.GetNanos() >= 1_000_000_000 || timeout.AsDuration() == 0 {
				t.Fatalf("connect_timeout = %v, want valid positive duration or nil", timeout)
			}
		}
		if c.GetType() != cluster.Cluster_EDS {
			t.Fatalf("cluster_discovery_type = %v, want EDS (3)", c.GetType())
		}
		edsCfg := c.GetEdsClusterConfig()
		if edsCfg == nil || edsCfg.GetServiceName() != clusterName {
			t.Fatalf("eds_cluster_config = %v, want service_name %q", edsCfg, clusterName)
		}
		if _, ok := edsCfg.GetEdsConfig().GetConfigSourceSpecifier().(*core.ConfigSource_Ads); !ok {
			t.Fatalf("eds_config = %v, want ADS config source", edsCfg.GetEdsConfig())
		}
		if c.GetLoadAssignment() != nil {
			t.Fatalf("EDS cluster must not have inline load_assignment: %v", c.GetLoadAssignment())
		}

		ts := c.GetTransportSocket()
		if !wantMTLS {
			if ts != nil {
				t.Fatalf("plaintext transport_socket = %v, want nil", ts)
			}
			return
		}
		if ts == nil || (ts.GetName() != "transport_sockets.tls" && ts.GetName() != "envoy.transport_sockets.tls") {
			t.Fatalf("transport_socket = %v, want TLS transport socket", ts)
		}
		const wantTLSTypeURL = "type.googleapis.com/extensions.transport_sockets.tls.v1.UpstreamTlsContext"
		typedCfg := ts.GetTypedConfig()
		if typedCfg == nil || typedCfg.GetTypeUrl() != wantTLSTypeURL {
			t.Fatalf("transport_socket typed_config = %v, want %q", typedCfg, wantTLSTypeURL)
		}
		tlsContext := &tlsv1.UpstreamTlsContext{}
		if err := typedCfg.UnmarshalTo(tlsContext); err != nil {
			t.Fatalf("unmarshal UpstreamTlsContext: %v", err)
		}
		common := tlsContext.GetCommonTlsContext()
		if common == nil {
			t.Fatal("common_tls_context is nil")
		}
		if alpn := common.GetAlpnProtocols(); len(alpn) != 1 || alpn[0] != "h2" {
			t.Fatalf("alpn_protocols = %v, want [h2]", alpn)
		}
		if inst := common.GetTlsCertificateCertificateProviderInstance().GetInstanceName(); inst != "default" {
			t.Fatalf("tls cert provider instance = %q, want default", inst)
		}
		combined := common.GetCombinedValidationContext()
		if combined == nil {
			t.Fatal("combined_validation_context is nil")
		}
		if inst := combined.GetValidationContextCertificateProviderInstance().GetInstanceName(); inst != "default" {
			t.Fatalf("validation root provider instance = %q, want default", inst)
		}
		if combined.GetDefaultValidationContext().GetTrustedCa() != nil {
			t.Fatalf("default_validation_context.trusted_ca = %v, want nil", combined.GetDefaultValidationContext().GetTrustedCa())
		}
		if sans := combined.GetDefaultValidationContext().GetMatchSubjectAltNames(); len(sans) != 1 || sans[0] != backendSAN {
			t.Fatalf("match_subject_alt_names = %v, want [%s]", sans, backendSAN)
		}
	}

	for _, tc := range []struct {
		name     string
		configs  []config.Config
		wantMTLS bool
	}{
		{name: "plaintext", configs: nil, wantMTLS: false},
		{name: "strict-mtls", configs: []config.Config{strictConfig}, wantMTLS: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			push := newRDSTestPushContext(t, tc.configs, []*model.Service{service})
			resources := (&TripleConfigGenerator{}).BuildClusters(proxy, push, []string{clusterName})
			if len(resources) != 1 {
				t.Fatalf("resources = %d, want 1", len(resources))
			}
			generated := &cluster.Cluster{}
			if err := resources[0].GetResource().UnmarshalTo(generated); err != nil {
				t.Fatalf("unmarshal cluster: %v", err)
			}
			assertClusterContract(t, generated, tc.wantMTLS)
		})
	}
}

func TestStrictPeerAuthenticationUsesEndpointServiceAccounts(t *testing.T) {
	const hostName = host.Name("payment.app.svc.cluster.local")
	const clusterName = "outbound|8080||" + string(hostName)
	const wantPrimarySAN = "spiffe://cluster.local/ns/app/sa/payment"
	const wantCanarySAN = "spiffe://cluster.local/ns/app/sa/payment-canary"

	service := newRDSTestService("payment", "app", string(hostName), 8080)
	env := newRDSTestEnv(t, []config.Config{
		newPeerAuthenticationConfig("strict", "app", security.PeerAuthentication_MutualTLS_STRICT),
	}, []*model.Service{service})

	initialPush := model.NewPushContext()
	initialPush.InitContext(env, nil, nil)

	// Simulate endpoints arriving into EndpointIndex after the initial snapshot, followed
	// by a non-Service full push (e.g. PeerAuthentication update).
	env.EndpointIndex.UpdateServiceEndpoints(model.ShardKey{}, string(hostName), "app", []*model.DubboEndpoint{
		{
			Addresses:       []string{"10.0.0.10"},
			EndpointPort:    8080,
			ServicePortName: "http",
			ServiceAccount:  wantPrimarySAN,
			HealthStatus:    model.Healthy,
		},
		{
			Addresses:       []string{"10.0.0.11"},
			EndpointPort:    8080,
			ServicePortName: "http",
			ServiceAccount:  "payment-canary",
			HealthStatus:    model.Healthy,
		},
	}, false)

	updatedPush := model.NewPushContext()
	updatedPush.InitContext(env, initialPush, &model.PushRequest{
		Full: true,
		ConfigsUpdated: sets.New(model.ConfigKey{
			Kind:      kind.PeerAuthentication,
			Name:      "strict",
			Namespace: "app",
		}),
	})

	proxy := &model.Proxy{
		ID:              "inherent~10.0.0.2~caller.app~app.svc.cluster.local",
		Type:            model.Inherent,
		ConfigNamespace: "app",
	}
	resources := (&TripleConfigGenerator{}).BuildClusters(proxy, updatedPush, []string{clusterName})
	if len(resources) != 1 {
		t.Fatalf("resources = %d, want 1", len(resources))
	}
	generated := &cluster.Cluster{}
	if err := resources[0].GetResource().UnmarshalTo(generated); err != nil {
		t.Fatalf("unmarshal cluster: %v", err)
	}
	if generated.GetTransportSocket() == nil {
		t.Fatal("expected mTLS transport_socket when endpoint ServiceAccounts are populated")
	}
	tlsContext := &tlsv1.UpstreamTlsContext{}
	if err := generated.GetTransportSocket().GetTypedConfig().UnmarshalTo(tlsContext); err != nil {
		t.Fatalf("unmarshal upstream TLS context: %v", err)
	}
	got := tlsContext.GetCommonTlsContext().
		GetCombinedValidationContext().
		GetDefaultValidationContext().
		GetMatchSubjectAltNames()
	want := []string{wantPrimarySAN, wantCanarySAN}
	if !slices.Equal(got, want) {
		t.Fatalf("match_subject_alt_names = %v, want %v", got, want)
	}
}

func TestStrictPeerAuthenticationSkipsMTLSWithoutServiceAccounts(t *testing.T) {
	const hostName = host.Name("payment.app.svc.cluster.local")
	const clusterName = "outbound|8080||" + string(hostName)

	for _, tc := range []struct {
		name            string
		serviceAccounts []string
		endpointSA      string
	}{
		{name: "no endpoints and no service accounts"},
		{name: "endpoint without service account"},
		{name: "malformed empty spiffe service account", endpointSA: "spiffe://cluster.local/ns/app/sa/"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			service := newRDSTestService("payment", "app", string(hostName), 8080)
			service.ServiceAccounts = tc.serviceAccounts
			env := newRDSTestEnv(t, []config.Config{
				newPeerAuthenticationConfig("strict", "app", security.PeerAuthentication_MutualTLS_STRICT),
			}, []*model.Service{service})
			if tc.name != "no endpoints and no service accounts" {
				env.EndpointIndex.UpdateServiceEndpoints(model.ShardKey{}, string(hostName), "app", []*model.DubboEndpoint{{
					Addresses:       []string{"10.0.0.10"},
					EndpointPort:    8080,
					ServicePortName: "http",
					ServiceAccount:  tc.endpointSA,
					HealthStatus:    model.Healthy,
				}}, false)
			}
			push := model.NewPushContext()
			push.InitContext(env, nil, nil)

			resources := (&TripleConfigGenerator{}).BuildClusters(&model.Proxy{
				ID:              "inherent~10.0.0.2~caller.app~app.svc.cluster.local",
				Type:            model.Inherent,
				ConfigNamespace: "app",
			}, push, []string{clusterName})
			if len(resources) != 1 {
				t.Fatalf("resources = %d, want 1", len(resources))
			}
			generated := &cluster.Cluster{}
			if err := resources[0].GetResource().UnmarshalTo(generated); err != nil {
				t.Fatalf("unmarshal cluster: %v", err)
			}
			if generated.GetTransportSocket() != nil {
				t.Fatalf("expected nil transport_socket when no valid SPIFFE SANs exist, got %v", generated.GetTransportSocket())
			}
		})
	}
}

func TestNormalizePeerSPIFFESANs(t *testing.T) {
	if got := normalizePeerSPIFFESANs(nil, "", "app", nil); len(got) != 0 {
		t.Fatalf("nil input = %v, want empty", got)
	}

	got := normalizePeerSPIFFESANs(
		[]string{
			"",
			"   ",
			"spiffe://cluster.local/ns/app/sa/",
			"https://not-spiffe.example/sa/payment",
			"ns/app",
			" spiffe://cluster.local/ns/app/sa/payment ",
			"payment",
			"reviews",
			"reviews",
		},
		"",
		"app",
		[]string{"corp.example"},
	)
	want := []string{
		"spiffe://cluster.local/ns/app/sa/payment",
		"spiffe://cluster.local/ns/app/sa/reviews",
		"spiffe://corp.example/ns/app/sa/payment",
		"spiffe://corp.example/ns/app/sa/reviews",
	}
	if !slices.Equal(got, want) {
		t.Fatalf("normalizePeerSPIFFESANs() = %v, want %v", got, want)
	}

	if gotNoNS := normalizePeerSPIFFESANs([]string{"reviews"}, "cluster.local", "", nil); len(gotNoNS) != 0 {
		t.Fatalf("bare SA with empty namespace = %v, want empty", gotNoNS)
	}
}

func TestDeletingLastEndpointForSARemovesSANFromCluster(t *testing.T) {
	const hostName = host.Name("payment.app.svc.cluster.local")
	const clusterName = "outbound|8080||" + string(hostName)
	const primarySAN = "spiffe://cluster.local/ns/app/sa/payment"
	const canarySAN = "spiffe://cluster.local/ns/app/sa/payment-canary"

	service := newRDSTestService("payment", "app", string(hostName), 8080)
	env := newRDSTestEnv(t, []config.Config{
		newPeerAuthenticationConfig("strict", "app", security.PeerAuthentication_MutualTLS_STRICT),
	}, []*model.Service{service})
	env.EndpointIndex.UpdateServiceEndpoints(model.ShardKey{}, string(hostName), "app", []*model.DubboEndpoint{
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
	}, true)

	proxy := &model.Proxy{
		ID:              "inherent~10.0.0.2~caller.app~app.svc.cluster.local",
		Type:            model.Inherent,
		ConfigNamespace: "app",
	}
	gen := &TripleConfigGenerator{}
	buildMTLS := func(prev *model.PushContext) (*model.PushContext, *cluster.Cluster, []string) {
		push := model.NewPushContext()
		push.InitContext(env, prev, &model.PushRequest{
			Full: true,
			ConfigsUpdated: sets.New(model.ConfigKey{
				Kind:      kind.Service,
				Name:      string(hostName),
				Namespace: "app",
			}),
		})
		resources := gen.BuildClusters(proxy, push, []string{clusterName})
		if len(resources) != 1 {
			t.Fatalf("resources = %d, want 1", len(resources))
		}
		generated := &cluster.Cluster{}
		if err := resources[0].GetResource().UnmarshalTo(generated); err != nil {
			t.Fatalf("unmarshal cluster: %v", err)
		}
		if generated.GetTransportSocket() == nil {
			return push, generated, nil
		}
		tlsContext := &tlsv1.UpstreamTlsContext{}
		if err := generated.GetTransportSocket().GetTypedConfig().UnmarshalTo(tlsContext); err != nil {
			t.Fatalf("unmarshal upstream TLS context: %v", err)
		}
		return push, generated, tlsContext.GetCommonTlsContext().
			GetCombinedValidationContext().
			GetDefaultValidationContext().
			GetMatchSubjectAltNames()
	}

	push1, generated1, sans1 := buildMTLS(nil)
	if generated1.GetTransportSocket() == nil || !slices.Equal(sans1, []string{primarySAN, canarySAN}) {
		t.Fatalf("initial transport=%v sans=%v, want [%s %s]", generated1.GetTransportSocket() != nil, sans1, primarySAN, canarySAN)
	}

	// Delete the last endpoint running under canarySAN; primarySAN remains.
	if pushType := env.EndpointIndex.UpdateServiceEndpoints(model.ShardKey{}, string(hostName), "app", []*model.DubboEndpoint{{
		Addresses:       []string{"10.0.0.10"},
		EndpointPort:    8080,
		ServicePortName: "http",
		ServiceAccount:  primarySAN,
		HealthStatus:    model.Healthy,
	}}, true); pushType != model.FullPush {
		t.Fatalf("removing canary endpoint pushType = %v, want FullPush", pushType)
	}
	push2, generated2, sans2 := buildMTLS(push1)
	if generated2.GetTransportSocket() == nil || !slices.Equal(sans2, []string{primarySAN}) {
		t.Fatalf("after removing canary SAN transport=%v sans=%v, want [%s]", generated2.GetTransportSocket() != nil, sans2, primarySAN)
	}

	// Delete remaining endpoint via nil slice; SANs are cleared and mTLS is omitted.
	if pushType := env.EndpointIndex.UpdateServiceEndpoints(model.ShardKey{}, string(hostName), "app", nil, true); pushType != model.FullPush {
		t.Fatalf("deleting final endpoint pushType = %v, want FullPush", pushType)
	}
	push3, generated3, sans3 := buildMTLS(push2)
	if generated3.GetTransportSocket() != nil || len(sans3) != 0 {
		t.Fatalf("after deleting final endpoint expected nil transport_socket, got transport=%v sans=%v", generated3.GetTransportSocket(), sans3)
	}

	// Verify UpdateServiceEndpoints(..., nil, false) deleting the last endpoint for an SA triggers FullPush and removes the SA from CDS SANs.
	primaryShard := model.ShardKey{Provider: provider.Kubernetes}
	canaryShard := model.ShardKey{Provider: provider.External}
	env.EndpointIndex.UpdateServiceEndpoints(primaryShard, string(hostName), "app", []*model.DubboEndpoint{{
		Addresses:       []string{"10.0.0.10"},
		EndpointPort:    8080,
		ServicePortName: "http",
		ServiceAccount:  primarySAN,
		HealthStatus:    model.Healthy,
	}}, false)
	env.EndpointIndex.UpdateServiceEndpoints(canaryShard, string(hostName), "app", []*model.DubboEndpoint{{
		Addresses:       []string{"10.0.0.11"},
		EndpointPort:    8080,
		ServicePortName: "http",
		ServiceAccount:  canarySAN,
		HealthStatus:    model.Healthy,
	}}, false)
	push4, generated4, sans4 := buildMTLS(push3)
	if generated4.GetTransportSocket() == nil || !slices.Equal(sans4, []string{primarySAN, canarySAN}) {
		t.Fatalf("repopulated multi-shard transport=%v sans=%v, want [%s %s]", generated4.GetTransportSocket() != nil, sans4, primarySAN, canarySAN)
	}

	if pushType := env.EndpointIndex.UpdateServiceEndpoints(canaryShard, string(hostName), "app", nil, false); pushType != model.FullPush {
		t.Fatalf("UpdateServiceEndpoints(canaryShard, nil, false) pushType = %v, want FullPush", pushType)
	}
	push5, generated5, sans5 := buildMTLS(push4)
	if generated5.GetTransportSocket() == nil || !slices.Equal(sans5, []string{primarySAN}) {
		t.Fatalf("after deleting canaryShard via nil,false transport=%v sans=%v, want [%s]", generated5.GetTransportSocket() != nil, sans5, primarySAN)
	}

	if pushType := env.EndpointIndex.UpdateServiceEndpoints(primaryShard, string(hostName), "app", nil, false); pushType != model.FullPush {
		t.Fatalf("UpdateServiceEndpoints(primaryShard, nil, false) pushType = %v, want FullPush", pushType)
	}
	_, generated6, sans6 := buildMTLS(push5)
	if generated6.GetTransportSocket() != nil || len(sans6) != 0 {
		t.Fatalf("after deleting final shard via nil,false expected nil transport_socket, got transport=%v sans=%v", generated6.GetTransportSocket(), sans6)
	}
}
