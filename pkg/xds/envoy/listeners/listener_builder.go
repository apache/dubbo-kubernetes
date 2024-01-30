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
	envoy_listener_v3 "github.com/envoyproxy/go-control-plane/envoy/config/listener/v3"

	"github.com/pkg/errors"
)

import (
	core_xds "github.com/apache/dubbo-kubernetes/pkg/core/xds"
	"github.com/apache/dubbo-kubernetes/pkg/xds/envoy"
	v3 "github.com/apache/dubbo-kubernetes/pkg/xds/envoy/listeners/v3"
	envoy_names "github.com/apache/dubbo-kubernetes/pkg/xds/envoy/names"
)

// ListenerBuilderOpt is a configuration option for ListenerBuilder.
//
// The goal of ListenerBuilderOpt is to facilitate fluent ListenerBuilder API.
type ListenerBuilderOpt interface {
	// ApplyTo adds ListenerConfigurer(s) to the ListenerBuilder.
	ApplyTo(builder *ListenerBuilder)
}

func NewListenerBuilder(apiVersion core_xds.APIVersion, name string) *ListenerBuilder {
	return &ListenerBuilder{
		apiVersion: apiVersion,
		name:       name,
	}
}

// NewInboundListenerBuilder creates an Inbound ListenBuilder
// with a default name: inbound:address:port
func NewInboundListenerBuilder(
	apiVersion core_xds.APIVersion,
	address string,
	port uint32,
	protocol core_xds.SocketAddressProtocol,
) *ListenerBuilder {
	listenerName := envoy_names.GetInboundListenerName(address, port)

	return NewListenerBuilder(apiVersion, listenerName).
		Configure(InboundListener(address, port, protocol))
}

// NewOutboundListenerBuilder creates an Outbound ListenBuilder
// with a default name: outbound:address:port
func NewOutboundListenerBuilder(
	apiVersion core_xds.APIVersion,
	address string,
	port uint32,
	protocol core_xds.SocketAddressProtocol,
) *ListenerBuilder {
	listenerName := envoy_names.GetOutboundListenerName(address, port)

	return NewListenerBuilder(apiVersion, listenerName).
		Configure(OutboundListener(address, port, protocol))
}

func (b *ListenerBuilder) WithOverwriteName(name string) *ListenerBuilder {
	b.name = name
	return b
}

// ListenerBuilder is responsible for generating an Envoy listener
// by applying a series of ListenerConfigurers.
type ListenerBuilder struct {
	apiVersion  core_xds.APIVersion
	configurers []v3.ListenerConfigurer
	name        string
}

// Configure configures ListenerBuilder by adding individual ListenerConfigurers.
func (b *ListenerBuilder) Configure(opts ...ListenerBuilderOpt) *ListenerBuilder {
	for _, opt := range opts {
		opt.ApplyTo(b)
	}

	return b
}

// Build generates an Envoy listener by applying a series of ListenerConfigurers.
func (b *ListenerBuilder) Build() (envoy.NamedResource, error) {
	switch b.apiVersion {
	case core_xds.APIVersion(envoy.APIV3):
		listener := envoy_listener_v3.Listener{
			Name: b.name,
		}
		for _, configurer := range b.configurers {
			if err := configurer.Configure(&listener); err != nil {
				return nil, err
			}
		}
		if listener.GetName() == "" {
			return nil, errors.New("listener name is required, but it was not provided")
		}
		return &listener, nil
	default:
		return nil, errors.New("unknown API")
	}
}

func (b *ListenerBuilder) MustBuild() envoy.NamedResource {
	listener, err := b.Build()
	if err != nil {
		panic(errors.Wrap(err, "failed to build Envoy Listener").Error())
	}

	return listener
}

func (b *ListenerBuilder) GetName() string {
	return b.name
}

// AddConfigurer appends a given ListenerConfigurer to the end of the chain.
func (b *ListenerBuilder) AddConfigurer(configurer v3.ListenerConfigurer) {
	b.configurers = append(b.configurers, configurer)
}

// ListenerBuilderOptFunc is a convenience type adapter.
type ListenerBuilderOptFunc func(builder *ListenerBuilder)

func (f ListenerBuilderOptFunc) ApplyTo(builder *ListenerBuilder) {
	if f != nil {
		f(builder)
	}
}

// AddListenerConfigurer produces an option that applies the given
// configurer to the listener.
func AddListenerConfigurer(c v3.ListenerConfigurer) ListenerBuilderOpt {
	return ListenerBuilderOptFunc(func(builder *ListenerBuilder) {
		builder.AddConfigurer(c)
	})
}
