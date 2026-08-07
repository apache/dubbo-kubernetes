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

package activation

import (
	"context"
	"testing"
	"time"

	"github.com/apache/dubbo-kubernetes/dubbod/discovery/pkg/activation/externalscaler"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

func ref(scalerMetadata map[string]string) *externalscaler.ScaledObjectRef {
	return &externalscaler.ScaledObjectRef{
		Name:           "orders",
		Namespace:      "app",
		ScalerMetadata: scalerMetadata,
	}
}

func serviceRef() *externalscaler.ScaledObjectRef {
	return ref(map[string]string{serviceMetadataKey: "orders"})
}

func TestIsActiveReflectsPendingDemand(t *testing.T) {
	registry := NewRegistry()
	scaler := NewScaler(registry)

	response, err := scaler.IsActive(context.Background(), serviceRef())
	if err != nil {
		t.Fatalf("IsActive() error = %v", err)
	}
	if response.GetResult() {
		t.Fatal("IsActive() = true with no pending requests")
	}

	registry.Report("gateway-a", orders, 1)
	response, err = scaler.IsActive(context.Background(), serviceRef())
	if err != nil {
		t.Fatalf("IsActive() error = %v", err)
	}
	if !response.GetResult() {
		t.Fatal("IsActive() = false while a request is held")
	}
}

// Guessing the target from the ScaledObject name would silently activate the
// wrong workload whenever the two differ, so the metadata is required.
func TestTriggerMetadataIsValidated(t *testing.T) {
	scaler := NewScaler(NewRegistry())

	tests := []struct {
		name string
		ref  *externalscaler.ScaledObjectRef
	}{
		{name: "nil reference", ref: nil},
		{name: "missing service", ref: ref(map[string]string{})},
		{name: "blank service", ref: ref(map[string]string{serviceMetadataKey: "  "})},
		{
			name: "unparsable target",
			ref: ref(map[string]string{
				serviceMetadataKey:       "orders",
				targetPendingMetadataKey: "many",
			}),
		},
		{
			name: "zero target would divide by zero in HPA",
			ref: ref(map[string]string{
				serviceMetadataKey:       "orders",
				targetPendingMetadataKey: "0",
			}),
		},
		{
			name: "negative target",
			ref: ref(map[string]string{
				serviceMetadataKey:       "orders",
				targetPendingMetadataKey: "-1",
			}),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if _, err := scaler.IsActive(context.Background(), test.ref); status.Code(err) != codes.InvalidArgument {
				t.Fatalf("IsActive() error = %v, want InvalidArgument", err)
			}
		})
	}
}

func TestNamespaceDefaultsToScaledObjectAndCanBeOverridden(t *testing.T) {
	registry := NewRegistry()
	scaler := NewScaler(registry)

	// Demand recorded in the ScaledObject's own namespace is picked up without
	// any namespace metadata.
	registry.Report("gateway-a", orders, 1)
	response, err := scaler.IsActive(context.Background(), serviceRef())
	if err != nil {
		t.Fatalf("IsActive() error = %v", err)
	}
	if !response.GetResult() {
		t.Fatal("IsActive() = false for a target in the ScaledObject namespace")
	}

	// With an override, the same ScaledObject must look elsewhere and see none.
	overridden := ref(map[string]string{
		serviceMetadataKey:   "orders",
		namespaceMetadataKey: "staging",
	})
	response, err = scaler.IsActive(context.Background(), overridden)
	if err != nil {
		t.Fatalf("IsActive() with override error = %v", err)
	}
	if response.GetResult() {
		t.Fatal("IsActive() = true for a namespace with no demand")
	}
}

func TestGetMetricSpecAdvertisesTargetAndName(t *testing.T) {
	scaler := NewScaler(NewRegistry())

	spec, err := scaler.GetMetricSpec(context.Background(), serviceRef())
	if err != nil {
		t.Fatalf("GetMetricSpec() error = %v", err)
	}
	if len(spec.GetMetricSpecs()) != 1 {
		t.Fatalf("metric specs = %d, want 1", len(spec.GetMetricSpecs()))
	}
	got := spec.GetMetricSpecs()[0]
	if want := "dubbo-activation-app-orders"; got.GetMetricName() != want {
		t.Fatalf("metric name = %q, want %q", got.GetMetricName(), want)
	}
	if got.GetTargetSizeFloat() != defaultTargetPendingRequests {
		t.Fatalf("targetSizeFloat = %v, want %v", got.GetTargetSizeFloat(), defaultTargetPendingRequests)
	}
	// Older KEDA releases still read the deprecated integer field, so it must
	// agree with the float rather than being left at zero.
	if got.GetTargetSize() != int64(defaultTargetPendingRequests) {
		t.Fatalf("targetSize = %d, want %d", got.GetTargetSize(), int64(defaultTargetPendingRequests))
	}

	custom := ref(map[string]string{
		serviceMetadataKey:       "orders",
		targetPendingMetadataKey: "5",
	})
	spec, err = scaler.GetMetricSpec(context.Background(), custom)
	if err != nil {
		t.Fatalf("GetMetricSpec() with custom target error = %v", err)
	}
	if got := spec.GetMetricSpecs()[0].GetTargetSizeFloat(); got != 5 {
		t.Fatalf("targetSizeFloat = %v, want 5", got)
	}
}

func TestGetMetricsReportsPendingUnderTheAdvertisedName(t *testing.T) {
	registry := NewRegistry()
	scaler := NewScaler(registry)
	registry.Report("gateway-a", orders, 3)

	spec, err := scaler.GetMetricSpec(context.Background(), serviceRef())
	if err != nil {
		t.Fatalf("GetMetricSpec() error = %v", err)
	}
	name := spec.GetMetricSpecs()[0].GetMetricName()

	metrics, err := scaler.GetMetrics(context.Background(), &externalscaler.GetMetricsRequest{
		ScaledObjectRef: serviceRef(),
		MetricName:      name,
	})
	if err != nil {
		t.Fatalf("GetMetrics() error = %v", err)
	}
	if len(metrics.GetMetricValues()) != 1 {
		t.Fatalf("metric values = %d, want 1", len(metrics.GetMetricValues()))
	}
	value := metrics.GetMetricValues()[0]
	if value.GetMetricName() != name {
		t.Fatalf("metric name = %q, want %q", value.GetMetricName(), name)
	}
	if value.GetMetricValueFloat() != 3 {
		t.Fatalf("metricValueFloat = %v, want 3", value.GetMetricValueFloat())
	}
	if value.GetMetricValue() != 3 {
		t.Fatalf("metricValue = %d, want 3", value.GetMetricValue())
	}
}

// A name GetMetricSpec never advertised means the two calls disagree about what
// is being measured; scaling on that would use the wrong signal.
func TestGetMetricsRejectsUnknownMetricName(t *testing.T) {
	scaler := NewScaler(NewRegistry())

	_, err := scaler.GetMetrics(context.Background(), &externalscaler.GetMetricsRequest{
		ScaledObjectRef: serviceRef(),
		MetricName:      "some-other-metric",
	})
	if status.Code(err) != codes.InvalidArgument {
		t.Fatalf("GetMetrics() error = %v, want InvalidArgument", err)
	}
}

// KEDA expects the current state as soon as the stream opens, not only on the
// next change; otherwise a workload with requests already waiting stays at zero.
func TestStreamIsActiveSendsCurrentStateImmediately(t *testing.T) {
	registry := NewRegistry()
	scaler := NewScaler(registry)
	registry.Report("gateway-a", orders, 2)

	stream := newFakeStream(context.Background())
	go func() { _ = scaler.StreamIsActive(serviceRef(), stream) }()

	if got := stream.next(t); !got {
		t.Fatal("first streamed value = false, want true for existing demand")
	}
}

func TestStreamIsActivePushesTransitions(t *testing.T) {
	registry := NewRegistry()
	scaler := NewScaler(registry)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	stream := newFakeStream(ctx)
	errCh := make(chan error, 1)
	go func() { errCh <- scaler.StreamIsActive(serviceRef(), stream) }()

	if got := stream.next(t); got {
		t.Fatal("first streamed value = true, want false with no demand")
	}

	registry.Report("gateway-a", orders, 1)
	if got := stream.next(t); !got {
		t.Fatal("streamed value after demand = false, want true")
	}

	registry.Report("gateway-a", orders, 0)
	if got := stream.next(t); got {
		t.Fatal("streamed value after drain = true, want false")
	}

	cancel()
	select {
	case <-errCh:
	case <-time.After(2 * time.Second):
		t.Fatal("StreamIsActive did not return after the stream context was canceled")
	}
}

func TestStreamIsActiveRejectsInvalidTrigger(t *testing.T) {
	scaler := NewScaler(NewRegistry())
	err := scaler.StreamIsActive(ref(map[string]string{}), newFakeStream(context.Background()))
	if status.Code(err) != codes.InvalidArgument {
		t.Fatalf("StreamIsActive() error = %v, want InvalidArgument", err)
	}
}

// fakeStream stands in for the gRPC server stream. Sends are buffered so the
// scaler is never blocked by the test's read pace.
type fakeStream struct {
	ctx      context.Context
	messages chan bool
}

func newFakeStream(ctx context.Context) *fakeStream {
	return &fakeStream{ctx: ctx, messages: make(chan bool, 16)}
}

func (s *fakeStream) next(t *testing.T) bool {
	t.Helper()
	select {
	case value := <-s.messages:
		return value
	case <-time.After(2 * time.Second):
		t.Fatal("no message streamed")
		return false
	}
}

func (s *fakeStream) Send(response *externalscaler.IsActiveResponse) error {
	s.messages <- response.GetResult()
	return nil
}

func (s *fakeStream) Context() context.Context     { return s.ctx }
func (s *fakeStream) SetHeader(metadata.MD) error  { return nil }
func (s *fakeStream) SendHeader(metadata.MD) error { return nil }
func (s *fakeStream) SetTrailer(metadata.MD)       {}
func (s *fakeStream) SendMsg(any) error            { return nil }
func (s *fakeStream) RecvMsg(any) error            { return nil }
