package xds

import (
	"github.com/apache/dubbo-kubernetes/sail/pkg/model"
	"github.com/apache/dubbo-kubernetes/sail/pkg/networking/core"
)

type CdsGenerator struct {
	ConfigGenerator core.ConfigGenerator
}

var _ model.XdsDeltaResourceGenerator = &CdsGenerator{}

func cdsNeedsPush(req *model.PushRequest, proxy *model.Proxy) (*model.PushRequest, bool) {
	if res, ok := xdsNeedsPush(req, proxy); ok {
		return req, res
	}

	if !req.Full {
		return req, false
	}
	// TODO ?
	return nil, false
}

func (c CdsGenerator) Generate(proxy *model.Proxy, w *model.WatchedResource, req *model.PushRequest) (model.Resources, model.XdsLogDetails, error) {
	req, needsPush := cdsNeedsPush(req, proxy)
	if !needsPush {
		return nil, model.DefaultXdsLogDetails, nil
	}
	clusters, logs := c.ConfigGenerator.BuildClusters(proxy, req)
	return clusters, logs, nil
}

// GenerateDeltas for CDS currently only builds deltas when services change. todo implement changes for DestinationRule, etc
func (c CdsGenerator) GenerateDeltas(proxy *model.Proxy, req *model.PushRequest,
	w *model.WatchedResource,
) (model.Resources, model.DeletedResources, model.XdsLogDetails, bool, error) {
	req, needsPush := cdsNeedsPush(req, proxy)
	if !needsPush {
		return nil, nil, model.DefaultXdsLogDetails, false, nil
	}
	updatedClusters, removedClusters, logs, usedDelta := c.ConfigGenerator.BuildDeltaClusters(proxy, req, w)
	return updatedClusters, removedClusters, logs, usedDelta, nil
}
