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

package v3

import (
	mesh_proto "github.com/apache/dubbo-kubernetes/api/mesh/v1alpha1"
	envoy_core "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	listenerv3 "github.com/envoyproxy/go-control-plane/envoy/config/listener/v3"
)

type AdditionalAddressConfigurer struct {
	Addresses []mesh_proto.OutboundInterface
}

func (c *AdditionalAddressConfigurer) Configure(l *listenerv3.Listener) error {
	if len(c.Addresses) < 1 || l.Address == nil {
		return nil
	}

	var addresses []*listenerv3.AdditionalAddress
	for _, addr := range c.Addresses {
		address := makeSocketAddress(addr.DataplaneIP, addr.DataplanePort, l.Address.GetSocketAddress().GetProtocol())
		addresses = append(addresses, address)
	}
	l.AdditionalAddresses = addresses
	return nil
}

func makeSocketAddress(addr string, port uint32, protocol envoy_core.SocketAddress_Protocol) *listenerv3.AdditionalAddress {
	return &listenerv3.AdditionalAddress{
		Address: &envoy_core.Address{
			Address: &envoy_core.Address_SocketAddress{
				SocketAddress: &envoy_core.SocketAddress{
					Protocol: protocol,
					Address:  addr,
					PortSpecifier: &envoy_core.SocketAddress_PortValue{
						PortValue: port,
					},
				},
			},
		},
	}
}
