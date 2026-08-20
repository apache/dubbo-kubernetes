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
	"encoding/json"
	"testing"

	"github.com/apache/dubbo-kubernetes/pkg/kube"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
)

func TestInjectServiceDoesNotRewriteApplicationTargetPort(t *testing.T) {
	service := corev1.Service{
		ObjectMeta: metav1.ObjectMeta{Name: "payment", Namespace: "app"},
		Spec: corev1.ServiceSpec{
			Selector: map[string]string{"app": "payment"},
			Ports: []corev1.ServicePort{{
				Name:       "grpc",
				Port:       80,
				TargetPort: intstr.FromInt(8080),
				Protocol:   corev1.ProtocolTCP,
			}},
		},
	}
	raw, err := json.Marshal(service)
	if err != nil {
		t.Fatal(err)
	}
	response := (&Webhook{}).injectService(&kube.AdmissionReview{
		Request: &kube.AdmissionRequest{
			UID:       types.UID("test"),
			Kind:      metav1.GroupVersionKind{Version: "v1", Kind: "Service"},
			Namespace: "app",
			Operation: kube.Create,
			Object:    runtime.RawExtension{Raw: raw},
		},
	}, "/inject")
	if !response.Allowed {
		t.Fatal("service admission rejected")
	}
	if len(response.Patch) != 0 {
		t.Fatalf("service patch = %s, want no targetPort rewrite", response.Patch)
	}
}
