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

package endpoints

import (
	"sort"
)

import (
	envoy_core "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	envoy_endpoint "github.com/envoyproxy/go-control-plane/envoy/config/endpoint/v3"

	proto_wrappers "github.com/golang/protobuf/ptypes/wrappers"
)

import (
	core_xds "github.com/apache/dubbo-kubernetes/pkg/core/xds"
	envoy "github.com/apache/dubbo-kubernetes/pkg/xds/envoy/metadata/v3"
)

func CreateClusterLoadAssignment(clusterName string, endpoints []core_xds.Endpoint) *envoy_endpoint.ClusterLoadAssignment {
	localityLbEndpoints := LocalityLbEndpointsMap{}

	for _, ep := range endpoints {
		var address *envoy_core.Address
		if ep.UnixDomainPath != "" {
			address = &envoy_core.Address{
				Address: &envoy_core.Address_Pipe{
					Pipe: &envoy_core.Pipe{
						Path: ep.UnixDomainPath,
					},
				},
			}
		} else {
			address = &envoy_core.Address{
				Address: &envoy_core.Address_SocketAddress{
					SocketAddress: &envoy_core.SocketAddress{
						Protocol: envoy_core.SocketAddress_TCP,
						Address:  ep.Target,
						PortSpecifier: &envoy_core.SocketAddress_PortValue{
							PortValue: ep.Port,
						},
					},
				},
			}
		}
		lbEndpoint := &envoy_endpoint.LbEndpoint{
			Metadata: envoy.EndpointMetadata(ep.Tags),
			HostIdentifier: &envoy_endpoint.LbEndpoint_Endpoint{
				Endpoint: &envoy_endpoint.Endpoint{
					Address: address,
				},
			},
		}
		if ep.Weight > 0 {
			lbEndpoint.LoadBalancingWeight = &proto_wrappers.UInt32Value{
				Value: ep.Weight,
			}
		}
		localityLbEndpoints.append(ep, lbEndpoint)
	}

	for _, lbEndpoints := range localityLbEndpoints {
		// sort the slice to ensure stable Envoy configuration
		sortLbEndpoints(lbEndpoints.LbEndpoints)
	}

	return &envoy_endpoint.ClusterLoadAssignment{
		ClusterName: clusterName,
		Endpoints:   localityLbEndpoints.asSlice(),
	}
}

type LocalityLbEndpointsMap map[string]*envoy_endpoint.LocalityLbEndpoints

func (l LocalityLbEndpointsMap) append(ep core_xds.Endpoint, endpoint *envoy_endpoint.LbEndpoint) {
	key := ep.LocalityString()
	if _, ok := l[key]; !ok {
		var locality *envoy_core.Locality
		priority := uint32(0)
		lbWeight := uint32(0)
		if ep.HasLocality() {
			locality = &envoy_core.Locality{
				Zone:    ep.Locality.Zone,
				SubZone: ep.Locality.SubZone,
			}
			priority = ep.Locality.Priority
			lbWeight = ep.Locality.Weight
		}

		localityLbEndpoint := &envoy_endpoint.LocalityLbEndpoints{
			LbEndpoints: make([]*envoy_endpoint.LbEndpoint, 0),
			Locality:    locality,
			Priority:    priority,
		}
		if lbWeight > 0 {
			localityLbEndpoint.LoadBalancingWeight = &proto_wrappers.UInt32Value{Value: lbWeight}
		}
		l[key] = localityLbEndpoint
	}
	l[key].LbEndpoints = append(l[key].LbEndpoints, endpoint)
}

func (l LocalityLbEndpointsMap) asSlice() []*envoy_endpoint.LocalityLbEndpoints {
	slice := make([]*envoy_endpoint.LocalityLbEndpoints, 0, len(l))

	for _, lle := range l {
		sortLbEndpoints(lle.LbEndpoints)
		slice = append(slice, lle)
	}

	// sort the slice to ensure stable Envoy configuration
	sort.Slice(slice, func(i, j int) bool {
		left, right := slice[i], slice[j]
		leftLocality := left.GetLocality().GetRegion() + left.GetLocality().GetZone() + left.GetLocality().GetSubZone()
		rightLocality := right.GetLocality().GetRegion() + right.GetLocality().GetZone() + right.GetLocality().GetSubZone()
		if leftLocality != "" || rightLocality != "" {
			return leftLocality < rightLocality
		}
		return len(left.LbEndpoints) < len(right.LbEndpoints)
	})

	return slice
}

func sortLbEndpoints(lbEndpoints []*envoy_endpoint.LbEndpoint) {
	sort.Slice(lbEndpoints, func(i, j int) bool {
		left, right := lbEndpoints[i], lbEndpoints[j]
		leftAddr := left.GetEndpoint().GetAddress().GetSocketAddress().GetAddress()
		rightAddr := right.GetEndpoint().GetAddress().GetSocketAddress().GetAddress()
		if leftAddr == rightAddr {
			return left.GetEndpoint().GetAddress().GetSocketAddress().GetPortValue() < right.GetEndpoint().GetAddress().GetSocketAddress().GetPortValue()
		}
		return leftAddr < rightAddr
	})
}
