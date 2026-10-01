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

package render

import (
	"strings"
	"testing"

	"github.com/apache/dubbo-kubernetes/operator/pkg/manifest"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func TestGenerateManifestUsesEmbeddedInstallerAssets(t *testing.T) {
	manifests, _, err := GenerateManifest(nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("GenerateManifest() error = %v", err)
	}
	if len(manifests) == 0 {
		t.Fatal("GenerateManifest() returned zero manifest sets")
	}
}

// TestGenerateManifestDefaultsToHighlyAvailableControlPlane pins the default
// posture: a single dubbod makes every node drain or upgrade a control-plane
// outage, during which no pod can be injected and no xDS update is served.
func TestGenerateManifestDefaultsToHighlyAvailableControlPlane(t *testing.T) {
	manifests, _, err := GenerateManifest(nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("GenerateManifest() error = %v", err)
	}

	deployment := findManifest(t, manifests, "Deployment", "dubbod")
	replicas, ok, err := unstructured.NestedInt64(deployment.Object, "spec", "replicas")
	if err != nil || !ok {
		t.Fatalf("replicas missing: ok=%v err=%v", ok, err)
	}
	if replicas < 2 {
		t.Fatalf("dubbod replicas = %d, want at least 2", replicas)
	}

	// The budget only protects the control plane if it is actually rendered at
	// the default replica count.
	budget := findManifest(t, manifests, "PodDisruptionBudget", "dubbod")
	minAvailable, ok, err := unstructured.NestedInt64(budget.Object, "spec", "minAvailable")
	if err != nil || !ok {
		t.Fatalf("minAvailable missing: ok=%v err=%v", ok, err)
	}
	if minAvailable != 1 {
		t.Fatalf("dubbod PodDisruptionBudget minAvailable = %d, want 1", minAvailable)
	}

	constraints, ok, err := unstructured.NestedSlice(deployment.Object, "spec", "template", "spec", "topologySpreadConstraints")
	if err != nil || !ok || len(constraints) == 0 {
		t.Fatalf("topologySpreadConstraints missing: ok=%v err=%v", ok, err)
	}
	for _, raw := range constraints {
		constraint := raw.(map[string]interface{})
		// Spreading must stay advisory, or a single-node cluster leaves the
		// second replica permanently Pending.
		if policy, _, _ := unstructured.NestedString(constraint, "whenUnsatisfiable"); policy != "ScheduleAnyway" {
			t.Fatalf("whenUnsatisfiable = %q, want ScheduleAnyway", policy)
		}
	}
}

// TestGenerateManifestSingleReplicaOmitsDisruptionBudget guards the other
// direction: a budget of minAvailable 1 over a single replica blocks every
// voluntary eviction, so node drains hang instead of proceeding.
func TestGenerateManifestSingleReplicaOmitsDisruptionBudget(t *testing.T) {
	manifests, _, err := GenerateManifest(nil, []string{"values.replicaCount=1"}, nil, nil)
	if err != nil {
		t.Fatalf("GenerateManifest() error = %v", err)
	}
	for _, set := range manifests {
		for _, item := range set.Manifests {
			if item.GetKind() == "PodDisruptionBudget" && item.GetName() == "dubbod" {
				t.Fatal("PodDisruptionBudget rendered for a single-replica control plane")
			}
		}
	}
}

// TestGenerateManifestPassesGatewayReplicaDefault walks the whole chain the
// gateway replica default travels: chart values, the operator API schema, and
// the environment variable dubbod reads at runtime. A break anywhere in it is
// silent, because the controller simply falls back to its own constant.
func TestGenerateManifestPassesGatewayReplicaDefault(t *testing.T) {
	for _, test := range []struct {
		name string
		set  []string
		want string
	}{
		{name: "default", want: "2"},
		{name: "override", set: []string{"values.global.gateway.replicaCount=3"}, want: "3"},
	} {
		t.Run(test.name, func(t *testing.T) {
			manifests, _, err := GenerateManifest(nil, test.set, nil, nil)
			if err != nil {
				t.Fatalf("GenerateManifest() error = %v", err)
			}
			deployment := findManifest(t, manifests, "Deployment", "dubbod")
			containers, ok, err := unstructured.NestedSlice(deployment.Object, "spec", "template", "spec", "containers")
			if err != nil || !ok || len(containers) == 0 {
				t.Fatalf("deployment containers missing: ok=%v err=%v", ok, err)
			}
			env, ok, err := unstructured.NestedSlice(containers[0].(map[string]interface{}), "env")
			if err != nil || !ok {
				t.Fatalf("container env missing: ok=%v err=%v", ok, err)
			}
			for _, raw := range env {
				entry := raw.(map[string]interface{})
				if name, _, _ := unstructured.NestedString(entry, "name"); name != "DUBBO_TRANSIT_REPLICAS" {
					continue
				}
				if value, _, _ := unstructured.NestedString(entry, "value"); value != test.want {
					t.Fatalf("DUBBO_TRANSIT_REPLICAS = %q, want %q", value, test.want)
				}
				return
			}
			t.Fatal("DUBBO_TRANSIT_REPLICAS not rendered")
		})
	}
}

func TestTelemetryValidationWebhookDoesNotRequireRevisionLabel(t *testing.T) {
	manifests, _, err := GenerateManifest(nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("GenerateManifest() error = %v", err)
	}
	webhook := findManifest(t, manifests, "ValidatingWebhookConfiguration", "dubbo-validator-dubbo-system")
	webhooks, ok, err := unstructured.NestedSlice(webhook.Object, "webhooks")
	if err != nil || !ok {
		t.Fatalf("webhooks missing: ok=%v err=%v", ok, err)
	}
	for _, raw := range webhooks {
		entry := raw.(map[string]interface{})
		name, _, _ := unstructured.NestedString(entry, "name")
		if name != "telemetry.validation.dubbo.apache.org" {
			continue
		}
		if _, found := entry["objectSelector"]; found {
			t.Fatal("Telemetry validation webhook must match resources without revision labels")
		}
		return
	}
	t.Fatal("Telemetry validation webhook not rendered")
}

func hasArgValue(args []string, flag, value string) bool {
	for i := 0; i < len(args)-1; i++ {
		if args[i] == flag && args[i+1] == value {
			return true
		}
	}
	return false
}

func TestGenerateManifestExposesDubbodPrometheusScrapeEndpoint(t *testing.T) {
	manifests, _, err := GenerateManifest(nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("GenerateManifest() error = %v", err)
	}

	for _, set := range manifests {
		for _, m := range set.Manifests {
			if m.GetKind() != "Service" || m.GetName() != "dubbod" || m.GetNamespace() != "dubbo-system" {
				continue
			}
			annotations := m.GetAnnotations()
			if annotations["prometheus.io/scrape"] != "true" {
				t.Fatalf("prometheus.io/scrape = %q, want true", annotations["prometheus.io/scrape"])
			}
			if annotations["prometheus.io/path"] != "/metrics" {
				t.Fatalf("prometheus.io/path = %q, want /metrics", annotations["prometheus.io/path"])
			}
			if annotations["prometheus.io/port"] != "8080" {
				t.Fatalf("prometheus.io/port = %q, want 8080", annotations["prometheus.io/port"])
			}

			ports, ok, err := unstructured.NestedSlice(m.Object, "spec", "ports")
			if err != nil || !ok {
				t.Fatalf("spec.ports missing: ok=%v err=%v", ok, err)
			}
			for _, port := range ports {
				p, ok := port.(map[string]interface{})
				if !ok {
					continue
				}
				name, _, _ := unstructured.NestedString(p, "name")
				number, _, _ := unstructured.NestedInt64(p, "port")
				if name == "http-monitoring" && number == 8080 {
					return
				}
			}
			t.Fatal("dubbod Service missing http-monitoring port 8080")
		}
	}
	t.Fatal("dubbod Service not rendered")
}

func TestGenerateManifestSetOverridesHelmValues(t *testing.T) {
	manifests, _, err := GenerateManifest(nil, []string{
		"values.global.management.port=26081",
		"values.global.proxy.clusterDomain=example.local",
		"values.global.statusPort=26021",
		"values.global.configValidation=false",
		"values.revision=canary",
	}, nil, nil)
	if err != nil {
		t.Fatalf("GenerateManifest() error = %v", err)
	}

	managementService := findManifest(t, manifests, "Service", "dubbod-management")
	managementPort := findPort(t, managementService, "management")
	port, _, _ := unstructured.NestedInt64(managementPort, "port")
	targetPort, _, _ := unstructured.NestedInt64(managementPort, "targetPort")
	if port != 26081 || targetPort != 26081 {
		t.Fatalf("dubbod-management port=%d targetPort=%d, want 26081/26081", port, targetPort)
	}

	deployment := findManifest(t, manifests, "Deployment", "dubbod")
	containers, ok, err := unstructured.NestedSlice(deployment.Object, "spec", "template", "spec", "containers")
	if err != nil || !ok || len(containers) == 0 {
		t.Fatalf("deployment containers missing: ok=%v err=%v", ok, err)
	}
	execute := containers[0].(map[string]interface{})
	containerPorts, ok, err := unstructured.NestedSlice(execute, "ports")
	if err != nil || !ok {
		t.Fatalf("container ports missing: ok=%v err=%v", ok, err)
	}
	if !hasContainerPort(containerPorts, "management", 26081) {
		t.Fatal("dubbod deployment missing management containerPort 26081")
	}

	configMap := findManifest(t, manifests, "ConfigMap", "dubbo")
	if rev := configMap.GetLabels()["dubbo.apache.org/rev"]; rev != "canary" {
		t.Fatalf("dubbo configmap revision label = %q, want canary", rev)
	}
	mesh, _, _ := unstructured.NestedString(configMap.Object, "data", "mesh")
	for _, want := range []string{"statusPort: 26021", "trustDomain: example.local"} {
		if !strings.Contains(mesh, want) {
			t.Fatalf("mesh config missing %q:\n%s", want, mesh)
		}
	}
	if hasManifest(manifests, "ValidatingWebhookConfiguration", "dubbo-validator-dubbo-system") {
		t.Fatal("validating webhook rendered even though values.global.configValidation=false")
	}
}

func TestGenerateManifestRejectsRemovedInstallSurface(t *testing.T) {
	tests := []struct {
		name string
		set  string
	}{
		{
			name: "removed component",
			set:  "components.nacos.enabled=true",
		},
		{
			name: "removed helm value",
			set:  "values.nacos.enabled=true",
		},
		{
			name: "removed values profile",
			set:  "values.profile=demo",
		},
		{
			name: "removed base value",
			set:  "values.base.enableDubboConfigCRD=false",
		},
		{
			name: "removed resources value",
			set:  "values.resources.requests.cpu=100m",
		},
		{
			name: "removed injector webhook value",
			set:  "values.injectorWebhook.defaultTemplates=grpc-engine",
		},
		{
			name: "removed global namespace value",
			set:  "values.global.dubboNamespace=custom-system",
		},
		{
			name: "removed proxy image value",
			set:  "values.global.proxy.image=dubml/dubbod:test",
		},
		{
			name: "removed inherent value",
			set:  "values.global.inherent.cni.enabled=false",
		},
		{
			name: "removed prometheus component",
			set:  "components.prometheus.enabled=true",
		},
		{
			name: "removed grafana component",
			set:  "components.grafana.enabled=true",
		},
		{
			name: "removed tracing component",
			set:  "components.tracing.enabled=true",
		},
		{
			name: "removed otel collector component",
			set:  "components.otelCollector.enabled=true",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, _, err := GenerateManifest(nil, []string{test.set}, nil, nil)
			if err == nil {
				t.Fatal("GenerateManifest() returned nil error")
			}
			if !strings.Contains(err.Error(), "unmarshal") {
				t.Fatalf("GenerateManifest() error = %v, want unmarshal error", err)
			}
		})
	}
}

func findManifest(t *testing.T, manifests []manifest.ManifestSet, kind, name string) manifest.Manifest {
	t.Helper()
	if m, ok := findOptionalManifest(manifests, kind, name); ok {
		return m
	}
	t.Fatalf("%s %s not rendered", kind, name)
	return manifest.Manifest{}
}

func hasManifest(manifests []manifest.ManifestSet, kind, name string) bool {
	_, ok := findOptionalManifest(manifests, kind, name)
	return ok
}

func findOptionalManifest(manifests []manifest.ManifestSet, kind, name string) (manifest.Manifest, bool) {
	for _, set := range manifests {
		for _, m := range set.Manifests {
			if m.GetKind() == kind && m.GetName() == name && (m.GetNamespace() == "" || m.GetNamespace() == "dubbo-system") {
				return m, true
			}
		}
	}
	return manifest.Manifest{}, false
}

func findPort(t *testing.T, m manifest.Manifest, name string) map[string]interface{} {
	t.Helper()
	ports, ok, err := unstructured.NestedSlice(m.Object, "spec", "ports")
	if err != nil || !ok {
		t.Fatalf("%s/%s spec.ports missing: ok=%v err=%v", m.GetKind(), m.GetName(), ok, err)
	}
	for _, port := range ports {
		p, ok := port.(map[string]interface{})
		if !ok {
			continue
		}
		gotName, _, _ := unstructured.NestedString(p, "name")
		if gotName == name {
			return p
		}
	}
	t.Fatalf("%s/%s missing port %s", m.GetKind(), m.GetName(), name)
	return nil
}

func hasContainerPort(ports []interface{}, name string, number int64) bool {
	for _, port := range ports {
		p, ok := port.(map[string]interface{})
		if !ok {
			continue
		}
		gotName, _, _ := unstructured.NestedString(p, "name")
		gotNumber, _, _ := unstructured.NestedInt64(p, "containerPort")
		if gotName == name && gotNumber == number {
			return true
		}
	}
	return false
}
