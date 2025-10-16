package xds

import (
	"github.com/apache/dubbo-kubernetes/pkg/config/schema/kind"
	"github.com/apache/dubbo-kubernetes/pkg/util/sets"
	"github.com/apache/dubbo-kubernetes/sail/pkg/model"
)

func (s *DiscoveryServer) ServiceUpdate(shard model.ShardKey, hostname string, namespace string, event model.Event) {
	// When a service deleted, we should cleanup the endpoint shards and also remove keys from EndpointIndex to
	// prevent memory leaks.
	if event == model.EventDelete {
		s.Env.EndpointIndex.DeleteServiceShard(shard, hostname, namespace, false)
	} else {
	}
}

func (s *DiscoveryServer) EDSUpdate(shard model.ShardKey, serviceName string, namespace string,
	dubboEndpoints []*model.DubboEndpoint,
) {
	// Update the endpoint shards
	pushType := s.Env.EndpointIndex.UpdateServiceEndpoints(shard, serviceName, namespace, dubboEndpoints, true)
	if pushType == model.IncrementalPush || pushType == model.FullPush {
		// Trigger a push
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
	// Update the endpoint shards
	s.Env.EndpointIndex.UpdateServiceEndpoints(shard, serviceName, namespace, istioEndpoints, false)
}
