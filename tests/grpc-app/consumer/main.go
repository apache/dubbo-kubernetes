/*
 * Licensed to the Apache Software Foundation (ASF) under one or more
 * contributor license agreements.  See the NOTICE file distributed with
 * this work for additional information regarding copyright ownership.
 * The ASF licenses this file to You under the Apache License, Version 2.0
 * (the "License"); you may not use this file except in compliance with
 * the License.  You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"regexp"
	"strings"
	"syscall"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	xdscreds "google.golang.org/grpc/credentials/xds"
	"google.golang.org/grpc/grpclog"
	"google.golang.org/grpc/reflection"
	"google.golang.org/grpc/xds"

	pb "github.com/apache/dubbo-kubernetes/test/grpc-app/proto"
)

var (
	port = flag.Int("port", 17070, "gRPC server port")
)

type echoServer struct {
	pb.UnimplementedEchoServiceServer
	pb.UnimplementedEchoTestServiceServer
	hostname string
}

func (s *echoServer) Echo(ctx context.Context, req *pb.EchoRequest) (*pb.EchoResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("request is nil")
	}
	log.Printf("Received: %v", req.Message)
	return &pb.EchoResponse{
		Message:  req.Message,
		Hostname: s.hostname,
	}, nil
}

func (s *echoServer) StreamEcho(req *pb.EchoRequest, stream pb.EchoService_StreamEchoServer) error {
	if req == nil {
		return fmt.Errorf("request is nil")
	}
	if stream == nil {
		return fmt.Errorf("stream is nil")
	}
	log.Printf("StreamEcho received: %v", req.Message)
	for i := 0; i < 3; i++ {
		if err := stream.Send(&pb.EchoResponse{
			Message:  fmt.Sprintf("%s [%d]", req.Message, i),
			Hostname: s.hostname,
		}); err != nil {
			log.Printf("StreamEcho send error: %v", err)
			return err
		}
	}
	return nil
}

func (s *echoServer) ForwardEcho(ctx context.Context, req *pb.ForwardEchoRequest) (*pb.ForwardEchoResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("request is nil")
	}

	count := req.Count
	if count < 0 {
		count = 0
	}
	if count > 100 {
		count = 100
	}

	log.Printf("ForwardEcho called: url=%s, count=%d", req.Url, count)

	output := make([]string, 0, count)
	for i := int32(0); i < count; i++ {
		line := fmt.Sprintf("[%d body] Hostname=%s", i, s.hostname)
		output = append(output, line)
	}

	return &pb.ForwardEchoResponse{
		Output: output,
	}, nil
}

// grpcLogger filters out xDS informational logs that are incorrectly marked as ERROR
type grpcLogger struct {
	logger *log.Logger
}

var (
	// Regex to match gRPC formatting errors like %!p(...)
	formatErrorRegex = regexp.MustCompile(`%!p\([^)]+\)`)
)

// cleanMessage removes formatting errors from gRPC logs
// Fixes issues like: "\u003c%!p(networktype.keyType=grpc.internal.transport.networktype)\u003e": "unix"
func cleanMessage(msg string) string {
	// Replace %!p(...) patterns with a cleaner representation
	msg = formatErrorRegex.ReplaceAllStringFunc(msg, func(match string) string {
		// Extract the key from %!p(networktype.keyType=...)
		if strings.Contains(match, "networktype.keyType") {
			return `"networktype"`
		}
		// For other cases, just remove the error pattern
		return ""
	})
	// Also clean up Unicode escape sequences that appear with formatting errors
	// Replace \u003c (which is <) and \u003e (which is >) when they appear with formatting errors
	msg = strings.ReplaceAll(msg, `\u003c`, "<")
	msg = strings.ReplaceAll(msg, `\u003e`, ">")
	// Clean up patterns like <...>: "unix" to just show the value
	msg = regexp.MustCompile(`<[^>]*>:\s*"unix"`).ReplaceAllString(msg, `"networktype": "unix"`)
	return msg
}

func (l *grpcLogger) Info(args ...interface{}) {
	msg := fmt.Sprint(args...)
	// Filter out xDS "entering mode: SERVING" logs
	if strings.Contains(msg, "entering mode") && strings.Contains(msg, "SERVING") {
		return
	}
	msg = cleanMessage(msg)
	l.logger.Print("INFO: ", msg)
}

func (l *grpcLogger) Infoln(args ...interface{}) {
	msg := fmt.Sprintln(args...)
	if strings.Contains(msg, "entering mode") && strings.Contains(msg, "SERVING") {
		return
	}
	msg = cleanMessage(msg)
	l.logger.Print("INFO: ", msg)
}

func (l *grpcLogger) Infof(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	if strings.Contains(msg, "entering mode") && strings.Contains(msg, "SERVING") {
		return
	}
	msg = cleanMessage(msg)
	l.logger.Printf("INFO: %s", msg)
}

func (l *grpcLogger) Warning(args ...interface{}) {
	msg := cleanMessage(fmt.Sprint(args...))
	l.logger.Print("WARNING: ", msg)
}

func (l *grpcLogger) Warningln(args ...interface{}) {
	msg := cleanMessage(fmt.Sprintln(args...))
	l.logger.Print("WARNING: ", msg)
}

func (l *grpcLogger) Warningf(format string, args ...interface{}) {
	msg := cleanMessage(fmt.Sprintf(format, args...))
	l.logger.Printf("WARNING: %s", msg)
}

func (l *grpcLogger) Error(args ...interface{}) {
	msg := fmt.Sprint(args...)
	// Filter out xDS "entering mode: SERVING" logs that are incorrectly marked as ERROR
	if strings.Contains(msg, "entering mode") && strings.Contains(msg, "SERVING") {
		return
	}
	// Filter out common connection reset errors - these are normal network behavior
	// when clients disconnect before completing the HTTP/2 handshake
	if strings.Contains(msg, "connection reset by peer") ||
		strings.Contains(msg, "failed to receive the preface from client") ||
		strings.Contains(msg, "connection error") {
		// These are normal network events, log at DEBUG level instead of ERROR
		return
	}
	msg = cleanMessage(msg)
	l.logger.Print("ERROR: ", msg)
}

func (l *grpcLogger) Errorln(args ...interface{}) {
	msg := fmt.Sprintln(args...)
	if strings.Contains(msg, "entering mode") && strings.Contains(msg, "SERVING") {
		return
	}
	// Filter out common connection reset errors - these are normal network behavior
	if strings.Contains(msg, "connection reset by peer") ||
		strings.Contains(msg, "failed to receive the preface from client") ||
		strings.Contains(msg, "connection error") {
		return
	}
	msg = cleanMessage(msg)
	l.logger.Print("ERROR: ", msg)
}

func (l *grpcLogger) Errorf(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	if strings.Contains(msg, "entering mode") && strings.Contains(msg, "SERVING") {
		return
	}
	// Filter out common connection reset errors - these are normal network behavior
	if strings.Contains(msg, "connection reset by peer") ||
		strings.Contains(msg, "failed to receive the preface from client") ||
		strings.Contains(msg, "connection error") {
		return
	}
	msg = cleanMessage(msg)
	l.logger.Printf("ERROR: %s", msg)
}

func (l *grpcLogger) Fatal(args ...interface{}) {
	l.logger.Fatal(args...)
}

func (l *grpcLogger) Fatalln(args ...interface{}) {
	l.logger.Fatal(args...)
}

func (l *grpcLogger) Fatalf(format string, args ...interface{}) {
	l.logger.Fatalf(format, args...)
}

func (l *grpcLogger) V(level int) bool {
	return level <= 0
}

// waitForBootstrapFile waits for the grpc-bootstrap.json file to exist
// This is necessary because the dubbo-proxy sidecar needs time to generate the file
func waitForBootstrapFile(bootstrapPath string, maxWait time.Duration) error {
	log.Printf("Waiting for bootstrap file to exist: %s (max wait: %v)", bootstrapPath, maxWait)

	ctx, cancel := context.WithTimeout(context.Background(), maxWait)
	defer cancel()

	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	startTime := time.Now()
	for {
		// Check if file exists and is not empty
		if info, err := os.Stat(bootstrapPath); err == nil && info.Size() > 0 {
			log.Printf("Bootstrap file found after %v: %s", time.Since(startTime), bootstrapPath)
			return nil
		}

		// Check for timeout
		select {
		case <-ctx.Done():
			return fmt.Errorf("timeout waiting for bootstrap file: %s (waited %v)", bootstrapPath, time.Since(startTime))
		case <-ticker.C:
			// Continue waiting
		}
	}
}

func main() {
	flag.Parse()

	// Set custom gRPC logger to filter out xDS informational logs
	// The "ERROR: [xds] Listener entering mode: SERVING" is actually an informational log
	grpclog.SetLoggerV2(&grpcLogger{
		logger: log.New(os.Stderr, "", log.LstdFlags),
	})

	hostname, _ := os.Hostname()
	if hostname == "" {
		hostname = "unknown"
	}

	// Get bootstrap file path from environment variable or use default
	bootstrapPath := os.Getenv("GRPC_XDS_BOOTSTRAP")
	if bootstrapPath == "" {
		bootstrapPath = "/etc/dubbo/proxy/grpc-bootstrap.json"
		log.Printf("GRPC_XDS_BOOTSTRAP not set, using default: %s", bootstrapPath)
	}

	// Wait for bootstrap file to exist before creating xDS server
	// The dubbo-proxy sidecar needs time to generate this file
	if err := waitForBootstrapFile(bootstrapPath, 60*time.Second); err != nil {
		log.Fatalf("Failed to wait for bootstrap file: %v", err)
	}

	// Create xDS-enabled gRPC server
	// For proxyless gRPC, we use xds.NewGRPCServer() instead of grpc.NewServer()
	creds, err := xdscreds.NewServerCredentials(xdscreds.ServerOptions{
		FallbackCreds: insecure.NewCredentials(),
	})
	if err != nil {
		log.Fatalf("Failed to create xDS server credentials: %v", err)
	}

	server, err := xds.NewGRPCServer(grpc.Creds(creds))
	if err != nil {
		log.Fatalf("Failed to create xDS gRPC server: %v", err)
	}

	es := &echoServer{hostname: hostname}
	pb.RegisterEchoServiceServer(server, es)
	pb.RegisterEchoTestServiceServer(server, es)
	// Enable reflection API for grpcurl to discover services
	reflection.Register(server)

	lis, err := net.Listen("tcp", fmt.Sprintf("0.0.0.0:%d", *port))
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}

	log.Printf("Starting gRPC proxyless server on port %d (hostname: %s)", *port, hostname)

	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
		<-sigChan
		log.Println("Shutting down server...")
		server.GracefulStop()
	}()

	// Serve the gRPC server
	// Note: server.Serve returns when the listener is closed, which is normal during shutdown
	// Connection reset errors are handled by the gRPC library and logged separately
	if err := server.Serve(lis); err != nil {
		// Only log as fatal if it's not a normal shutdown (listener closed)
		if !strings.Contains(err.Error(), "use of closed network connection") {
			log.Fatalf("Failed to serve: %v", err)
		}
		log.Printf("Server stopped: %v", err)
	}
}
