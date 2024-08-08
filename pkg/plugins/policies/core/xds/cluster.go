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

package xds

import (
	"errors"
	"fmt"
	"github.com/apache/dubbo-kubernetes/pkg/xds/envoy"
)

import (
	"github.com/apache/dubbo-kubernetes/pkg/xds/envoy/tags"
)

type Cluster struct {
	service           string
	name              string
	tags              tags.Tags
	mesh              string
	isExternalService bool
	selector          []envoy.EndpointSelector
}

func (c *Cluster) Service() string                    { return c.service }
func (c *Cluster) Name() string                       { return c.name }
func (c *Cluster) Tags() tags.Tags                    { return c.tags }
func (c *Cluster) Selector() []envoy.EndpointSelector { return c.selector }

// Mesh returns a non-empty string only if the cluster is in a different mesh
// from the context.
func (c *Cluster) Mesh() string            { return c.mesh }
func (c *Cluster) IsExternalService() bool { return c.isExternalService }
func (c *Cluster) Hash() string            { return fmt.Sprintf("%s-%s", c.name, c.tags.String()) }

type NewClusterOpt interface {
	apply(cluster *Cluster)
}

type newClusterOptFunc func(cluster *Cluster)

func (f newClusterOptFunc) apply(cluster *Cluster) {
	f(cluster)
}

type ClusterBuilder struct {
	opts []NewClusterOpt
}

func NewClusterBuilder() *ClusterBuilder {
	return &ClusterBuilder{}
}

func (b *ClusterBuilder) Build() *Cluster {
	c := &Cluster{}
	for _, opt := range b.opts {
		opt.apply(c)
	}
	if err := c.validate(); err != nil {
		panic(err)
	}
	return c
}

func (b *ClusterBuilder) WithService(service string) *ClusterBuilder {
	b.opts = append(b.opts, newClusterOptFunc(func(cluster *Cluster) {
		cluster.service = service
		if len(cluster.name) == 0 {
			cluster.name = service
		}
	}))
	return b
}

func (b *ClusterBuilder) WithName(name string) *ClusterBuilder {
	b.opts = append(b.opts, newClusterOptFunc(func(cluster *Cluster) {
		cluster.name = name
		if len(cluster.service) == 0 {
			cluster.service = name
		}
	}))
	return b
}

func (b *ClusterBuilder) WithMesh(mesh string) *ClusterBuilder {
	b.opts = append(b.opts, newClusterOptFunc(func(cluster *Cluster) {
		cluster.mesh = mesh
	}))
	return b
}

func (b *ClusterBuilder) WithTags(tags tags.Tags) *ClusterBuilder {
	b.opts = append(b.opts, newClusterOptFunc(func(cluster *Cluster) {
		cluster.tags = tags
	}))
	return b
}

func (b *ClusterBuilder) WithExternalService(isExternalService bool) *ClusterBuilder {
	b.opts = append(b.opts, newClusterOptFunc(func(cluster *Cluster) {
		cluster.isExternalService = isExternalService
	}))
	return b
}

func (c *Cluster) validate() error {
	if c.service == "" || c.name == "" {
		return errors.New("either WithService() or WithName() should be called")
	}
	return nil
}

func (b *ClusterBuilder) WithSelectors(selector ...envoy.EndpointSelector) *ClusterBuilder {
	b.opts = append(b.opts, newClusterOptFunc(func(cluster *Cluster) {
		if cluster.selector == nil {
			cluster.selector = make([]envoy.EndpointSelector, 0)
		}
		cluster.selector = append(cluster.selector, selector...)
	}))
	return b
}
