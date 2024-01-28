package v3

import (
	util_proto "github.com/apache/dubbo-kubernetes/pkg/util/proto"
	envoy_listener "github.com/envoyproxy/go-control-plane/envoy/config/listener/v3"
)

type OriginalDstForwarderConfigurer struct{}

var _ ListenerConfigurer = &OriginalDstForwarderConfigurer{}

func (c *OriginalDstForwarderConfigurer) Configure(l *envoy_listener.Listener) error {
	l.UseOriginalDst = util_proto.Bool(true)
	return nil
}
