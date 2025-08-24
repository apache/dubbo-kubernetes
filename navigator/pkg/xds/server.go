package xds

import (
	discovery "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v3"
	"time"
)

type DiscoveryStream = discovery.AggregatedDiscoveryService_StreamAggregatedResourcesServer

type Connection struct {
	peerAddr    string
	connectedAt time.Time
	conID       string
	pushChannel chan any
	stream      DiscoveryStream
	initialized chan struct{}
	stop        chan struct{}
	reqChan     chan *discovery.DiscoveryRequest
	errorChan   chan error
}

func NewConnection(peerAddr string, stream DiscoveryStream) Connection {
	return Connection{
		pushChannel: make(chan any),
		initialized: make(chan struct{}),
		stop:        make(chan struct{}),
		reqChan:     make(chan *discovery.DiscoveryRequest, 1),
		errorChan:   make(chan error, 1),
		peerAddr:    peerAddr,
		connectedAt: time.Now(),
		stream:      stream,
	}
}
