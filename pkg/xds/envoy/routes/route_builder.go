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

package routes

import (
	envoy_config_route_v3 "github.com/envoyproxy/go-control-plane/envoy/config/route/v3"

	"google.golang.org/protobuf/types/known/anypb"
)

import (
	core_xds "github.com/apache/dubbo-kubernetes/pkg/core/xds"
	"github.com/apache/dubbo-kubernetes/pkg/xds/envoy"
)

type RouteConfigurer interface {
	Configure(*envoy_config_route_v3.Route) error
}

type RouteBuilder struct {
	apiVersion  core_xds.APIVersion
	configurers []RouteConfigurer
	name        string
}

func NewRouteBuilder(apiVersion core_xds.APIVersion, name string) *RouteBuilder {
	return &RouteBuilder{
		apiVersion: apiVersion,
		name:       name,
	}
}

func (r *RouteBuilder) Configure(opts ...RouteConfigurer) *RouteBuilder {
	r.configurers = append(r.configurers, opts...)
	return r
}

func (r *RouteBuilder) Build() (envoy.NamedResource, error) {
	route := &envoy_config_route_v3.Route{
		Match:                &envoy_config_route_v3.RouteMatch{},
		Name:                 r.name,
		TypedPerFilterConfig: map[string]*anypb.Any{},
	}

	for _, c := range r.configurers {
		if err := c.Configure(route); err != nil {
			return nil, err
		}
	}

	return route, nil
}

type RouteConfigureFunc func(*envoy_config_route_v3.Route) error

func (f RouteConfigureFunc) Configure(r *envoy_config_route_v3.Route) error {
	if f != nil {
		return f(r)
	}

	return nil
}

type RouteMustConfigureFunc func(*envoy_config_route_v3.Route)

func (f RouteMustConfigureFunc) Configure(r *envoy_config_route_v3.Route) error {
	if f != nil {
		f(r)
	}

	return nil
}
