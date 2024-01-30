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
	envoy_listener "github.com/envoyproxy/go-control-plane/envoy/config/listener/v3"
	envoy_resource "github.com/envoyproxy/go-control-plane/pkg/resource/v3"
)

import (
	mesh_proto "github.com/apache/dubbo-kubernetes/api/mesh/v1alpha1"
	"github.com/apache/dubbo-kubernetes/pkg/core/xds"
	core_rules "github.com/apache/dubbo-kubernetes/pkg/plugins/policies/core/rules"
	"github.com/apache/dubbo-kubernetes/pkg/xds/generator"
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
