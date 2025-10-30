package core

import (
	"github.com/apache/dubbo-kubernetes/sail/pkg/model"
	core "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	listener "github.com/envoyproxy/go-control-plane/envoy/config/listener/v3"
	discovery "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v3"
	meshconfig "istio.io/api/mesh/v1alpha1"
)

type ConfigGenerator interface {
	BuildListeners(node *model.Proxy, push *model.PushContext) []*listener.Listener
	BuildClusters(node *model.Proxy, req *model.PushRequest) ([]*discovery.Resource, model.XdsLogDetails)
	BuildDeltaClusters(proxy *model.Proxy, updates *model.PushRequest,
		watched *model.WatchedResource) ([]*discovery.Resource, []string, model.XdsLogDetails, bool)
	BuildHTTPRoutes(node *model.Proxy, req *model.PushRequest, routeNames []string) ([]*discovery.Resource, model.XdsLogDetails)
	BuildExtensionConfiguration(node *model.Proxy, push *model.PushContext, extensionConfigNames []string,
		pullSecrets map[string][]byte) []*core.TypedExtensionConfig
	MeshConfigChanged(mesh *meshconfig.MeshConfig)
}

type ConfigGeneratorImpl struct {
	Cache model.XdsCache
}

func NewConfigGenerator(cache model.XdsCache) *ConfigGeneratorImpl {
	return &ConfigGeneratorImpl{
		Cache: cache,
	}
}

func (configgen *ConfigGeneratorImpl) MeshConfigChanged(_ *meshconfig.MeshConfig) {
	return
}
