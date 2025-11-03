package xds

import (
	"fmt"

	"github.com/apache/dubbo-kubernetes/pkg/config/schema/kind"
	"github.com/apache/dubbo-kubernetes/pkg/util/sets"
	"github.com/apache/dubbo-kubernetes/sail/pkg/model"
	"github.com/apache/dubbo-kubernetes/sail/pkg/util/protoconv"
	"github.com/apache/dubbo-kubernetes/sail/pkg/xds/endpoints"
	discovery "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v3"
)

var _ model.XdsDeltaResourceGenerator = &EdsGenerator{}

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
	kind.PeerAuthentication,
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

// canSendPartialFullPushes checks if a request contains *only* endpoints updates except `skippedEdsConfigs`.
// This allows us to perform more efficient pushes where we only update the endpoints that did change.
func canSendPartialFullPushes(req *model.PushRequest) bool {
	// If we don't know what configs are updated, just send a full push
	if req.Forced {
		return false
	}
	for cfg := range req.ConfigsUpdated {
		if skippedEdsConfigs.Contains(cfg.Kind) {
			// the updated config does not impact EDS, skip it
			// this happens when push requests are merged due to debounce
			continue
		}
		if cfg.Kind != kind.ServiceEntry {
			return false
		}
	}
	return true
}

func (eds *EdsGenerator) buildEndpoints(proxy *model.Proxy,
	req *model.PushRequest,
	w *model.WatchedResource,
) (model.Resources, model.XdsLogDetails) {
	var edsUpdatedServices map[string]struct{}
	// canSendPartialFullPushes determines if we can send a partial push (ie a subset of known CLAs).
	// This is safe when only Services has changed, as this implies that only the CLAs for the
	// associated Service changed. Note when a multi-network Service changes it triggers a push with
	// ConfigsUpdated=ALL, so in this case we would not enable a partial push.
	// Despite this code existing on the SotW code path, sending these partial pushes is still allowed;
	// see https://www.envoyproxy.io/docs/envoy/latest/api-docs/xds_protocol#grouping-resources-into-responses
	if !req.Full || canSendPartialFullPushes(req) {
		edsUpdatedServices = model.ConfigNamesOfKind(req.ConfigsUpdated, kind.ServiceEntry)
	}
	var resources model.Resources
	empty := 0
	cached := 0
	regenerated := 0
	for clusterName := range w.ResourceNames {
		if edsUpdatedServices != nil {
			if _, ok := edsUpdatedServices[model.ParseSubsetKeyHostname(clusterName)]; !ok {
				// Cluster was not updated, skip recomputing. This happens when we get an incremental update for a
				// specific Hostname. On connect or for full push edsUpdatedServices will be empty.
				continue
			}
		}
		builder := endpoints.NewEndpointBuilder(clusterName, proxy, req.Push)
		if builder == nil {
			continue
		}

		// Try to get from cache
		if eds.Cache != nil {
			cachedEndpoint := eds.Cache.Get(builder)
			if cachedEndpoint != nil {
				resources = append(resources, cachedEndpoint)
				cached++
				continue
			}
		}

		// generate eds from beginning
		l := builder.BuildClusterLoadAssignment(eds.EndpointIndex)
		if l == nil {
			continue
		}
		regenerated++

		if len(l.Endpoints) == 0 {
			empty++
		}
		resource := &discovery.Resource{
			Name:     l.ClusterName,
			Resource: protoconv.MessageToAny(l),
		}
		resources = append(resources, resource)
		if eds.Cache != nil {
			eds.Cache.Add(builder, req, resource)
		}
	}
	return resources, model.XdsLogDetails{
		Incremental:    len(edsUpdatedServices) != 0,
		AdditionalInfo: fmt.Sprintf("empty:%v cached:%v/%v", empty, cached, cached+regenerated),
	}
}

func shouldUseDeltaEds(req *model.PushRequest) bool {
	if !req.Full {
		return false
	}
	return canSendPartialFullPushes(req)
}

func (eds *EdsGenerator) GenerateDeltas(proxy *model.Proxy, req *model.PushRequest,
	w *model.WatchedResource,
) (model.Resources, model.DeletedResources, model.XdsLogDetails, bool, error) {
	if !edsNeedsPush(req, proxy) {
		return nil, nil, model.DefaultXdsLogDetails, false, nil
	}
	if !shouldUseDeltaEds(req) {
		resources, logDetails := eds.buildEndpoints(proxy, req, w)
		return resources, nil, logDetails, false, nil
	}

	resources, removed, logs := eds.buildDeltaEndpoints(proxy, req, w)
	return resources, removed, logs, true, nil
}

// buildDeltaEndpoints builds endpoints for delta XDS
func (eds *EdsGenerator) buildDeltaEndpoints(proxy *model.Proxy,
	req *model.PushRequest,
	w *model.WatchedResource,
) (model.Resources, []string, model.XdsLogDetails) {
	edsUpdatedServices := model.ConfigNamesOfKind(req.ConfigsUpdated, kind.ServiceEntry)
	var resources model.Resources
	var removed []string
	empty := 0
	cached := 0
	regenerated := 0

	for clusterName := range w.ResourceNames {
		// filter out eds that are not updated for clusters
		if len(edsUpdatedServices) > 0 {
			if _, ok := edsUpdatedServices[model.ParseSubsetKeyHostname(clusterName)]; !ok {
				continue
			}
		}

		builder := endpoints.NewEndpointBuilder(clusterName, proxy, req.Push)
		if builder == nil {
			removed = append(removed, clusterName)
			continue
		}

		// if a service is not found, it means the cluster is removed
		if !builder.ServiceFound() {
			removed = append(removed, clusterName)
			continue
		}

		// Try to get from cache
		if eds.Cache != nil {
			cachedEndpoint := eds.Cache.Get(builder)
			if cachedEndpoint != nil {
				resources = append(resources, cachedEndpoint)
				cached++
				continue
			}
		}

		// generate new eds cache
		l := builder.BuildClusterLoadAssignment(eds.EndpointIndex)
		if l == nil || len(l.Endpoints) == 0 {
			removed = append(removed, clusterName)
			continue
		}
		regenerated++
		if len(l.Endpoints) == 0 {
			empty++
		}
		resource := &discovery.Resource{
			Name:     l.ClusterName,
			Resource: protoconv.MessageToAny(l),
		}
		resources = append(resources, resource)
		if eds.Cache != nil {
			eds.Cache.Add(builder, req, resource)
		}
	}
	return resources, removed, model.XdsLogDetails{
		Incremental:    len(edsUpdatedServices) != 0,
		AdditionalInfo: fmt.Sprintf("empty:%v cached:%v/%v", empty, cached, cached+regenerated),
	}
}
