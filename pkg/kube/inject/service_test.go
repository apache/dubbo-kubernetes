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
	"strings"
	"testing"

	"github.com/apache/dubbo-kubernetes/pkg/kube"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
)

func TestRewriteProxylessServiceTargetPortsRoutesTCPServiceToXServer(t *testing.T) {
	svc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{Name: "nginx", Namespace: "app"},
		Spec: corev1.ServiceSpec{
			Selector: map[string]string{"app": "nginx"},
			Ports: []corev1.ServicePort{
				{Name: "http", Port: 80, TargetPort: intstr.FromInt(80), Protocol: corev1.ProtocolTCP},
				{Name: "metrics", Port: 9090, TargetPort: intstr.FromString("metrics"), Protocol: corev1.ProtocolTCP},
			},
		},
	}

	if !rewriteProxylessServiceTargetPorts(svc) {
		t.Fatalf("rewriteProxylessServiceTargetPorts() = false, want true")
	}
	for _, port := range svc.Spec.Ports {
		if got := port.TargetPort.IntVal; got != ProxylessXServerPort {
			t.Fatalf("port %s targetPort = %d, want %d", port.Name, got, ProxylessXServerPort)
		}
	}
}

func TestRewriteProxylessServiceTargetPortsSkipsUnsafeServices(t *testing.T) {
	cases := []struct {
		name string
		svc  corev1.Service
	}{
		{
			name: "selectorless",
			svc: corev1.Service{
				ObjectMeta: metav1.ObjectMeta{Name: "manual", Namespace: "app"},
				Spec: corev1.ServiceSpec{
					Ports: []corev1.ServicePort{{Name: "http", Port: 80, TargetPort: intstr.FromInt(80)}},
				},
			},
		},
		{
			name: "external-name",
			svc: corev1.Service{
				ObjectMeta: metav1.ObjectMeta{Name: "external", Namespace: "app"},
				Spec: corev1.ServiceSpec{
					Type:         corev1.ServiceTypeExternalName,
					ExternalName: "example.com",
					Selector:     map[string]string{"app": "external"},
					Ports:        []corev1.ServicePort{{Name: "http", Port: 80, TargetPort: intstr.FromInt(80)}},
				},
			},
		},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			svc := tt.svc
			if rewriteProxylessServiceTargetPorts(&svc) {
				t.Fatalf("rewriteProxylessServiceTargetPorts() = true, want false")
			}
			if got := svc.Spec.Ports[0].TargetPort.IntVal; got != 80 {
				t.Fatalf("targetPort = %d, want 80", got)
			}
		})
	}
}

func TestRewriteProxylessServiceTargetPortsSkipsNonTCPPorts(t *testing.T) {
	svc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{Name: "dns", Namespace: "app"},
		Spec: corev1.ServiceSpec{
			Selector: map[string]string{"app": "dns"},
			Ports: []corev1.ServicePort{
				{Name: "dns", Port: 53, TargetPort: intstr.FromInt(53), Protocol: corev1.ProtocolUDP},
			},
		},
	}

	if rewriteProxylessServiceTargetPorts(svc) {
		t.Fatalf("rewriteProxylessServiceTargetPorts() = true, want false")
	}
	if got := svc.Spec.Ports[0].TargetPort.IntVal; got != 53 {
		t.Fatalf("targetPort = %d, want 53", got)
	}
}

func TestInjectServiceCreatesXServerTargetPortPatch(t *testing.T) {
	svc := corev1.Service{
		ObjectMeta: metav1.ObjectMeta{Name: "nginx", Namespace: "app"},
		Spec: corev1.ServiceSpec{
			Selector: map[string]string{"app": "nginx"},
			Ports: []corev1.ServicePort{
				{Name: "http", Port: 80, TargetPort: intstr.FromInt(80), Protocol: corev1.ProtocolTCP},
			},
		},
	}
	raw, err := json.Marshal(svc)
	if err != nil {
		t.Fatalf("json.Marshal() failed: %v", err)
	}

	wh := &Webhook{Config: &Config{Policy: InjectionPolicyEnabled}}
	resp := wh.injectService(&kube.AdmissionReview{
		Request: &kube.AdmissionRequest{
			UID:       types.UID("test"),
			Kind:      metav1.GroupVersionKind{Version: "v1", Kind: "Service"},
			Namespace: "app",
			Operation: kube.Create,
			Object:    runtime.RawExtension{Raw: raw},
		},
	}, "/inject")

	if !resp.Allowed {
		t.Fatalf("Allowed = false, want true")
	}
	if resp.PatchType == nil || *resp.PatchType != "JSONPatch" {
		t.Fatalf("PatchType = %v, want JSONPatch", resp.PatchType)
	}
	if !strings.Contains(string(resp.Patch), `"value":15080`) {
		t.Fatalf("patch = %s, want targetPort 15080", string(resp.Patch))
	}
}

func TestInjectServiceSkipsProxylessOptOutService(t *testing.T) {
	svc := corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "dxgate-gateway",
			Namespace: "app",
			Labels: map[string]string{
				"proxyless.dubbo.apache.org/inject": "false",
			},
		},
		Spec: corev1.ServiceSpec{
			Selector: map[string]string{"app.kubernetes.io/name": "dxgate"},
			Ports: []corev1.ServicePort{
				{Name: "http", Port: 80, TargetPort: intstr.FromString("http"), Protocol: corev1.ProtocolTCP},
			},
		},
	}
	raw, err := json.Marshal(svc)
	if err != nil {
		t.Fatalf("json.Marshal() failed: %v", err)
	}

	wh := &Webhook{Config: &Config{Policy: InjectionPolicyEnabled}}
	resp := wh.injectService(&kube.AdmissionReview{
		Request: &kube.AdmissionRequest{
			UID:       types.UID("test"),
			Kind:      metav1.GroupVersionKind{Version: "v1", Kind: "Service"},
			Namespace: "app",
			Operation: kube.Create,
			Object:    runtime.RawExtension{Raw: raw},
		},
	}, "/inject")

	if !resp.Allowed {
		t.Fatalf("Allowed = false, want true")
	}
	if len(resp.Patch) != 0 {
		t.Fatalf("patch = %s, want no patch", string(resp.Patch))
	}
}
