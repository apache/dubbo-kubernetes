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
	envoy_listener "github.com/envoyproxy/go-control-plane/envoy/config/listener/v3"

	"google.golang.org/protobuf/types/known/wrapperspb"

	mesh_proto "github.com/apache/dubbo-kubernetes/api/mesh/v1alpha1"

	core_xds "github.com/apache/dubbo-kubernetes/pkg/core/xds"

	v3 "github.com/apache/dubbo-kubernetes/pkg/xds/envoy/listeners/v3"
)

func TLSInspector() ListenerBuilderOpt {
	return AddListenerConfigurer(&v3.TLSInspectorConfigurer{})
}

func OriginalDstForwarder() ListenerBuilderOpt {
	return AddListenerConfigurer(&v3.OriginalDstForwarderConfigurer{})
}

func InboundListener(address string, port uint32, protocol core_xds.SocketAddressProtocol) ListenerBuilderOpt {
	return AddListenerConfigurer(&v3.InboundListenerConfigurer{
		Protocol: protocol,
		Address:  address,
		Port:     port,
	})
}

func OutboundListener(address string, port uint32, protocol core_xds.SocketAddressProtocol) ListenerBuilderOpt {
	return AddListenerConfigurer(&v3.OutboundListenerConfigurer{
		Protocol: protocol,
		Address:  address,
		Port:     port,
	})
}

func PipeListener(socketPath string) ListenerBuilderOpt {
	return AddListenerConfigurer(&v3.PipeListenerConfigurer{
		SocketPath: socketPath,
	})
}

func NoBindToPort() ListenerBuilderOpt {
	return AddListenerConfigurer(&v3.TransparentProxyingConfigurer{})
}

func FilterChain(builder *FilterChainBuilder) ListenerBuilderOpt {
	return AddListenerConfigurer(
		v3.ListenerConfigureFunc(func(listener *envoy_listener.Listener) error {
			filterChain, err := builder.Build()
			if err != nil {
				return err
			}
			listener.FilterChains = append(listener.FilterChains, filterChain.(*envoy_listener.FilterChain))
			return nil
		}),
	)
}

func ConnectionBufferLimit(bytes uint32) ListenerBuilderOpt {
	return AddListenerConfigurer(
		v3.ListenerMustConfigureFunc(func(l *envoy_listener.Listener) {
			l.PerConnectionBufferLimitBytes = wrapperspb.UInt32(bytes)
		}))
}

func EnableReusePort(enable bool) ListenerBuilderOpt {
	return AddListenerConfigurer(
		v3.ListenerMustConfigureFunc(func(l *envoy_listener.Listener) {
			l.EnableReusePort = &wrapperspb.BoolValue{Value: enable}
		}))
}

func EnableFreebind(enable bool) ListenerBuilderOpt {
	return AddListenerConfigurer(
		v3.ListenerMustConfigureFunc(func(l *envoy_listener.Listener) {
			l.Freebind = wrapperspb.Bool(enable)
		}))
}

func TagsMetadata(tags map[string]string) ListenerBuilderOpt {
	return AddListenerConfigurer(&v3.TagsMetadataConfigurer{
		Tags: tags,
	})
}

func AdditionalAddresses(addresses []mesh_proto.OutboundInterface) ListenerBuilderOpt {
	return AddListenerConfigurer(&v3.AdditionalAddressConfigurer{
		Addresses: addresses,
	})
}
