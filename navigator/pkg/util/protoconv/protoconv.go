package protoconv

import (
	"github.com/apache/dubbo-kubernetes/navigator/pkg/features"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
)

// MessageToAnyWithError converts from proto message to proto Any
func MessageToAnyWithError(msg proto.Message) (*anypb.Any, error) {
	b, err := marshal(msg)
	if err != nil {
		return nil, err
	}
	return &anypb.Any{
		// nolint: staticcheck
		TypeUrl: "type.googleapis.com/" + string(msg.ProtoReflect().Descriptor().FullName()),
		Value:   b,
	}, nil
}

func marshal(msg proto.Message) ([]byte, error) {
	if features.EnableVtprotobuf {
		if vt, ok := msg.(vtStrictMarshal); ok {
			// Attempt to use more efficient implementation
			// "Strict" is the equivalent to Deterministic=true below
			return vt.MarshalVTStrict()
		}
	}
	// If not available, fallback to normal implementation
	return proto.MarshalOptions{Deterministic: true}.Marshal(msg)
}

// MessageToAny converts from proto message to proto Any
func MessageToAny(msg proto.Message) *anypb.Any {
	out, err := MessageToAnyWithError(msg)
	if err != nil {
		return nil
	}
	return out
}

// https://github.com/planetscale/vtprotobuf#available-features
type vtStrictMarshal interface {
	MarshalVTStrict() ([]byte, error)
}
