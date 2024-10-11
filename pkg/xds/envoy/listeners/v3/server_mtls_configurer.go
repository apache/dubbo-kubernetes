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
	envoy_core "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	envoy_listener "github.com/envoyproxy/go-control-plane/envoy/config/listener/v3"

	core_mesh "github.com/apache/dubbo-kubernetes/pkg/core/resources/apis/mesh"
	core_xds "github.com/apache/dubbo-kubernetes/pkg/core/xds"
	"github.com/apache/dubbo-kubernetes/pkg/util/proto"
	tls "github.com/apache/dubbo-kubernetes/pkg/xds/envoy/tls/v3"
)

type ServerSideMTLSConfigurer struct {
	Mesh           *core_mesh.MeshResource
	SecretsTracker core_xds.SecretsTracker
}

var _ FilterChainConfigurer = &ServerSideMTLSConfigurer{}

func (c *ServerSideMTLSConfigurer) Configure(filterChain *envoy_listener.FilterChain) error {
	if !c.Mesh.MTLSEnabled() {
		return nil
	}
	tlsContext, err := tls.CreateDownstreamTlsContext(c.SecretsTracker.RequestCa(c.Mesh.GetMeta().GetName()), c.SecretsTracker.RequestIdentityCert())
	if err != nil {
		return err
	}
	if tlsContext != nil {
		pbst, err := proto.MarshalAnyDeterministic(tlsContext)
		if err != nil {
			return err
		}
		filterChain.TransportSocket = &envoy_core.TransportSocket{
			Name: "envoy.transport_sockets.tls",
			ConfigType: &envoy_core.TransportSocket_TypedConfig{
				TypedConfig: pbst,
			},
		}
	}
	return nil
}
