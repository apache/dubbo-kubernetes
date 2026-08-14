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

package main

import (
	"context"
	"log"
	"net"
	"net/http"
	"os"
	"time"

	runtimetelemetry "github.com/kdubbo/xds-api/grpc/telemetry"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/health"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/protobuf/types/known/emptypb"
)

func main() {
	if len(os.Args) < 2 {
		log.Fatal("usage: activation-e2e server|sleep")
	}
	switch os.Args[1] {
	case "server":
		runServer()
	case "telemetry":
		runTelemetryApplication()
	case "sleep":
		time.Sleep(5 * time.Second)
	default:
		log.Fatalf("unknown mode %q", os.Args[1])
	}
}

func runServer() {
	grpcServer := grpc.NewServer()
	checker := health.NewServer()
	checker.SetServingStatus("", healthpb.HealthCheckResponse_SERVING)
	healthpb.RegisterHealthServer(grpcServer, checker)
	handler := h2c.NewHandler(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		if request.ProtoMajor == 2 && request.Header.Get("Content-Type") == "application/grpc" {
			grpcServer.ServeHTTP(response, request)
			return
		}
		response.WriteHeader(http.StatusOK)
		_, _ = response.Write([]byte("payment-ok\n"))
	}), &http2.Server{})
	log.Printf("SERVING address=:8080")
	log.Fatal(http.ListenAndServe(":8080", handler))
}

func runTelemetryApplication() {
	runtime := runtimetelemetry.Default()
	listener, err := net.Listen("tcp", ":8080")
	if err != nil {
		log.Fatal(err)
	}
	grpcServer := grpc.NewServer(runtime.ServerOption())
	checker := health.NewServer()
	checker.SetServingStatus("", healthpb.HealthCheckResponse_SERVING)
	checker.SetServingStatus("telemetry-load", healthpb.HealthCheckResponse_SERVING)
	healthpb.RegisterHealthServer(grpcServer, checker)
	go func() {
		log.Printf("SERVING grpc=:8080")
		log.Fatal(grpcServer.Serve(listener))
	}()

	mux := http.NewServeMux()
	mux.Handle("/metrics", runtime.Handler())
	mux.HandleFunc("/healthz", func(response http.ResponseWriter, _ *http.Request) {
		response.WriteHeader(http.StatusOK)
	})
	go func() {
		log.Printf("SERVING metrics=:9090")
		log.Fatal(http.ListenAndServe(":9090", mux))
	}()

	connection, err := grpc.NewClient(
		"passthrough:///127.0.0.1:8080",
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		runtime.ClientDialOption(),
	)
	if err != nil {
		log.Fatal(err)
	}
	defer connection.Close()

	client := healthpb.NewHealthClient(connection)
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	for range ticker.C {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		_, successErr := client.Check(ctx, &healthpb.HealthCheckRequest{Service: "telemetry-load"})
		// The unknown method supplies a stable non-OK status for label and
		// aggregation verification without opening another connection.
		failureErr := connection.Invoke(ctx, "/telemetry.v1.Probe/Missing", &emptypb.Empty{}, &emptypb.Empty{})
		cancel()
		if successErr != nil {
			log.Printf("health request failed: %v", successErr)
		}
		if failureErr == nil {
			log.Printf("unknown method unexpectedly succeeded")
		}
	}
}
