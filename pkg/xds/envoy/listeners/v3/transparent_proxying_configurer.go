package v3

import (
	util_proto "github.com/apache/dubbo-kubernetes/pkg/util/proto"
	envoy_listener "github.com/envoyproxy/go-control-plane/envoy/config/listener/v3"
)

type TransparentProxyingConfigurer struct{}

func (c *TransparentProxyingConfigurer) Configure(l *envoy_listener.Listener) error {
	l.BindToPort = util_proto.Bool(false)
	return nil
}
