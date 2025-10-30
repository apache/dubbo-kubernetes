package core

import (
	"github.com/apache/dubbo-kubernetes/sail/pkg/model"
	listener "github.com/envoyproxy/go-control-plane/envoy/config/listener/v3"
)

func (configgen *ConfigGeneratorImpl) BuildListeners(node *model.Proxy,
	push *model.PushContext,
) []*listener.Listener {
	return nil
}
