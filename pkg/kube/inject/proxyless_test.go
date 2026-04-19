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
	"testing"

	meshv1alpha1 "github.com/kdubbo/api/mesh/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

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

func hasMount(mounts []corev1.VolumeMount, name, path string, readOnly bool) bool {
	for _, mount := range mounts {
		if mount.Name == name && mount.MountPath == path && mount.ReadOnly == readOnly {
			return true
		}
	}
	return false
}
