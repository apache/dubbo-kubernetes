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

	"golang.org/x/exp/maps"
)

import (
	mesh_proto "github.com/apache/dubbo-kubernetes/api/mesh/v1alpha1"
	"github.com/apache/dubbo-kubernetes/pkg/core/logger"
	core_plugins "github.com/apache/dubbo-kubernetes/pkg/core/plugins"
	core_mesh "github.com/apache/dubbo-kubernetes/pkg/core/resources/apis/mesh"
	core_model "github.com/apache/dubbo-kubernetes/pkg/core/resources/model"
	core_store "github.com/apache/dubbo-kubernetes/pkg/core/resources/store"
	core_xds "github.com/apache/dubbo-kubernetes/pkg/core/xds"
	"github.com/apache/dubbo-kubernetes/pkg/plugins/policies/core/ordered"
	"github.com/apache/dubbo-kubernetes/pkg/plugins/runtime/k8s/util"
	"github.com/apache/dubbo-kubernetes/pkg/util/sets"
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
	vsl := meshContext.Resources.ListOrEmpty(core_mesh.VirtualServiceType)
	if vsl == nil || len(vsl.GetItems()) == 0 {
		return res
	}
	getVs := func(serviceName string) *mesh_proto.VirtualService {
		for _, item := range vsl.GetItems() {
			item, ok := item.GetSpec().(*mesh_proto.VirtualService)
			if !ok {
				logger.Errorf("meshContext build error, item need to be *api.VirtualService,but got %T", item)
				continue
			}
			for _, host := range item.GetHosts() {
				if getNamesFromHost(host).Contains(serviceName) {
					return item
				}
			}
		}
		return nil
	}
	dnl := meshContext.Resources.ListOrEmpty(core_mesh.DestinationRuleType)
	// k8s-ServiceName to Selectors
	buildSet := map[string][]core_xds.ClusterSelectorList{}
	for _, outbound := range dataplane.Spec.Networking.Outbound {
		serviceName := outbound.GetService()
		name, _, err := util.ParseServiceTag(serviceName)
		if err != nil {
			logger.Errorf("meshContext build error, service_tag parse error, service_name=%s, err=%v", serviceName, err)
			continue
		}
		vs := getVs(name.Name)
		if vs == nil {
			continue
		}
		if buildSet[name.Name] != nil {
			res[serviceName] = buildSet[name.Name]
		} else {
			i := generateVirtualService(vs, func(host string) *mesh_proto.DestinationRule {
				for _, item := range dnl.GetItems() {
					if item, ok := item.GetSpec().(*mesh_proto.DestinationRule); !ok {
						logger.Errorf("meshContext build error, item need to be *api.VirtualService,but got %T", item)
						continue
					} else if item.Host == host {
						return item
					}
				}
				return nil
			}, func(host, subset string) *mesh_proto.Subset {
				for _, item := range dnl.GetItems() {
					if item, ok := item.GetSpec().(*mesh_proto.DestinationRule); !ok {
						logger.Errorf("meshContext build error, item need to be *api.VirtualService,but got %T", item)
						continue
					} else if item.Host == host {
						for _, s := range item.Subsets {
							if s.Name == subset {
								return s
							}
						}
					}
				}
				return nil
			}, dataplane, meshContext)
			res[serviceName] = i
			buildSet[name.Name] = i
		}
	}

	return res
}

func generateVirtualService(vs *mesh_proto.VirtualService, getDestinationRuleFunc func(host string) *mesh_proto.DestinationRule, getDestinationSubsetFunc func(host string, subset string) *mesh_proto.Subset, dataplane *core_mesh.DataplaneResource, meshContext xds_context.MeshContext) []core_xds.ClusterSelectorList {
	res := make([]core_xds.ClusterSelectorList, 0, len(vs.Http))
	for _, route := range vs.Http {
		csl := core_xds.ClusterSelectorList{
			ModifyInfo:   mesh_proto.TrafficRoute_Http_Modify{},
			EndSelectors: make([]core_xds.ClusterSelector, 0, len(route.Route)),
		}
		// not support redirect, directResponse , mirror
		csl.ModifyInfo.RequestHeaders, csl.ModifyInfo.ResponseHeaders = handleResquestResponeHeaderOption(route.Headers)
		if route.Timeout != nil {
			csl.ModifyInfo.TimeOut = route.Timeout
		}
		if route.Rewrite != nil {
			if route.Rewrite.UriRegexRewrite == nil {
				csl.ModifyInfo.Path = &mesh_proto.TrafficRoute_Http_Modify_Path{
					Type: &mesh_proto.TrafficRoute_Http_Modify_Path_Regex{
						Regex: &mesh_proto.TrafficRoute_Http_Modify_RegexReplace{
							Pattern:      route.Rewrite.UriRegexRewrite.Match,
							Substitution: route.Rewrite.UriRegexRewrite.Rewrite,
						},
					},
				}
			} else if route.Rewrite.Uri != "" {
				csl.ModifyInfo.Path = &mesh_proto.TrafficRoute_Http_Modify_Path{
					Type: &mesh_proto.TrafficRoute_Http_Modify_Path_Rewrite{
						Rewrite: route.Rewrite.Uri,
					},
				}
			}
		}
		for _, destination := range route.Route {
			cs := core_xds.ClusterSelector{
				ConfigInfo: core_xds.TrafficRouteConfig{
					RequestHeaders:    nil,
					ResponseHeaders:   nil,
					DestinationSubSet: nil,
					Policy:            nil,
					Weight:            0,
				},
				TagSelect: mesh_proto.TagSelector{},
			}
			if destination.Headers != nil {
				cs.ConfigInfo.RequestHeaders, cs.ConfigInfo.ResponseHeaders = handleResquestResponeHeaderOption(destination.Headers)
			}
			cs.ConfigInfo.Weight = destination.Weight
			cs.ConfigInfo.DestinationSubSet = destination.Destination

			ds := getDestinationSubsetFunc(destination.Destination.Host, destination.Destination.Subset)
			if ds == nil {
				continue
			}

			if ds.TrafficPolicy != nil {
				cs.ConfigInfo.Policy = ds.TrafficPolicy
			} else {
				cs.ConfigInfo.Policy = getDestinationRuleFunc(destination.Destination.Host).TrafficPolicy
			}
			cs.TagSelect = maps.Clone(ds.Labels)

			csl.EndSelectors = append(csl.EndSelectors, cs)
		}
		for _, match := range route.Match {
			csl := core_xds.ClusterSelectorList{
				MatchInfo:    mesh_proto.TrafficRoute_Http_Match{},
				ModifyInfo:   csl.ModifyInfo,
				EndSelectors: csl.EndSelectors,
			}
			if match.Uri != nil {
				csl.MatchInfo.Path = match.Uri
			}
			if match.Method != nil {
				csl.MatchInfo.Method = match.Method
			}
			if match.Headers != nil && len(match.Headers) != 0 {
				csl.MatchInfo.Headers = maps.Clone(match.Headers)
			}
			if match.QueryParams != nil && len(match.QueryParams) != 0 {
				csl.MatchInfo.Params = maps.Clone(match.QueryParams)
			}
			res = append(res, csl)
		}
	}
	return res
}

func handleResquestResponeHeaderOption(header *mesh_proto.Headers) (request *mesh_proto.TrafficRoute_Http_Modify_Headers, response *mesh_proto.TrafficRoute_Http_Modify_Headers) {
	if header != nil {
		if header.Request != nil {
			// unsupported header.Request.Set
			request = &mesh_proto.TrafficRoute_Http_Modify_Headers{}
			for k, v := range header.Request.Add {
				request.Add = append(request.Add, &mesh_proto.TrafficRoute_Http_Modify_Headers_Add{
					Name:  k,
					Value: v,
				})
			}
			for _, v := range header.Request.Remove {
				request.Remove = append(request.Remove, &mesh_proto.TrafficRoute_Http_Modify_Headers_Remove{
					Name: v,
				})
			}
		}
		if header.Response != nil {
			// unsupported header.Response.Set
			response = &mesh_proto.TrafficRoute_Http_Modify_Headers{}
			for k, v := range header.Response.Add {
				response.Add = append(response.Add, &mesh_proto.TrafficRoute_Http_Modify_Headers_Add{
					Name:  k,
					Value: v,
				})
			}
			for _, v := range header.Response.Remove {
				response.Remove = append(response.Remove, &mesh_proto.TrafficRoute_Http_Modify_Headers_Remove{
					Name: v,
				})
			}
		}
	}
	return request, response
}

//
//func handleResquestResponeHeaderOption(cs core_xds.ClusterSelectorList, route *mesh_proto.HTTPRoute) {
//
//}

func getNamesFromHost(host string) sets.String {
	// todo: match more possible host name,
	return map[string]struct{}{host: {}}
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
