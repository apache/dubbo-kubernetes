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
	"testing"

	meshv1alpha1 "github.com/kdubbo/api/mesh/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestInstallerGRPCEngineTemplateInjectsDirectXDSConnection(t *testing.T) {
	_, currentFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatalf("runtime.Caller() failed")
	}
	templatePath := filepath.Join(filepath.Dir(currentFile), "../../..", "dubboinstaller/charts/dubbod/files/grpc-engine.yaml")
	templateBytes, err := os.ReadFile(templatePath)
	if err != nil {
		t.Fatalf("failed to read grpc-engine.yaml: %v", err)
	}
	templates, err := ParseTemplates(RawTemplates{
		ProxylessGRPCTemplateName: string(templateBytes),
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
			Annotations: map[string]string{
				ProxylessInjectTemplatesAnnoName: ProxylessGRPCTemplateName,
			},
		},
		Spec: corev1.PodSpec{
			ServiceAccountName: "grpc-sa",
			Containers: []corev1.Container{{
				Name: "app",
			}},
		},
	}
	req := InjectionParameters{
		pod:          pod,
		templates:    templates,
		valuesConfig: valuesConfig,
		meshGlobalConfig: &meshv1alpha1.MeshGlobalConfig{
			TrustDomain: "cluster.local",
			DefaultConfig: &meshv1alpha1.ProxyConfig{
				DiscoveryAddress: "dubbod.dubbo-system.svc:15012",
			},
		},
		proxyConfig: &meshv1alpha1.ProxyConfig{
			DiscoveryAddress: "dubbod.dubbo-system.svc:15012",
		},
	}

	mergedPod, injectedPod, err := RunTemplate(req)
	if err != nil {
		t.Fatalf("RunTemplate() failed: %v", err)
	}

	if len(injectedPod.Spec.Containers) != 1 {
		t.Fatalf("template containers = %d, want 1 application container overlay", len(injectedPod.Spec.Containers))
	}
	assertDirectXDSConnection(t, injectedPod, pod.Name)

	if len(mergedPod.Spec.Containers) != 1 {
		t.Fatalf("containers = %d, want 1 application container", len(mergedPod.Spec.Containers))
	}
	assertDirectXDSConnection(t, mergedPod, pod.Name)
}

func assertDirectXDSConnection(t *testing.T, pod *corev1.Pod, podName string) {
	t.Helper()

	container := pod.Spec.Containers[0]
	if container.Name != "app" {
		t.Fatalf("container name = %q, want app", container.Name)
	}
	if !hasEnv(container.Env, "GRPC_XDS_BOOTSTRAP", ProxylessGRPCBootstrapPath) {
		t.Fatalf("GRPC_XDS_BOOTSTRAP env missing")
	}
	if !hasEnv(container.Env, "DUBBO_GRPC_XDS_RESOLVER", "xds:///") {
		t.Fatalf("DUBBO_GRPC_XDS_RESOLVER env missing")
	}
	if !hasEnv(container.Env, "DUBBO_GRPC_XDS_CREDENTIALS", "true") {
		t.Fatalf("DUBBO_GRPC_XDS_CREDENTIALS env missing")
	}
	if !hasEnv(container.Env, "CA_ADDR", "dubbod.dubbo-system.svc:15012") {
		t.Fatalf("CA_ADDR env missing")
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
	if !hasMount(container.VolumeMounts, ProxylessXDSVolumeName, ProxylessXDSMountPath, true) {
		t.Fatalf("proxyless xds mount missing")
	}
	if len(pod.Spec.Volumes) != 1 {
		t.Fatalf("volumes = %d, want 1", len(pod.Spec.Volumes))
	}
	if got, want := pod.Spec.Volumes[0].Name, ProxylessXDSVolumeName; got != want {
		t.Fatalf("volume name = %q, want %q", got, want)
	}
	if pod.Spec.Volumes[0].Secret == nil {
		t.Fatalf("volume secret = nil, want SecretVolumeSource")
	}
	if got, want := pod.Spec.Volumes[0].Secret.SecretName, ProxylessGRPCSecretName(podName); got != want {
		t.Fatalf("secret name = %q, want %q", got, want)
	}
	if pod.Spec.Volumes[0].Secret.DefaultMode == nil {
		t.Fatalf("secret defaultMode = nil, want 420")
	}
	if got, want := *pod.Spec.Volumes[0].Secret.DefaultMode, int32(420); got != want {
		t.Fatalf("secret defaultMode = %d, want %d", got, want)
	}
}

func TestAddApplicationContainerConfigInjectsProxylessGRPCContract(t *testing.T) {
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "grpc-provider-6d4c7b8c9f-abcde",
			Namespace: "grpc-app",
			Annotations: map[string]string{
				ProxylessInjectTemplatesAnnoName: ProxylessGRPCTemplateName,
			},
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{{
				Name: "app",
			}},
		},
	}

	req := InjectionParameters{
		meshGlobalConfig: &meshv1alpha1.MeshGlobalConfig{
			TrustDomain: "cluster.local",
			DefaultConfig: &meshv1alpha1.ProxyConfig{
				DiscoveryAddress: "dubbod.dubbo-system.svc:15012",
			},
		},
		proxyConfig: &meshv1alpha1.ProxyConfig{
			DiscoveryAddress: "dubbod.dubbo-system.svc:15012",
		},
	}

	addApplicationContainerConfig(pod, req)

	if len(pod.Spec.Volumes) != 1 {
		t.Fatalf("volumes = %d, want 1", len(pod.Spec.Volumes))
	}
	vol := pod.Spec.Volumes[0]
	if got, want := vol.Name, ProxylessXDSVolumeName; got != want {
		t.Fatalf("volume name = %q, want %q", got, want)
	}
	if vol.Secret == nil {
		t.Fatalf("volume secret = nil, want SecretVolumeSource")
	}
	if got, want := vol.Secret.SecretName, ProxylessGRPCSecretName(pod.Name); got != want {
		t.Fatalf("secret name = %q, want %q", got, want)
	}

	container := pod.Spec.Containers[0]
	if !hasEnv(container.Env, "GRPC_XDS_BOOTSTRAP", ProxylessGRPCBootstrapPath) {
		t.Fatalf("GRPC_XDS_BOOTSTRAP env missing")
	}
	if !hasEnv(container.Env, "GRPC_XDS_EXPERIMENTAL_SECURITY_SUPPORT", "true") {
		t.Fatalf("GRPC_XDS_EXPERIMENTAL_SECURITY_SUPPORT env missing")
	}
	if !hasEnv(container.Env, "DUBBO_GRPC_XDS_CREDENTIALS", "true") {
		t.Fatalf("DUBBO_GRPC_XDS_CREDENTIALS env missing")
	}
	if !hasEnv(container.Env, "DUBBO_GRPC_XDS_RESOLVER", "xds:///") {
		t.Fatalf("DUBBO_GRPC_XDS_RESOLVER env missing")
	}
	if !hasMount(container.VolumeMounts, ProxylessXDSVolumeName, ProxylessXDSMountPath, true) {
		t.Fatalf("proxyless xds mount missing")
	}
}

func TestProxylessGRPCSecretNameFitsKubernetesLengthLimit(t *testing.T) {
	name := ProxylessGRPCSecretName("grpc-provider-012345678901234567890123456789012345678901234567890123")
	if len(name) > 63 {
		t.Fatalf("secret name length = %d, want <= 63", len(name))
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
