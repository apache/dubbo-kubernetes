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

package gateway

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/apache/dubbo-kubernetes/pkg/config"
	"github.com/apache/dubbo-kubernetes/pkg/config/schema/gvk"
	"github.com/apache/dubbo-kubernetes/pkg/kube/inject"
	"github.com/google/go-cmp/cmp"
	networking "github.com/kdubbo/api/networking/v1alpha3"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/wrapperspb"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/yaml"

	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"
)

func TestBuildDxgateRuntimeConfigFromHTTPRoute(t *testing.T) {
	pathType := gatewayv1.PathMatchPathPrefix
	path := "/orders"
	backendPort := gatewayv1.PortNumber(8080)
	weight80 := int32(80)
	weight20 := int32(20)
	hostname := gatewayv1.Hostname("api.example.com")
	sectionName := gatewayv1.SectionName("http")

	gw := gatewayv1.Gateway{
		ObjectMeta: metav1.ObjectMeta{
			Name:            "public",
			Namespace:       "app",
			ResourceVersion: "10",
		},
		Spec: gatewayv1.GatewaySpec{
			GatewayClassName: "dubbo",
			Listeners: []gatewayv1.Listener{
				{
					Name:     "http",
					Protocol: gatewayv1.HTTPProtocolType,
					Port:     80,
				},
			},
		},
	}
	route := &gatewayv1.HTTPRoute{
		ObjectMeta: metav1.ObjectMeta{
			Name:            "orders",
			Namespace:       "app",
			ResourceVersion: "20",
		},
		Spec: gatewayv1.HTTPRouteSpec{
			CommonRouteSpec: gatewayv1.CommonRouteSpec{
				ParentRefs: []gatewayv1.ParentReference{
					{
						Name:        "public",
						SectionName: &sectionName,
					},
				},
			},
			Hostnames: []gatewayv1.Hostname{hostname},
			Rules: []gatewayv1.HTTPRouteRule{
				{
					Matches: []gatewayv1.HTTPRouteMatch{
						{
							Path: &gatewayv1.HTTPPathMatch{
								Type:  &pathType,
								Value: &path,
							},
							Headers: []gatewayv1.HTTPHeaderMatch{
								{
									Name:  "x-env",
									Value: "prod",
								},
							},
						},
					},
					BackendRefs: []gatewayv1.HTTPBackendRef{
						{
							BackendRef: gatewayv1.BackendRef{
								BackendObjectReference: gatewayv1.BackendObjectReference{
									Name: "orders-v1",
									Port: &backendPort,
								},
								Weight: &weight80,
							},
						},
						{
							BackendRef: gatewayv1.BackendRef{
								BackendObjectReference: gatewayv1.BackendObjectReference{
									Name: "orders-v2",
									Port: &backendPort,
								},
								Weight: &weight20,
							},
						},
					},
				},
			},
		},
	}

	raw, hash, err := buildDxgateRuntimeConfig(gw, []*gatewayv1.HTTPRoute{route}, nil, "cluster.local")
	if err != nil {
		t.Fatal(err)
	}
	if hash == "" {
		t.Fatal("expected runtime config hash")
	}

	var cfg dxgateRuntimeConfig
	if err := yaml.Unmarshal([]byte(raw), &cfg); err != nil {
		t.Fatal(err)
	}

	if diff := cmp.Diff([]string{"api.example.com"}, cfg.Listeners[0].VirtualHosts[0].Domains); diff != "" {
		t.Fatalf("unexpected domains (-want +got):\n%s", diff)
	}
	if got := cfg.Listeners[0].Bind; got != "0.0.0.0:80" {
		t.Fatalf("unexpected listener bind: %s", got)
	}
	routeCfg := cfg.Listeners[0].VirtualHosts[0].Routes[0]
	if diff := cmp.Diff([]dxgateWeightedCluster{
		{Name: "app-orders-0-0", Weight: 80},
		{Name: "app-orders-0-1", Weight: 20},
	}, routeCfg.WeightedClusters); diff != "" {
		t.Fatalf("unexpected weighted clusters (-want +got):\n%s", diff)
	}
	if got := cfg.Clusters[0].Endpoints[0].Address; got != "orders-v1.app.svc.cluster.local" {
		t.Fatalf("unexpected backend address: %s", got)
	}
	if got := routeCfg.Matches[0].Path; got != (dxgatePathMatch{Type: "prefix", Value: "/orders"}) {
		t.Fatalf("unexpected path match: %#v", got)
	}
	if got := routeCfg.Matches[0].Headers; len(got) != 1 || got[0].Name != "x-env" || got[0].Value != "prod" {
		t.Fatalf("unexpected header matches: %#v", got)
	}
}

func TestBuildDxgateRuntimeConfigAppliesCircuitBreakerPolicy(t *testing.T) {
	backendPort := gatewayv1.PortNumber(9080)
	gw := gatewayv1.Gateway{
		ObjectMeta: metav1.ObjectMeta{Name: "public", Namespace: "app", ResourceVersion: "10"},
		Spec: gatewayv1.GatewaySpec{
			GatewayClassName: "dubbo",
			Listeners: []gatewayv1.Listener{
				{Name: "http", Protocol: gatewayv1.HTTPProtocolType, Port: 80},
			},
		},
	}
	route := &gatewayv1.HTTPRoute{
		ObjectMeta: metav1.ObjectMeta{Name: "reviews", Namespace: "app", ResourceVersion: "20"},
		Spec: gatewayv1.HTTPRouteSpec{
			CommonRouteSpec: gatewayv1.CommonRouteSpec{
				ParentRefs: []gatewayv1.ParentReference{{Name: "public"}},
			},
			Rules: []gatewayv1.HTTPRouteRule{
				{
					BackendRefs: []gatewayv1.HTTPBackendRef{
						{
							BackendRef: gatewayv1.BackendRef{
								BackendObjectReference: gatewayv1.BackendObjectReference{
									Name: "reviews",
									Port: &backendPort,
								},
							},
						},
					},
				},
			},
		},
	}
	policies := []config.Config{
		{
			Meta: config.Meta{
				GroupVersionKind:  gvk.CircuitBreakerPolicy,
				Name:              "reviews-circuit-breaker",
				Namespace:         "app",
				CreationTimestamp: time.Unix(1, 0),
			},
			Spec: &networking.CircuitBreakerPolicy{
				TargetRefs: []*networking.PolicyTargetReference{
					{Kind: "Service", Name: "reviews"},
				},
				ConnectionPool: &networking.ConnectionPoolSettings{
					MaxConnections:          1,
					Http1MaxPendingRequests: 1,
					Http2MaxRequests:        2,
					MaxRetries:              3,
				},
				OutlierDetection: &networking.OutlierDetection{
					Consecutive_5XxErrors: wrapperspb.UInt32(1),
					Interval:              durationpb.New(time.Second),
					BaseEjectionTime:      durationpb.New(30 * time.Second),
					MaxEjectionPercent:    100,
				},
			},
		},
	}

	raw, _, err := buildDxgateRuntimeConfig(gw, []*gatewayv1.HTTPRoute{route}, policies, "cluster.local")
	if err != nil {
		t.Fatal(err)
	}
	var cfg dxgateRuntimeConfig
	if err := yaml.Unmarshal([]byte(raw), &cfg); err != nil {
		t.Fatal(err)
	}
	if len(cfg.Clusters) != 1 {
		t.Fatalf("clusters = %d, want 1", len(cfg.Clusters))
	}
	cb := cfg.Clusters[0].CircuitBreaker
	if cb == nil {
		t.Fatal("cluster circuit_breaker is nil")
	}
	if cb.MaxConnections != 1 || cb.HTTP1MaxPendingRequests != 1 || cb.HTTP2MaxRequests != 2 || cb.MaxRetries != 3 {
		t.Fatalf("unexpected circuit breaker: %#v", cb)
	}
	outlier := cfg.Clusters[0].Outlier
	if outlier == nil {
		t.Fatal("cluster outlier_detection is nil")
	}
	if outlier.Consecutive5xxErrors != 1 || outlier.Interval != "1s" || outlier.BaseEjectionTime != "30s" || outlier.MaxEjectionPercent != 100 {
		t.Fatalf("unexpected outlier detection: %#v", outlier)
	}
}

func TestBuildDxgateRuntimeConfigFiltersUnattachedHTTPRoutes(t *testing.T) {
	backendPort := gatewayv1.PortNumber(8080)
	gw := gatewayv1.Gateway{
		ObjectMeta: metav1.ObjectMeta{Name: "public", Namespace: "app"},
		Spec: gatewayv1.GatewaySpec{
			GatewayClassName: "dubbo",
			Listeners: []gatewayv1.Listener{
				{Name: "http", Protocol: gatewayv1.HTTPProtocolType, Port: 80},
			},
		},
	}
	route := &gatewayv1.HTTPRoute{
		ObjectMeta: metav1.ObjectMeta{Name: "orders", Namespace: "app"},
		Spec: gatewayv1.HTTPRouteSpec{
			CommonRouteSpec: gatewayv1.CommonRouteSpec{
				ParentRefs: []gatewayv1.ParentReference{{Name: "other"}},
			},
			Rules: []gatewayv1.HTTPRouteRule{
				{
					BackendRefs: []gatewayv1.HTTPBackendRef{
						{
							BackendRef: gatewayv1.BackendRef{
								BackendObjectReference: gatewayv1.BackendObjectReference{Name: "orders", Port: &backendPort},
							},
						},
					},
				},
			},
		},
	}

	raw, _, err := buildDxgateRuntimeConfig(gw, []*gatewayv1.HTTPRoute{route}, nil, "cluster.local")
	if err != nil {
		t.Fatal(err)
	}
	var cfg dxgateRuntimeConfig
	if err := yaml.Unmarshal([]byte(raw), &cfg); err != nil {
		t.Fatal(err)
	}
	if len(cfg.Clusters) != 0 {
		t.Fatalf("expected no clusters for unattached route, got %#v", cfg.Clusters)
	}
}

func TestBuildDxgateBootstrapConfig(t *testing.T) {
	raw, hash, err := buildDxgateBootstrapConfig(
		"http://dubbod.dubbo-system.svc:26010",
		[]string{"public-dubbo.app.svc.cluster.local:80"},
		"Kubernetes",
		"cluster.local",
	)
	if err != nil {
		t.Fatal(err)
	}
	if hash == "" {
		t.Fatal("expected bootstrap config hash")
	}

	var cfg dxgateBootstrapConfig
	if err := json.Unmarshal([]byte(raw), &cfg); err != nil {
		t.Fatal(err)
	}
	if cfg.XDSAddress != "http://dubbod.dubbo-system.svc:26010" {
		t.Fatalf("unexpected xDS address: %s", cfg.XDSAddress)
	}
	if diff := cmp.Diff([]string{"public-dubbo.app.svc.cluster.local:80"}, cfg.ListenerNames); diff != "" {
		t.Fatalf("unexpected listener names (-want +got):\n%s", diff)
	}
	if cfg.ClusterID != "Kubernetes" || cfg.DNSDomain != "cluster.local" {
		t.Fatalf("unexpected bootstrap identity: %#v", cfg)
	}
}

func TestDeploymentControllerBuildDxgateBootstrapConfigUsesXDSAddressAnnotation(t *testing.T) {
	controller := &DeploymentController{
		clusterID:       "remote",
		systemNamespace: "dubbo-system",
	}
	raw, _, err := controller.buildDxgateBootstrapConfig(gatewayv1.Gateway{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "eastwest",
			Namespace: "dubbo-system",
			Annotations: map[string]string{
				xdsAddressAnnotation: "http://192.168.15.164:32010",
			},
		},
	}, "dxgate-gateway", []corev1.ServicePort{{Name: "http-eastwest", Port: 15443, TargetPort: intstr.FromInt(15080)}})
	if err != nil {
		t.Fatal(err)
	}

	var cfg dxgateBootstrapConfig
	if err := json.Unmarshal([]byte(raw), &cfg); err != nil {
		t.Fatal(err)
	}
	if cfg.XDSAddress != "http://192.168.15.164:32010" {
		t.Fatalf("xdsAddress = %q, want annotation", cfg.XDSAddress)
	}
	if cfg.ClusterID != "remote" {
		t.Fatalf("clusterID = %q, want remote", cfg.ClusterID)
	}
}

func TestExtractServicePortsTargetsDxgateContainerPorts(t *testing.T) {
	gw := gatewayv1.Gateway{
		Spec: gatewayv1.GatewaySpec{
			Listeners: []gatewayv1.Listener{
				{Name: "http", Protocol: gatewayv1.HTTPProtocolType, Port: 8080},
				{Name: "tls", Protocol: gatewayv1.TLSProtocolType, Port: 8443},
			},
		},
	}

	ports := extractServicePorts(gw)
	if len(ports) != 1 {
		t.Fatalf("expected only the http listener port, got %#v", ports)
	}
	if ports[0].Name != "http" || ports[0].Port != 8080 || ports[0].TargetPort.String() != "http" {
		t.Fatalf("unexpected http service port: %#v", ports[0])
	}
}

func TestExtractServicePortsTargetsGRPCInboundForEastWestGateway(t *testing.T) {
	gw := gatewayv1.Gateway{
		ObjectMeta: metav1.ObjectMeta{
			Annotations: map[string]string{
				eastWestGatewayAnnotation: "true",
				serviceNodePortAnnotation: "32443",
			},
		},
		Spec: gatewayv1.GatewaySpec{
			Listeners: []gatewayv1.Listener{
				{Name: "http-eastwest", Protocol: gatewayv1.HTTPProtocolType, Port: 15443},
			},
		},
	}

	ports := extractServicePorts(gw)
	if len(ports) != 1 {
		t.Fatalf("ports = %d, want 1", len(ports))
	}
	if ports[0].TargetPort.IntValue() != 15080 {
		t.Fatalf("targetPort = %s, want 15080", ports[0].TargetPort.String())
	}
	if ports[0].NodePort != 32443 {
		t.Fatalf("nodePort = %d, want 32443", ports[0].NodePort)
	}
}

func TestServiceTypeForGatewayUsesAnnotation(t *testing.T) {
	gw := gatewayv1.Gateway{
		ObjectMeta: metav1.ObjectMeta{
			Annotations: map[string]string{serviceTypeAnnotation: "NodePort"},
		},
	}

	if got := serviceTypeForGateway(gw, corev1.ServiceTypeLoadBalancer); got != corev1.ServiceTypeNodePort {
		t.Fatalf("serviceType = %s, want NodePort", got)
	}
}

func TestGetDefaultNameUsesFixedDxgateGatewayName(t *testing.T) {
	spec := &gatewayv1.GatewaySpec{GatewayClassName: "dubbo"}
	if got := getDefaultName("httpbin-gateway", spec, false); got != "dxgate-gateway" {
		t.Fatalf("default name = %q, want dxgate-gateway", got)
	}
	if got := getDefaultName("foo-gateway", spec, true); got != "dxgate-gateway" {
		t.Fatalf("default name with suffix disabled = %q, want dxgate-gateway", got)
	}
}

func TestGetLegacyDefaultNameKeepsOldGatewayDerivedName(t *testing.T) {
	spec := &gatewayv1.GatewaySpec{GatewayClassName: "dubbo"}
	if got := getLegacyDefaultName("httpbin-gateway", spec, false); got != "httpbin-gateway-dubbo" {
		t.Fatalf("legacy default name = %q, want httpbin-gateway-dubbo", got)
	}
}

func TestKubeGatewayTemplateRendersDxgateResources(t *testing.T) {
	templatePath := filepath.Join("..", "..", "..", "..", "..", "..", "manifests", "charts", "dubbod", "files", "kube-gateway.yaml")
	raw, err := os.ReadFile(templatePath)
	if err != nil {
		t.Fatal(err)
	}
	templates, err := inject.ParseTemplates(inject.RawTemplates{"gateway": string(raw)})
	if err != nil {
		t.Fatal(err)
	}

	controller := &DeploymentController{
		injectConfig: func() inject.Config {
			return inject.Config{Templates: templates}
		},
	}
	rendered, err := controller.render("gateway", TemplateInput{
		Gateway: &gatewayv1.Gateway{
			ObjectMeta: metav1.ObjectMeta{Name: "public", Namespace: "app"},
		},
		DeploymentName:      "public-dubbo",
		ServiceAccount:      "public-dubbo",
		Ports:               []corev1.ServicePort{{Name: "http", Port: 80, TargetPort: intstr.FromString("http")}},
		ServiceType:         corev1.ServiceTypeLoadBalancer,
		Revision:            "default",
		BootstrapConfig:     "{\n  \"xds_address\": \"http://dubbod.dubbo-system.svc:26010\",\n  \"listener_names\": [\n    \"public-dubbo.app.svc.cluster.local:80\"\n  ],\n  \"cluster_id\": \"Kubernetes\",\n  \"dns_domain\": \"cluster.local\"\n}\n",
		BootstrapConfigHash: "abc123",
		DxgateImage:         "kdubbo/dxgate:test",
		SystemNamespace:     "dubbo-system",
		ClusterID:           "Kubernetes",
		DomainSuffix:        "cluster.local",
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(rendered) != 4 {
		t.Fatalf("expected 4 rendered resources, got %d", len(rendered))
	}
	for i, doc := range rendered {
		var obj map[string]any
		if err := yaml.Unmarshal([]byte(doc), &obj); err != nil {
			t.Fatalf("rendered resource %d is invalid YAML: %v\n%s", i, err, doc)
		}
	}
	if strings.Contains(strings.Join(rendered, "\n---\n"), "{{") {
		t.Fatalf("hardcoded gateway template still contains template delimiters:\n%s", strings.Join(rendered, "\n---\n"))
	}
	if !strings.Contains(rendered[0], "bootstrap.json") || !strings.Contains(rendered[0], `"xds_address": "http://dubbod.dubbo-system.svc:26010"`) {
		t.Fatalf("configmap did not render dxgate bootstrap:\n%s", rendered[0])
	}
	if !strings.Contains(rendered[2], "image: kdubbo/dxgate:test") {
		t.Fatalf("deployment did not render dxgate image:\n%s", rendered[2])
	}
	if !strings.Contains(rendered[2], "DXGATE_BOOTSTRAP") || strings.Contains(rendered[2], "DXGATE_STATIC_CONFIG") {
		t.Fatalf("deployment did not switch from static config to bootstrap:\n%s", rendered[2])
	}
	if !strings.Contains(rendered[2], `proxyless.dubbo.apache.org/inject: "true"`) {
		t.Fatalf("deployment pod template did not enable proxyless injection for dxgate mTLS certs:\n%s", rendered[2])
	}
	if !strings.Contains(rendered[2], "inject.dubbo.apache.org/templates: grpc-engine") {
		t.Fatalf("deployment pod template did not request grpc-engine injection:\n%s", rendered[2])
	}
	if !strings.Contains(rendered[2], "app.kubernetes.io/instance: public-dubbo") {
		t.Fatalf("deployment did not render stable dxgate instance label:\n%s", rendered[2])
	}
	if !strings.Contains(rendered[3], "targetPort: http") {
		t.Fatalf("service did not target dxgate http port:\n%s", rendered[3])
	}
	if !strings.Contains(rendered[3], `proxyless.dubbo.apache.org/inject: "false"`) {
		t.Fatalf("service did not opt out of proxyless targetPort rewriting:\n%s", rendered[3])
	}
	if !strings.Contains(rendered[3], "app.kubernetes.io/instance: public-dubbo") {
		t.Fatalf("service did not render stable dxgate instance selector:\n%s", rendered[3])
	}
}
