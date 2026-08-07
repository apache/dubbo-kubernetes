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
				if name, _, _ := unstructured.NestedString(entry, "name"); name != "DUBBO_DXGATE_REPLICAS" {
					continue
				}
				if value, _, _ := unstructured.NestedString(entry, "value"); value != test.want {
					t.Fatalf("DUBBO_DXGATE_REPLICAS = %q, want %q", value, test.want)
				}
				return
			}
			t.Fatal("DUBBO_DXGATE_REPLICAS not rendered")
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

// The activation scaler is only reachable if the flag, the container port and
// the Service KEDA dials all agree on one number. Each is written in a
// different template, so a mismatch renders cleanly and fails only at runtime.
func TestGenerateManifestWiresActivationScalerEndToEnd(t *testing.T) {
	manifests, _, err := GenerateManifest(nil, []string{"values.global.activation.port=26031"}, nil, nil)
	if err != nil {
		t.Fatalf("GenerateManifest() error = %v", err)
	}

	deployment := findManifest(t, manifests, "Deployment", "dubbod")
	containers, ok, err := unstructured.NestedSlice(deployment.Object, "spec", "template", "spec", "containers")
	if err != nil || !ok || len(containers) == 0 {
		t.Fatalf("deployment containers missing: ok=%v err=%v", ok, err)
	}
	execute := containers[0].(map[string]interface{})

	args, _, _ := unstructured.NestedStringSlice(execute, "args")
	if !hasArgValue(args, "--activationAddr", ":26031") {
		t.Fatalf("args = %v, want --activationAddr :26031", args)
	}

	ports, ok, err := unstructured.NestedSlice(execute, "ports")
	if err != nil || !ok {
		t.Fatalf("container ports missing: ok=%v err=%v", ok, err)
	}
	if !hasContainerPort(ports, "activation", 26031) {
		t.Fatal("dubbod deployment missing activation containerPort 26031")
	}

	service := findManifest(t, manifests, "Service", "dubbod-activation")
	port, _, _ := unstructured.NestedInt64(findPort(t, service, "grpc-activation"), "port")
	if port != 26031 {
		t.Fatalf("dubbod-activation port = %d, want 26031", port)
	}

	// Gateways broadcast their demand to every replica, so they need a name
	// that resolves to all of them. A load-balanced Service would deliver each
	// report to one arbitrary replica, and KEDA's stream is on another.
	replicas := findManifest(t, manifests, "Service", "dubbod-activation-replicas")
	clusterIP, _, _ := unstructured.NestedString(replicas.Object, "spec", "clusterIP")
	if clusterIP != "None" {
		t.Fatalf("dubbod-activation-replicas clusterIP = %q, want None", clusterIP)
	}
	notReady, _, _ := unstructured.NestedBool(replicas.Object, "spec", "publishNotReadyAddresses")
	if !notReady {
		t.Fatal("dubbod-activation-replicas must publish not-ready addresses so a held request is not blocked on readiness")
	}
}

// Port zero is the documented off switch. It has to remove the listener, the
// port and the Service together; leaving a Service behind would publish an
// endpoint that refuses every connection.
func TestGenerateManifestDisablesActivationAtPortZero(t *testing.T) {
	manifests, _, err := GenerateManifest(nil, []string{"values.global.activation.port=0"}, nil, nil)
	if err != nil {
		t.Fatalf("GenerateManifest() error = %v", err)
	}
	if hasManifest(manifests, "Service", "dubbod-activation") {
		t.Fatal("dubbod-activation Service rendered while activation is disabled")
	}
	if hasManifest(manifests, "Service", "dubbod-activation-replicas") {
		t.Fatal("dubbod-activation-replicas Service rendered while activation is disabled")
	}

	deployment := findManifest(t, manifests, "Deployment", "dubbod")
	containers, _, _ := unstructured.NestedSlice(deployment.Object, "spec", "template", "spec", "containers")
	execute := containers[0].(map[string]interface{})
	args, _, _ := unstructured.NestedStringSlice(execute, "args")
	if !hasArgValue(args, "--activationAddr", "") {
		t.Fatalf("args = %v, want --activationAddr with an empty value", args)
	}
	ports, _, _ := unstructured.NestedSlice(execute, "ports")
	for _, raw := range ports {
		if name, _, _ := unstructured.NestedString(raw.(map[string]interface{}), "name"); name == "activation" {
			t.Fatal("activation containerPort rendered while activation is disabled")
		}
	}
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

func TestGenerateManifestRemoteAccessServiceIsOptional(t *testing.T) {
	manifests, _, err := GenerateManifest(nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("GenerateManifest() error = %v", err)
	}
	if hasManifest(manifests, "Service", "dubbod-remote") {
		t.Fatal("dubbod-remote rendered while remote access is disabled")
	}

	manifests, _, err = GenerateManifest(nil, []string{
		"values.global.multicluster.remoteAccess.enabled=true",
		"values.global.multicluster.remoteAccess.serviceType=NodePort",
		"values.global.multicluster.remoteAccess.grpcPort=32010",
		"values.global.multicluster.remoteAccess.xdsPort=26112",
		"values.global.multicluster.remoteAccess.webhookPort=30443",
	}, nil, nil)
	if err != nil {
		t.Fatalf("GenerateManifest() with remote access error = %v", err)
	}
	service := findManifest(t, manifests, "Service", "dubbod-remote")
	serviceType, _, _ := unstructured.NestedString(service.Object, "spec", "type")
	if serviceType != "NodePort" {
		t.Fatalf("dubbod-remote service type = %q, want NodePort", serviceType)
	}
	grpcPort, _, _ := unstructured.NestedInt64(findPort(t, service, "grpc-xds"), "port")
	xdsPort, _, _ := unstructured.NestedInt64(findPort(t, service, "tls-xds"), "port")
	webhookPort, _, _ := unstructured.NestedInt64(findPort(t, service, "https-webhook"), "port")
	if grpcPort != 32010 || xdsPort != 26112 || webhookPort != 30443 {
		t.Fatalf("dubbod-remote ports = %d/%d/%d, want 32010/26112/30443", grpcPort, xdsPort, webhookPort)
	}
}

func TestGenerateManifestRemoteAccessCertificateHosts(t *testing.T) {
	manifests, _, err := GenerateManifest(nil, []string{
		"values.global.multicluster.remoteAccess.enabled=true",
		"values.global.multicluster.remoteAccess.certificateHosts[0]=192.168.15.164",
		"values.global.multicluster.remoteAccess.certificateHosts[1]=dubbod.example.internal",
	}, nil, nil)
	if err != nil {
		t.Fatalf("GenerateManifest() with remote access certificate hosts error = %v", err)
	}

	deployment := findManifest(t, manifests, "Deployment", "dubbod")
	containers, ok, err := unstructured.NestedSlice(deployment.Object, "spec", "template", "spec", "containers")
	if err != nil || !ok || len(containers) == 0 {
		t.Fatalf("deployment containers missing: ok=%v err=%v", ok, err)
	}
	execute := containers[0].(map[string]interface{})
	env, ok, err := unstructured.NestedSlice(execute, "env")
	if err != nil || !ok {
		t.Fatalf("deployment env missing: ok=%v err=%v", ok, err)
	}
	for _, item := range env {
		entry, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
		name, _, _ := unstructured.NestedString(entry, "name")
		if name != "DUBBOD_CUSTOM_HOST" {
			continue
		}
		value, _, _ := unstructured.NestedString(entry, "value")
		if value != "192.168.15.164,dubbod.example.internal" {
			t.Fatalf("DUBBOD_CUSTOM_HOST = %q", value)
		}
		return
	}
	t.Fatal("dubbod deployment missing DUBBOD_CUSTOM_HOST")
}

func TestGenerateManifestEastWestGatewayConfig(t *testing.T) {
	manifests, _, err := GenerateManifest(nil, []string{
		"values.global.multicluster.eastWestGateway.enabled=true",
		"values.global.multicluster.eastWestGateway.serviceType=NodePort",
		"values.global.multicluster.eastWestGateway.port=15443",
		"values.global.multicluster.eastWestGateway.targetPort=15080",
		"values.global.multicluster.eastWestGateway.nodePort=32443",
		"values.global.multicluster.eastWestGateway.xdsAddress=http://192.168.15.164:32010",
		"values.global.multicluster.eastWestGateway.gateways[0].clusterName=remote",
		"values.global.multicluster.eastWestGateway.gateways[0].address=192.168.15.155",
		"values.global.multicluster.eastWestGateway.gateways[0].port=15443",
	}, nil, nil)
	if err != nil {
		t.Fatalf("GenerateManifest() with east-west gateway error = %v", err)
	}

	gateway := findManifest(t, manifests, "Gateway", "dubbod-eastwest-gateway")
	if got := gateway.GetAnnotations()["gateway.dubbo.apache.org/service-type"]; got != "NodePort" {
		t.Fatalf("east-west gateway service-type annotation = %q, want NodePort", got)
	}
	if got := gateway.GetAnnotations()["gateway.dubbo.apache.org/target-port"]; got != "15080" {
		t.Fatalf("east-west gateway target-port annotation = %q, want 15080", got)
	}
	if got := gateway.GetAnnotations()["gateway.dubbo.apache.org/node-port"]; got != "32443" {
		t.Fatalf("east-west gateway node-port annotation = %q, want 32443", got)
	}
	if got := gateway.GetAnnotations()["gateway.dubbo.apache.org/xds-address"]; got != "http://192.168.15.164:32010" {
		t.Fatalf("east-west gateway xds-address annotation = %q, want override", got)
	}
	listeners, ok, err := unstructured.NestedSlice(gateway.Object, "spec", "listeners")
	if err != nil || !ok || len(listeners) != 1 {
		t.Fatalf("gateway listeners missing: ok=%v err=%v", ok, err)
	}
	listener := listeners[0].(map[string]interface{})
	port, _, _ := unstructured.NestedInt64(listener, "port")
	if port != 15443 {
		t.Fatalf("east-west listener port = %d, want 15443", port)
	}

	deployment := findManifest(t, manifests, "Deployment", "dubbod")
	if got := deploymentEnvValue(t, deployment, "DUBBO_EASTWEST_GATEWAYS"); got != "remote=192.168.15.155:15443" {
		t.Fatalf("DUBBO_EASTWEST_GATEWAYS = %q", got)
	}
}

func TestGenerateManifestProxylessCNIIsGlobalByDefault(t *testing.T) {
	manifests, _, err := GenerateManifest(nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("GenerateManifest() error = %v", err)
	}
	if !hasManifest(manifests, "DaemonSet", "dubbo-cni-node") {
		t.Fatal("dubbo-cni-node not rendered by default")
	}

	manifests, _, err = GenerateManifest(nil, []string{"values.global.proxyless.cni.enabled=false"}, nil, nil)
	if err != nil {
		t.Fatalf("GenerateManifest() with proxyless CNI disabled error = %v", err)
	}
	if hasManifest(manifests, "DaemonSet", "dubbo-cni-node") {
		t.Fatal("dubbo-cni-node rendered while proxyless CNI is explicitly disabled")
	}

	manifests, _, err = GenerateManifest(nil, []string{
		"values.global.proxyless.cni.enabled=true",
		"values.global.proxyless.cni.image=kdubbo/dubbod:cni-test",
		"values.global.proxyless.cni.binDir=/var/lib/cni/bin",
		"values.global.proxyless.cni.confDir=/var/lib/cni/net.d",
		"values.global.proxyless.cni.stateDir=/run/dubbo-cni",
		"values.global.proxyless.cni.grpcInboundPort=16080",
		"values.global.proxyless.cni.ipsetPath=/usr/sbin/ipset",
		"values.global.proxyless.cni.refreshInterval=30s",
	}, nil, nil)
	if err != nil {
		t.Fatalf("GenerateManifest() with proxyless CNI error = %v", err)
	}
	daemonSet := findManifest(t, manifests, "DaemonSet", "dubbo-cni-node")
	containers, ok, err := unstructured.NestedSlice(daemonSet.Object, "spec", "template", "spec", "containers")
	if err != nil || !ok || len(containers) == 0 {
		t.Fatalf("daemonset containers missing: ok=%v err=%v", ok, err)
	}
	installer := containers[0].(map[string]interface{})
	image, _, _ := unstructured.NestedString(installer, "image")
	if image != "kdubbo/dubbod:cni-test" {
		t.Fatalf("CNI installer image = %q, want kdubbo/dubbod:cni-test", image)
	}
	args, ok, err := unstructured.NestedStringSlice(installer, "args")
	if err != nil || !ok {
		t.Fatalf("installer args missing: ok=%v err=%v", ok, err)
	}
	for _, want := range []string{
		"--bin-dir=/var/lib/cni/bin",
		"--conf-dir=/var/lib/cni/net.d",
		"--state-dir=/run/dubbo-cni",
		"--grpc-inbound-port=16080",
		"--ipset-path=/usr/sbin/ipset",
		"--refresh-interval=30s",
	} {
		if !containsString(args, want) {
			t.Fatalf("installer args missing %q: %v", want, args)
		}
	}
	volumes, ok, err := unstructured.NestedSlice(daemonSet.Object, "spec", "template", "spec", "volumes")
	if err != nil || !ok {
		t.Fatalf("daemonset volumes missing: ok=%v err=%v", ok, err)
	}
	for _, want := range []string{"/var/lib/cni/bin", "/var/lib/cni/net.d", "/run/dubbo-cni"} {
		if !hasHostPathVolume(volumes, want) {
			t.Fatalf("daemonset missing hostPath volume %s", want)
		}
	}
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
			set:  "values.global.proxy.image=kdubbo/dubbod:test",
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

func deploymentEnvValue(t *testing.T, deployment manifest.Manifest, name string) string {
	t.Helper()
	containers, ok, err := unstructured.NestedSlice(deployment.Object, "spec", "template", "spec", "containers")
	if err != nil || !ok || len(containers) == 0 {
		t.Fatalf("deployment containers missing: ok=%v err=%v", ok, err)
	}
	execute := containers[0].(map[string]interface{})
	env, ok, err := unstructured.NestedSlice(execute, "env")
	if err != nil || !ok {
		t.Fatalf("deployment env missing: ok=%v err=%v", ok, err)
	}
	for _, item := range env {
		entry, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
		gotName, _, _ := unstructured.NestedString(entry, "name")
		if gotName != name {
			continue
		}
		value, _, _ := unstructured.NestedString(entry, "value")
		return value
	}
	t.Fatalf("deployment missing env %s", name)
	return ""
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

func containsString(items []string, want string) bool {
	for _, item := range items {
		if item == want {
			return true
		}
	}
	return false
}

func hasHostPathVolume(volumes []interface{}, path string) bool {
	for _, item := range volumes {
		volume, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
		got, _, _ := unstructured.NestedString(volume, "hostPath", "path")
		if got == path {
			return true
		}
	}
	return false
}
