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
