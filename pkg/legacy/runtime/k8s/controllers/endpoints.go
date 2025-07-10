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

package controllers

import (
	"sort"

	mesh_proto "github.com/apache/dubbo-kubernetes/api/mesh/v1alpha1"
	core_mesh "github.com/apache/dubbo-kubernetes/pkg/core/resources/apis/mesh"
)

type Endpoint struct {
	Address  string
	Port     uint32
	Instance string
}

type EndpointsByService map[string][]Endpoint

func (e EndpointsByService) Services() []string {
	list := make([]string, 0, len(e))
	for key := range e {
		list = append(list, key)
	}
	sort.Strings(list)
	return list
}

func endpointsByService(dataplanes []*core_mesh.DataplaneResource) EndpointsByService {
	result := EndpointsByService{}
	for _, other := range dataplanes {
		for _, inbound := range other.Spec.Networking.GetInbound() {
			if inbound.State == mesh_proto.Dataplane_Networking_Inbound_Ignored {
				continue
			}
			svc, ok := inbound.GetTags()[mesh_proto.ServiceTag]
			if !ok {
				continue
			}
			endpoint := Endpoint{
				Port:     inbound.Port,
				Instance: inbound.GetTags()[mesh_proto.InstanceTag],
			}
			if inbound.Address != "" {
				endpoint.Address = inbound.Address
			} else {
				endpoint.Address = other.Spec.Networking.Address
			}
			result[svc] = append(result[svc], endpoint)
		}
	}
	return result
}
