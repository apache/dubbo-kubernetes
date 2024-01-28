package topology

import (
	mesh_proto "github.com/apache/dubbo-kubernetes/api/mesh/v1alpha1"
	core_mesh "github.com/apache/dubbo-kubernetes/pkg/core/resources/apis/mesh"
	core_xds "github.com/apache/dubbo-kubernetes/pkg/core/xds"
)

func BuildEdsEndpoint(
	localZone string,
	dataplanes []*core_mesh.DataplaneResource,
	zoneIngresses []*core_mesh.ZoneIngressResource,
) core_xds.EndpointMap {
	outbound := core_xds.EndpointMap{}

	fillDataplaneOutbounds(outbound, dataplanes, 1, localZone)

	return outbound
}

// endpointWeight defines default weight for in-cluster endpoint.
// Examples of having service "backend":
//  1. Single-zone deployment, 2 instances in one cluster (zone1)
//     All endpoints have to have the same weight (ex. 1) to achieve fair loadbalancing.
//     Endpoints:
//     * backend-zone1-1 - weight: 1
//     * backend-zone1-2 - weight: 1
//  2. Multi-zone deployment, 2 instances in "zone1" (local zone), 3 instances in "zone2" (remote zone) with 1 Ingress instance
//     Endpoints:
//     * backend-zone1-1 - weight: 1
//     * backend-zone1-2 - weight: 1
//     * ingress-zone2-1 - weight: 3 (all remote endpoints are aggregated to one Ingress, it needs to have weight of instances in other cluster)
//  3. Multi-zone deployment, 2 instances in "zone1" (local zone), 2 instances in "zone2" (remote zone) with 1 Ingress instance
//     Many instances of Ingress will forward the traffic to the same endpoints in "zone2" so we need to lower the weights.
//     Since weights are integers, we cannot put fractional on ingress endpoints weights, we need to adjust "default" weight for local zone
//     Endpoints:
//     * backend-zone1-1 - weight: 2
//     * backend-zone1-2 - weight: 2
//     * ingress-zone2-1 - weight: 3
//     * ingress-zone2-2 - weight: 3
func fillDataplaneOutbounds(
	outbound core_xds.EndpointMap,
	dataplanes []*core_mesh.DataplaneResource,
	endpointWeight uint32,
	localZone string,
) {
	for _, dataplane := range dataplanes {
		dpSpec := dataplane.Spec
		dpNetworking := dpSpec.GetNetworking()

		for _, inbound := range dpNetworking.GetHealthyInbounds() {
			inboundTags := cloneTags(inbound.GetTags())
			serviceName := inboundTags[mesh_proto.ServiceTag]
			inboundInterface := dpNetworking.ToInboundInterface(inbound)
			inboundAddress := inboundInterface.DataplaneAdvertisedIP
			inboundPort := inboundInterface.DataplanePort

			outbound[serviceName] = append(outbound[serviceName], core_xds.Endpoint{
				Target:   inboundAddress,
				Port:     inboundPort,
				Tags:     inboundTags,
				Weight:   endpointWeight,
				Locality: GetLocality(localZone, getZone(inboundTags), true),
			})
		}
	}
}

func cloneTags(tags map[string]string) map[string]string {
	result := map[string]string{}
	for tag, value := range tags {
		result[tag] = value
	}
	return result
}

const (
	// Constants for Locality Aware load balancing
	// The Highest priority 0 shall be assigned to all locally available services
	// A priority of 1 is for ExternalServices and services exposed on neighboring ingress-es
	priorityLocal  = 0
	priorityRemote = 1
)

func GetLocality(localZone string, otherZone *string, localityAwareness bool) *core_xds.Locality {
	if otherZone == nil {
		return nil
	}

	var priority uint32 = priorityLocal

	if localityAwareness && localZone != *otherZone {
		priority = priorityRemote
	}

	return &core_xds.Locality{
		Zone:     *otherZone,
		Priority: priority,
	}
}

func getZone(tags map[string]string) *string {
	if zone, ok := tags[mesh_proto.ZoneTag]; ok {
		return &zone
	}
	return nil
}
