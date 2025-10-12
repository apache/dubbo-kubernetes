/*
 * Licensed to the Apache Software Foundation (ASF) under one or more
 * contributor license agreements.  See the NOTICE file distributed with
 * this work for additional information regarding copyright ownership.
 * The ASF licenses this file to You under the Apache License, Version 2.0
 * (the "License"); you may not use this file except in compliance with
 * the License.  You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package controller

import (
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	"github.com/apache/dubbo-kubernetes/pkg/config/host"
	"github.com/apache/dubbo-kubernetes/sail/pkg/serviceregistry/provider"
)

// TestControllerInitialization tests that the controller can be initialized properly
func TestControllerInitialization(t *testing.T) {
	// Create controller options
	opts := Options{
		ClusterID:                 "test-cluster",
		EnableK8sServiceDiscovery: true,
		K8sServiceNamespaces:      []string{"default", "kube-system"},
		DubboAnnotationPrefix:     "dubbo.apache.org",
	}

	// Test controller creation with nil client (for testing)
	controller := NewController(opts, nil)
	if controller == nil {
		t.Fatal("Expected controller to be created, got nil")
	}

	// Verify controller properties
	if controller.opts.ClusterID != "test-cluster" {
		t.Errorf("Expected cluster ID 'test-cluster', got '%s'", controller.opts.ClusterID)
	}

	if !controller.opts.EnableK8sServiceDiscovery {
		t.Error("Expected k8s service discovery to be enabled")
	}
}

func TestConvertK8sServiceToDubboService(t *testing.T) {
	// Create controller with nil client for testing
	opts := Options{
		ClusterID:                 "test-cluster",
		EnableK8sServiceDiscovery: true,
		K8sServiceNamespaces:      []string{"default"},
		DubboAnnotationPrefix:     "dubbo.apache.org",
	}
	controller := NewController(opts, nil)

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
					CreationTimestamp: metav1.Now(),
				},
				Spec: corev1.ServiceSpec{
					Ports: []corev1.ServicePort{
						{
							Name:       "dubbo",
							Port:       20880,
							TargetPort: intstr.FromInt(20880),
							Protocol:   corev1.ProtocolTCP,
						},
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
