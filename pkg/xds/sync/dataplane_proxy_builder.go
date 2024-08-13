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
	"github.com/apache/dubbo-kubernetes/api/mesh/v1alpha1"
	"github.com/apache/dubbo-kubernetes/pkg/core/logger"
	envoy_common "github.com/apache/dubbo-kubernetes/pkg/xds/envoy"
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
		OutboundSelector:               trafficPolicyToSelector(dataplane, meshContext),
		OutboundTargets:                meshContext.EndpointMap,
		ExternalServiceOutboundTargets: endpointMap,
	}

	return routing
}

func trafficPolicyToSelector(dataplane *core_mesh.DataplaneResource, meshContext xds_context.MeshContext) core_xds.EndpointSelectorMap {
	res := core_xds.EndpointSelectorMap{}
	// first step focus on outbound
	tagRouteList := meshContext.Resources.ListOrEmpty(core_mesh.TagRouteType)
	getTagRoutePolicy := func(dubboApplicationName string) *v1alpha1.TagRoute {
		for _, resource := range tagRouteList.GetItems() {
			resource, ok := resource.GetSpec().(*v1alpha1.TagRoute)
			if !ok {
				logger.Errorf("get wroug type in meshcontext.list() ,somewhere error")
				continue
			}
			if resource.Key == dubboApplicationName {
				return resource
			}
		}
		return nil
	}
	dubboApplicationName := dataplane.Spec.GetApplication()
	if p := getTagRoutePolicy(dubboApplicationName); p != nil {
		for _, outbound := range dataplane.Spec.Networking.Outbound {
			serviceName := outbound.GetService()
			if serviceName == "" || res[serviceName] != nil {
				continue
			}
			res[serviceName] = generateTagSelector(p, meshContext)
		}
	}
	return res
}

func generateTagSelector(p *v1alpha1.TagRoute, meshContext xds_context.MeshContext) []envoy_common.EndpointSelector {
	list := meshContext.Resources.ListOrEmpty(core_mesh.MetaDataType)
	// match from appName and port
	getMetadata := func(endpoint core_xds.Endpoint) *v1alpha1.ServiceInfo {
		if list == nil || len(list.GetItems()) == 0 {
			return nil
		}
		for _, resource := range list.GetItems() {
			resource, ok := resource.GetSpec().(*v1alpha1.MetaData)
			if !ok || resource.App != endpoint.Tags[v1alpha1.AppTag] {
				continue
			}
			for _, info := range resource.Services {
				if info.Port == int64(endpoint.Port) {
					return info
				}
			}
		}
		return nil
	}
	res := make([]envoy_common.EndpointSelector, 0, len(p.Tags))
	for _, tag := range p.Tags {
		res = append(res, envoy_common.EndpointSelector{
			MatchInfo: envoy_common.TrafficRouteHttpMatch{
				Name: tag.Name,
				Params: map[string]envoy_common.TrafficRouteHttpMatchStringMatcher{
					"dubbo.tag": envoy_common.NewTrafficRouteHttpMatchStringMatcherExact(tag.Name),
				},
			},
			SelectFunc: func(endpoint core_xds.EndpointList) core_xds.EndpointList {
				res := core_xds.EndpointList{}
				for _, e := range endpoint {
					m := getMetadata(e)
					if m == nil {
						continue
					}
					matched := true
					for _, match := range tag.Match {
						if !match.Value.Match(m.Params[match.Key]) {
							matched = false
							break
						}
					}
					if matched {
						res = append(res, e)
					}
				}
				return res
			},
		})
	}
	return append(res, envoy_common.EndpointSelector{
		MatchInfo: envoy_common.TrafficRouteHttpMatch{},
		SelectFunc: func(endpoint core_xds.EndpointList) core_xds.EndpointList {
			return endpoint
		},
	})
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
