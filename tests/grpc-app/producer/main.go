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
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"regexp"
	"strings"
	"sync"
	"syscall"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/connectivity"
	"google.golang.org/grpc/credentials/insecure"
	xdscreds "google.golang.org/grpc/credentials/xds"
	"google.golang.org/grpc/grpclog"
	"google.golang.org/grpc/reflection"
	_ "google.golang.org/grpc/xds"

	pb "github.com/apache/dubbo-kubernetes/test/grpc-app/proto"
)

var (
	target     = flag.String("target", "", "Target service address with xds:/// scheme (optional)")
	count      = flag.Int("count", 5, "Number of requests to send")
	port       = flag.Int("port", 17171, "gRPC server port for ForwardEcho testing")
	testServer *grpc.Server
)

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
	if strings.Contains(msg, "entering mode") && strings.Contains(msg, "SERVING") {
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
	msg = cleanMessage(msg)
	l.logger.Print("ERROR: ", msg)
}

func (l *grpcLogger) Errorf(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	if strings.Contains(msg, "entering mode") && strings.Contains(msg, "SERVING") {
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

func main() {
	flag.Parse()

	// Set custom gRPC logger to filter out xDS informational logs
	// The "ERROR: [xds] Listener entering mode: SERVING" is actually an informational log
	grpclog.SetLoggerV2(&grpcLogger{
		logger: log.New(os.Stderr, "", log.LstdFlags),
	})

	go startTestServer(*port)

	if *target != "" {
		testDirectConnection(*target, *count)
	}

	log.Printf("Producer running. Test server listening on port %d for ForwardEcho", *port)

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan
	log.Println("Shutting down...")

	if testServer != nil {
		log.Println("Stopping test server...")
		testServer.GracefulStop()
	}
}

func testDirectConnection(target string, count int) {
	creds, err := xdscreds.NewClientCredentials(xdscreds.ClientOptions{
		FallbackCreds: insecure.NewCredentials(),
	})
	if err != nil {
		log.Fatalf("Failed to create xDS client credentials: %v", err)
	}

	ctx := context.Background()
	conn, err := grpc.DialContext(ctx, target, grpc.WithTransportCredentials(creds))
	if err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}
	defer conn.Close()

	client := pb.NewEchoServiceClient(conn)

	log.Printf("Connected to %s, sending %d requests...", target, count)

	for i := 0; i < count; i++ {
		req := &pb.EchoRequest{
			Message: fmt.Sprintf("Hello from producer [%d]", i+1),
		}

		reqCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		resp, err := client.Echo(reqCtx, req)
		cancel()

		if err != nil {
			log.Printf("Request %d failed: %v", i+1, err)
			continue
		}

		if resp == nil {
			log.Printf("Request %d failed: response is nil", i+1)
			continue
		}

		log.Printf("Request %d: Response=%s, Hostname=%s", i+1, resp.Message, resp.Hostname)
		time.Sleep(500 * time.Millisecond)
	}

	log.Println("All requests completed")
}

func startTestServer(port int) {
	lis, err := net.Listen("tcp", fmt.Sprintf("0.0.0.0:%d", port))
	if err != nil {
		log.Printf("Failed to listen on port %d: %v", port, err)
		return
	}

	testServer = grpc.NewServer()
	pb.RegisterEchoTestServiceServer(testServer, &testServerImpl{
		connCache: make(map[string]*grpc.ClientConn),
	})
	reflection.Register(testServer)

	log.Printf("Test server listening on port %d for ForwardEcho (reflection enabled)", port)
	if err := testServer.Serve(lis); err != nil {
		log.Printf("Test server error: %v", err)
	}
}

type testServerImpl struct {
	pb.UnimplementedEchoTestServiceServer
	// Connection cache: map from URL to gRPC connection
	connCache map[string]*grpc.ClientConn
	connMutex sync.RWMutex
}

func (s *testServerImpl) ForwardEcho(ctx context.Context, req *pb.ForwardEchoRequest) (*pb.ForwardEchoResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("request is nil")
	}

	if req.Url == "" {
		return nil, fmt.Errorf("url is required")
	}

	count := req.Count
	if count < 0 {
		count = 0
	}
	if count > 100 {
		count = 100
	}

	log.Printf("ForwardEcho: url=%s, count=%d", req.Url, count)

	// Check bootstrap configuration
	bootstrapPath := os.Getenv("GRPC_XDS_BOOTSTRAP")
	if bootstrapPath == "" {
		return nil, fmt.Errorf("GRPC_XDS_BOOTSTRAP environment variable is not set")
	}

	// Verify bootstrap file exists
	if _, err := os.Stat(bootstrapPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("bootstrap file does not exist: %s", bootstrapPath)
	}

	// Read bootstrap file to verify UDS socket
	// #nosec G304 -- bootstrapPath is a controlled path from environment variable
	bootstrapData, err := os.ReadFile(bootstrapPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read bootstrap file: %v", err)
	}

	var bootstrapJSON map[string]interface{}
	if err := json.Unmarshal(bootstrapData, &bootstrapJSON); err != nil {
		return nil, fmt.Errorf("failed to parse bootstrap file: %v", err)
	}

	// Extract UDS socket path
	var udsPath string
	if xdsServers, ok := bootstrapJSON["xds_servers"].([]interface{}); ok && len(xdsServers) > 0 {
		if server, ok := xdsServers[0].(map[string]interface{}); ok {
			if serverURI, ok := server["server_uri"].(string); ok {
				if strings.HasPrefix(serverURI, "unix://") {
					udsPath = strings.TrimPrefix(serverURI, "unix://")
					if _, err := os.Stat(udsPath); os.IsNotExist(err) {
						return nil, fmt.Errorf("UDS socket does not exist: %s", udsPath)
					}
				}
			}
		}
	}

	// CRITICAL: Reuse connections to avoid creating new xDS connections for each RPC call
	// This prevents the RDS request loop issue and ensures stable connection state
	s.connMutex.RLock()
	conn, exists := s.connCache[req.Url]
	s.connMutex.RUnlock()

	// Check if cached connection is still valid (not closed/shutdown)
	if exists && conn != nil {
		state := conn.GetState()
		if state == connectivity.Shutdown {
			// Connection is closed, remove from cache and create new one
			log.Printf("ForwardEcho: cached connection for %s is SHUTDOWN, removing from cache", req.Url)
			s.connMutex.Lock()
			delete(s.connCache, req.Url)
			conn = nil
			exists = false
			s.connMutex.Unlock()
		}
	}

	if !exists || conn == nil {
		// Create new connection
		s.connMutex.Lock()
		// Double-check after acquiring write lock
		if conn, exists = s.connCache[req.Url]; !exists || conn == nil {
			// Create xDS client credentials
			creds, err := xdscreds.NewClientCredentials(xdscreds.ClientOptions{
				FallbackCreds: insecure.NewCredentials(),
			})
			if err != nil {
				s.connMutex.Unlock()
				return nil, fmt.Errorf("failed to create xDS client credentials: %v", err)
			}

			// Dial with xDS URL - use background context, not the request context
			// The request context might timeout before xDS configuration is received
			log.Printf("ForwardEcho: creating new connection for %s...", req.Url)
			conn, err = grpc.DialContext(context.Background(), req.Url, grpc.WithTransportCredentials(creds))
			if err != nil {
				s.connMutex.Unlock()
				return nil, fmt.Errorf("failed to dial %s: %v", req.Url, err)
			}
			s.connCache[req.Url] = conn
			log.Printf("ForwardEcho: cached connection for %s", req.Url)
		}
		s.connMutex.Unlock()
	} else {
		log.Printf("ForwardEcho: reusing cached connection for %s (state: %v)", req.Url, conn.GetState())
	}

	initialState := conn.GetState()
	log.Printf("ForwardEcho: initial connection state: %v", initialState)

	// CRITICAL: If connection is already READY, use it directly without waiting
	// For cached connections, they should already be in READY state
	if initialState == connectivity.Ready {
		log.Printf("ForwardEcho: connection is already READY, proceeding with RPC calls")
	} else {
		// Only wait for new connections or connections that are not READY
		// For gRPC xDS proxyless, we need to wait for the client to receive and process LDS/CDS/EDS
		// The connection state may transition: IDLE -> CONNECTING -> READY (or TRANSIENT_FAILURE -> CONNECTING -> READY)
		log.Printf("ForwardEcho: waiting for xDS configuration to be processed and connection to be ready (30 seconds)...")

		// Wait for state changes with multiple attempts
		maxWait := 30 * time.Second

		// Wait for state changes, allowing multiple state transitions
		// CRITICAL: Don't exit on TRANSIENT_FAILURE - it may recover to READY
		stateChanged := false
		currentState := initialState
		startTime := time.Now()
		lastStateChangeTime := startTime

		for time.Since(startTime) < maxWait {
			if currentState == connectivity.Ready {
				log.Printf("ForwardEcho: connection is READY after %v", time.Since(startTime))
				stateChanged = true
				break
			}

			// Only exit on Shutdown, not on TransientFailure (it may recover)
			if currentState == connectivity.Shutdown {
				log.Printf("ForwardEcho: connection in %v state after %v, cannot recover", currentState, time.Since(startTime))
				break
			}

			// Wait for state change with remaining timeout
			remaining := maxWait - time.Since(startTime)
			if remaining <= 0 {
				break
			}

			// Use shorter timeout for each WaitForStateChange call to allow periodic checks
			waitTimeout := remaining
			if waitTimeout > 5*time.Second {
				waitTimeout = 5 * time.Second
			}

			stateCtx, stateCancel := context.WithTimeout(context.Background(), waitTimeout)
			if conn.WaitForStateChange(stateCtx, currentState) {
				newState := conn.GetState()
				elapsed := time.Since(startTime)
				log.Printf("ForwardEcho: connection state changed from %v to %v after %v", currentState, newState, elapsed)
				stateChanged = true
				currentState = newState
				lastStateChangeTime = time.Now()

				// If READY, we're done
				if newState == connectivity.Ready {
					stateCancel()
					break
				}

				// If we're in TRANSIENT_FAILURE, continue waiting - it may recover
				// gRPC xDS client will retry connection when endpoints become available
				if newState == connectivity.TransientFailure {
					log.Printf("ForwardEcho: connection in TRANSIENT_FAILURE, continuing to wait for recovery (remaining: %v)", maxWait-elapsed)
				}
			} else {
				// Timeout waiting for state change - check if we should continue
				elapsed := time.Since(startTime)
				if currentState == connectivity.TransientFailure {
					// If we've been in TRANSIENT_FAILURE for a while, continue waiting
					// The connection may recover when endpoints become available
					if time.Since(lastStateChangeTime) < 10*time.Second {
						log.Printf("ForwardEcho: still in TRANSIENT_FAILURE after %v, continuing to wait (remaining: %v)", elapsed, maxWait-elapsed)
					} else {
						log.Printf("ForwardEcho: no state change after %v, current state: %v (remaining: %v)", elapsed, currentState, maxWait-elapsed)
					}
				} else {
					log.Printf("ForwardEcho: no state change after %v, current state: %v (remaining: %v)", elapsed, currentState, maxWait-elapsed)
				}
			}
			stateCancel()
		}

		finalState := conn.GetState()
		log.Printf("ForwardEcho: final connection state: %v (stateChanged=%v, waited=%v)", finalState, stateChanged, time.Since(startTime))

		// If connection is not READY, log a warning but proceed anyway
		// The first RPC call may trigger connection establishment
		if finalState != connectivity.Ready {
			log.Printf("ForwardEcho: WARNING - connection is not READY (state=%v), but proceeding with RPC calls", finalState)
		}
	}

	// Create client and make RPC calls
	client := pb.NewEchoServiceClient(conn)
	output := make([]string, 0, count)

	log.Printf("ForwardEcho: sending %d requests...", count)
	for i := int32(0); i < count; i++ {
		echoReq := &pb.EchoRequest{
			Message: fmt.Sprintf("Request %d", i+1),
		}

		currentState := conn.GetState()
		log.Printf("ForwardEcho: sending request %d (connection state: %v)...", i+1, currentState)

		// Use longer timeout for first request to allow connection establishment
		// For subsequent requests, use shorter timeout but still allow for retries
		timeout := 30 * time.Second
		if i > 0 {
			timeout = 20 * time.Second
		}

		reqCtx, reqCancel := context.WithTimeout(context.Background(), timeout)
		reqStartTime := time.Now()
		resp, err := client.Echo(reqCtx, echoReq)
		duration := time.Since(reqStartTime)
		reqCancel()

		// Check connection state after RPC call
		stateAfterRPC := conn.GetState()
		log.Printf("ForwardEcho: request %d completed in %v, connection state: %v (was %v)", i+1, duration, stateAfterRPC, currentState)

		if err != nil {
			log.Printf("ForwardEcho: request %d failed: %v", i+1, err)
			output = append(output, fmt.Sprintf("[%d] Error: %v", i, err))

			// If connection is in TRANSIENT_FAILURE, wait a bit before next request
			// to allow gRPC client to retry and recover
			if stateAfterRPC == connectivity.TransientFailure && i < count-1 {
				waitTime := 2 * time.Second
				log.Printf("ForwardEcho: connection in TRANSIENT_FAILURE, waiting %v before next request...", waitTime)
				time.Sleep(waitTime)

				// Check if connection recovered
				newState := conn.GetState()
				if newState == connectivity.Ready {
					log.Printf("ForwardEcho: connection recovered to READY after wait")
				} else {
					log.Printf("ForwardEcho: connection state after wait: %v", newState)
				}
			}
			continue
		}

		if resp == nil {
			log.Printf("ForwardEcho: request %d failed: response is nil", i+1)
			output = append(output, fmt.Sprintf("[%d] Error: response is nil", i))
			continue
		}

		log.Printf("ForwardEcho: request %d succeeded: Hostname=%s", i+1, resp.Hostname)
		output = append(output, fmt.Sprintf("[%d body] Hostname=%s", i, resp.Hostname))

		// Small delay between successful requests to avoid overwhelming the server
		if i < count-1 {
			time.Sleep(100 * time.Millisecond)
		}
	}

	log.Printf("ForwardEcho: completed %d requests", count)

	return &pb.ForwardEchoResponse{
		Output: output,
	}, nil
}
