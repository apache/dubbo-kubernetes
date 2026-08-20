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

	api "github.com/kdubbo/api/telemetry/v1alpha3"
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
			}}, Metrics: []*api.Metrics{{
				Providers: []*api.Metrics_MetricsProvider{{Name: PrometheusProvider}},
				Rules: []*api.MetricRule{{
					Metric: api.StandardMetric_REQUEST_COUNT,
					Scope:  api.MetricScope_CLIENT_AND_SERVER,
					Tags: map[string]*api.TagOverride{
						"grpc_response_status": {Action: api.TagOverride_REMOVE},
					},
				}},
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
	if !got.MetricsEnabled() || got.MetricsProvider() != PrometheusProvider {
		t.Fatalf("metrics = enabled:%v provider:%q", got.MetricsEnabled(), got.MetricsProvider())
	}
	if len(got.MetricRules) != 1 ||
		got.MetricRules[0].Metric != api.StandardMetric_REQUEST_COUNT ||
		got.MetricRules[0].Scope != api.MetricScope_CLIENT_AND_SERVER ||
		len(got.MetricRules[0].Tags) != 1 ||
		got.MetricRules[0].Tags[0].Name != "grpc_response_status" ||
		got.MetricRules[0].Tags[0].Action != api.TagOverride_REMOVE {
		t.Fatalf("metric rules = %#v", got.MetricRules)
	}
}

func TestResolveMetricsDisableOverride(t *testing.T) {
	resources := []Resource{
		{
			Name: "mesh-default", Namespace: "dubbo-system",
			Spec: &api.Telemetry{Metrics: []*api.Metrics{{
				Providers: []*api.Metrics_MetricsProvider{{Name: PrometheusProvider}},
			}}},
		},
		{
			Name: "namespace-disable", Namespace: "myapp",
			Spec: &api.Telemetry{Metrics: []*api.Metrics{{
				Enabled: wrapperspb.Bool(false),
			}}},
		},
	}

	got := Resolve(resources, "dubbo-system", "myapp", nil)
	if got.MetricsEnabled() {
		t.Fatal("namespace override did not disable metrics")
	}
	if got.MetricsProvider() != PrometheusProvider {
		t.Fatalf("provider = %q, want inherited prometheus", got.MetricsProvider())
	}
}

func TestResolveMetricsRulesOverride(t *testing.T) {
	resources := []Resource{
		{
			Name: "mesh-default", Namespace: "dubbo-system",
			Spec: &api.Telemetry{Metrics: []*api.Metrics{{
				Providers: []*api.Metrics_MetricsProvider{{Name: PrometheusProvider}},
				Rules: []*api.MetricRule{{
					Metric: api.StandardMetric_REQUEST_COUNT,
					Scope:  api.MetricScope_CLIENT,
				}},
			}}},
		},
		{
			Name: "namespace-rules", Namespace: "myapp",
			Spec: &api.Telemetry{Metrics: []*api.Metrics{{
				Rules: []*api.MetricRule{{
					Metric: api.StandardMetric_REQUEST_COUNT,
					Scope:  api.MetricScope_SERVER,
				}},
			}}},
		},
	}

	got := Resolve(resources, "dubbo-system", "myapp", nil)
	if len(got.MetricRules) != 1 || got.MetricRules[0].Scope != api.MetricScope_SERVER {
		t.Fatalf("metric rules = %#v, want namespace replacement", got.MetricRules)
	}
}

func TestResolveLoggingOverride(t *testing.T) {
	resources := []Resource{
		{
			Name: "mesh-default", Namespace: "dubbo-system",
			Spec: &api.Telemetry{Logging: []*api.Logging{{
				Providers: []*api.Logging_LoggingProvider{{Name: OTELLogProvider}},
				Tags:      []*api.Logging_Tag{{Name: "mesh", Value: "default"}},
			}}},
		},
		{
			Name: "workload-override", Namespace: "myapp",
			Spec: &api.Telemetry{
				Selector: &typeapi.WorkloadSelector{MatchLabels: map[string]string{"app": "frontend"}},
				Logging: []*api.Logging{{
					Providers: []*api.Logging_LoggingProvider{{Name: OTELLogProvider}},
					Match:     &api.Logging_Match{Mode: api.Logging_Match_SERVER},
					Filter:    &api.Logging_Filter{Expression: "response.code >= 500"},
					Tags:      []*api.Logging_Tag{{Name: "environment", Value: "test"}},
				}},
			},
		},
	}

	got := Resolve(resources, "dubbo-system", "myapp", map[string]string{"app": "frontend"})
	if !got.LoggingConfigured || len(got.Logging) != 1 {
		t.Fatalf("logging = %#v", got.Logging)
	}
	rule := got.Logging[0]
	if len(rule.Providers) != 1 || rule.Providers[0] != OTELLogProvider ||
		rule.Mode != api.Logging_Match_SERVER ||
		rule.FilterExpression != "response.code >= 500" ||
		len(rule.Tags) != 1 || rule.Tags[0] != (Tag{Name: "environment", Value: "test"}) {
		t.Fatalf("logging rule = %#v", rule)
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
	if got, want := ProviderEndpoint("otel", "dubbo-system"), "http://opentelemetry-collector.dubbo-system.svc:4317"; got != want {
		t.Fatalf("logging endpoint = %q, want %q", got, want)
	}
}
