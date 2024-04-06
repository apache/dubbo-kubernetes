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

package generator

import (
	"context"
)

import (
	"github.com/pkg/errors"
)

import (
	"github.com/apache/dubbo-kubernetes/pkg/core/plugins"
	"github.com/apache/dubbo-kubernetes/pkg/core/xds"
	"github.com/apache/dubbo-kubernetes/pkg/plugins/policies/core/ordered"
	xds_context "github.com/apache/dubbo-kubernetes/pkg/xds/context"
	generator_core "github.com/apache/dubbo-kubernetes/pkg/xds/generator/core"
)

func NewGenerator() generator_core.ResourceGenerator {
	return generator{}
}

type generator struct{}

func (g generator) Generator(ctx context.Context, rs *xds.ResourceSet, xdsCtx xds_context.Context, proxy *xds.Proxy) (*xds.ResourceSet, error) {
	for _, policy := range plugins.Plugins().PolicyPlugins(ordered.Policies) {
		if err := policy.Plugin.Apply(rs, xdsCtx, proxy); err != nil {
			return nil, errors.Wrapf(err, "could not apply policy plugin %s", policy.Name)
		}
	}
	return rs, nil
}
