package xds

import (
	mesh_proto "github.com/apache/dubbo-kubernetes/api/mesh/v1alpha1"
	core_model "github.com/apache/dubbo-kubernetes/pkg/core/resources/model"
)

// TypedMatchingPolicies all policies of this type matching
type TypedMatchingPolicies struct {
	Type              core_model.ResourceType
	InboundPolicies   map[mesh_proto.InboundInterface][]core_model.Resource
	OutboundPolicies  map[mesh_proto.OutboundInterface][]core_model.Resource
	ServicePolicies   map[ServiceName][]core_model.Resource
	DataplanePolicies []core_model.Resource
}

type PluginOriginatedPolicies map[core_model.ResourceType]TypedMatchingPolicies

type MatchedPolicies struct {
	// Inbound(Listener) -> Policy

	// Service(Cluster) -> Policy

	// Outbound(Listener) -> Policy

	// Dataplane -> Policy

}
