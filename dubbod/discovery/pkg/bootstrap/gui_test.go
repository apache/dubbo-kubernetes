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
	"testing"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestGUIGatewayInstanceUsesDeploymentIdentity(t *testing.T) {
	desired := int32(2)
	deployment := appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "dxgate-gateway",
			Namespace: "default",
			Labels: map[string]string{
				"gateway.networking.k8s.io/gateway-name": "dxgate-gateway",
			},
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &desired,
		},
		Status: appsv1.DeploymentStatus{
			ReadyReplicas: 2,
		},
	}

	got := guiGatewayInstanceFromDeployment(deployment)
	if got.Name != "dxgate-gateway" {
		t.Fatalf("Name = %q, want deployment name without pod hash", got.Name)
	}
	if got.GatewayName != "dxgate-gateway" {
		t.Fatalf("GatewayName = %q, want dxgate-gateway", got.GatewayName)
	}
	if got.ReadyReplicas != 2 || got.DesiredReplicas != 2 || !got.IsReady {
		t.Fatalf("readiness = ready:%d desired:%d isReady:%v, want 2/2 true",
			got.ReadyReplicas, got.DesiredReplicas, got.IsReady)
	}
}

func TestGUILogTailLines(t *testing.T) {
	if got := guiLogTailLines(""); got != 200 {
		t.Fatalf("default tail lines = %d, want 200", got)
	}
	if got := guiLogTailLines("25"); got != 25 {
		t.Fatalf("tail lines = %d, want 25", got)
	}
	if got := guiLogTailLines("99999"); got != 2000 {
		t.Fatalf("capped tail lines = %d, want 2000", got)
	}
}

func TestGUILogContainersPrefersNamedContainer(t *testing.T) {
	pod := corev1.Pod{
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{Name: "setup"},
				{Name: "dxgate"},
			},
		},
	}

	got := guiLogContainers(pod, "dxgate")
	if len(got) != 1 || got[0] != "dxgate" {
		t.Fatalf("containers = %v, want [dxgate]", got)
	}

	got = guiLogContainers(pod, "missing")
	if len(got) != 2 || got[0] != "setup" || got[1] != "dxgate" {
		t.Fatalf("fallback containers = %v, want all containers", got)
	}
}
