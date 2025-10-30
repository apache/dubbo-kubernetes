package networking

import core "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"

type TransportProtocol uint8

const (
	// TransportProtocolTCP is a TCP listener
	TransportProtocolTCP = iota
	// TransportProtocolQUIC is a QUIC listener
	TransportProtocolQUIC
)

func (tp TransportProtocol) String() string {
	switch tp {
	case TransportProtocolTCP:
		return "tcp"
	case TransportProtocolQUIC:
		return "quic"
	}
	return "unknown"
}

func (tp TransportProtocol) ToEnvoySocketProtocol() core.SocketAddress_Protocol {
	if tp == TransportProtocolQUIC {
		return core.SocketAddress_UDP
	}
	return core.SocketAddress_TCP
}
