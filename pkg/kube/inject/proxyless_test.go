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
		t.Fatalf("template containers = %d, want 1 hard-coded application container overlay", len(injectedPod.Spec.Containers))
	}
	if err := postProcessPod(mergedPod, *injectedPod, req); err != nil {
		t.Fatalf("postProcessPod() failed: %v", err)
	}

	if len(mergedPod.Spec.Containers) != 1 {
		t.Fatalf("containers = %d, want 1 application container", len(mergedPod.Spec.Containers))
	}
	assertDirectXDSConnection(t, mergedPod, "app", ProxylessGRPCSecretNameForMeta(pod.ObjectMeta))
}

func TestInstallerGRPCEngineTemplateUsesGenerateNameForDeploymentPods(t *testing.T) {
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
			GenerateName: "nginx-95575cc5d-",
			Namespace:    "app",
			Annotations: map[string]string{
				ProxylessInjectTemplatesAnnoName: ProxylessGRPCTemplateName,
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
		templates:    templates,
		valuesConfig: valuesConfig,
		meshGlobalConfig: &meshv1alpha1.MeshGlobalConfig{
			TrustDomain: "cluster.local",
			DefaultConfig: &meshv1alpha1.ProxyConfig{
				DiscoveryAddress: "dubbod.dubbo-system.svc:15012",
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
		t.Fatalf("containers = %d, want original nginx container only", len(mergedPod.Spec.Containers))
	}
	assertDirectXDSConnection(t, mergedPod, "nginx", ProxylessGRPCSecretNameForMeta(pod.ObjectMeta))
	if got := mergedPod.Spec.Volumes[0].Secret.SecretName; got == ProxylessGRPCSecretName("") {
		t.Fatalf("secret name = %q, want generateName-based secret", got)
	}
}

func assertDirectXDSConnection(t *testing.T, pod *corev1.Pod, containerName, secretName string) {
	t.Helper()

	container := pod.Spec.Containers[0]
	if container.Name != containerName {
		t.Fatalf("container name = %q, want %q", container.Name, containerName)
	}
	if !hasEnv(container.Env, "GRPC_XDS_BOOTSTRAP", ProxylessGRPCBootstrapPath) {
		t.Fatalf("GRPC_XDS_BOOTSTRAP env missing")
	}
	if !hasEnv(container.Env, ProxylessXDSAddressEnvName, "dubbod.dubbo-system.svc:15012") {
		t.Fatalf("%s env missing", ProxylessXDSAddressEnvName)
	}
	if !hasEnv(container.Env, "DUBBO_GRPC_XDS_RESOLVER", "xds:///") {
		t.Fatalf("DUBBO_GRPC_XDS_RESOLVER env missing")
	}
	if !hasEnv(container.Env, "DUBBO_GRPC_XDS_CREDENTIALS", "true") {
		t.Fatalf("DUBBO_GRPC_XDS_CREDENTIALS env missing")
	}
	if !hasEnv(container.Env, "CA_ADDRESS", "dubbod.dubbo-system.svc:15012") {
		t.Fatalf("CA_ADDRESS env missing")
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

	if err := addApplicationContainerConfig(pod, req); err != nil {
		t.Fatalf("addApplicationContainerConfig() failed: %v", err)
	}

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
	if got, want := vol.Secret.SecretName, ProxylessGRPCSecretNameForMeta(pod.ObjectMeta); got != want {
		t.Fatalf("secret name = %q, want %q", got, want)
	}

	container := pod.Spec.Containers[0]
	if !hasEnv(container.Env, "GRPC_XDS_BOOTSTRAP", ProxylessGRPCBootstrapPath) {
		t.Fatalf("GRPC_XDS_BOOTSTRAP env missing")
	}
	if !hasEnv(container.Env, ProxylessXDSAddressEnvName, "dubbod.dubbo-system.svc:15012") {
		t.Fatalf("%s env missing", ProxylessXDSAddressEnvName)
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

func TestInstallerGRPCEngineTemplateConfiguresXDSClientForDubbodImage(t *testing.T) {
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
			Name:      "nginx-consumer-6d4c7b8c9f-abcde",
			Namespace: "app",
			Annotations: map[string]string{
				ProxylessInjectTemplatesAnnoName: ProxylessGRPCTemplateName,
			},
		},
		Spec: corev1.PodSpec{
			ServiceAccountName: "nginx",
			Containers: []corev1.Container{{
				Name:  "app",
				Image: "kdubbo/dubbod:debug",
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
	if err := postProcessPod(mergedPod, *injectedPod, req); err != nil {
		t.Fatalf("postProcessPod() failed: %v", err)
	}

	container := mergedPod.Spec.Containers[0]
	wantArgs := []string{"xds-client", "--watch"}
	if strings.Join(container.Args, ",") != strings.Join(wantArgs, ",") {
		t.Fatalf("args = %v, want %v", container.Args, wantArgs)
	}
}

func TestInstallerGRPCEngineTemplateDoesNotConfigureXDSClientForNonDubbodImage(t *testing.T) {
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
			Name:      "nginx-v1-6d4c7b8c9f-abcde",
			Namespace: "app",
			Annotations: map[string]string{
				ProxylessInjectTemplatesAnnoName: ProxylessGRPCTemplateName,
			},
		},
		Spec: corev1.PodSpec{
			ServiceAccountName: "nginx",
			Containers: []corev1.Container{{
				Name:  "app",
				Image: "nginx:1.27-alpine",
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
	if err := postProcessPod(mergedPod, *injectedPod, req); err != nil {
		t.Fatalf("postProcessPod() failed: %v", err)
	}
	assertNoArgs(t, mergedPod)
}

func TestEnsureProxylessGRPCTemplateAnnotation(t *testing.T) {
	pod := &corev1.Pod{}
	ensureProxylessGRPCTemplateAnnotation(pod)
	if got := pod.Annotations[ProxylessInjectTemplatesAnnoName]; got != ProxylessGRPCTemplateName {
		t.Fatalf("template annotation = %q, want %q", got, ProxylessGRPCTemplateName)
	}

	ensureProxylessGRPCTemplateAnnotation(pod)
	if got := pod.Annotations[ProxylessInjectTemplatesAnnoName]; got != ProxylessGRPCTemplateName {
		t.Fatalf("template annotation after second call = %q, want %q", got, ProxylessGRPCTemplateName)
	}

	pod.Annotations[ProxylessInjectTemplatesAnnoName] = "custom"
	ensureProxylessGRPCTemplateAnnotation(pod)
	if got, want := pod.Annotations[ProxylessInjectTemplatesAnnoName], "custom,"+ProxylessGRPCTemplateName; got != want {
		t.Fatalf("template annotation = %q, want %q", got, want)
	}
}

func TestProxylessGRPCSecretNameFitsKubernetesLengthLimit(t *testing.T) {
	name := ProxylessGRPCSecretName("grpc-provider-012345678901234567890123456789012345678901234567890123")
	if len(name) > 63 {
		t.Fatalf("secret name length = %d, want <= 63", len(name))
	}
}

func TestProxylessGRPCSecretNameForMetaPrefersGenerateName(t *testing.T) {
	meta := metav1.ObjectMeta{Name: "nginx-95575cc5d-kh98x", GenerateName: "nginx-95575cc5d-"}
	if got, want := ProxylessGRPCSecretNameForMeta(meta), ProxylessGRPCSecretName(meta.GenerateName); got != want {
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
