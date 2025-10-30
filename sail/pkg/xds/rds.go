package xds

import (
	"github.com/apache/dubbo-kubernetes/pkg/config/schema/kind"
	"github.com/apache/dubbo-kubernetes/pkg/util/sets"
	"github.com/apache/dubbo-kubernetes/sail/pkg/model"
	"github.com/apache/dubbo-kubernetes/sail/pkg/networking/core"
)

type RdsGenerator struct {
	ConfigGenerator core.ConfigGenerator
}

var _ model.XdsResourceGenerator = &RdsGenerator{}

var skippedRdsConfigs = sets.New[kind.Kind](
	kind.RequestAuthentication,
	kind.PeerAuthentication,
	kind.Secret,
	kind.DNSName,
)

func rdsNeedsPush(req *model.PushRequest, proxy *model.Proxy) bool {
	if res, ok := xdsNeedsPush(req, proxy); ok {
		return res
	}
	if !req.Full {
		return false
	}
	for config := range req.ConfigsUpdated {
		if !skippedRdsConfigs.Contains(config.Kind) {
			return true
		}
	}
	return false
}

func (c RdsGenerator) Generate(proxy *model.Proxy, w *model.WatchedResource, req *model.PushRequest) (model.Resources, model.XdsLogDetails, error) {
	if !rdsNeedsPush(req, proxy) {
		return nil, model.DefaultXdsLogDetails, nil
	}
	resources, logDetails := c.ConfigGenerator.BuildHTTPRoutes(proxy, req, w.ResourceNames.UnsortedList())
	return resources, logDetails, nil
}
