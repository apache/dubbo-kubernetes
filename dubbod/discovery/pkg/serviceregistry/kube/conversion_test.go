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
