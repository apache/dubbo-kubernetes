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

	telemetryconfig "github.com/apache/dubbo-kubernetes/pkg/config/telemetry"
	"github.com/apache/dubbo-kubernetes/pkg/kube/inject"
	apitelemetry "github.com/dubml/api/telemetry/v1alpha3"
	typeapi "github.com/dubml/api/type/v1alpha3"
	"github.com/google/go-cmp/cmp"
	"google.golang.org/protobuf/types/known/wrapperspb"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/yaml"

	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"
)

func TestBuildGatewayBootstrapConfig(t *testing.T) {
	raw, hash, err := buildGatewayBootstrapConfig(
		"https://dubbod.dubbo-system.svc:26012",
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

	var cfg gatewayBootstrapConfig
	if err := json.Unmarshal([]byte(raw), &cfg); err != nil {
		t.Fatal(err)
	}
	if cfg.XDSAddress != "https://dubbod.dubbo-system.svc:26012" {
		t.Fatalf("unexpected xDS address: %s", cfg.XDSAddress)
	}
	if diff := cmp.Diff([]string{"public-dubbo.app.svc.cluster.local:80"}, cfg.ListenerNames); diff != "" {
		t.Fatalf("unexpected listener names (-want +got):\n%s", diff)
	}
	if cfg.ClusterID != "Kubernetes" || cfg.DNSDomain != "cluster.local" {
		t.Fatalf("unexpected bootstrap identity: %#v", cfg)
	}
}

func TestManagedGatewayRequiresSecureADS(t *testing.T) {
	controller := &DeploymentController{systemNamespace: "dubbo-system"}
	raw, _, err := controller.buildGatewayBootstrapConfig(gatewayv1.Gateway{}, "gateway", nil)
	if err != nil || !strings.Contains(raw, "https://dubbod.dubbo-system.svc:26012") {
		t.Fatalf("managed gateway must default to authenticated ADS: %s, %v", raw, err)
	}
	if _, _, err := buildGatewayBootstrapConfig("http://dubbod.dubbo-system.svc:26010", nil, "", ""); err == nil {
		t.Fatal("managed gateway must reject plaintext ADS")
	}
}

func TestExtractServicePortsTargetsGRPCInbound(t *testing.T) {
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
	if ports[0].Name != "http" || ports[0].Port != 8080 || ports[0].TargetPort.IntValue() != 15080 {
		t.Fatalf("unexpected http service port: %#v", ports[0])
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

func TestGatewayReplicas(t *testing.T) {
	tests := []struct {
		name       string
		annotation string
		fallback   int32
		want       int32
	}{
		{name: "unset takes the installed default", fallback: 2, want: 2},
		{name: "installed default is honored", fallback: 5, want: 5},
		{name: "explicit count wins", annotation: "4", fallback: 2, want: 4},
		{name: "zero scales the gateway down deliberately", annotation: "0", fallback: 2, want: 0},
		{name: "negative falls back to the default", annotation: "-1", fallback: 2, want: 2},
		{name: "garbage falls back to the default", annotation: "many", fallback: 2, want: 2},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			meta := metav1.ObjectMeta{}
			if test.annotation != "" {
				meta.Annotations = map[string]string{replicasAnnotation: test.annotation}
			}
			got := gatewayReplicasWithDefault(gatewayv1.Gateway{ObjectMeta: meta}, test.fallback)
			if got != test.want {
				t.Fatalf("gatewayReplicasWithDefault() = %d, want %d", got, test.want)
			}
		})
	}
}

// TestInstalledGatewayReplicasRejectsUnusableSettings guards the mesh-wide
// default: zero would make every gateway default to serving no traffic at all,
// which is never what an operator means by "the default".
func TestInstalledGatewayReplicasRejectsUnusableSettings(t *testing.T) {
	if got := installedGatewayReplicas(); got < 1 {
		t.Fatalf("installedGatewayReplicas() = %d, want at least 1", got)
	}
	if got := gatewayReplicas(gatewayv1.Gateway{}); got < 1 {
		t.Fatalf("gatewayReplicas() with no annotation = %d, want at least 1", got)
	}
}

func TestObservabilityConfigForGatewayDefaults(t *testing.T) {
	cfg := resolveGatewayObservability(gatewayv1.Gateway{
		ObjectMeta: metav1.ObjectMeta{Name: "public", Namespace: "app"},
	}, "dubbo-system", nil)

	if cfg.OtelEndpoint != "" {
		t.Fatalf("otel endpoint = %q, want empty", cfg.OtelEndpoint)
	}
	if cfg.OtelServiceName != "gateway.app.public" {
		t.Fatalf("otel service name = %q", cfg.OtelServiceName)
	}
	if cfg.OtelSampling != "100" {
		t.Fatalf("otel sampling = %q", cfg.OtelSampling)
	}
	if cfg.AccessLog != "true" {
		t.Fatalf("access log = %q", cfg.AccessLog)
	}
	if cfg.AccessLogFormat != "text" {
		t.Fatalf("access log format = %q", cfg.AccessLogFormat)
	}
}

func TestObservabilityConfigForGatewayAccessLogAnnotations(t *testing.T) {
	cfg := resolveGatewayObservability(gatewayv1.Gateway{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "public",
			Namespace: "app",
			Annotations: map[string]string{
				accessLogAnnotation:       "false",
				accessLogFormatAnnotation: "json",
			},
		},
	}, "dubbo-system", nil)

	if cfg.OtelEndpoint != "" {
		t.Fatalf("otel endpoint = %q", cfg.OtelEndpoint)
	}
	if cfg.OtelServiceName != "gateway.app.public" {
		t.Fatalf("otel service name = %q", cfg.OtelServiceName)
	}
	if cfg.OtelSampling != "100" {
		t.Fatalf("otel sampling = %q", cfg.OtelSampling)
	}
	if cfg.AccessLog != "false" {
		t.Fatalf("access log = %q", cfg.AccessLog)
	}
	if cfg.AccessLogFormat != "json" {
		t.Fatalf("access log format = %q", cfg.AccessLogFormat)
	}
}

func TestObservabilityConfigForGatewayInvalidAccessLogAnnotationsFallBack(t *testing.T) {
	cfg := resolveGatewayObservability(gatewayv1.Gateway{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "public",
			Namespace: "app",
			Annotations: map[string]string{
				accessLogAnnotation:       "maybe",
				accessLogFormatAnnotation: "yaml",
			},
		},
	}, "dubbo-system", nil)
	if cfg.AccessLog != "true" {
		t.Fatalf("access log = %q, want true", cfg.AccessLog)
	}
	if cfg.AccessLogFormat != "text" {
		t.Fatalf("access log format = %q, want text", cfg.AccessLogFormat)
	}
}

func TestGetDefaultNameIsolatesGateways(t *testing.T) {
	spec := &gatewayv1.GatewaySpec{GatewayClassName: "dubbo"}
	if got := getDefaultName("test-gateway", spec, false); got != "test-gateway-dubbo" {
		t.Fatalf("derived name = %q, want test-gateway-dubbo", got)
	}
	if got := getDefaultName("public", spec, false); got != "public-dubbo" {
		t.Fatalf("derived name = %q, want public-dubbo", got)
	}
	if got := getDefaultName("private", spec, true); got != "private" {
		t.Fatalf("name with suffix disabled = %q, want private", got)
	}
}

func TestManagedGatewayResourceCleanupRequiresMatchingGateway(t *testing.T) {
	labels := map[string]string{
		"gateway.dubbo.apache.org/managed":       "dubbo.apache.org-gateway-controller",
		"gateway.networking.k8s.io/gateway-name": "public",
	}
	if !isManagedGatewayResourceFor(labels, "public") {
		t.Fatal("matching managed resource was not recognized")
	}
	if isManagedGatewayResourceFor(labels, "test-gateway") {
		t.Fatal("cleanup would delete another Gateway's resources")
	}
}

func TestKubeGatewayTemplateRendersGatewayResources(t *testing.T) {
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
	input := TemplateInput{
		Gateway: &gatewayv1.Gateway{
			ObjectMeta: metav1.ObjectMeta{Name: "public", Namespace: "app"},
		},
		DeploymentName:      "public-dubbo",
		ServiceAccount:      "public-dubbo",
		Ports:               []corev1.ServicePort{{Name: "http", Port: 80, TargetPort: intstr.FromInt(15080)}},
		ServiceType:         corev1.ServiceTypeLoadBalancer,
		Revision:            "default",
		BootstrapConfig:     "{\n  \"xds_address\": \"https://dubbod.dubbo-system.svc:26012\",\n  \"listener_names\": [\n    \"public-dubbo.app.svc.cluster.local:80\"\n  ],\n  \"cluster_id\": \"Kubernetes\",\n  \"dns_domain\": \"cluster.local\"\n}\n",
		BootstrapConfigHash: "abc123",
		GatewayImage:        "apache/dubbo-gateway:test",
		SystemNamespace:     "dubbo-system",
		ClusterID:           "Kubernetes",
		DomainSuffix:        "cluster.local",
		OtelServiceName:     "gateway.app.public",
		OtelSampling:        "100",
		AccessLog:           "true",
		AccessLogFormat:     "text",
	}
	rendered, err := controller.render("gateway", input)
	if err != nil {
		t.Fatal(err)
	}
	if len(rendered) != 6 {
		t.Fatalf("expected 6 rendered resources, got %d", len(rendered))
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
	if !strings.Contains(rendered[0], "bootstrap.json") || !strings.Contains(rendered[0], `"xds_address": "https://dubbod.dubbo-system.svc:26012"`) {
		t.Fatalf("configmap did not render gateway bootstrap:\n%s", rendered[0])
	}
	if !strings.Contains(rendered[1], "automountServiceAccountToken: true") ||
		!strings.Contains(rendered[2], "resources: [\"secrets\"]") ||
		!strings.Contains(rendered[3], "kind: RoleBinding") {
		t.Fatalf("credential Secret RBAC not rendered:\n%s", strings.Join(rendered[1:4], "\n---\n"))
	}
	if !strings.Contains(rendered[4], "image: apache/dubbo-gateway:test") {
		t.Fatalf("deployment did not render gateway image:\n%s", rendered[4])
	}
	if !strings.Contains(rendered[4], "GATEWAY_BOOTSTRAP") {
		t.Fatalf("deployment did not switch from static config to bootstrap:\n%s", rendered[4])
	}
	if !strings.Contains(rendered[4], `inherent.dubbo.apache.org/inject: "true"`) {
		t.Fatalf("deployment pod template did not enable inherent injection for gateway mTLS certs:\n%s", rendered[4])
	}
	if !strings.Contains(rendered[4], "inject.dubbo.apache.org/templates: grpc-engine") {
		t.Fatalf("deployment pod template did not request grpc-engine injection:\n%s", rendered[4])
	}
	if !strings.Contains(rendered[4], `prometheus.io/scrape: "true"`) ||
		!strings.Contains(rendered[4], "prometheus.io/path: /metrics") ||
		!strings.Contains(rendered[4], `prometheus.io/port: "26021"`) {
		t.Fatalf("deployment pod template did not render prometheus scrape annotations:\n%s", rendered[4])
	}
	if strings.Contains(rendered[4], "GATEWAY_OTEL_ENDPOINT") {
		t.Fatalf("deployment rendered OTEL endpoint without annotation:\n%s", rendered[4])
	}
	for _, want := range []string{
		"GATEWAY_NAME",
		`value: "public"`,
		"GATEWAY_OTEL_SERVICE_NAME",
		`value: "gateway.app.public"`,
		"GATEWAY_OTEL_SAMPLING_PERCENTAGE",
		`value: "100"`,
		"GATEWAY_ACCESS_LOG",
		"GATEWAY_ACCESS_LOG_FORMAT",
		`value: "text"`,
	} {
		if !strings.Contains(rendered[4], want) {
			t.Fatalf("deployment did not render observability env %q:\n%s", want, rendered[4])
		}
	}
	if !strings.Contains(rendered[4], "app.kubernetes.io/instance: public-dubbo") {
		t.Fatalf("deployment did not render stable gateway instance label:\n%s", rendered[4])
	}
	if !strings.Contains(rendered[4], "GATEWAY_HTTP_ADDR") ||
		!strings.Contains(rendered[4], "value: 0.0.0.0:15080") ||
		!strings.Contains(rendered[4], "containerPort: 15080") {
		t.Fatalf("deployment did not bind gateway to the managed gateway targetPort:\n%s", rendered[4])
	}
	if !strings.Contains(rendered[5], "targetPort: 15080") {
		t.Fatalf("service did not target the managed gateway listener:\n%s", rendered[5])
	}
	if !strings.Contains(rendered[5], `inherent.dubbo.apache.org/inject: "false"`) {
		t.Fatalf("service did not opt out of inherent targetPort rewriting:\n%s", rendered[5])
	}
	if !strings.Contains(rendered[5], "app.kubernetes.io/instance: public-dubbo") {
		t.Fatalf("service did not render stable gateway instance selector:\n%s", rendered[5])
	}

	input.OtelEndpoint = "http://tracing.dubbo-system.svc:4317"
	rendered, err = controller.render("gateway", input)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(rendered[4], "GATEWAY_OTEL_ENDPOINT") ||
		!strings.Contains(rendered[4], `value: "http://tracing.dubbo-system.svc:4317"`) {
		t.Fatalf("deployment did not render gateway OTEL endpoint:\n%s", rendered[4])
	}

}

// TestKubeGatewayTemplateRendersHighAvailabilityResources covers the path a
// real reconcile takes: gatewayReplicas defaults to 2, which must produce a
// disruption budget and spread the pods. At one replica the budget has to be
// omitted, or minAvailable 1 blocks every voluntary eviction of that pod.
func TestKubeGatewayTemplateRendersHighAvailabilityResources(t *testing.T) {
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
	input := TemplateInput{
		Gateway: &gatewayv1.Gateway{
			ObjectMeta: metav1.ObjectMeta{Name: "public", Namespace: "app"},
		},
		DeploymentName:      "public-dubbo",
		ServiceAccount:      "public-dubbo",
		Ports:               []corev1.ServicePort{{Name: "http", Port: 80, TargetPort: intstr.FromInt(15080)}},
		ServiceType:         corev1.ServiceTypeLoadBalancer,
		Replicas:            gatewayReplicas(gatewayv1.Gateway{}),
		Revision:            "default",
		BootstrapConfig:     "{}\n",
		BootstrapConfigHash: "abc123",
		GatewayImage:        "apache/dubbo-gateway:test",
	}

	rendered, err := controller.render("gateway", input)
	if err != nil {
		t.Fatal(err)
	}
	if len(rendered) != 7 {
		t.Fatalf("rendered %d resources, want 7 including credential RBAC and disruption budget", len(rendered))
	}
	joined := strings.Join(rendered, "\n---\n")
	for _, want := range []string{
		"replicas: 2",
		"kind: PodDisruptionBudget",
		"minAvailable: 1",
		"podAntiAffinity",
		"topologySpreadConstraints",
		"whenUnsatisfiable: ScheduleAnyway",
	} {
		if !strings.Contains(joined, want) {
			t.Fatalf("high availability output missing %q:\n%s", want, joined)
		}
	}

	input.Replicas = 1
	rendered, err = controller.render("gateway", input)
	if err != nil {
		t.Fatal(err)
	}
	if len(rendered) != 6 {
		t.Fatalf("rendered %d resources at one replica, want 6 without a disruption budget", len(rendered))
	}
	joined = strings.Join(rendered, "\n---\n")
	if strings.Contains(joined, "kind: PodDisruptionBudget") {
		t.Fatalf("disruption budget rendered for a single replica:\n%s", joined)
	}
	if strings.Contains(joined, "topologySpreadConstraints") {
		t.Fatalf("topology spread rendered for a single replica:\n%s", joined)
	}
}

func TestResolveGatewayObservabilityTelemetryHierarchy(t *testing.T) {
	resources := []telemetryconfig.Resource{
		{
			Name: "mesh-default", Namespace: "dubbo-system", CreationTimestamp: time.Unix(1, 0),
			Spec: &apitelemetry.Telemetry{Tracing: []*apitelemetry.Tracing{{
				Providers:                []*apitelemetry.Tracing_TracingProvider{{Name: "localtrace"}},
				Tags:                     []*apitelemetry.Tracing_Tag{{Name: "foo", Value: "bar"}},
				RandomSamplingPercentage: wrapperspb.Double(100),
			}}, Logging: []*apitelemetry.Logging{{
				Providers: []*apitelemetry.Logging_LoggingProvider{{Name: telemetryconfig.OTELLogProvider}},
				Match:     &apitelemetry.Logging_Match{Mode: apitelemetry.Logging_Match_SERVER},
				Filter:    &apitelemetry.Logging_Filter{Expression: "response.code >= 500"},
				Tags:      []*apitelemetry.Logging_Tag{{Name: "environment", Value: "test"}},
			}}},
		},
		{
			Name: "namespace-override", Namespace: "app", CreationTimestamp: time.Unix(2, 0),
			Spec: &apitelemetry.Telemetry{Tracing: []*apitelemetry.Tracing{{
				Tags:                     []*apitelemetry.Tracing_Tag{{Name: "userId", Value: "unknown"}},
				RandomSamplingPercentage: wrapperspb.Double(25),
			}}},
		},
	}
	gw := gatewayv1.Gateway{ObjectMeta: metav1.ObjectMeta{
		Name: "public", Namespace: "app", Labels: map[string]string{"app": "frontend"},
	}}

	cfg := resolveGatewayObservability(gw, "dubbo-system", resources)
	if want := "http://tracing.dubbo-system.svc:4317"; cfg.OtelEndpoint != want {
		t.Fatalf("otel endpoint = %q, want %q", cfg.OtelEndpoint, want)
	}
	if cfg.OtelSampling != "25" {
		t.Fatalf("otel sampling = %q, want 25", cfg.OtelSampling)
	}
	if cfg.OtelTags != `{"userId":"unknown"}` {
		t.Fatalf("otel tags = %q", cfg.OtelTags)
	}
	if cfg.OtelLogsEndpoint != "http://opentelemetry-collector.dubbo-system.svc:4317" ||
		cfg.AccessLog != "true" || cfg.AccessLogMode != "SERVER" ||
		cfg.AccessLogFilter != "response.code >= 500" ||
		cfg.AccessLogTags != `{"environment":"test"}` {
		t.Fatalf("access logging = %#v", cfg)
	}
	resources = append(resources, telemetryconfig.Resource{
		Name: "workload-override", Namespace: "app", CreationTimestamp: time.Unix(3, 0),
		Spec: &apitelemetry.Telemetry{
			Selector: &typeapi.WorkloadSelector{MatchLabels: map[string]string{"app": "frontend"}},
			Tracing:  []*apitelemetry.Tracing{{DisableSpanReporting: wrapperspb.Bool(true)}},
		},
	})
	if cfg := resolveGatewayObservability(gw, "dubbo-system", resources); cfg.OtelEndpoint != "" {
		t.Fatalf("otel endpoint = %q, want empty when workload disables reporting", cfg.OtelEndpoint)
	}
}
