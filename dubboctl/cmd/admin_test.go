package cmd

import (
	"context"
	"reflect"
	"testing"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

func TestDeploymentPodsUsesDeploymentSelector(t *testing.T) {
	client := fake.NewSimpleClientset(
		&appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{Name: "dubbod", Namespace: "dubbo-system"},
			Spec: appsv1.DeploymentSpec{
				Selector: &metav1.LabelSelector{MatchLabels: map[string]string{"app": "dubbod"}},
				Template: corev1.PodTemplateSpec{
					ObjectMeta: metav1.ObjectMeta{Labels: map[string]string{"app": "dubbod"}},
				},
			},
		},
		&corev1.Pod{ObjectMeta: metav1.ObjectMeta{Name: "dubbod-b", Namespace: "dubbo-system", Labels: map[string]string{"app": "dubbod"}}},
		&corev1.Pod{ObjectMeta: metav1.ObjectMeta{Name: "dubbod-a", Namespace: "dubbo-system", Labels: map[string]string{"app": "dubbod"}}},
		&corev1.Pod{ObjectMeta: metav1.ObjectMeta{Name: "other", Namespace: "dubbo-system", Labels: map[string]string{"app": "other"}}},
	)

	pods, err := deploymentPods(context.Background(), client, "dubbo-system", "dubbod")
	if err != nil {
		t.Fatalf("deploymentPods() returned error: %v", err)
	}
	got := []string{}
	for _, pod := range pods {
		got = append(got, pod.Name)
	}
	want := []string{"dubbod-a", "dubbod-b"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("deploymentPods() = %v, want %v", got, want)
	}
}

func TestLogContainersPrefersRequestedContainer(t *testing.T) {
	pod := corev1.Pod{
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{{Name: "app"}, {Name: "execute"}},
		},
	}
	if got := logContainers(pod, "execute"); !reflect.DeepEqual(got, []string{"execute"}) {
		t.Fatalf("logContainers() = %v, want [execute]", got)
	}
	if got := logContainers(pod, "missing"); !reflect.DeepEqual(got, []string{"app", "execute"}) {
		t.Fatalf("logContainers() fallback = %v, want all containers", got)
	}
}
