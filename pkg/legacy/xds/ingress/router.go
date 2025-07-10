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

package ingress

import (
	mesh_proto "github.com/apache/dubbo-kubernetes/api/mesh/v1alpha1"
	core_mesh "github.com/apache/dubbo-kubernetes/pkg/core/resources/apis/mesh"
	core_xds "github.com/apache/dubbo-kubernetes/pkg/core/xds"
)

func BuildDestinationMap(mesh string, ingress *core_mesh.ZoneIngressResource) core_xds.DestinationMap {
	destinations := core_xds.DestinationMap{}
	for _, svc := range ingress.Spec.GetAvailableServices() {
		if mesh != svc.GetMesh() {
			continue
		}
		service := svc.Tags[mesh_proto.ServiceTag]
		destinations[service] = destinations[service].Add(mesh_proto.MatchTags(svc.Tags))
	}
	return destinations
}
