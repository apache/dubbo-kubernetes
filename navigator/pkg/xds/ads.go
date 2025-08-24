package xds

import (
	"github.com/apache/dubbo-kubernetes/navigator/pkg/model"
	"github.com/apache/dubbo-kubernetes/navigator/pkg/xds"
	discovery "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v3"
)

type Connection struct {
	xds.Connection
	node         *core.Node
	proxy        *model.Proxy
	deltaStream  DeltaDiscoveryStream
	deltaReqChan chan *discovery.DeltaDiscoveryRequest
	s            *DiscoveryServer
	ids          []string
}

func (conn *Connection) XdsConnection() *xds.Connection {
	return &conn.Connection
}

func (conn *Connection) Proxy() *model.Proxy {
	return conn.proxy
}
