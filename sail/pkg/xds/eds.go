package xds

import (
	"github.com/apache/dubbo-kubernetes/sail/pkg/model"
)

func (s *DiscoveryServer) EDSUpdate(shard model.ShardKey, serviceName string, namespace string,
	dubboEndpoints []*model.DubboEndpoint,
) {
	// Update the endpoint shards
	pushType := s.Env.EndpointIndex.UpdateServiceEndpoints(shard, serviceName, namespace, dubboEndpoints, true)
	if pushType == model.IncrementalPush || pushType == model.FullPush {
		// Trigger a push
		s.ConfigUpdate(&model.PushRequest{
			Full:           pushType == model.FullPush,
			ConfigsUpdated: nil,
			Reason:         model.NewReasonStats(model.EndpointUpdate),
		})
	}
}
