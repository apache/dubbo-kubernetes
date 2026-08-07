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

	"github.com/apache/dubbo-kubernetes/dubbod/discovery/pkg/activation/demandpb"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
)

var reviews = Target{Namespace: "app", Name: "reviews"}

func startDemand(t *testing.T, registry *Registry) demandpb.ActivationDemandClient {
	t.Helper()

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	server := grpc.NewServer()
	demandpb.RegisterActivationDemandServer(server, NewDemandService(registry))
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
	return demandpb.NewActivationDemandClient(connection)
}

func snapshot(reporter string, targets ...*demandpb.TargetDemand) *demandpb.DemandSnapshot {
	return &demandpb.DemandSnapshot{Reporter: reporter, Targets: targets}
}

func demandFor(target Target, pending int64) *demandpb.TargetDemand {
	return &demandpb.TargetDemand{
		Namespace: target.Namespace,
		Service:   target.Name,
		Pending:   pending,
	}
}

func waitFor(t *testing.T, condition func() bool, message string) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if condition() {
			return
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatal(message)
}

func TestReportStreamFeedsTheRegistry(t *testing.T) {
	registry := NewRegistry()
	client := startDemand(t, registry)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	stream, err := client.Report(ctx)
	if err != nil {
		t.Fatalf("Report() error = %v", err)
	}
	if err := stream.Send(snapshot("gateway-a", demandFor(orders, 3), demandFor(reviews, 1))); err != nil {
		t.Fatalf("Send() error = %v", err)
	}
	waitFor(t, func() bool { return registry.Pending(orders) == 3 && registry.Pending(reviews) == 1 },
		"registry did not pick up the first snapshot")

	// A target dropped from the snapshot has drained: there is no separate
	// clear message, so its absence has to be what clears it.
	if err := stream.Send(snapshot("gateway-a", demandFor(orders, 2))); err != nil {
		t.Fatalf("Send() error = %v", err)
	}
	waitFor(t, func() bool { return registry.Pending(orders) == 2 && registry.Pending(reviews) == 0 },
		"omitted target was not cleared")

	summary, err := stream.CloseAndRecv()
	if err != nil {
		t.Fatalf("CloseAndRecv() error = %v", err)
	}
	if summary.GetSnapshots() != 2 {
		t.Fatalf("snapshots = %d, want 2", summary.GetSnapshots())
	}
}

// The stream is the gateway's liveness. When it ends the demand has to go with
// it, or a rolling gateway update holds the workload up for a whole TTL after
// the old pod is gone.
func TestClosingTheStreamForgetsTheGateway(t *testing.T) {
	registry := NewRegistry()
	client := startDemand(t, registry)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	stream, err := client.Report(ctx)
	if err != nil {
		t.Fatalf("Report() error = %v", err)
	}
	if err := stream.Send(snapshot("gateway-a", demandFor(orders, 5))); err != nil {
		t.Fatalf("Send() error = %v", err)
	}
	waitFor(t, func() bool { return registry.Pending(orders) == 5 }, "snapshot was not recorded")

	if _, err := stream.CloseAndRecv(); err != nil {
		t.Fatalf("CloseAndRecv() error = %v", err)
	}
	waitFor(t, func() bool { return registry.Pending(orders) == 0 },
		"demand survived the stream closing")
}

// Every replica gets the same broadcast, so two gateways reporting the same
// target must add up rather than overwrite one another.
func TestReportsFromSeveralGatewaysAccumulate(t *testing.T) {
	registry := NewRegistry()
	client := startDemand(t, registry)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	first, err := client.Report(ctx)
	if err != nil {
		t.Fatal(err)
	}
	second, err := client.Report(ctx)
	if err != nil {
		t.Fatal(err)
	}

	if err := first.Send(snapshot("gateway-a", demandFor(orders, 2))); err != nil {
		t.Fatal(err)
	}
	if err := second.Send(snapshot("gateway-b", demandFor(orders, 3))); err != nil {
		t.Fatal(err)
	}
	waitFor(t, func() bool { return registry.Pending(orders) == 5 },
		"reports from two gateways did not accumulate")

	if _, err := first.CloseAndRecv(); err != nil {
		t.Fatal(err)
	}
	waitFor(t, func() bool { return registry.Pending(orders) == 3 },
		"closing one gateway removed more than its own demand")
}

func TestReportRejectsMalformedSnapshots(t *testing.T) {
	tests := []struct {
		name string
		send *demandpb.DemandSnapshot
	}{
		{name: "no reporter", send: snapshot("", demandFor(orders, 1))},
		{name: "blank reporter", send: snapshot("   ", demandFor(orders, 1))},
		{
			name: "target without namespace",
			send: snapshot("gateway-a", &demandpb.TargetDemand{Service: "orders", Pending: 1}),
		},
		{
			name: "target without service",
			send: snapshot("gateway-a", &demandpb.TargetDemand{Namespace: "app", Pending: 1}),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			client := startDemand(t, NewRegistry())
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			stream, err := client.Report(ctx)
			if err != nil {
				t.Fatal(err)
			}
			// Send may or may not observe the error depending on when the
			// server rejects it; CloseAndRecv is where it always surfaces.
			_ = stream.Send(test.send)
			if _, err := stream.CloseAndRecv(); status.Code(err) != codes.InvalidArgument {
				t.Fatalf("CloseAndRecv() error = %v, want InvalidArgument", err)
			}
		})
	}
}

// One stream is one gateway. Allowing a second identity would strand the
// first one's demand with nothing left to refresh it.
func TestReportRejectsAChangedReporter(t *testing.T) {
	client := startDemand(t, NewRegistry())
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	stream, err := client.Report(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if err := stream.Send(snapshot("gateway-a", demandFor(orders, 1))); err != nil {
		t.Fatal(err)
	}
	_ = stream.Send(snapshot("gateway-b", demandFor(orders, 1)))

	if _, err := stream.CloseAndRecv(); status.Code(err) != codes.InvalidArgument {
		t.Fatalf("CloseAndRecv() error = %v, want InvalidArgument", err)
	}
}

// Demand reported to this replica has to be visible to the KEDA stream this
// replica is serving; that is the whole reason both live on one listener.
func TestReportedDemandActivatesTheScalerOnTheSameReplica(t *testing.T) {
	server := NewServer()

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	address := listener.Addr().String()
	_ = listener.Close()

	stop := make(chan struct{})
	if err := server.Serve(address, stop); err != nil {
		t.Fatalf("Serve() error = %v", err)
	}
	t.Cleanup(func() { close(stop) })

	connection, err := grpc.NewClient(address, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = connection.Close() })

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	stream, err := demandpb.NewActivationDemandClient(connection).Report(ctx)
	if err != nil {
		t.Fatalf("Report() error = %v", err)
	}
	if err := stream.Send(snapshot("gateway-a", demandFor(orders, 1))); err != nil {
		t.Fatal(err)
	}

	waitFor(t, func() bool { return server.Registry().Pending(orders) == 1 },
		"demand did not reach the registry")
}
