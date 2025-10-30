package core

import (
	"github.com/apache/dubbo-kubernetes/sail/pkg/model"
	discovery "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v3"
)

func (configgen *ConfigGeneratorImpl) BuildClusters(proxy *model.Proxy, req *model.PushRequest) ([]*discovery.Resource, model.XdsLogDetails) {
	return nil, model.XdsLogDetails{}
}

func (configgen *ConfigGeneratorImpl) BuildDeltaClusters(proxy *model.Proxy, updates *model.PushRequest,
	watched *model.WatchedResource,
) ([]*discovery.Resource, []string, model.XdsLogDetails, bool) {
	return nil, nil, model.XdsLogDetails{}, false
}
