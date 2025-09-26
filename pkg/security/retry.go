package security

import (
	"github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/retry"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
)

var CARetryOptions = []retry.CallOption{
	retry.WithMax(5),
	retry.WithCodes(codes.Canceled, codes.DeadlineExceeded, codes.ResourceExhausted, codes.Aborted, codes.Internal, codes.Unavailable),
}

func CARetryInterceptor() grpc.DialOption {
	return grpc.WithUnaryInterceptor(retry.UnaryClientInterceptor(CARetryOptions...))
}
