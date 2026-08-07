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
	"net"
	"testing"
	"time"

	"github.com/apache/dubbo-kubernetes/dubbod/discovery/pkg/activation/externalscaler"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
)

// startScaler serves the scaler over a real gRPC connection, so the wire
// contract is exercised the way KEDA drives it rather than through a stub.
func startScaler(t *testing.T, registry *Registry) externalscaler.ExternalScalerClient {
	t.Helper()

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	server := grpc.NewServer()
	externalscaler.RegisterExternalScalerServer(server, NewScaler(registry))
	go func() { _ = server.Serve(listener) }()

	connection, err := grpc.NewClient(listener.Addr().String(),
		grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		server.Stop()
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = connection.Close()
		server.Stop()
	})
	return externalscaler.NewExternalScalerClient(connection)
}

func TestScalerOverGRPCDrivesActivation(t *testing.T) {
	registry := NewRegistry()
	client := startScaler(t, registry)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	stream, err := client.StreamIsActive(ctx, serviceRef())
	if err != nil {
		t.Fatalf("StreamIsActive() error = %v", err)
	}

	first, err := stream.Recv()
	if err != nil {
		t.Fatalf("first Recv() error = %v", err)
	}
	if first.GetResult() {
		t.Fatal("first streamed value = true, want false with no demand")
	}

	registry.Report("gateway-a", orders, 4)
	next, err := stream.Recv()
	if err != nil {
		t.Fatalf("Recv() after demand error = %v", err)
	}
	if !next.GetResult() {
		t.Fatal("streamed value after demand = false, want true")
	}

	// The polling path has to agree with the pushed one; KEDA falls back to it
	// whenever it rebuilds its scaler state.
	active, err := client.IsActive(ctx, serviceRef())
	if err != nil {
		t.Fatalf("IsActive() error = %v", err)
	}
	if !active.GetResult() {
		t.Fatal("IsActive() = false while the stream reported active")
	}

	spec, err := client.GetMetricSpec(ctx, serviceRef())
	if err != nil {
		t.Fatalf("GetMetricSpec() error = %v", err)
	}
	metrics, err := client.GetMetrics(ctx, &externalscaler.GetMetricsRequest{
		ScaledObjectRef: serviceRef(),
		MetricName:      spec.GetMetricSpecs()[0].GetMetricName(),
	})
	if err != nil {
		t.Fatalf("GetMetrics() error = %v", err)
	}
	if got := metrics.GetMetricValues()[0].GetMetricValueFloat(); got != 4 {
		t.Fatalf("metricValueFloat = %v, want 4", got)
	}
}

func TestScalerOverGRPCReportsInvalidTriggerAsStatusError(t *testing.T) {
	client := startScaler(t, NewRegistry())

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if _, err := client.IsActive(ctx, ref(map[string]string{})); status.Code(err) != codes.InvalidArgument {
		t.Fatalf("IsActive() error = %v, want InvalidArgument", err)
	}

	stream, err := client.StreamIsActive(ctx, ref(map[string]string{}))
	if err != nil {
		t.Fatalf("StreamIsActive() error = %v", err)
	}
	if _, err := stream.Recv(); status.Code(err) != codes.InvalidArgument {
		t.Fatalf("Recv() error = %v, want InvalidArgument", err)
	}
}

// StreamMetricSpec is optional. KEDA falls back to polling GetMetricSpec when
// it is unimplemented, so returning Unimplemented has to be the actual
// behavior rather than a crash or an empty stream.
func TestStreamMetricSpecIsUnimplemented(t *testing.T) {
	client := startScaler(t, NewRegistry())

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	stream, err := client.StreamMetricSpec(ctx, serviceRef())
	if err != nil {
		t.Fatalf("StreamMetricSpec() error = %v", err)
	}
	if _, err := stream.Recv(); status.Code(err) != codes.Unimplemented {
		t.Fatalf("Recv() error = %v, want Unimplemented", err)
	}
}
