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

package app

import (
	"context"
	"fmt"
	"io"
	"net"
	neturl "net/url"
	"os"
	"strconv"
	"strings"
	"time"

	testproto "github.com/apache/dubbo-kubernetes/dubbod/discovery/cmd/app/testdata"
	"github.com/apache/dubbo-kubernetes/pkg/config/constants"
	"github.com/apache/dubbo-kubernetes/pkg/log"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/reflection"
	"google.golang.org/grpc/status"
)

const defaultXDSTestGRPCAddr = ":17171"

type xdsTestGRPCServer struct {
	testproto.UnimplementedXDSTestServiceServer
}

func startXDSTestGRPCServer(stop <-chan struct{}) error {
	addr := firstNonEmpty(os.Getenv("XDS_TEST_GRPC_ADDR"), defaultXDSTestGRPCAddr)
	if strings.EqualFold(addr, "disabled") {
		return nil
	}
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}
	server := grpc.NewServer()
	testproto.RegisterXDSTestServiceServer(server, &xdsTestGRPCServer{})
	reflection.Register(server)
	go func() {
		<-stop
		server.GracefulStop()
	}()
	go func() {
		if err := server.Serve(listener); err != nil {
			log.Warnf("xDS test gRPC server stopped: %v", err)
		}
	}()
	log.Infof("xDS test gRPC server listening on %s", addr)
	return nil
}

func (s *xdsTestGRPCServer) ForwardHTTP(ctx context.Context, req *testproto.ForwardHTTPRequest) (*testproto.ForwardHTTPResponse, error) {
	opts, err := grpcOutboundOptionsFromForwardHTTPRequest(req)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	expected, err := parseExpectedWeights(opts.expect)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	client, err := newSampleADSClient(ctx, opts)
	if err != nil {
		return nil, status.Error(codes.Unavailable, err.Error())
	}
	defer client.close()
	if err := client.start(); err != nil {
		return nil, status.Error(codes.Unavailable, err.Error())
	}
	snapshot, err := client.waitForRoute(ctx, expected, opts.waitTimeout)
	if err != nil {
		return nil, status.Error(codes.FailedPrecondition, err.Error())
	}
	output, err := runSampleRequestsWithOutput(ctx, client, snapshot, opts.count, opts.requestInterval, opts.requestTimeout, io.Discard)
	if err != nil {
		return nil, status.Error(codes.Unknown, err.Error())
	}
	return &testproto.ForwardHTTPResponse{Output: output}, nil
}

func grpcOutboundOptionsFromForwardHTTPRequest(req *testproto.ForwardHTTPRequest) (*grpcOutboundOptions, error) {
	if req == nil {
		return nil, fmt.Errorf("request is required")
	}
	host, port, err := parseXDSForwardURL(req.GetUrl())
	if err != nil {
		return nil, err
	}
	namespace := firstNonEmpty(req.GetNamespace(), namespaceFromServiceHost(host), os.Getenv("POD_NAMESPACE"), "default")
	serviceAccount := firstNonEmpty(req.GetServiceAccount(), namespace)
	requestHeaders, err := parseRequestHeaders(req.GetHeaders())
	if err != nil {
		return nil, err
	}
	requestInterval, err := durationField(req.GetRequestInterval(), 0)
	if err != nil {
		return nil, fmt.Errorf("invalid requestInterval: %w", err)
	}
	requestTimeout, err := durationField(req.GetRequestTimeout(), 5*time.Second)
	if err != nil {
		return nil, fmt.Errorf("invalid requestTimeout: %w", err)
	}
	waitTimeout, err := durationField(req.GetWaitTimeout(), 30*time.Second)
	if err != nil {
		return nil, fmt.Errorf("invalid waitTimeout: %w", err)
	}
	count := int(req.GetCount())
	if count <= 0 {
		count = 1
	}
	return &grpcOutboundOptions{
		host:            host,
		port:            port,
		path:            normalizeRequestPath(req.GetPath()),
		requestHeaders:  requestHeaders,
		target:          net.JoinHostPort(host, strconv.Itoa(port)),
		xdsAddress:      firstNonEmpty(os.Getenv("XDS_TEST_XDS_ADDRESS"), "127.0.0.1:26010"),
		namespace:       namespace,
		podName:         firstNonEmpty(os.Getenv("POD_NAME"), os.Getenv("HOSTNAME"), "grpc-outbound"),
		podIP:           firstNonEmpty(os.Getenv("INSTANCE_IP"), os.Getenv("POD_IP"), "127.0.0.1"),
		serviceAccount:  serviceAccount,
		trustDomain:     firstNonEmpty(os.Getenv("TRUST_DOMAIN"), constants.DefaultClusterLocalDomain),
		domainSuffix:    firstNonEmpty(os.Getenv("DOMAIN_SUFFIX"), constants.DefaultClusterLocalDomain),
		clusterID:       firstNonEmpty(os.Getenv("DUBBO_META_CLUSTER_ID"), string(constants.DefaultClusterName)),
		count:           count,
		expect:          req.GetExpect(),
		waitTimeout:     waitTimeout,
		requestInterval: requestInterval,
		requestTimeout:  requestTimeout,
		insecure:        true,
	}, nil
}

func parseXDSForwardURL(raw string) (string, int, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", 0, fmt.Errorf("url is required")
	}
	parsed, err := neturl.Parse(raw)
	if err != nil {
		return "", 0, err
	}
	if parsed.Scheme != "xds" {
		return "", 0, fmt.Errorf("unsupported url scheme %q, want xds", parsed.Scheme)
	}
	target := parsed.Host
	if target == "" {
		target = strings.TrimPrefix(parsed.Path, "/")
	}
	target = strings.TrimSuffix(target, "/")
	host, portValue, err := net.SplitHostPort(target)
	if err != nil {
		return "", 0, fmt.Errorf("invalid xds url target %q: %w", target, err)
	}
	port, err := strconv.Atoi(portValue)
	if err != nil {
		return "", 0, fmt.Errorf("invalid xds url port %q: %w", portValue, err)
	}
	return host, port, nil
}

func namespaceFromServiceHost(host string) string {
	parts := strings.Split(host, ".")
	if len(parts) >= 4 && parts[2] == "svc" && parts[1] != "" {
		return parts[1]
	}
	return ""
}

func durationField(value string, defaultValue time.Duration) (time.Duration, error) {
	if strings.TrimSpace(value) == "" {
		return defaultValue, nil
	}
	return time.ParseDuration(value)
}
