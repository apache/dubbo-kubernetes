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

package grpc

import (
	"io"
	"math"
	"strings"

	dubbokeepalive "github.com/apache/dubbo-kubernetes/pkg/keepalive"
	"github.com/apache/dubbo-kubernetes/pkg/util/sets"
	"github.com/apache/dubbo-kubernetes/sail/pkg/features"
	middleware "github.com/grpc-ecosystem/go-grpc-middleware"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/keepalive"
	"google.golang.org/grpc/status"
)

type ErrorType string

const (
	UnexpectedError     ErrorType = "unexpectedError"
	ExpectedError       ErrorType = "expectedError"
	GracefulTermination ErrorType = "gracefulTermination"
)

const (
	defaultClientMaxReceiveMessageSize = math.MaxInt32
	defaultInitialConnWindowSize       = 1024 * 1024 // default gRPC InitialWindowSize
	defaultInitialWindowSize           = 1024 * 1024 // default gRPC ConnWindowSize
)

var expectedGrpcFailureMessages = sets.New(
	"client disconnected",
	"error reading from server: EOF",
	"transport is closing",
)

func ClientOptions(options *dubbokeepalive.Options, tlsOpts *TLSOptions) ([]grpc.DialOption, error) {
	if options == nil {
		options = dubbokeepalive.DefaultOption()
	}
	keepaliveOption := grpc.WithKeepaliveParams(keepalive.ClientParameters{
		Time:    options.Time,
		Timeout: options.Timeout,
	})

	initialWindowSizeOption := grpc.WithInitialWindowSize(int32(defaultInitialWindowSize))
	initialConnWindowSizeOption := grpc.WithInitialConnWindowSize(int32(defaultInitialConnWindowSize))
	msgSizeOption := grpc.WithDefaultCallOptions(grpc.MaxCallRecvMsgSize(defaultClientMaxReceiveMessageSize))
	var tlsDialOpts grpc.DialOption
	var err error
	if tlsOpts != nil {
		tlsDialOpts, err = getTLSDialOption(tlsOpts)
		if err != nil {
			return nil, err
		}
	} else {
		tlsDialOpts = grpc.WithTransportCredentials(insecure.NewCredentials())
	}
	return []grpc.DialOption{keepaliveOption, initialWindowSizeOption, initialConnWindowSizeOption, msgSizeOption, tlsDialOpts}, nil
}

func ServerOptions(options *dubbokeepalive.Options, interceptors ...grpc.UnaryServerInterceptor) []grpc.ServerOption {
	maxStreams := features.MaxConcurrentStreams
	maxRecvMsgSize := features.MaxRecvMsgSize

	grpcOptions := []grpc.ServerOption{
		grpc.UnaryInterceptor(middleware.ChainUnaryServer(interceptors...)),
		grpc.MaxConcurrentStreams(uint32(maxStreams)),
		grpc.MaxRecvMsgSize(maxRecvMsgSize),
		// Ensure we allow clients sufficient ability to send keep alives. If this is higher than client
		// keep alive setting, it will prematurely get a GOAWAY sent.
		grpc.KeepaliveEnforcementPolicy(keepalive.EnforcementPolicy{
			MinTime: options.Time / 2,
		}),
		grpc.KeepaliveParams(keepalive.ServerParameters{
			Time:                  options.Time,
			Timeout:               options.Timeout,
			MaxConnectionAge:      options.MaxServerConnectionAge,
			MaxConnectionAgeGrace: options.MaxServerConnectionAgeGrace,
		}),
	}

	return grpcOptions
}

func GRPCErrorType(err error) ErrorType {
	if err == io.EOF {
		return GracefulTermination
	}

	if s, ok := status.FromError(err); ok {
		if s.Code() == codes.Canceled || s.Code() == codes.DeadlineExceeded {
			return ExpectedError
		}
		if s.Code() == codes.Unavailable && containsExpectedMessage(s.Message()) {
			return ExpectedError
		}
	}
	// If this is not a gRPCStatus we should just error message.
	if strings.Contains(err.Error(), "stream terminated by RST_STREAM with error code: NO_ERROR") {
		return ExpectedError
	}
	if strings.Contains(err.Error(), "received prior goaway: code: NO_ERROR") {
		return ExpectedError
	}

	return UnexpectedError
}

func containsExpectedMessage(msg string) bool {
	for m := range expectedGrpcFailureMessages {
		if strings.Contains(msg, m) {
			return true
		}
	}
	return false
}
