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
	"google.golang.org/grpc/credentials/insecure"
)

// An empty address is how a cluster without KEDA runs unchanged: no listener,
// no error, and the rest of the control plane comes up as usual.
func TestServeWithoutAddressIsANoOp(t *testing.T) {
	server := NewServer()
	stop := make(chan struct{})
	defer close(stop)

	if err := server.Serve("", stop); err != nil {
		t.Fatalf("Serve() with no address error = %v", err)
	}
}

func TestServeReportsAnUnusableAddress(t *testing.T) {
	// Occupy a port, then ask the scaler for the same one.
	taken, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = taken.Close() }()

	server := NewServer()
	stop := make(chan struct{})
	defer close(stop)

	if err := server.Serve(taken.Addr().String(), stop); err == nil {
		t.Fatal("Serve() on an occupied address returned no error")
	}
}

// The registry the gateways report into and the scaler KEDA dials have to be
// the same one, or demand would be recorded where nothing reads it.
func TestServerSharesOneRegistryBetweenReportsAndScaler(t *testing.T) {
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
	client := externalscaler.NewExternalScalerClient(connection)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	server.Registry().Report("gateway-a", orders, 2)
	active, err := client.IsActive(ctx, serviceRef())
	if err != nil {
		t.Fatalf("IsActive() error = %v", err)
	}
	if !active.GetResult() {
		t.Fatal("IsActive() = false after demand was reported to the server registry")
	}
}
