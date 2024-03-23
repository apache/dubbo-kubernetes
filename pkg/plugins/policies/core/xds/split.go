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
	"github.com/apache/dubbo-kubernetes/pkg/xds/envoy/tags"
)

type Split struct {
	clusterName string
	weight      uint32
	lbMetadata  tags.Tags

	hasExternalService bool
}

func (s *Split) ClusterName() string      { return s.clusterName }
func (s *Split) Weight() uint32           { return s.weight }
func (s *Split) LBMetadata() tags.Tags    { return s.lbMetadata }
func (s *Split) HasExternalService() bool { return s.hasExternalService }

type NewSplitOpt interface {
	apply(s *Split)
}

type newSplitOptFunc func(s *Split)

func (f newSplitOptFunc) apply(s *Split) {
	f(s)
}

type SplitBuilder struct {
	opts []NewSplitOpt
}

func NewSplitBuilder() *SplitBuilder {
	return &SplitBuilder{}
}

func (b *SplitBuilder) Build() *Split {
	s := &Split{}
	for _, opt := range b.opts {
		opt.apply(s)
	}
	return s
}

func (b *SplitBuilder) WithClusterName(clusterName string) *SplitBuilder {
	b.opts = append(b.opts, newSplitOptFunc(func(s *Split) {
		s.clusterName = clusterName
	}))
	return b
}

func (b *SplitBuilder) WithWeight(weight uint32) *SplitBuilder {
	b.opts = append(b.opts, newSplitOptFunc(func(s *Split) {
		s.weight = weight
	}))
	return b
}

func (b *SplitBuilder) WithExternalService(hasExternalService bool) *SplitBuilder {
	b.opts = append(b.opts, newSplitOptFunc(func(s *Split) {
		s.hasExternalService = hasExternalService
	}))
	return b
}
