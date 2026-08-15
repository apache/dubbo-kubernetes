// Licensed to the Apache Software Foundation (ASF) under one or more
// contributor license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright ownership.
// The ASF licenses this file to You under the Apache License, Version 2.0.

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
