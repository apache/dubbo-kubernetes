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
	"strings"
)

import (
	"github.com/pkg/errors"
)

import (
	"github.com/apache/dubbo-kubernetes/api/mesh/v1alpha1"
	"github.com/apache/dubbo-kubernetes/pkg/core/logger"
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

func trafficPolicyToSelector(dataplane *core_mesh.DataplaneResource, meshContext xds_context.MeshContext) core_xds.ServiceSelectorMap {
	res := core_xds.ServiceSelectorMap{}
	// first step focus on outbound
	tagRouteList := meshContext.Resources.ListOrEmpty(core_mesh.TagRouteType)
	DynamicConfigList := meshContext.Resources.ListOrEmpty(core_mesh.DynamicConfigType)
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

	getDynamicConfigPolicy := func(dubboApplicationName, dubboServiceName string) (
		dubboAppWideConfig *v1alpha1.DynamicConfig, dubboServiceWideConfig *v1alpha1.DynamicConfig,
	) {
		for _, resource := range DynamicConfigList.GetItems() {
			resource, ok := resource.GetSpec().(*v1alpha1.DynamicConfig)
			if !ok {
				logger.Errorf("get wroug type in meshcontext.list() ,somewhere error")
				continue
			}
			if dubboAppWideConfig == nil &&
				resource.Key == dubboApplicationName &&
				strings.ToLower(resource.Scope) == "application" {
				dubboAppWideConfig = resource
			}
			if dubboServiceWideConfig == nil &&
				resource.Key == dubboServiceName &&
				strings.ToLower(resource.Scope) == "service" {
				dubboServiceWideConfig = resource
			}
		}
		return
	}

	// match from appName and port
	list := meshContext.Resources.ListOrEmpty(core_mesh.MetaDataType)
	getAppNameFromEndpoint := func(endpoint core_xds.Endpoint) *v1alpha1.ServiceInfo {
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

	ListServiceInfoFromServiceName := func(serviceName string) []struct {
		App  string
		Info *v1alpha1.ServiceInfo
	} {
		if list == nil || len(list.GetItems()) == 0 {
			return nil
		}
		res := make([]struct {
			App  string
			Info *v1alpha1.ServiceInfo
		}, 0)
		for _, resource := range list.GetItems() {
			resource, ok := resource.GetSpec().(*v1alpha1.MetaData)
			if !ok {
				continue
			}
			for _, info := range resource.Services {
				if info.Name == serviceName {
					res = append(res, struct {
						App  string
						Info *v1alpha1.ServiceInfo
					}{App: resource.App, Info: info})
				}
			}
		}
		return res
	}

	for _, outbound := range dataplane.Spec.Networking.Outbound {
		if p := getTagRoutePolicy(outbound.Tags[v1alpha1.AppTag]); p != nil {
			serviceName := outbound.GetService()
			if serviceName == "" {
				continue
			}
			generateTagSelector(serviceName, res, ListServiceInfoFromServiceName, getServiceInfoFromEndpoint, getTagRoutePolicy)
		}
	}

	for _, outbound := range dataplane.Spec.Networking.Outbound {
		dubboServiceName := outbound.GetService()
		ap, sp := getDynamicConfigPolicy(outbound.Tags[v1alpha1.AppTag], dubboServiceName)
		generateDynamicConfigToSelector(ap, sp, dubboServiceName, res, getServiceInfoFromEndpoint)
	}
	return res
}

func generateDynamicConfigToSelector(
	dubboApplicationPolicy *v1alpha1.DynamicConfig,
	dubboServicePolicy *v1alpha1.DynamicConfig,
	serviceName string,
	selectorMap core_xds.ServiceSelectorMap,
	getMetadata func(endpoint core_xds.Endpoint) *v1alpha1.ServiceInfo,
) {

}

func generateTagSelector(
	serviceName string,
	selectorMap core_xds.ServiceSelectorMap,
	getServiceInfoFromName func(serviceName string) []struct {
		App  string
		Info *v1alpha1.ServiceInfo
	},
	getMetadataFromEndpoint func(endpoint core_xds.Endpoint) *v1alpha1.ServiceInfo,
	GetPolicyFunc func(dubboApplicationName string) *v1alpha1.TagRoute,
) {
	infolist := getServiceInfoFromName(serviceName)
	for _, s := range infolist {

		cs := make([]core_xds.ClusterSelectorList, 0, len(p.Tags))
		for _, tag := range p.Tags {
			cs = append(cs, core_xds.ClusterSelectorList{
				MatchInfo: core_xds.TrafficRouteHttpMatch{
					Params: map[string]core_xds.TrafficRouteHttpStringMatcher{
						"dubbo.tag": core_xds.NewTrafficRouteHttpMatcherExact(tag.Name),
					},
				},
				EndSelectors: []core_xds.ClusterSelector{{
					ConfigInfo: core_xds.TrafficRouteConfig{
						ExternalTags: map[string]string{
							"dubbo.tag": tag.Name,
						},
						Weight: 100,
					},
					SelectFunc: func(endpoint core_xds.EndpointList) core_xds.EndpointList {
						res := core_xds.EndpointList{}
						for _, e := range endpoint {
							// get metadata belong to endpoint
							m := getMetadataFromEndpoint(e)
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
				}},
			})
		}
	}

	if i, ok := selectorMap[serviceName]; ok {
		selectorMap[serviceName] = core_xds.MergeClusterSelectorList(i, cs)
	} else {
		selectorMap[serviceName] = cs
	}
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
