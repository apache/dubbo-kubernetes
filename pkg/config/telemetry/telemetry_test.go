//
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

package telemetry

import (
	"testing"
	"time"

	api "github.com/kdubbo/api/telemetry/v1alpha1"
	typeapi "github.com/kdubbo/api/type/v1alpha3"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

func TestResolveTelemetryLevels(t *testing.T) {
	resources := []Resource{
		{
			Name: "mesh-default", Namespace: "dubbo-system", CreationTimestamp: time.Unix(1, 0),
			Spec: &api.Telemetry{Tracing: []*api.Tracing{{
				Providers:                []*api.Tracing_TracingProvider{{Name: "localtrace"}},
				Tags:                     []*api.Tracing_Tag{{Name: "foo", Value: "bar"}},
				RandomSamplingPercentage: wrapperspb.Double(100),
			}}},
		},
		{
			Name: "namespace-override", Namespace: "myapp", CreationTimestamp: time.Unix(2, 0),
			Spec: &api.Telemetry{Tracing: []*api.Tracing{{
				Tags: []*api.Tracing_Tag{{Name: "userId", Value: "unknown"}},
			}}},
		},
		{
			Name: "workload-override", Namespace: "myapp", CreationTimestamp: time.Unix(3, 0),
			Spec: &api.Telemetry{
				Selector: &typeapi.WorkloadSelector{MatchLabels: map[string]string{"app": "frontend"}},
				Tracing:  []*api.Tracing{{DisableSpanReporting: wrapperspb.Bool(true)}},
			},
		},
	}

	got := Resolve(resources, "dubbo-system", "myapp", map[string]string{"app": "frontend"})
	if got.Provider() != "localtrace" {
		t.Fatalf("provider = %q, want localtrace", got.Provider())
	}
	if got.SamplingPercentage() != 100 {
		t.Fatalf("sampling = %v, want 100", got.SamplingPercentage())
	}
	if got.ResourceAttributes() != "userId=unknown" {
		t.Fatalf("tags = %q, want namespace replacement", got.ResourceAttributes())
	}
	if !got.Disabled() {
		t.Fatal("workload override did not disable span reporting")
	}
}

func TestMeshlevelSelectorIsIgnored(t *testing.T) {
	resources := []Resource{{
		Name: "invalid", Namespace: "dubbo-system",
		Spec: &api.Telemetry{
			Selector: &typeapi.WorkloadSelector{MatchLabels: map[string]string{"app": "frontend"}},
			Tracing:  []*api.Tracing{{Providers: []*api.Tracing_TracingProvider{{Name: "localtrace"}}}},
		},
	}}
	got := Resolve(resources, "dubbo-system", "myapp", map[string]string{"app": "frontend"})
	if got.Configured {
		t.Fatal("selector in mesh namespace must not apply")
	}
}

func TestProviderEndpoint(t *testing.T) {
	if got, want := ProviderEndpoint("localtrace", "dubbo-system"), "http://tracing.dubbo-system.svc:4317"; got != want {
		t.Fatalf("endpoint = %q, want %q", got, want)
	}
}
