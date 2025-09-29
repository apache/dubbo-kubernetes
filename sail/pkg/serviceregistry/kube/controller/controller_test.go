package controller

import (
	"testing"

	"github.com/apache/dubbo-kubernetes/pkg/config/host"
	"github.com/apache/dubbo-kubernetes/sail/pkg/serviceregistry/provider"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestConvertK8sServiceToDubboService(t *testing.T) {
	controller := &Controller{}

	tests := []struct {
		name         string
		k8sService   *corev1.Service
		expectNil    bool
		expectedHost host.Name
	}{
		{
			name:       "nil service",
			k8sService: nil,
			expectNil:  true,
		},
		{
			name: "service without dubbo annotations",
			k8sService: &corev1.Service{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-service",
					Namespace: "default",
				},
			},
			expectNil: true,
		},
		{
			name: "service with dubbo annotations",
			k8sService: &corev1.Service{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-service",
					Namespace: "default",
					Annotations: map[string]string{
						"dubbo.apache.org/service-name": "com.example.UserService",
						"dubbo.apache.org/version":      "1.0.0",
						"dubbo.apache.org/group":        "test",
					},
				},
			},
			expectNil:    false,
			expectedHost: "com.example.UserService",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := controller.convertK8sServiceToDubboService(tt.k8sService)
			if tt.expectNil {
				if result != nil {
					t.Errorf("expected nil result, got %v", result)
				}
			} else {
				if result == nil {
					t.Fatal("expected non-nil result")
				}
				if result.Hostname != tt.expectedHost {
					t.Errorf("expected hostname %s, got %s", tt.expectedHost, result.Hostname)
				}
				if result.Attributes.ServiceRegistry != provider.Kubernetes {
					t.Errorf("expected provider %s, got %s", provider.Kubernetes, result.Attributes.ServiceRegistry)
				}
			}
		})
	}
}
