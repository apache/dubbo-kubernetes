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

package inject

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	telemetryconfig "github.com/apache/dubbo-kubernetes/pkg/config/telemetry"
	meshv1alpha1 "github.com/dubml/api/mesh/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestInstallerDefaultTripleTemplateInjectsDirectXDSConnection(t *testing.T) {
	_, currentFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatalf("runtime.Caller() failed")
	}
	templatePath := filepath.Join(filepath.Dir(currentFile), "../../..", "manifests/charts/dubbod/files/triple-engine.yaml")
	templateBytes, err := os.ReadFile(templatePath)
	if err != nil {
		t.Fatalf("failed to read triple-engine.yaml: %v", err)
	}
	templates, err := ParseTemplates(RawTemplates{
		InherentTripleTemplateName: string(templateBytes),
	})
	if err != nil {
		t.Fatalf("ParseTemplates() failed: %v", err)
	}
	valuesConfig, err := NewValuesConfig("{}")
	if err != nil {
		t.Fatalf("NewValuesConfig() failed: %v", err)
	}

	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "grpc-provider-6d4c7b8c9f-abcde",
			Namespace: "grpc-app",
		},
		Spec: corev1.PodSpec{
			ServiceAccountName: "grpc-sa",
			Containers: []corev1.Container{{
				Name: "app",
			}},
		},
	}
	req := InjectionParameters{
		pod:             pod,
		defaultTemplate: []string{InherentTripleTemplateName},
		templates:       templates,
		valuesConfig:    valuesConfig,
		meshConfig: &meshv1alpha1.MeshConfig{
			TrustDomain: "cluster.local",
			DefaultConfig: &meshv1alpha1.ProxyConfig{
				DiscoveryAddress: "dubbod.dubbo-system.svc:26012",
			},
		},
		proxyConfig: &meshv1alpha1.ProxyConfig{
			DiscoveryAddress: "dubbod.dubbo-system.svc:26012",
		},
	}

	mergedPod, injectedPod, err := RunTemplate(req)
	if err != nil {
		t.Fatalf("RunTemplate() failed: %v", err)
	}

	if len(injectedPod.Spec.Containers) != 1 {
		t.Fatalf("template containers = %d, want application overlay only", len(injectedPod.Spec.Containers))
	}
	if err := postProcessPod(mergedPod, *injectedPod, req); err != nil {
		t.Fatalf("postProcessPod() failed: %v", err)
	}

	if len(mergedPod.Spec.Containers) != 1 {
		t.Fatalf("containers = %d, want original application container only", len(mergedPod.Spec.Containers))
	}
	assertDirectXDSConnection(t, mergedPod, "app", InherentXDSSecretNameForMeta(pod.ObjectMeta))
}

func TestInstallerTripleEngineTemplateAllocatesUniqueNameForDeploymentPods(t *testing.T) {
	_, currentFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatalf("runtime.Caller() failed")
	}
	templatePath := filepath.Join(filepath.Dir(currentFile), "../../..", "manifests/charts/dubbod/files/triple-engine.yaml")
	templateBytes, err := os.ReadFile(templatePath)
	if err != nil {
		t.Fatalf("failed to read triple-engine.yaml: %v", err)
	}
	templates, err := ParseTemplates(RawTemplates{
		InherentTripleTemplateName: string(templateBytes),
	})
	if err != nil {
		t.Fatalf("ParseTemplates() failed: %v", err)
	}
	valuesConfig, err := NewValuesConfig("{}")
	if err != nil {
		t.Fatalf("NewValuesConfig() failed: %v", err)
	}

	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "nginx-95575cc5d-",
			Namespace:    "app",
			Annotations: map[string]string{
				InherentInjectTemplatesAnnoName: InherentTripleTemplateName,
			},
		},
		Spec: corev1.PodSpec{
			ServiceAccountName: "nginx",
			Containers: []corev1.Container{{
				Name:  "nginx",
				Image: "nginx:1.27-alpine",
			}},
		},
	}
	req := InjectionParameters{
		pod:          pod,
		requestUID:   "admission-1",
		templates:    templates,
		valuesConfig: valuesConfig,
		meshConfig: &meshv1alpha1.MeshConfig{
			TrustDomain: "cluster.local",
			DefaultConfig: &meshv1alpha1.ProxyConfig{
				DiscoveryAddress: "dubbod.dubbo-system.svc:26012",
			},
		},
	}

	mergedPod, injectedPod, err := RunTemplate(req)
	if err != nil {
		t.Fatalf("RunTemplate() failed: %v", err)
	}
	if err := postProcessPod(mergedPod, *injectedPod, req); err != nil {
		t.Fatalf("postProcessPod() failed: %v", err)
	}
	if len(mergedPod.Spec.Containers) != 1 {
		t.Fatalf("containers = %d, want original application container only", len(mergedPod.Spec.Containers))
	}
	assertDirectXDSConnection(t, mergedPod, "nginx", InherentXDSSecretNameForMeta(mergedPod.ObjectMeta))
	if mergedPod.Name == "" || !strings.HasPrefix(mergedPod.Name, pod.GenerateName) {
		t.Fatalf("Pod name = %q, want allocated name with deployment prefix", mergedPod.Name)
	}
	patch, err := injectPod(req)
	if err != nil || !strings.Contains(string(patch), `"path":"/metadata/name"`) {
		t.Fatalf("injection must patch the allocated Pod name: patch=%s error=%v", patch, err)
	}
	other := pod.DeepCopy()
	otherReq := req
	otherReq.requestUID = "admission-2"
	if err := postProcessPod(other, *injectedPod, otherReq); err != nil {
		t.Fatal(err)
	}
	if InherentXDSSecretNameForMeta(other.ObjectMeta) == InherentXDSSecretNameForMeta(mergedPod.ObjectMeta) {
		t.Fatal("replicas must not share credentials")
	}
	name := mergedPod.Name
	if err := postProcessPod(mergedPod, *injectedPod, otherReq); err != nil || mergedPod.Name != name {
		t.Fatalf("reinvocation must preserve Pod identity: name=%q error=%v", mergedPod.Name, err)
	}
}

func assertDirectXDSConnection(t *testing.T, pod *corev1.Pod, containerName, secretName string) {
	t.Helper()

	container := pod.Spec.Containers[0]
	bootstrapEnv, bootstrapPath := InherentTripleBootstrapEnvName, InherentTripleBootstrapPath
	if container.Name != containerName {
		t.Fatalf("container name = %q, want %q", container.Name, containerName)
	}
	if !hasEnv(container.Env, bootstrapEnv, bootstrapPath) {
		t.Fatalf("%s env missing", bootstrapEnv)
	}
	if !hasEnv(container.Env, InherentTripleStatusAddressEnvName, InherentTripleStatusAddress) {
		t.Fatalf("%s env missing", InherentTripleStatusAddressEnvName)
	}
	for _, env := range container.Env {
		if strings.HasPrefix(env.Name, "DUBBO_GRPC_") || env.Name == "GRPC_XDS_BOOTSTRAP" {
			t.Fatalf("legacy gRPC engine env injected: %s", env.Name)
		}
	}

	foundMetricsPort := false
	for _, port := range container.Ports {
		if port.Name == InherentMetricsPortName && port.ContainerPort == InherentMetricsPort {
			foundMetricsPort = true
		}
	}
	if !foundMetricsPort {
		t.Fatalf("metrics port %d missing", InherentMetricsPort)
	}
	if got := pod.Annotations["prometheus.io/scrape"]; got != "true" {
		t.Fatalf("prometheus scrape annotation = %q, want true", got)
	}

	if !hasEnv(container.Env, "TRUST_DOMAIN", "cluster.local") {
		t.Fatalf("TRUST_DOMAIN env missing")
	}
	if !hasFieldRefEnv(container.Env, "POD_NAMESPACE", "metadata.namespace") {
		t.Fatalf("POD_NAMESPACE fieldRef env missing")
	}
	if !hasFieldRefEnv(container.Env, "INSTANCE_IP", "status.podIP") {
		t.Fatalf("INSTANCE_IP fieldRef env missing")
	}
	if !hasMount(container.VolumeMounts, InherentXDSVolumeName, InherentXDSMountPath, true) {
		t.Fatalf("inherent xds mount missing")
	}
	if len(pod.Spec.Volumes) != 1 {
		t.Fatalf("volumes = %d, want 1", len(pod.Spec.Volumes))
	}
	if got, want := pod.Spec.Volumes[0].Name, InherentXDSVolumeName; got != want {
		t.Fatalf("volume name = %q, want %q", got, want)
	}
	if pod.Spec.Volumes[0].Secret == nil {
		t.Fatalf("volume secret = nil, want SecretVolumeSource")
	}
	if got, want := pod.Spec.Volumes[0].Secret.SecretName, secretName; got != want {
		t.Fatalf("secret name = %q, want %q", got, want)
	}
	if pod.Spec.Volumes[0].Secret.DefaultMode == nil {
		t.Fatalf("secret defaultMode = nil, want 420")
	}
	if got, want := *pod.Spec.Volumes[0].Secret.DefaultMode, int32(420); got != want {
		t.Fatalf("secret defaultMode = %d, want %d", got, want)
	}
}

func assertNoArgs(t *testing.T, pod *corev1.Pod) {
	t.Helper()
	if len(pod.Spec.Containers) == 0 {
		t.Fatalf("containers = 0, want at least 1")
	}
	if len(pod.Spec.Containers[0].Args) != 0 {
		t.Fatalf("args = %v, want no launcher args", pod.Spec.Containers[0].Args)
	}
}

func TestEnsureInherentTripleTemplateAnnotation(t *testing.T) {
	pod := &corev1.Pod{}
	ensureInherentTemplateAnnotation(pod, InherentTripleTemplateName)
	if got := pod.Annotations[InherentInjectTemplatesAnnoName]; got != InherentTripleTemplateName {
		t.Fatalf("template annotation = %q, want %q", got, InherentTripleTemplateName)
	}

	ensureInherentTemplateAnnotation(pod, InherentTripleTemplateName)
	if got := pod.Annotations[InherentInjectTemplatesAnnoName]; got != InherentTripleTemplateName {
		t.Fatalf("template annotation after second call = %q, want %q", got, InherentTripleTemplateName)
	}

	pod.Annotations[InherentInjectTemplatesAnnoName] = "custom"
	ensureInherentTemplateAnnotation(pod, InherentTripleTemplateName)
	if got, want := pod.Annotations[InherentInjectTemplatesAnnoName], "custom,"+InherentTripleTemplateName; got != want {
		t.Fatalf("template annotation = %q, want %q", got, want)
	}
}

func TestInherentXDSSecretNameFitsKubernetesLengthLimit(t *testing.T) {
	name := InherentXDSSecretName("grpc-provider-012345678901234567890123456789012345678901234567890123")
	if len(name) > 63 {
		t.Fatalf("secret name length = %d, want <= 63", len(name))
	}
}

func TestInherentXDSSecretNameForMetaUsesPodName(t *testing.T) {
	meta := metav1.ObjectMeta{Name: "nginx-95575cc5d-kh98x", GenerateName: "nginx-95575cc5d-"}
	if got, want := InherentXDSSecretNameForMeta(meta), InherentXDSSecretName(meta.Name); got != want {
		t.Fatalf("secret name = %q, want %q", got, want)
	}
}

func hasEnv(envs []corev1.EnvVar, name, value string) bool {
	for _, env := range envs {
		if env.Name == name && env.Value == value {
			return true
		}
	}
	return false
}

func hasFieldRefEnv(envs []corev1.EnvVar, name, fieldPath string) bool {
	for _, env := range envs {
		if env.Name != name || env.ValueFrom == nil || env.ValueFrom.FieldRef == nil {
			continue
		}
		if env.ValueFrom.FieldRef.FieldPath == fieldPath {
			return true
		}
	}
	return false
}

func hasMount(mounts []corev1.VolumeMount, name, path string, readOnly bool) bool {
	for _, mount := range mounts {
		if mount.Name == name && mount.MountPath == path && mount.ReadOnly == readOnly {
			return true
		}
	}
	return false
}

func TestInstallerTripleEngineTemplateInjectsTelemetryEnv(t *testing.T) {
	_, currentFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatalf("runtime.Caller() failed")
	}
	templatePath := filepath.Join(filepath.Dir(currentFile), "../../..", "manifests/charts/dubbod/files/triple-engine.yaml")
	templateBytes, err := os.ReadFile(templatePath)
	if err != nil {
		t.Fatalf("failed to read triple-engine.yaml: %v", err)
	}
	templates, err := ParseTemplates(RawTemplates{
		InherentTripleTemplateName: string(templateBytes),
	})
	if err != nil {
		t.Fatalf("ParseTemplates() failed: %v", err)
	}
	valuesConfig, err := NewValuesConfig("{}")
	if err != nil {
		t.Fatalf("NewValuesConfig() failed: %v", err)
	}

	newPod := func() *corev1.Pod {
		return &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "grpc-provider-6d4c7b8c9f-abcde",
				Namespace: "grpc-app",
				Annotations: map[string]string{
					InherentInjectTemplatesAnnoName: InherentTripleTemplateName,
				},
			},
			Spec: corev1.PodSpec{
				ServiceAccountName: "grpc-sa",
				Containers:         []corev1.Container{{Name: "app"}},
			},
		}
	}
	newParams := func(effective telemetryconfig.EffectiveTracing) InjectionParameters {
		return InjectionParameters{
			pod:          newPod(),
			templates:    templates,
			valuesConfig: valuesConfig,
			meshConfig:   &meshv1alpha1.MeshConfig{TrustDomain: "cluster.local"},
			telemetry:    effective,
			proxyConfig: &meshv1alpha1.ProxyConfig{
				DiscoveryAddress: "dubbod.dubbo-system.svc:26012",
			},
		}
	}
	envValue := func(pod *corev1.Pod, name string) string {
		app := FindContainer("app", pod.Spec.Containers)
		if app == nil {
			t.Fatalf("app container not found")
		}
		for _, e := range app.Env {
			if e.Name == name {
				return e.Value
			}
		}
		return ""
	}

	sampling := 100.0
	disabled := false
	tracing := telemetryconfig.EffectiveTracing{
		Configured:               true,
		Providers:                []string{"localtrace"},
		Tags:                     []telemetryconfig.Tag{{Name: "foo", Value: "bar"}},
		RandomSamplingPercentage: &sampling,
		DisableSpanReporting:     &disabled,
	}
	mergedPod, _, err := RunTemplate(newParams(tracing))
	if err != nil {
		t.Fatalf("RunTemplate() failed: %v", err)
	}
	if err := postProcessPod(mergedPod, corev1.Pod{}, newParams(tracing)); err != nil {
		t.Fatalf("postProcessPod() failed: %v", err)
	}
	if got := envValue(mergedPod, InherentTripleStatusAddressEnvName); got != InherentTripleStatusAddress {
		t.Fatalf("%s = %q, want %q", InherentTripleStatusAddressEnvName, got, InherentTripleStatusAddress)
	}
	app := FindContainer("app", mergedPod.Spec.Containers)
	if app == nil {
		t.Fatal("app container not found")
	}
	foundMetricsPort := false
	for _, port := range app.Ports {
		if port.Name == InherentMetricsPortName && port.ContainerPort == InherentMetricsPort {
			foundMetricsPort = true
		}
	}
	if !foundMetricsPort {
		t.Fatalf("application metrics port %d missing", InherentMetricsPort)
	}
	for key, want := range map[string]string{
		"prometheus.io/scrape": "true",
		"prometheus.io/path":   "/metrics",
		"prometheus.io/port":   "9090",
	} {
		if got := mergedPod.Annotations[key]; got != want {
			t.Fatalf("annotation %s = %q, want %q", key, got, want)
		}
	}
	if got, want := envValue(mergedPod, "OTEL_EXPORTER_OTLP_ENDPOINT"), "http://tracing.dubbo-system.svc:4317"; got != want {
		t.Fatalf("OTEL_EXPORTER_OTLP_ENDPOINT = %q, want %q", got, want)
	}
	if got := envValue(mergedPod, "OTEL_TRACES_EXPORTER"); got != "otlp" {
		t.Fatalf("OTEL_TRACES_EXPORTER = %q, want otlp", got)
	}
	if got := envValue(mergedPod, "OTEL_TRACES_SAMPLER_ARG"); got != "1" {
		t.Fatalf("OTEL_TRACES_SAMPLER_ARG = %q, want 1", got)
	}
	if got := envValue(mergedPod, "OTEL_RESOURCE_ATTRIBUTES"); got != "foo=bar" {
		t.Fatalf("OTEL_RESOURCE_ATTRIBUTES = %q, want foo=bar", got)
	}

	mergedPod, _, err = RunTemplate(newParams(telemetryconfig.EffectiveTracing{}))
	if err != nil {
		t.Fatalf("RunTemplate() failed: %v", err)
	}
	if got := envValue(mergedPod, "OTEL_EXPORTER_OTLP_ENDPOINT"); got != "" {
		t.Fatalf("OTEL_EXPORTER_OTLP_ENDPOINT = %q, want empty without Telemetry", got)
	}

	disabled = true
	tracing.DisableSpanReporting = &disabled
	mergedPod, _, err = RunTemplate(newParams(tracing))
	if err != nil {
		t.Fatalf("RunTemplate() failed: %v", err)
	}
	if got := envValue(mergedPod, "OTEL_TRACES_EXPORTER"); got != "none" {
		t.Fatalf("OTEL_TRACES_EXPORTER = %q, want none", got)
	}
}
