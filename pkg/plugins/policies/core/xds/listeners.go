package xds

import (
	mesh_proto "github.com/apache/dubbo-kubernetes/api/mesh/v1alpha1"
	"github.com/apache/dubbo-kubernetes/pkg/core/xds"
	core_rules "github.com/apache/dubbo-kubernetes/pkg/plugins/policies/core/rules"
	"github.com/apache/dubbo-kubernetes/pkg/xds/generator"
	envoy_listener "github.com/envoyproxy/go-control-plane/envoy/config/listener/v3"
	envoy_resource "github.com/envoyproxy/go-control-plane/pkg/resource/v3"
)

type Listeners struct {
	Inbound         map[core_rules.InboundListener]*envoy_listener.Listener
	Outbound        map[mesh_proto.OutboundInterface]*envoy_listener.Listener
	Ipv4Passthrough *envoy_listener.Listener
	Ipv6Passthrough *envoy_listener.Listener
}

func GatherListeners(rs *xds.ResourceSet) Listeners {
	listeners := Listeners{
		Inbound:  map[core_rules.InboundListener]*envoy_listener.Listener{},
		Outbound: map[mesh_proto.OutboundInterface]*envoy_listener.Listener{},
	}

	for _, res := range rs.Resources(envoy_resource.ListenerType) {
		listener := res.Resource.(*envoy_listener.Listener)
		address := listener.GetAddress().GetSocketAddress()

		switch res.Origin {
		case generator.OriginOutbound:
			listeners.Outbound[mesh_proto.OutboundInterface{
				DataplaneIP:   address.GetAddress(),
				DataplanePort: address.GetPortValue(),
			}] = listener
		case generator.OriginInbound:
			listeners.Inbound[core_rules.InboundListener{
				Address: address.GetAddress(),
				Port:    address.GetPortValue(),
			}] = listener
		default:
			continue
		}
	}
	return listeners
}
