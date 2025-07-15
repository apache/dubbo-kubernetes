/*
 * Licensed to the Apache Software Foundation (ASF) under one or more
 * contributor license agreements.  See the NOTICE file distributed with
 * this work for additional information regarding copyright ownership.
 * The ASF licenses this file to You under the Apache License, Version 2.0
 * (the "License"); you may not use this file except in compliance with
 * the License.  You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

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
