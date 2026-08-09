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

package grpcgen

import (
	"testing"
	"time"

	"github.com/apache/dubbo-kubernetes/dubbod/discovery/pkg/model"
	"github.com/apache/dubbo-kubernetes/pkg/config"
	"github.com/apache/dubbo-kubernetes/pkg/config/host"
	"github.com/apache/dubbo-kubernetes/pkg/config/schema/gvk"
	networking "github.com/kdubbo/api/networking/v1alpha3"
	security "github.com/kdubbo/api/security/v1alpha3"
	cluster "github.com/kdubbo/xds-api/cluster/v1"
	tlsv1 "github.com/kdubbo/xds-api/extensions/transport_sockets/tls/v1"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"
)

func TestActivationPinsBackendAndActivatorSANsInOneCDSContext(t *testing.T) {
	hostName := host.Name("payment.app.svc.cluster.local")
	service := newRDSTestService("payment", "app", string(hostName), 8080)
	backendSAN := "spiffe://cluster.local/ns/app/sa/payment"
	push := newRDSTestPushContext(t, []config.Config{
		newActivationPolicyConfig("payment", "app", "payment"),
	}, []*model.Service{service})

	context := (&clusterBuilder{
		push:     push,
		hostname: hostName,
		svc:      service,
	}).buildUpstreamTLSContext(&cluster.Cluster{Name: "outbound|8080||" + string(hostName)})
	got := context.GetCommonTlsContext().
		GetCombinedValidationContext().
		GetDefaultValidationContext().
		GetMatchSubjectAltNames()

	activatorSANs := push.ActivationGatewaySANs("app")
	if len(activatorSANs) != 1 {
		t.Fatalf("Activator SANs = %v, want one identity", activatorSANs)
	}
	if !contains(got, backendSAN) {
		t.Fatalf("SAN pins = %v, want backend %q", got, backendSAN)
	}
	if !contains(got, activatorSANs[0]) {
		t.Fatalf("SAN pins = %v, want Activator %q", got, activatorSANs[0])
	}
}

func TestStrictPeerAuthenticationEmitsActivationSANPinnedMTLSCluster(t *testing.T) {
	hostName := host.Name("payment.app.svc.cluster.local")
	service := newRDSTestService("payment", "app", string(hostName), 8080)
	backendSAN := "spiffe://cluster.local/ns/app/sa/payment"
	push := newRDSTestPushContext(t, []config.Config{
		newActivationPolicyConfig("payment", "app", "payment"),
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

	resources := (&GrpcConfigGenerator{}).BuildClusters(&model.Proxy{
		ID:              "proxyless~10.0.0.2~caller.app~app.svc.cluster.local",
		Type:            model.Proxyless,
		ConfigNamespace: "app",
	}, push, []string{"outbound|8080||" + string(hostName)})
	if len(resources) != 1 {
		t.Fatalf("resources = %d, want 1", len(resources))
	}
	generated := &cluster.Cluster{}
	if err := resources[0].GetResource().UnmarshalTo(generated); err != nil {
		t.Fatalf("unmarshal cluster: %v", err)
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
	if !contains(got, backendSAN) || !contains(got, push.ActivationGatewaySANs("app")[0]) {
		t.Fatalf("SAN pins = %v, want backend and Activator identities", got)
	}
}

func TestBuildClustersExternalNameBackendTLSPolicyUsesSimpleTLS(t *testing.T) {
	hostName := "httpbin-egress.app.svc.cluster.local"
	service := newRDSTestService("httpbin-egress", "app", hostName, 443)
	service.Resolution = model.Alias
	service.Attributes.ExternalName = "httpbin.org"
	push := newRDSTestPushContext(t, []config.Config{
		newBackendTLSPolicyConfig("httpbin-tls", "app", "httpbin-egress", "httpbin.org"),
	}, []*model.Service{service})

	resources := (&GrpcConfigGenerator{}).BuildClusters(&model.Proxy{
		ID:   "router~10.0.0.1~dxgate.app~app.svc.cluster.local",
		Type: model.Router,
	}, push, []string{"outbound|443||" + hostName})

	if len(resources) != 1 {
		t.Fatalf("resources = %d, want 1", len(resources))
	}
	c := &cluster.Cluster{}
	if err := resources[0].GetResource().UnmarshalTo(c); err != nil {
		t.Fatalf("unmarshal cluster: %v", err)
	}
	if c.GetTransportSocket() == nil {
		t.Fatal("transport socket is nil")
	}
	tlsContext := &tlsv1.UpstreamTlsContext{}
	if err := c.GetTransportSocket().GetTypedConfig().UnmarshalTo(tlsContext); err != nil {
		t.Fatalf("unmarshal upstream tls context: %v", err)
	}
	if tlsContext.GetSni() != "httpbin.org" {
		t.Fatalf("SNI = %q, want httpbin.org", tlsContext.GetSni())
	}
	if tlsContext.GetCommonTlsContext().GetTlsCertificateCertificateProviderInstance() != nil {
		t.Fatal("simple TLS should not configure a workload certificate provider")
	}
}

func newBackendTLSPolicyConfig(name, namespace, serviceName, hostname string) config.Config {
	wellKnown := gatewayv1.WellKnownCACertificatesSystem
	return config.Config{
		Meta: config.Meta{
			GroupVersionKind:  gvk.BackendTLSPolicy,
			Name:              name,
			Namespace:         namespace,
			CreationTimestamp: time.Unix(1, 0),
		},
		Spec: &gatewayv1.BackendTLSPolicySpec{
			TargetRefs: []gatewayv1.LocalPolicyTargetReferenceWithSectionName{
				{
					LocalPolicyTargetReference: gatewayv1.LocalPolicyTargetReference{
						Group: gatewayv1.Group(""),
						Kind:  gatewayv1.Kind("Service"),
						Name:  gatewayv1.ObjectName(serviceName),
					},
				},
			},
			Validation: gatewayv1.BackendTLSPolicyValidation{
				WellKnownCACertificates: &wellKnown,
				Hostname:                gatewayv1.PreciseHostname(hostname),
			},
		},
	}
}

func newActivationPolicyConfig(name, namespace, serviceName string) config.Config {
	return config.Config{
		Meta: config.Meta{
			GroupVersionKind: gvk.ServiceActivationPolicy,
			Name:             name,
			Namespace:        namespace,
		},
		Spec: &networking.ServiceActivationPolicy{
			TargetRef: &networking.PolicyTargetReference{
				Kind: "Service",
				Name: serviceName,
			},
			AutoscalerRef: &networking.AutoscalerReference{Name: serviceName},
			BackendServiceAccounts: []string{
				"payment",
			},
		},
	}
}
