// Licensed to the Apache Software Foundation (ASF) under one or more
// contributor license agreements.  See the NOTICE file distributed with
// this work for additional information regarding copyright ownership.
// The ASF licenses this file to You under the Apache License, Version 2.0.

package crd

import (
	"testing"

	"github.com/apache/dubbo-kubernetes/pkg/config/schema/gvk"
	networking "github.com/kdubbo/api/networking/v1alpha3"
)

func TestParseInputsServiceEntryAndWorkloadEntry(t *testing.T) {
	configs, others, err := ParseInputs(`
apiVersion: networking.dubbo.apache.org/v1alpha3
kind: ServiceEntry
metadata:
  name: payment
  namespace: default
spec:
  hosts: [payment.mesh.test]
  addresses: [240.0.0.20]
  ports:
  - name: http
    number: 8080
    protocol: HTTP
  resolution: STATIC
  workloadSelector:
    matchLabels:
      app: payment
---
apiVersion: networking.dubbo.apache.org/v1alpha3
kind: WorkloadEntry
metadata:
  name: payment-local
  namespace: default
spec:
  address: 127.0.0.1
  ports:
    http: 18080
  labels:
    app: payment
`)
	if err != nil {
		t.Fatal(err)
	}
	if len(others) != 0 || len(configs) != 2 {
		t.Fatalf("configs=%d others=%d, want configs=2 others=0", len(configs), len(others))
	}
	if configs[0].GroupVersionKind != gvk.ServiceEntry {
		t.Fatalf("first kind = %v, want ServiceEntry", configs[0].GroupVersionKind)
	}
	service := configs[0].Spec.(*networking.ServiceEntry)
	if service.GetResolution() != networking.ServiceEntry_STATIC || service.GetPorts()[0].GetProtocol() != "HTTP" {
		t.Fatalf("unexpected ServiceEntry: %#v", service)
	}
	if configs[1].GroupVersionKind != gvk.WorkloadEntry {
		t.Fatalf("second kind = %v, want WorkloadEntry", configs[1].GroupVersionKind)
	}
	workload := configs[1].Spec.(*networking.WorkloadEntry)
	if workload.GetAddress() != "127.0.0.1" || workload.GetPorts()["http"] != 18080 {
		t.Fatalf("unexpected WorkloadEntry: %#v", workload)
	}
}
