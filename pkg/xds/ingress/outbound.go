package ingress

import (
	mesh_proto "github.com/apache/dubbo-kubernetes/api/mesh/v1alpha1"
	core_mesh "github.com/apache/dubbo-kubernetes/pkg/core/resources/apis/mesh"
	core_xds "github.com/apache/dubbo-kubernetes/pkg/core/xds"
)

func BuildEndpointMap(
	destinations core_xds.DestinationMap,
	dataplanes []*core_mesh.DataplaneResource,
) core_xds.EndpointMap {
	if len(destinations) == 0 {
		return nil
	}

	outbound := core_xds.EndpointMap{}
	for _, dataplane := range dataplanes {
		for _, inbound := range dataplane.Spec.GetNetworking().GetHealthyInbounds() {
			service := inbound.Tags[mesh_proto.ServiceTag]
			selectors, ok := destinations[service]
			if !ok {
				continue
			}
			if !selectors.Matches(inbound.Tags) {
				continue
			}
			iface := dataplane.Spec.GetNetworking().ToInboundInterface(inbound)
			outbound[service] = append(outbound[service], core_xds.Endpoint{
				Target: iface.DataplaneAdvertisedIP,
				Port:   iface.DataplanePort,
				Tags:   inbound.Tags,
				Weight: 1,
			})
		}
	}

	return outbound
}
