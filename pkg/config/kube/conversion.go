package kube

import (
	"github.com/apache/dubbo-kubernetes/pkg/config/protocol"
	"github.com/apache/dubbo-kubernetes/pkg/util/sets"
	corev1 "k8s.io/api/core/v1"
	"strings"
)

var (
	grpcWeb    = string(protocol.GRPCWeb)
	grpcWebLen = len(grpcWeb)
)

const (
	DNS = 53
)

var wellKnownPorts = sets.New[int32](DNS)

func ConvertProtocol(port int32, portName string, proto corev1.Protocol, appProto *string) protocol.Instance {
	if proto == corev1.ProtocolUDP {
		return protocol.UDP
	}

	// If application protocol is set, we will use that
	// If not, use the port name
	name := portName
	if appProto != nil {
		name = *appProto
		// Kubernetes has a few AppProtocol specific standard names defined in the Service spec
		// Handle these only for AppProtocol (name cannot have these values, anyways).
		// https://github.com/kubernetes/kubernetes/blob/b4140391cf39ea54cd0227e294283bfb5718c22d/staging/src/k8s.io/api/core/v1/generated.proto#L1245-L1248
		switch name {
		// "http2 over cleartext", which is also what our HTTP2 port is
		case "kubernetes.io/h2c":
			return protocol.HTTP2
		// WebSocket over cleartext
		case "kubernetes.io/ws":
			return protocol.HTTP
		// WebSocket over TLS
		case "kubernetes.io/wss":
			return protocol.HTTPS
		}
	}

	// Check if the port name prefix is "grpc-web". Need to do this before the general
	// prefix check below, since it contains a hyphen.
	if len(name) >= grpcWebLen && strings.EqualFold(name[:grpcWebLen], grpcWeb) {
		return protocol.GRPCWeb
	}

	// Parse the port name to find the prefix, if any.
	i := strings.IndexByte(name, '-')
	if i >= 0 {
		name = name[:i]
	}

	p := protocol.Parse(name)
	if p == protocol.Unsupported {
		// Make TCP as default protocol for well know ports if protocol is not specified.
		if wellKnownPorts.Contains(port) {
			return protocol.TCP
		}
	}
	return p
}
