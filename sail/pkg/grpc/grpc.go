package grpc

import (
	dubbokeepalive "github.com/apache/dubbo-kubernetes/pkg/keepalive"
	"github.com/apache/dubbo-kubernetes/pkg/util/sets"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/keepalive"
	"google.golang.org/grpc/status"
	"io"
	"math"
	"strings"
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
