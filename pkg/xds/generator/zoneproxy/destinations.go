package zoneproxy

import (
	"reflect"
)

import (
	"golang.org/x/exp/slices"
)

import (
	mesh_proto "github.com/apache/dubbo-kubernetes/api/mesh/v1alpha1"
	xds_context "github.com/apache/dubbo-kubernetes/pkg/xds/context"
	envoy_tags "github.com/apache/dubbo-kubernetes/pkg/xds/envoy/tags"
)

func BuildMeshDestinations(
	availableServices []*mesh_proto.ZoneIngress_AvailableService, // available services for a single mesh
	res xds_context.Resources,
) map[string][]envoy_tags.Tags {
	destForMesh := map[string][]envoy_tags.Tags{}
	addTrafficFlowByDefaultDestination(destForMesh)
	return destForMesh
}

// addTrafficFlowByDefaultDestination Make sure that when
// at least one MeshHTTPRoute policy exists there will be a "match all"
// destination pointing to all services (dubbo.io/service:* -> dubbo.io/service:*)
func addTrafficFlowByDefaultDestination(
	destinations map[string][]envoy_tags.Tags,
) {
	// We need to add a destination to route any service to any instance of
	// that service
	matchAllTags := envoy_tags.Tags{mesh_proto.ServiceTag: mesh_proto.MatchAllTag}
	matchAllDestinations := destinations[mesh_proto.MatchAllTag]
	foundAllServicesDestination := slices.ContainsFunc(
		matchAllDestinations,
		func(tagsElem envoy_tags.Tags) bool {
			return reflect.DeepEqual(tagsElem, matchAllTags)
		},
	)

	if !foundAllServicesDestination {
		matchAllDestinations = append(matchAllDestinations, matchAllTags)
	}

	destinations[mesh_proto.MatchAllTag] = matchAllDestinations
}
