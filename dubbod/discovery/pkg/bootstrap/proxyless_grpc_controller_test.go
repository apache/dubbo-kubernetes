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

package bootstrap

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/apache/dubbo-kubernetes/pkg/config/constants"
	"github.com/apache/dubbo-kubernetes/pkg/dubboagency/grpcxds"
	"github.com/apache/dubbo-kubernetes/pkg/kube/inject"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

func TestShouldManageProxylessGRPCPod(t *testing.T) {
	tests := []struct {
		name string
		pod  *corev1.Pod
		want bool
	}{
		{
			name: "explicit grpc-engine template",
			pod: &corev1.Pod{ObjectMeta: metav1.ObjectMeta{
				Annotations: map[string]string{
					inject.ProxylessInjectTemplatesAnnoName: inject.ProxylessGRPCTemplateName,
				},
			}},
			want: true,
		},
		{
			name: "default injected pod marked by bootstrap env",
			pod: &corev1.Pod{Spec: corev1.PodSpec{Containers: []corev1.Container{{
				Env: []corev1.EnvVar{{
					Name:  "GRPC_XDS_BOOTSTRAP",
					Value: inject.ProxylessGRPCBootstrapPath,
				}},
			}}}},
			want: true,
		},
		{
			name: "default injected pod marked by runtime config env",
			pod: &corev1.Pod{Spec: corev1.PodSpec{Containers: []corev1.Container{{
				Env: []corev1.EnvVar{{
					Name:  inject.ProxylessGRPCConfigEnvName,
					Value: inject.ProxylessGRPCConfigPath,
				}},
			}}}},
			want: true,
		},
		{
			name: "default injected pod marked by xds secret volume",
			pod: &corev1.Pod{Spec: corev1.PodSpec{Volumes: []corev1.Volume{{
				Name: inject.ProxylessXDSVolumeName,
				VolumeSource: corev1.VolumeSource{
					Secret: &corev1.SecretVolumeSource{SecretName: "dubbo-xds-pod"},
				},
			}}}},
			want: true,
		},
		{
			name: "not proxyless grpc",
			pod:  &corev1.Pod{},
			want: false,
		},
		{
			name: "nil pod",
			pod:  nil,
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := shouldManageProxylessGRPCPod(tt.pod); got != tt.want {
				t.Fatalf("shouldManageProxylessGRPCPod() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestBuildProxylessGRPCSecretIncludesRuntimeConfig(t *testing.T) {
	pod := &corev1.Pod{ObjectMeta: metav1.ObjectMeta{
		Name:      "nginx",
		Namespace: "app",
		UID:       "pod-uid",
	}}

	secret := buildProxylessGRPCSecret(pod, []byte("bootstrap"), []byte("runtime"), []byte("cert"), []byte("key"), []byte("root"))
	if secret.Name != inject.ProxylessGRPCSecretNameForMeta(pod.ObjectMeta) {
		t.Fatalf("secret name = %q, want proxyless secret name", secret.Name)
	}
	if got := string(secret.Data[inject.ProxylessGRPCBootstrapFileName]); got != "bootstrap" {
		t.Fatalf("bootstrap data = %q, want bootstrap", got)
	}
	if got := string(secret.Data[inject.ProxylessGRPCConfigFileName]); got != "runtime" {
		t.Fatalf("runtime config data = %q, want runtime", got)
	}
	if got := string(secret.Data[constants.CACertNamespaceConfigMapDataName]); got != "root" {
		t.Fatalf("root cert data = %q, want root", got)
	}
	if len(secret.OwnerReferences) != 1 || secret.OwnerReferences[0].UID != pod.UID {
		t.Fatalf("owner references = %+v, want pod owner", secret.OwnerReferences)
	}
}

func TestBuildRuntimeConfigJSON(t *testing.T) {
	workload := &proxylessGRPCWorkloadContext{
		nodeID:           "proxyless~10.0.0.1~nginx.app~app.svc.cluster.local",
		podName:          "nginx",
		podNamespace:     "app",
		podIP:            "10.0.0.1",
		serviceAccount:   "nginx",
		trustDomain:      "cluster.local",
		discoveryAddress: "dubbod.dubbo-system.svc:15012",
	}

	data, err := buildRuntimeConfigJSON(workload, nil, nil)
	if err != nil {
		t.Fatalf("buildRuntimeConfigJSON() failed: %v", err)
	}

	var got proxylessGRPCRuntimeConfig
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("runtime config is not valid JSON: %v", err)
	}

	if got.Version != proxylessGRPCRuntimeConfigVersion {
		t.Fatalf("version = %q, want %q", got.Version, proxylessGRPCRuntimeConfigVersion)
	}
	if got.Mode != "proxyless-grpc" {
		t.Fatalf("mode = %q, want proxyless-grpc", got.Mode)
	}
	if got.Env["GRPC_XDS_BOOTSTRAP"] != inject.ProxylessGRPCBootstrapPath {
		t.Fatalf("GRPC_XDS_BOOTSTRAP = %q, want %q", got.Env["GRPC_XDS_BOOTSTRAP"], inject.ProxylessGRPCBootstrapPath)
	}
	if got.Env[inject.ProxylessGRPCConfigEnvName] != inject.ProxylessGRPCConfigPath {
		t.Fatalf("%s = %q, want %q", inject.ProxylessGRPCConfigEnvName, got.Env[inject.ProxylessGRPCConfigEnvName], inject.ProxylessGRPCConfigPath)
	}
	if got.Bootstrap.DiscoveryAddress != workload.discoveryAddress {
		t.Fatalf("discoveryAddress = %q, want %q", got.Bootstrap.DiscoveryAddress, workload.discoveryAddress)
	}
	if got.Certificates.Provider != grpcxds.FileWatcherCertProviderName {
		t.Fatalf("cert provider = %q, want %q", got.Certificates.Provider, grpcxds.FileWatcherCertProviderName)
	}
	if got.Certificates.CertChain != inject.ProxylessXDSMountPath+"/"+constants.CertChainFilename {
		t.Fatalf("certChain = %q, want mounted cert-chain path", got.Certificates.CertChain)
	}
	if got.Workload.NodeID != workload.nodeID {
		t.Fatalf("nodeId = %q, want %q", got.Workload.NodeID, workload.nodeID)
	}
	if got.Workload.PodIP != workload.podIP {
		t.Fatalf("podIP = %q, want %q", got.Workload.PodIP, workload.podIP)
	}
}

func TestNextRotationTime(t *testing.T) {
	now := time.Now()
	rotations := map[types.NamespacedName]time.Time{
		{Name: "late", Namespace: "app"}:  now.Add(time.Hour),
		{Name: "early", Namespace: "app"}: now.Add(time.Minute),
	}

	got, found := nextRotationTime(rotations)
	if !found {
		t.Fatalf("nextRotationTime() found = false, want true")
	}
	if !got.Equal(now.Add(time.Minute)) {
		t.Fatalf("nextRotationTime() = %v, want %v", got, now.Add(time.Minute))
	}

	if _, found := nextRotationTime(nil); found {
		t.Fatalf("nextRotationTime(nil) found = true, want false")
	}
}

func TestScheduleRotationTracksEarliestTimer(t *testing.T) {
	now := time.Now()
	controller := &proxylessGRPCWorkloadController{
		rotations: make(map[types.NamespacedName]time.Time),
	}
	defer controller.stopAllTimers()

	late := types.NamespacedName{Name: "late", Namespace: "app"}
	early := types.NamespacedName{Name: "early", Namespace: "app"}
	controller.scheduleRotation(late, now.Add(100*time.Hour))
	controller.scheduleRotation(early, now.Add(10*time.Hour))

	controller.rotationMu.Lock()
	next := controller.nextRotation
	controller.rotationMu.Unlock()
	if next.Before(now.Add(7*time.Hour)) || next.After(now.Add(9*time.Hour)) {
		t.Fatalf("nextRotation = %v, want around 8h from now", next.Sub(now))
	}

	controller.clearRotation(early)
	controller.rotationMu.Lock()
	next = controller.nextRotation
	controller.rotationMu.Unlock()
	if next.Before(now.Add(79*time.Hour)) || next.After(now.Add(81*time.Hour)) {
		t.Fatalf("nextRotation after clearing earliest = %v, want around 80h from now", next.Sub(now))
	}
}
