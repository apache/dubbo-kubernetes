package xds

import "github.com/apache/dubbo-kubernetes/sail/pkg/model"

func (s *DiscoveryServer) SvcUpdate(shard model.ShardKey, hostname string, namespace string, event model.Event) {
	// When a service deleted, we should cleanup the endpoint shards and also remove keys from EndpointIndex to
	// prevent memory leaks.
	if event == model.EventDelete {
		s.Env.EndpointIndex.DeleteServiceShard(shard, hostname, namespace, false)
	} else {
	}
}
