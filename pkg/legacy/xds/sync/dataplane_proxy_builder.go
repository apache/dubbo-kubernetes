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

package sync

import (
	"context"
)

import (
	"github.com/pkg/errors"
)

import (
	core_plugins "github.com/apache/dubbo-kubernetes/pkg/core/plugins"
	core_mesh "github.com/apache/dubbo-kubernetes/pkg/core/resources/apis/mesh"
	core_model "github.com/apache/dubbo-kubernetes/pkg/core/resources/model"
	core_store "github.com/apache/dubbo-kubernetes/pkg/core/resources/store"
	core_xds "github.com/apache/dubbo-kubernetes/pkg/core/xds"
	"github.com/apache/dubbo-kubernetes/pkg/plugins/policies/core/ordered"
	xds_context "github.com/apache/dubbo-kubernetes/pkg/xds/context"
)

type DataplaneProxyBuilder struct {
	Zone       string
	APIVersion core_xds.APIVersion
}

func (p *DataplaneProxyBuilder) Build(ctx context.Context, key core_model.ResourceKey, meshContext xds_context.MeshContext) (*core_xds.Proxy, error) {
	dp, found := meshContext.DataplanesByName[key.Name]
	if !found {
		return nil, core_store.ErrorResourceNotFound(core_mesh.DataplaneType, key.Name, key.Mesh)
	}

	routing := p.resolveRouting(ctx, meshContext, dp)

	matchedPolicies, err := p.matchPolicies(meshContext, dp, nil)
	if err != nil {
		return nil, errors.Wrap(err, "could not match policies")
	}
	proxy := &core_xds.Proxy{
		Id:         core_xds.FromResourceKey(key),
		APIVersion: p.APIVersion,
		Policies:   *matchedPolicies,
		Dataplane:  dp,
		Routing:    *routing,
		Zone:       p.Zone,
	}
	return proxy, nil
}

func (p *DataplaneProxyBuilder) resolveRouting(
	ctx context.Context,
	meshContext xds_context.MeshContext,
	dataplane *core_mesh.DataplaneResource,
) *core_xds.Routing {
	// external services may not necessarily be in the same mesh
	endpointMap := core_xds.EndpointMap{}

	routing := &core_xds.Routing{
		OutboundTargets:                meshContext.EndpointMap,
		ExternalServiceOutboundTargets: endpointMap,
	}

	return routing
}

func (p *DataplaneProxyBuilder) matchPolicies(meshContext xds_context.MeshContext, dataplane *core_mesh.DataplaneResource, outboundSelectors core_xds.DestinationMap) (*core_xds.MatchedPolicies, error) {
	resources := meshContext.Resources
	matchedPolicies := &core_xds.MatchedPolicies{
		Dynamic: core_xds.PluginOriginatedPolicies{},
	}
	for _, p := range core_plugins.Plugins().PolicyPlugins(ordered.Policies) {
		res, err := p.Plugin.MatchedPolicies(dataplane, resources)
		if err != nil {
			return nil, errors.Wrapf(err, "could not apply policy plugin %s", p.Name)
		}
		if res.Type == "" {
			return nil, errors.Wrapf(err, "matched policy didn't set type for policy plugin %s", p.Name)
		}
		matchedPolicies.Dynamic[res.Type] = res
	}
	return matchedPolicies, nil
}
