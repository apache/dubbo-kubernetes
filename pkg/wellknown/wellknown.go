package wellknown

// Package wellknown contains common names for filters, listeners, etc.
// copied from envoyproxy/go-control-plane.
// TODO: remove this package

// Network filter names
const (
	// HTTPConnectionManager network filter
	HTTPConnectionManager = "envoy.filters.network.http_connection_manager"
	// TCPProxy network filter
	TCPProxy = "envoy.filters.network.tcp_proxy"
)
