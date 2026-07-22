// Licensed to the Apache Software Foundation (ASF) under one or more
// contributor license agreements.  See the NOTICE file distributed with
// this work for additional information regarding copyright ownership.
// The ASF licenses this file to You under the Apache License, Version 2.0.

package crd

import (
	"testing"

	"github.com/apache/dubbo-kubernetes/pkg/config/schema/collections"
	telemetry "github.com/kdubbo/api/telemetry/v1alpha1"
)

func TestFromJSONTelemetryScalarWrappers(t *testing.T) {
	spec, err := FromJSON(collections.Telemetry, `{
		"tracing": [{
			"randomSamplingPercentage": 100,
			"disableSpanReporting": true
		}]
	}`)
	if err != nil {
		t.Fatalf("FromJSON() error = %v", err)
	}

	got := spec.(*telemetry.Telemetry).GetTracing()[0]
	if got.GetRandomSamplingPercentage().GetValue() != 100 {
		t.Fatalf("randomSamplingPercentage = %v, want 100", got.GetRandomSamplingPercentage().GetValue())
	}
	if !got.GetDisableSpanReporting().GetValue() {
		t.Fatal("disableSpanReporting = false, want true")
	}
}
