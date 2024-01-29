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

package virtualhosts

import (
	envoy_config_route_v3 "github.com/envoyproxy/go-control-plane/envoy/config/route/v3"

	"github.com/pkg/errors"
)

import (
	core_xds "github.com/apache/dubbo-kubernetes/pkg/core/xds"
	"github.com/apache/dubbo-kubernetes/pkg/xds/envoy"
)

// VirtualHostConfigurer is responsible for configuring a single aspect of the entire Envoy VirtualHost,
// such as Route, CORS, etc.
type VirtualHostConfigurer interface {
	// Configure configures a single aspect on a given Envoy VirtualHost.
	Configure(virtualHost *envoy_config_route_v3.VirtualHost) error
}

// VirtualHostConfigureFunc adapts a configuration function to the
// VirtualHostConfigurer interface.
type VirtualHostConfigureFunc func(vh *envoy_config_route_v3.VirtualHost) error

func (f VirtualHostConfigureFunc) Configure(vh *envoy_config_route_v3.VirtualHost) error {
	if f != nil {
		return f(vh)
	}

	return nil
}

// VirtualHostMustConfigureFunc adapts a configuration function that
// never fails to the VirtualHostConfigurer interface.
type VirtualHostMustConfigureFunc func(vh *envoy_config_route_v3.VirtualHost)

func (f VirtualHostMustConfigureFunc) Configure(vh *envoy_config_route_v3.VirtualHost) error {
	if f != nil {
		f(vh)
	}

	return nil
}

// VirtualHostBuilderOpt is a configuration option for VirtualHostBuilder.
//
// The goal of VirtualHostBuilderOpt is to facilitate fluent VirtualHostBuilder API.
type VirtualHostBuilderOpt interface {
	// ApplyTo adds VirtualHostConfigurer(s) to the VirtualHostBuilder.
	ApplyTo(builder *VirtualHostBuilder)
}

func NewVirtualHostBuilder(apiVersion core_xds.APIVersion, name string) *VirtualHostBuilder {
	return &VirtualHostBuilder{
		apiVersion: apiVersion,
		name:       name,
	}
}

// VirtualHostBuilder is responsible for generating an Envoy VirtualHost
// by applying a series of VirtualHostConfigurers.
type VirtualHostBuilder struct {
	apiVersion  core_xds.APIVersion
	configurers []VirtualHostConfigurer
	name        string
}

// Configure configures VirtualHostBuilder by adding individual VirtualHostConfigurers.
func (b *VirtualHostBuilder) Configure(opts ...VirtualHostBuilderOpt) *VirtualHostBuilder {
	for _, opt := range opts {
		opt.ApplyTo(b)
	}

	return b
}

// Build generates an Envoy VirtualHost by applying a series of VirtualHostConfigurers.
func (b *VirtualHostBuilder) Build() (envoy.NamedResource, error) {
	switch b.apiVersion {
	case core_xds.APIVersion(envoy.APIV3):
		virtualHost := envoy_config_route_v3.VirtualHost{
			Name:    b.name,
			Domains: []string{"*"},
		}
		for _, configurer := range b.configurers {
			if err := configurer.Configure(&virtualHost); err != nil {
				return nil, err
			}
		}
		if virtualHost.GetName() == "" {
			return nil, errors.New("virtual host name is required, but it was not provided")
		}
		return &virtualHost, nil
	default:
		return nil, errors.New("unknown API")
	}
}

// AddConfigurer appends a given VirtualHostConfigurer to the end of the chain.
func (b *VirtualHostBuilder) AddConfigurer(configurer VirtualHostConfigurer) {
	b.configurers = append(b.configurers, configurer)
}

// VirtualHostBuilderOptFunc is a convenience type adapter.
type VirtualHostBuilderOptFunc func(builder *VirtualHostBuilder)

func (f VirtualHostBuilderOptFunc) ApplyTo(builder *VirtualHostBuilder) {
	if f != nil {
		f(builder)
	}
}

// AddVirtualHostConfigurer production an option that adds the given
// configurer to the virtual host builder.
func AddVirtualHostConfigurer(c VirtualHostConfigurer) VirtualHostBuilderOpt {
	return VirtualHostBuilderOptFunc(func(builder *VirtualHostBuilder) {
		builder.AddConfigurer(c)
	})
}
