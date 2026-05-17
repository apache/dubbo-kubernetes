package cmd

import (
	"strings"
	"testing"
	"time"

	"github.com/apache/dubbo-kubernetes/pkg/kube/inject"
	"github.com/kdubbo/api/annotation"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestIsInjectedPod(t *testing.T) {
	tests := []struct {
		name string
		pod  corev1.Pod
		want bool
	}{
		{
			name: "template annotation",
			pod: corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						annotation.OrgApacheDubboInjectTemplates.Name: inject.ProxylessGRPCTemplateName,
					},
				},
			},
			want: true,
		},
		{
			name: "xserver container",
			pod: corev1.Pod{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{{Name: inject.ProxylessXServerContainerName}},
				},
			},
			want: true,
		},
		{
			name: "proxyless env",
			pod: corev1.Pod{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{{
						Name: "app",
						Env:  []corev1.EnvVar{{Name: "GRPC_XDS_BOOTSTRAP", Value: inject.ProxylessGRPCBootstrapPath}},
					}},
				},
			},
			want: true,
		},
		{
			name: "plain pod",
			pod: corev1.Pod{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{{Name: "app"}},
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := isInjectedPod(test.pod); got != test.want {
				t.Fatalf("isInjectedPod() = %v, want %v", got, test.want)
			}
		})
	}
}

func TestPrintInjectedPods(t *testing.T) {
	pod := corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "app-1",
			Namespace:         "app",
			CreationTimestamp: metav1.NewTime(time.Unix(100, 0)),
			Annotations: map[string]string{
				annotation.OrgApacheDubboInjectTemplates.Name: inject.ProxylessGRPCTemplateName,
			},
		},
		Spec: corev1.PodSpec{
			NodeName:   "node-a",
			Containers: []corev1.Container{{Name: "app"}, {Name: inject.ProxylessXServerContainerName}},
		},
		Status: corev1.PodStatus{
			Phase: corev1.PodRunning,
			PodIP: "10.0.0.7",
			ContainerStatuses: []corev1.ContainerStatus{
				{Name: "app", Ready: true, RestartCount: 1},
				{Name: inject.ProxylessXServerContainerName, Ready: true},
			},
		},
	}

	var out strings.Builder
	if err := printInjectedPods(&out, []corev1.Pod{pod}, time.Unix(160, 0)); err != nil {
		t.Fatalf("printInjectedPods() returned error: %v", err)
	}
	got := out.String()
	for _, want := range []string{"NAMESPACE", "app-1", "2/2", "Running", "10.0.0.7", inject.ProxylessGRPCTemplateName} {
		if !strings.Contains(got, want) {
			t.Fatalf("output %q missing %q", got, want)
		}
	}
}
