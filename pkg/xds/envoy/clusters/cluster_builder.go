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

package clusters

import (
	core_xds "github.com/apache/dubbo-kubernetes/pkg/core/xds"
	"github.com/pkg/errors"

	envoy_api "github.com/envoyproxy/go-control-plane/envoy/config/cluster/v3"

	"github.com/apache/dubbo-kubernetes/pkg/xds/envoy"
	v3 "github.com/apache/dubbo-kubernetes/pkg/xds/envoy/clusters/v3"
)

// ClusterBuilderOpt is a configuration option for ClusterBuilder.
//
// The goal of ClusterBuilderOpt is to facilitate fluent ClusterBuilder API.
type ClusterBuilderOpt interface {
	// ApplyTo adds ClusterConfigurer(s) to the ClusterBuilder.
	ApplyTo(builder *ClusterBuilder)
}

func NewClusterBuilder(apiVersion core_xds.APIVersion, name string) *ClusterBuilder {
	return &ClusterBuilder{
		apiVersion: apiVersion,
		name:       name,
	}
}

// ClusterBuilder is responsible for generating an Envoy cluster
// by applying a series of ClusterConfigurers.
type ClusterBuilder struct {
	apiVersion core_xds.APIVersion
	// A series of ClusterConfigurers to apply to Envoy cluster.
	configurers []v3.ClusterConfigurer
	name        string
}

// Configure configures ClusterBuilder by adding individual ClusterConfigurers.
func (b *ClusterBuilder) Configure(opts ...ClusterBuilderOpt) *ClusterBuilder {
	for _, opt := range opts {
		opt.ApplyTo(b)
	}
	return b
}

// Build generates an Envoy cluster by applying a series of ClusterConfigurers.
func (b *ClusterBuilder) Build() (envoy.NamedResource, error) {
	switch b.apiVersion {
	case core_xds.APIVersion(envoy.APIV3):
		cluster := envoy_api.Cluster{
			Name: b.name,
		}
		for _, configurer := range b.configurers {
			if err := configurer.Configure(&cluster); err != nil {
				return nil, err
			}
		}
		if len(cluster.GetName()) == 0 {
			return nil, errors.New("cluster name is undefined")
		}
		return &cluster, nil
	default:
		return nil, errors.New("unknown API")
	}
}

func (b *ClusterBuilder) MustBuild() envoy.NamedResource {
	cluster, err := b.Build()
	if err != nil {
		panic(errors.Wrap(err, "failed to build Envoy Cluster").Error())
	}

	return cluster
}

// AddConfigurer appends a given ClusterConfigurer to the end of the chain.
func (b *ClusterBuilder) AddConfigurer(configurer v3.ClusterConfigurer) {
	b.configurers = append(b.configurers, configurer)
}

// ClusterBuilderOptFunc is a convenience type adapter.
type ClusterBuilderOptFunc func(config *ClusterBuilder)

func (f ClusterBuilderOptFunc) ApplyTo(builder *ClusterBuilder) {
	f(builder)
}
