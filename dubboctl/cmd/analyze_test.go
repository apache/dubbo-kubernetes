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

package cmd

import (
	"strings"
	"testing"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	policyv1 "k8s.io/api/policy/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func controlPlaneDeployment(replicas int32) appsv1.Deployment {
	return appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{Name: "dubbod", Namespace: "dubbo-system"},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{MatchLabels: map[string]string{"app": "dubbod"}},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{Labels: map[string]string{"app": "dubbod"}},
			},
		},
	}
}

func controlPlaneBudget() policyv1.PodDisruptionBudget {
	return policyv1.PodDisruptionBudget{
		ObjectMeta: metav1.ObjectMeta{Name: "dubbod", Namespace: "dubbo-system"},
		Spec: policyv1.PodDisruptionBudgetSpec{
			Selector: &metav1.LabelSelector{MatchLabels: map[string]string{"app": "dubbod"}},
		},
	}
}

func runningPod(name, node string, labels map[string]string) corev1.Pod {
	return corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: "dubbo-system", Labels: labels},
		Spec:       corev1.PodSpec{NodeName: node},
		Status:     corev1.PodStatus{Phase: corev1.PodRunning},
	}
}

func messagesContain(msgs []analyzeMessage, substring string) bool {
	for _, msg := range msgs {
		if strings.Contains(msg.Message, substring) {
			return true
		}
	}
	return false
}

func TestAnalyzeHighAvailabilityFlagsSingleReplicaControlPlane(t *testing.T) {
	msgs := analyzeHighAvailability(
		[]appsv1.Deployment{controlPlaneDeployment(1)},
		nil,
		[]corev1.Pod{runningPod("dubbod-a", "node-1", map[string]string{"app": "dubbod"})},
	)
	if len(msgs) != 1 {
		t.Fatalf("messages = %d, want 1:\n%v", len(msgs), msgs)
	}
	if !messagesContain(msgs, "control plane runs a single replica") {
		t.Fatalf("unexpected message: %v", msgs)
	}
}

func TestAnalyzeHighAvailabilityFlagsMissingDisruptionBudget(t *testing.T) {
	msgs := analyzeHighAvailability(
		[]appsv1.Deployment{controlPlaneDeployment(2)},
		nil,
		[]corev1.Pod{
			runningPod("dubbod-a", "node-1", map[string]string{"app": "dubbod"}),
			runningPod("dubbod-b", "node-2", map[string]string{"app": "dubbod"}),
		},
	)
	if !messagesContain(msgs, "no PodDisruptionBudget") {
		t.Fatalf("missing disruption budget not reported: %v", msgs)
	}
}

// Replica count alone proves nothing: two pods on one node still go down
// together, which is exactly what the topology spread is meant to prevent.
func TestAnalyzeHighAvailabilityFlagsReplicasOnOneNode(t *testing.T) {
	msgs := analyzeHighAvailability(
		[]appsv1.Deployment{controlPlaneDeployment(2)},
		[]policyv1.PodDisruptionBudget{controlPlaneBudget()},
		[]corev1.Pod{
			runningPod("dubbod-a", "node-1", map[string]string{"app": "dubbod"}),
			runningPod("dubbod-b", "node-1", map[string]string{"app": "dubbod"}),
		},
	)
	if len(msgs) != 1 {
		t.Fatalf("messages = %d, want 1:\n%v", len(msgs), msgs)
	}
	if !messagesContain(msgs, "scheduled on node node-1") {
		t.Fatalf("unexpected message: %v", msgs)
	}
}

func TestAnalyzeHighAvailabilityAcceptsHealthyControlPlane(t *testing.T) {
	msgs := analyzeHighAvailability(
		[]appsv1.Deployment{controlPlaneDeployment(2)},
		[]policyv1.PodDisruptionBudget{controlPlaneBudget()},
		[]corev1.Pod{
			runningPod("dubbod-a", "node-1", map[string]string{"app": "dubbod"}),
			runningPod("dubbod-b", "node-2", map[string]string{"app": "dubbod"}),
		},
	)
	if len(msgs) != 0 {
		t.Fatalf("healthy control plane reported %d messages:\n%v", len(msgs), msgs)
	}
}

func TestAnalyzeHighAvailabilityIgnoresScaledDownAndUnrelatedWorkloads(t *testing.T) {
	zero := int32(0)
	scaledDown := controlPlaneDeployment(1)
	scaledDown.Spec.Replicas = &zero

	unrelated := appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{Name: "httpbin", Namespace: "backend"},
		Spec: appsv1.DeploymentSpec{
			Selector: &metav1.LabelSelector{MatchLabels: map[string]string{"app": "httpbin"}},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{Labels: map[string]string{"app": "httpbin"}},
			},
		},
	}

	msgs := analyzeHighAvailability([]appsv1.Deployment{scaledDown, unrelated}, nil, nil)
	if len(msgs) != 0 {
		t.Fatalf("expected no messages, got:\n%v", msgs)
	}
}

func TestAnalyzeHighAvailabilityFlagsSingleReplicaGateway(t *testing.T) {
	replicas := int32(1)
	gateway := appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{Name: "public-dubbo", Namespace: "app"},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{MatchLabels: map[string]string{
				"app.kubernetes.io/name":     "dxgate",
				"app.kubernetes.io/instance": "public-dubbo",
			}},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{Labels: map[string]string{
					"app.kubernetes.io/name":     "dxgate",
					"app.kubernetes.io/instance": "public-dubbo",
				}},
			},
		},
	}

	msgs := analyzeHighAvailability([]appsv1.Deployment{gateway}, nil, nil)
	if !messagesContain(msgs, "gateway runs a single replica") {
		t.Fatalf("single replica gateway not reported: %v", msgs)
	}
}
