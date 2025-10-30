package core

import (
	"github.com/apache/dubbo-kubernetes/sail/pkg/model"
	core "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
)

func (configgen *ConfigGeneratorImpl) BuildExtensionConfiguration(
	proxy *model.Proxy, push *model.PushContext, extensionConfigNames []string, pullSecrets map[string][]byte,
) []*core.TypedExtensionConfig {
	return nil
}
