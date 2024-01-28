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
