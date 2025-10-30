package core

import (
	"github.com/apache/dubbo-kubernetes/sail/pkg/model"
	discovery "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v3"
)

func (configgen *ConfigGeneratorImpl) BuildHTTPRoutes(
	node *model.Proxy,
	req *model.PushRequest,
	routeNames []string,
) ([]*discovery.Resource, model.XdsLogDetails) {
	return nil, model.XdsLogDetails{}
}
