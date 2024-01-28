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

package listeners

import (
	util_proto "github.com/apache/dubbo-kubernetes/pkg/util/proto"
	v3 "github.com/apache/dubbo-kubernetes/pkg/xds/envoy/listeners/v3"
	envoy_core "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	envoy_listener "github.com/envoyproxy/go-control-plane/envoy/config/listener/v3"
)

// MatchTransportProtocol sets the transport protocol match for the filter chain.
func MatchTransportProtocol(transport string) FilterChainBuilderOpt {
	return AddFilterChainConfigurer(
		v3.FilterChainMustConfigureFunc(func(chain *envoy_listener.FilterChain) {
			if chain.FilterChainMatch == nil {
				chain.FilterChainMatch = &envoy_listener.FilterChainMatch{}
			}

			chain.FilterChainMatch.TransportProtocol = transport
		}),
	)
}

// MatchServerNames appends the giver server names to the filter chain
// match. These names are matches against the client SNI name for TLS
// sockets.
func MatchServerNames(names ...string) FilterChainBuilderOpt {
	return AddFilterChainConfigurer(
		v3.FilterChainMustConfigureFunc(func(chain *envoy_listener.FilterChain) {
			if chain.FilterChainMatch == nil {
				chain.FilterChainMatch = &envoy_listener.FilterChainMatch{}
			}

			for _, name := range names {
				// "" or "*" means match all, but Envoy supports only supports *.domain or more specific
				if name != "" && name != "*" {
					chain.FilterChainMatch.ServerNames = append(chain.FilterChainMatch.ServerNames, name)
				}
			}
		}),
	)
}

// MatchApplicationProtocols appends the given ALPN protocol names to the filter chain match.
func MatchApplicationProtocols(alpn ...string) FilterChainBuilderOpt {
	return AddFilterChainConfigurer(
		v3.FilterChainMustConfigureFunc(func(chain *envoy_listener.FilterChain) {
			if chain.FilterChainMatch == nil {
				chain.FilterChainMatch = &envoy_listener.FilterChainMatch{}
			}

			chain.FilterChainMatch.ApplicationProtocols = append(chain.FilterChainMatch.ApplicationProtocols, alpn...)
		}),
	)
}

// MatchSourceAddress appends an exact filter chain match for the given source IP address.
func MatchSourceAddress(address string) FilterChainBuilderOpt {
	return AddFilterChainConfigurer(
		v3.FilterChainMustConfigureFunc(func(chain *envoy_listener.FilterChain) {
			if chain.FilterChainMatch == nil {
				chain.FilterChainMatch = &envoy_listener.FilterChainMatch{}
			}

			chain.FilterChainMatch.SourcePrefixRanges = append(
				chain.FilterChainMatch.SourcePrefixRanges,
				&envoy_core.CidrRange{
					AddressPrefix: address,
					PrefixLen:     util_proto.UInt32(32),
				},
			)
		}),
	)
}
