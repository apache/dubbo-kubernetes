package options

import naviagent "github.com/apache/dubbo-kubernetes/pkg/navi-agent"

// ProxyArgs provides all of the configuration parameters for the Navi proxy.
type ProxyArgs struct {
	naviagent.Proxy
}

func NewProxyArgs() ProxyArgs {
	p := ProxyArgs{}
	return p
}
