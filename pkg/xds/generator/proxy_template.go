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
	"fmt"
)

import (
	model "github.com/apache/dubbo-kubernetes/pkg/core/xds"
	"github.com/apache/dubbo-kubernetes/pkg/plugins/policies/core/generator"
	xds_context "github.com/apache/dubbo-kubernetes/pkg/xds/context"
	"github.com/apache/dubbo-kubernetes/pkg/xds/generator/core"
)

const (
	DefaultProxy = "default-proxy"
	IngressProxy = "ingress-proxy"
)

type ProxyTemplateGenerator struct {
	ProfileName []string
}

func (g *ProxyTemplateGenerator) Generate(ctx context.Context, xdsCtx xds_context.Context, proxy *model.Proxy) (*model.ResourceSet, error) {
	resources := model.NewResourceSet()

	for _, name := range g.ProfileName {
		p, ok := predefinedProfiles[name]
		if !ok {
			return nil, fmt.Errorf("profile{name=%q}: unknown profile", g.ProfileName)
		}
		if rs, err := p.Generator(ctx, resources, xdsCtx, proxy); err != nil {
			return nil, err
		} else {
			resources.AddSet(rs)
		}
	}

	return resources, nil
}

func NewDefaultProxyProfile() core.ResourceGenerator {
	return core.CompositeResourceGenerator{
		InboundProxyGenerator{},
		OutboundProxyGenerator{},
		generator.NewGenerator(),
	}
}

var predefinedProfiles = make(map[string]core.ResourceGenerator)

func init() {
	RegisterProfile(DefaultProxy, NewDefaultProxyProfile())
	RegisterProfile(IngressProxy, core.CompositeResourceGenerator{IngressGenerator{}})
}

func RegisterProfile(profileName string, generator core.ResourceGenerator) {
	predefinedProfiles[profileName] = generator
}
