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
	envoy_hcm "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/network/http_connection_manager/v3"

	"github.com/pkg/errors"

	"google.golang.org/protobuf/types/known/anypb"
)

import (
	core_xds "github.com/apache/dubbo-kubernetes/pkg/core/xds"
	"github.com/apache/dubbo-kubernetes/pkg/xds/envoy"
	v3 "github.com/apache/dubbo-kubernetes/pkg/xds/envoy/listeners/v3"
)

// FilterChainBuilderOpt is a configuration option for FilterChainBuilder.
//
// The goal of FilterChainBuilderOpt is to facilitate fluent FilterChainBuilder API.
type FilterChainBuilderOpt interface {
	// ApplyTo adds FilterChainConfigurer(s) to the FilterChainBuilder.
	ApplyTo(builder *FilterChainBuilder)
}

func NewFilterChainBuilder(apiVersion core_xds.APIVersion, name string) *FilterChainBuilder {
	return &FilterChainBuilder{
		apiVersion: apiVersion,
		name:       name,
	}
}

// FilterChainBuilder is responsible for generating an Envoy filter chain
// by applying a series of FilterChainConfigurers.
type FilterChainBuilder struct {
	apiVersion  core_xds.APIVersion
	configurers []v3.FilterChainConfigurer
	name        string
}

// Configure configures FilterChainBuilder by adding individual FilterChainConfigurers.
func (b *FilterChainBuilder) Configure(opts ...FilterChainBuilderOpt) *FilterChainBuilder {
	for _, opt := range opts {
		opt.ApplyTo(b)
	}

	return b
}

func (b *FilterChainBuilder) ConfigureIf(condition bool, opts ...FilterChainBuilderOpt) *FilterChainBuilder {
	if !condition {
		return b
	}
	for _, opt := range opts {
		opt.ApplyTo(b)
	}

	return b
}

// Build generates an Envoy filter chain by applying a series of FilterChainConfigurers.
func (b *FilterChainBuilder) Build() (envoy.NamedResource, error) {
	switch b.apiVersion {
	case core_xds.APIVersion(envoy.APIV3):
		filterChain := envoy_listener_v3.FilterChain{
			Name: b.name,
		}

		for _, configurer := range b.configurers {
			if err := configurer.Configure(&filterChain); err != nil {
				return nil, err
			}
		}

		// Ensure there is always an HTTP router terminating the filter chain.
		_ = v3.UpdateHTTPConnectionManager(&filterChain, func(hcm *envoy_hcm.HttpConnectionManager) error {
			for _, filter := range hcm.HttpFilters {
				if filter.Name == "envoy.filters.http.router" {
					return nil
				}
			}
			router := &envoy_hcm.HttpFilter{
				Name: "envoy.filters.http.router",
				ConfigType: &envoy_hcm.HttpFilter_TypedConfig{
					TypedConfig: &anypb.Any{
						TypeUrl: "type.googleapis.com/envoy.extensions.filters.http.router.v3.Router",
					},
				},
			}
			hcm.HttpFilters = append(hcm.HttpFilters, router)
			return nil
		})

		return &filterChain, nil

	default:
		return nil, errors.New("unknown API")
	}
}

// AddConfigurer appends a given FilterChainConfigurer to the end of the chain.
func (b *FilterChainBuilder) AddConfigurer(configurer v3.FilterChainConfigurer) {
	b.configurers = append(b.configurers, configurer)
}

// FilterChainBuilderOptFunc is a convenience type adapter.
type FilterChainBuilderOptFunc func(builder *FilterChainBuilder)

func (f FilterChainBuilderOptFunc) ApplyTo(builder *FilterChainBuilder) {
	if f != nil {
		f(builder)
	}
}

// AddFilterChainConfigurer produces an option that applies the given
// configurer to the filter chain.
func AddFilterChainConfigurer(c v3.FilterChainConfigurer) FilterChainBuilderOpt {
	return FilterChainBuilderOptFunc(func(builder *FilterChainBuilder) {
		builder.AddConfigurer(c)
	})
}
