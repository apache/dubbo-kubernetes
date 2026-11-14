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

package kube

import (
	"testing"

	"github.com/apache/dubbo-kubernetes/pkg/config/mesh"
	"github.com/apache/dubbo-kubernetes/pkg/slices"
	meshv1alpha1 "github.com/dubml/api/mesh/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestConvertServicePreservesDualStackAddresses(t *testing.T) {
	service := corev1.Service{
		ObjectMeta: metav1.ObjectMeta{Name: "app", Namespace: "default"},
		Spec: corev1.ServiceSpec{
			ClusterIP:  "10.0.0.1",
			ClusterIPs: []string{"10.0.0.1", "fd00::1"},
		},
	}
	converted := ConvertService(service, "cluster.local", mesh.DefaultMeshConfig())
	if !slices.Equal(converted.Addresses, service.Spec.ClusterIPs) {
		t.Fatalf("addresses = %v, want %v", converted.Addresses, service.Spec.ClusterIPs)
	}
	if converted.DefaultAddress != service.Spec.ClusterIP {
		t.Fatalf("default address = %q", converted.DefaultAddress)
	}
}

func TestSecureNamingSANDefaults(t *testing.T) {
	for _, tc := range []struct {
		name string
		pod  *corev1.Pod
		mesh *meshv1alpha1.MeshConfig
		want string
	}{
		{
			name: "nil pod",
			pod:  nil,
			mesh: mesh.DefaultMeshConfig(),
			want: "",
		},
		{
			name: "empty namespace",
			pod: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{Name: "app-0"},
				Spec:       corev1.PodSpec{ServiceAccountName: "payment"},
			},
			mesh: mesh.DefaultMeshConfig(),
			want: "",
		},
		{
			name: "defaults empty service account and nil mesh trust domain",
			pod: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{Name: "app-0", Namespace: "app"},
			},
			mesh: nil,
			want: "spiffe://cluster.local/ns/app/sa/default",
		},
		{
			name: "defaults empty trust domain in mesh config",
			pod: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{Name: "app-0", Namespace: "app"},
			},
			mesh: &meshv1alpha1.MeshConfig{},
			want: "spiffe://cluster.local/ns/app/sa/default",
		},
		{
			name: "sanitizes explicit trust domain and keeps service account",
			pod: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{Name: "app-0", Namespace: "app"},
				Spec:       corev1.PodSpec{ServiceAccountName: "payment"},
			},
			mesh: &meshv1alpha1.MeshConfig{TrustDomain: "corp@example"},
			want: "spiffe://corp.example/ns/app/sa/payment",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := SecureNamingSAN(tc.pod, tc.mesh); got != tc.want {
				t.Fatalf("SecureNamingSAN() = %q, want %q", got, tc.want)
			}
		})
	}
}
