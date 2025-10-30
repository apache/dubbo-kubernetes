package xds

import (
	"github.com/apache/dubbo-kubernetes/pkg/config/schema/kind"
	"github.com/apache/dubbo-kubernetes/pkg/util/sets"
	"github.com/apache/dubbo-kubernetes/sail/pkg/model"
)

type EdsGenerator struct {
	Cache         model.XdsCache
	EndpointIndex *model.EndpointIndex
}

func (s *DiscoveryServer) ServiceUpdate(shard model.ShardKey, hostname string, namespace string, event model.Event) {
	if event == model.EventDelete {
		s.Env.EndpointIndex.DeleteServiceShard(shard, hostname, namespace, false)
	} else {
	}
}

func (s *DiscoveryServer) EDSUpdate(shard model.ShardKey, serviceName string, namespace string,
	dubboEndpoints []*model.DubboEndpoint,
) {
	pushType := s.Env.EndpointIndex.UpdateServiceEndpoints(shard, serviceName, namespace, dubboEndpoints, true)
	if pushType == model.IncrementalPush || pushType == model.FullPush {
		s.ConfigUpdate(&model.PushRequest{
			Full:           pushType == model.FullPush,
			ConfigsUpdated: sets.New(model.ConfigKey{Kind: kind.ServiceEntry, Name: serviceName, Namespace: namespace}),
			Reason:         model.NewReasonStats(model.EndpointUpdate),
		})
	}
}

func (s *DiscoveryServer) EDSCacheUpdate(shard model.ShardKey, serviceName string, namespace string,
	istioEndpoints []*model.DubboEndpoint,
) {
	s.Env.EndpointIndex.UpdateServiceEndpoints(shard, serviceName, namespace, istioEndpoints, false)
}

var skippedEdsConfigs = sets.New(
	kind.VirtualService,
	kind.RequestAuthentication,
	kind.Secret,
	kind.DNSName,
)

func edsNeedsPush(req *model.PushRequest, proxy *model.Proxy) bool {
	if res, ok := xdsNeedsPush(req, proxy); ok {
		return res
	}
	for config := range req.ConfigsUpdated {
		if !skippedEdsConfigs.Contains(config.Kind) {
			return true
		}
	}
	return false
}

func (eds *EdsGenerator) Generate(proxy *model.Proxy, w *model.WatchedResource, req *model.PushRequest) (model.Resources, model.XdsLogDetails, error) {
	if !edsNeedsPush(req, proxy) {
		return nil, model.DefaultXdsLogDetails, nil
	}
	resources, logDetails := eds.buildEndpoints(proxy, req, w)
	return resources, logDetails, nil
}

func (eds *EdsGenerator) buildEndpoints(proxy *model.Proxy,
	req *model.PushRequest,
	w *model.WatchedResource,
) (model.Resources, model.XdsLogDetails) {
	return nil, model.XdsLogDetails{}
}
