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
	"google.golang.org/grpc/status"
	_ "google.golang.org/grpc/xds"

	pb "github.com/apache/dubbo-kubernetes/tests/grpc-app/proto"
)

var (
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

	log.Printf("Consumer running. Test server listening on port %d for ForwardEcho", *port)

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan
	log.Println("Shutting down...")

	if testServer != nil {
		log.Println("Stopping test server...")
		testServer.GracefulStop()
	}
}

func startTestServer(port int) {
	lis, err := net.Listen("tcp", fmt.Sprintf("0.0.0.0:%d", port))
	if err != nil {
		log.Printf("Failed to listen on port %d: %v", port, err)
		return
	}

	testServer = grpc.NewServer()
	pb.RegisterEchoTestServiceServer(testServer, &testServerImpl{
		connCache: make(map[string]*cachedConnection),
	})
	reflection.Register(testServer)

	log.Printf("Test server listening on port %d for ForwardEcho (reflection enabled)", port)
	if err := testServer.Serve(lis); err != nil {
		log.Printf("Test server error: %v", err)
	}
}

type cachedConnection struct {
	conn      *grpc.ClientConn
	createdAt time.Time
}

type testServerImpl struct {
	pb.UnimplementedEchoTestServiceServer
	// Connection cache: map from URL to cached connection
	connCache map[string]*cachedConnection
	connMutex sync.RWMutex
}

// formatGRPCError formats gRPC errors similar to grpcurl output
func formatGRPCError(err error, index int32, total int32) string {
	if err == nil {
		return ""
	}

	// Extract gRPC status code and message using status package
	code := "Unknown"
	message := err.Error()

	// Try to extract gRPC status
	if st, ok := status.FromError(err); ok {
		code = st.Code().String()
		message = st.Message()
	} else {
		// Fallback: try to extract from error string
		if strings.Contains(message, "code = ") {
			// Extract code like "code = Unavailable"
			codeMatch := regexp.MustCompile(`code = (\w+)`).FindStringSubmatch(message)
			if len(codeMatch) > 1 {
				code = codeMatch[1]
			}
			// Extract message after "desc = "
			descMatch := regexp.MustCompile(`desc = "?([^"]+)"?`).FindStringSubmatch(message)
			if len(descMatch) > 1 {
				message = descMatch[1]
			} else {
				// If no desc, try to extract message after code
				parts := strings.SplitN(message, "desc = ", 2)
				if len(parts) > 1 {
					message = strings.Trim(parts[1], `"`)
				}
			}
		}
	}

	// Format similar to grpcurl (single line format)
	if total == 1 {
		return fmt.Sprintf("ERROR:\nCode: %s\nMessage: %s", code, message)
	}
	return fmt.Sprintf("[%d] Error: rpc error: code = %s desc = %s", index, code, message)
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

	// Reuse connections to avoid creating new xDS connections for each RPC call
	// This prevents the RDS request loop issue and ensures stable connection state
	s.connMutex.RLock()
	cached, exists := s.connCache[req.Url]
	var conn *grpc.ClientConn
	if exists && cached != nil {
		conn = cached.conn
	}
	s.connMutex.RUnlock()

	// Check if cached connection is still valid and not too old.
	// When xDS config changes (e.g., TLS is added/removed), gRPC xDS client should update connections,
	// but if the connection was established before xDS config was received, it may use old configuration.
	// To ensure we use the latest xDS config, we clear connections older than 10 seconds.
	// rebuilt quickly to use the new configuration.
	const maxConnectionAge = 10 * time.Second
	if exists && conn != nil {
		state := conn.GetState()
		if state == connectivity.Shutdown {
			// Connection is closed, remove from cache
			log.Printf("ForwardEcho: cached connection for %s is SHUTDOWN, removing from cache", req.Url)
			s.connMutex.Lock()
			delete(s.connCache, req.Url)
			conn = nil
			exists = false
			s.connMutex.Unlock()
		} else if time.Since(cached.createdAt) > maxConnectionAge {
			// Connection is too old, may be using stale xDS config (e.g., plaintext when TLS is required)
			// Clear cache to force reconnection with latest xDS config
			log.Printf("ForwardEcho: cached connection for %s is too old (%v), clearing cache to use latest xDS config", req.Url, time.Since(cached.createdAt))
			s.connMutex.Lock()
			if cachedConn, stillExists := s.connCache[req.Url]; stillExists && cachedConn != nil && cachedConn.conn != nil {
				cachedConn.conn.Close()
			}
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
		if cached, exists = s.connCache[req.Url]; !exists || cached == nil || cached.conn == nil {
			conn = nil
			// When TLS is configured (DestinationRule ISTIO_MUTUAL), gRPC xDS client needs
			// to fetch certificates from CertificateProvider. The CertificateProvider uses file_watcher
			// to read certificate files. If the files are not ready or CertificateProvider is not
			// initialized, certificate fetching will timeout.
			// We wait a short time to ensure CertificateProvider is ready and certificate files are accessible.
			// This is especially important when DestinationRule is just created and TLS is enabled.
			// The CertificateProvider may need time to initialize, especially on first connection.
			// We wait 3 seconds to give CertificateProvider enough time to initialize (reduced from 5s for faster startup).
			log.Printf("ForwardEcho: waiting 3 seconds to ensure CertificateProvider is ready...")
			time.Sleep(3 * time.Second)

			// Create xDS client credentials
			// NOTE: FallbackCreds is REQUIRED by gRPC xDS library for initial connection
			// before xDS configuration is available. However, once xDS configures TLS,
			// the client will use TLS and will NOT fallback to plaintext if TLS fails.
			// FallbackCreds is only used when xDS has not yet provided TLS configuration.
			creds, err := xdscreds.NewClientCredentials(xdscreds.ClientOptions{
				FallbackCreds: insecure.NewCredentials(),
			})
			if err != nil {
				s.connMutex.Unlock()
				return nil, fmt.Errorf("failed to create xDS client credentials: %v", err)
			}

			// Dial with xDS URL - use background context, not the request context
			// The request context might timeout before xDS configuration is received
			// When TLS is configured (DestinationRule ISTIO_MUTUAL), gRPC xDS client needs
			// to fetch certificates from CertificateProvider. This may take time, especially on
			// first connection. We use a longer timeout context to allow certificate fetching.
			log.Printf("ForwardEcho: creating new connection for %s...", req.Url)
			dialCtx, dialCancel := context.WithTimeout(context.Background(), 60*time.Second)
			conn, err = grpc.DialContext(dialCtx, req.Url, grpc.WithTransportCredentials(creds))
			dialCancel()
			if err != nil {
				s.connMutex.Unlock()
				return nil, fmt.Errorf("failed to dial %s: %v", req.Url, err)
			}
			s.connCache[req.Url] = &cachedConnection{
				conn:      conn,
				createdAt: time.Now(),
			}
			log.Printf("ForwardEcho: cached connection for %s", req.Url)
		}
		s.connMutex.Unlock()
	} else {
		log.Printf("ForwardEcho: reusing cached connection for %s (state: %v)", req.Url, conn.GetState())
		// NOTE: We reuse the cached connection. If xDS config changes (e.g., TLS is added/removed),
		// gRPC xDS client should automatically update the connection. However, if the connection
		// was established before xDS config was received, it may still be using old configuration.
		// We rely on the TLS mismatch detection logic in the RPC error handling to handle this case.
	}

	initialState := conn.GetState()
	log.Printf("ForwardEcho: initial connection state: %v", initialState)

	// Even if connection is READY, we need to verify it's still valid
	// because xDS configuration may have changed (e.g., from plaintext to TLS)
	// and the cached connection might be using old configuration.
	// gRPC xDS client should automatically update connections, but if the connection
	// was established before xDS config was received, it might be using FallbackCreds (plaintext).
	// We'll proceed with RPC calls, but if they fail with TLS/plaintext mismatch errors,
	// we'll clear the cache and retry.
	// When TLS is configured (DestinationRule ISTIO_MUTUAL), gRPC xDS client needs
	// to fetch certificates from CertificateProvider during TLS handshake. The TLS handshake
	// happens when the connection state transitions to READY. If CertificateProvider is not ready,
	// the TLS handshake will timeout. We need to wait for the connection to be READY, which
	// indicates that TLS handshake has completed successfully.
	if initialState == connectivity.Ready {
		log.Printf("ForwardEcho: connection is already READY, proceeding with RPC calls (will retry with new connection if TLS/plaintext mismatch detected)")
	} else {
		// Only wait for new connections or connections that are not READY
		// For gRPC xDS proxyless, we need to wait for the client to receive and process LDS/CDS/EDS
		// The connection state may transition: IDLE -> CONNECTING -> READY (or TRANSIENT_FAILURE -> CONNECTING -> READY)
		// When TLS is configured, the TLS handshake happens during this state transition.
		// If CertificateProvider is not ready, the TLS handshake will timeout and connection will fail.
		// We use a longer timeout (60 seconds) to allow CertificateProvider to fetch certificates.
		log.Printf("ForwardEcho: waiting for xDS configuration to be processed and connection to be ready (60 seconds)...")

		// Wait for state changes with multiple attempts
		maxWait := 60 * time.Second

		// Wait for state changes, allowing multiple state transitions
		// Don't exit on TRANSIENT_FAILURE - it may recover to READY
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
	errorCount := 0
	firstError := ""
	for i := int32(0); i < count; i++ {
		echoReq := &pb.EchoRequest{
			Message: fmt.Sprintf("Request %d", i+1),
		}

		currentState := conn.GetState()
		log.Printf("ForwardEcho: sending request %d (connection state: %v)...", i+1, currentState)

		// Use longer timeout for requests to allow TLS handshake completion
		// When mTLS is configured, certificate fetching and TLS handshake may take time
		// Use 30 seconds for all requests to ensure TLS handshake has enough time
		timeout := 30 * time.Second

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
			errorCount++
			if firstError == "" {
				firstError = err.Error()
			}

			// Format error similar to grpcurl output
			errMsg := formatGRPCError(err, i, count)
			output = append(output, errMsg)

			// Only clear cache if we detect specific TLS/plaintext mismatch errors.
			// TRANSIENT_FAILURE can occur for many reasons (e.g., xDS config updates, endpoint changes),
			// so we should NOT clear cache on every TRANSIENT_FAILURE.
			// Only clear cache when we detect explicit TLS-related errors that indicate a mismatch.
			errStr := err.Error()
			isTLSMismatch := false

			// Check for specific TLS/plaintext mismatch indicators:
			// - "tls: first record does not look like a TLS handshake" - client uses TLS but server is plaintext
			// - "authentication handshake failed" with TLS context - TLS handshake failed
			// - "context deadline exceeded" during authentication handshake - may indicate TLS handshake timeout
			//   (e.g., client using plaintext but server requiring TLS, or vice versa)
			// - "fetching trusted roots from CertificateProvider failed" - CertificateProvider not ready yet
			//   This is a temporary error that should be retried after waiting for CertificateProvider to be ready
			// These errors indicate that client and server TLS configuration are mismatched, or CertificateProvider is not ready
			if strings.Contains(errStr, "tls: first record does not look like a TLS handshake") ||
				(strings.Contains(errStr, "authentication handshake failed") && strings.Contains(errStr, "tls:")) ||
				(strings.Contains(errStr, "authentication handshake failed") && strings.Contains(errStr, "context deadline exceeded")) ||
				strings.Contains(errStr, "fetching trusted roots from CertificateProvider failed") {
				isTLSMismatch = true
				log.Printf("ForwardEcho: detected TLS/plaintext mismatch or CertificateProvider not ready error: %v", err)
			}

			// When TLS mismatch is detected, immediately clear cache and force reconnection.
			// This ensures that:
			// 1. If client config changed (plaintext -> TLS), new connection uses TLS
			// 2. If server config changed (TLS -> plaintext), new connection uses plaintext
			// 3. Connection behavior is consistent with current xDS configuration
			// - When only client TLS (DestinationRule ISTIO_MUTUAL) but server plaintext: connection SHOULD FAIL
			// - When client TLS + server mTLS (PeerAuthentication STRICT): connection SHOULD SUCCEED
			// - When both plaintext: connection SHOULD SUCCEED
			// By clearing cache and reconnecting, we ensure connection uses current xDS config.
			if isTLSMismatch {
				// Check if this is a CertificateProvider not ready error (temporary) vs configuration mismatch (persistent)
				isCertProviderNotReady := strings.Contains(errStr, "fetching trusted roots from CertificateProvider failed")

				// Clear cache and force reconnection on every TLS mismatch detection
				// This ensures we always try to use the latest xDS configuration
				if isCertProviderNotReady {
					log.Printf("ForwardEcho: WARNING - CertificateProvider not ready yet: %v", err)
					log.Printf("ForwardEcho: NOTE - This is a temporary error. CertificateProvider needs time to initialize.")
					log.Printf("ForwardEcho: Clearing connection cache and waiting for CertificateProvider to be ready...")
				} else {
					log.Printf("ForwardEcho: WARNING - detected TLS/plaintext mismatch error: %v", err)
					log.Printf("ForwardEcho: NOTE - This error indicates that client and server TLS configuration are mismatched")
					log.Printf("ForwardEcho: This usually happens when:")
					log.Printf("ForwardEcho:   1. DestinationRule with ISTIO_MUTUAL exists but PeerAuthentication with STRICT does not (client TLS, server plaintext)")
					log.Printf("ForwardEcho:   2. DestinationRule was deleted but cached connection still uses TLS")
					log.Printf("ForwardEcho: Clearing connection cache to force reconnection with updated xDS config...")
				}

				s.connMutex.Lock()
				if cachedConn, stillExists := s.connCache[req.Url]; stillExists && cachedConn != nil && cachedConn.conn != nil {
					cachedConn.conn.Close()
				}
				delete(s.connCache, req.Url)
				conn = nil
				s.connMutex.Unlock()

				// Wait for xDS config to propagate and be processed by gRPC xDS client.
				// When CDS/LDS config changes, it takes time for:
				// 1. Control plane to push new config to gRPC xDS client
				// 2. gRPC xDS client to process and apply new config
				// 3. CertificateProvider to be ready and certificate files to be accessible
				// 4. New connections to use updated config
				// For CertificateProvider not ready errors, we wait shorter time (3 seconds) as it's usually faster.
				// For configuration mismatch, we wait longer (10 seconds) to ensure config has propagated.
				// Reduced wait times for faster recovery while still ensuring reliability.
				if isCertProviderNotReady {
					log.Printf("ForwardEcho: waiting 3 seconds for CertificateProvider to be ready...")
					time.Sleep(3 * time.Second)
				} else {
					log.Printf("ForwardEcho: waiting 10 seconds for xDS config to propagate and CertificateProvider to be ready...")
					time.Sleep(10 * time.Second)
				}

				// Recreate connection - this will use current xDS config
				// When TLS is configured, gRPC xDS client needs to fetch certificates
				// from CertificateProvider. Use a longer timeout to allow certificate fetching.
				log.Printf("ForwardEcho: recreating connection with current xDS config...")
				creds, credErr := xdscreds.NewClientCredentials(xdscreds.ClientOptions{
					FallbackCreds: insecure.NewCredentials(),
				})
				if credErr != nil {
					log.Printf("ForwardEcho: failed to create xDS client credentials: %v", credErr)
					// Continue to record error
				} else {
					// Wait additional time before dialing to ensure CertificateProvider is ready
					// Reduced from 3s to 2s for faster recovery
					log.Printf("ForwardEcho: waiting 2 seconds before dialing to ensure CertificateProvider is ready...")
					time.Sleep(2 * time.Second)

					dialCtx, dialCancel := context.WithTimeout(context.Background(), 60*time.Second)
					newConn, dialErr := grpc.DialContext(dialCtx, req.Url, grpc.WithTransportCredentials(creds))
					dialCancel()
					if dialErr != nil {
						log.Printf("ForwardEcho: failed to dial %s: %v", req.Url, dialErr)
						// Continue to record error
					} else {
						s.connMutex.Lock()
						s.connCache[req.Url] = &cachedConnection{
							conn:      newConn,
							createdAt: time.Now(),
						}
						conn = newConn
						client = pb.NewEchoServiceClient(conn)
						s.connMutex.Unlock()
						log.Printf("ForwardEcho: connection recreated, retrying request %d", i+1)
						// Retry this request with new connection
						continue
					}
				}
			}
			if i < count-1 {
				waitTime := 2 * time.Second
				log.Printf("ForwardEcho: waiting %v before next request...", waitTime)
				time.Sleep(waitTime)
			}
			continue
		}

		if resp == nil {
			log.Printf("ForwardEcho: request %d failed: response is nil", i+1)
			output = append(output, fmt.Sprintf("[%d] Error: response is nil", i))
			continue
		}

		log.Printf("ForwardEcho: request %d succeeded: Hostname=%s ServiceVersion=%s Namespace=%s IP=%s",
			i+1, resp.Hostname, resp.ServiceVersion, resp.Namespace, resp.Ip)

		lineParts := []string{
			fmt.Sprintf("[%d body] Hostname=%s", i, resp.Hostname),
		}
		if resp.ServiceVersion != "" {
			lineParts = append(lineParts, fmt.Sprintf("ServiceVersion=%s", resp.ServiceVersion))
		}
		if resp.Namespace != "" {
			lineParts = append(lineParts, fmt.Sprintf("Namespace=%s", resp.Namespace))
		}
		if resp.Ip != "" {
			lineParts = append(lineParts, fmt.Sprintf("IP=%s", resp.Ip))
		}
		if resp.Cluster != "" {
			lineParts = append(lineParts, fmt.Sprintf("Cluster=%s", resp.Cluster))
		}
		if resp.ServicePort > 0 {
			lineParts = append(lineParts, fmt.Sprintf("ServicePort=%d", resp.ServicePort))
		}

		output = append(output, strings.Join(lineParts, " "))

		// Small delay between successful requests to avoid overwhelming the server
		if i < count-1 {
			time.Sleep(100 * time.Millisecond)
		}
	}

	log.Printf("ForwardEcho: completed %d requests", count)

	// If all requests failed, add summary similar to grpcurl
	if errorCount > 0 && errorCount == int(count) && firstError != "" {
		summary := fmt.Sprintf("ERROR:\nCode: Unknown\nMessage: %d/%d requests had errors; first error: %s", errorCount, count, firstError)
		// Prepend summary to output
		output = append([]string{summary}, output...)
	}

	return &pb.ForwardEchoResponse{
		Output: output,
	}, nil
}
