package features

import "github.com/apache/dubbo-kubernetes/pkg/env"

var (
	MaxConcurrentStreams = env.Register(
		"DUBBO_GPRC_MAXSTREAMS",
		100000,
		"Sets the maximum number of concurrent grpc streams.",
	).Get()
)
