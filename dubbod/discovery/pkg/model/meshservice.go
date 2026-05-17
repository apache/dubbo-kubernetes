//
// Licensed to the Apache Software Foundation (ASF) under one or more
// contributor license agreements.  See the NOTICE file distributed with
// this work for additional information regarding copyright ownership.
// The ASF licenses this file to You under the Apache License, Version 2.0
// (the "License"); you may not use this file except in compliance with
// the License.  You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package model

import (
	"github.com/apache/dubbo-kubernetes/pkg/config"
	networking "github.com/kdubbo/api/networking/v1alpha3"
)

func meshServiceToVirtualServiceConfig(msConfig config.Config) config.Config {
	r := msConfig.DeepCopy()
	ms, ok := r.Spec.(*networking.MeshService)
	if !ok || ms == nil {
		return r
	}
	vs := &networking.VirtualService{
		Hosts: append([]string(nil), ms.Hosts...),
	}
	for _, rule := range ms.Rules {
		if rule == nil {
			continue
		}
		appendMeshServiceHTTPRoutes(&vs.Http, ms, msConfig.Meta, rule.Route, rule.Match)
		appendMeshServiceHTTPRoutes(&vs.Http, ms, msConfig.Meta, rule.Routes, nil)
	}
	appendMeshServiceHTTPRoutes(&vs.Http, ms, msConfig.Meta, ms.Routes, nil)
	r.Spec = vs
	return resolveVirtualServiceShortnames(r)
}

func meshServiceToDestinationRuleConfigs(msConfig config.Config) []config.Config {
	r := msConfig.DeepCopy()
	ms, ok := r.Spec.(*networking.MeshService)
	if !ok || ms == nil {
		return []config.Config{r}
	}
	rulesByHost := map[string]*networking.DestinationRule{}
	order := []string{}
	ruleForHost := func(hostname string) *networking.DestinationRule {
		if hostname == "" {
			return nil
		}
		resolved := string(ResolveShortnameToFQDN(hostname, r.Meta))
		if dr := rulesByHost[resolved]; dr != nil {
			return dr
		}
		dr := &networking.DestinationRule{
			Host:          resolved,
			TrafficPolicy: ms.TrafficPolicy,
		}
		rulesByHost[resolved] = dr
		order = append(order, resolved)
		return dr
	}

	forEachMeshServiceDestination(ms, func(svc *networking.ServiceDestination) {
		if svc == nil {
			return
		}
		if serviceDestinationUsesLabels(svc) {
			dr := ruleForHost(hostForLabelDestination(ms, svc))
			if dr == nil || svc.Name == "" || hasSubset(dr, svc.Name) {
				return
			}
			dr.Subsets = append(dr.Subsets, &networking.Subset{
				Name:          svc.Name,
				Labels:        cloneStringMap(svc.Labels),
				TrafficPolicy: svc.TrafficPolicy,
			})
			return
		}
		if svc.TrafficPolicy != nil {
			if dr := ruleForHost(destinationHostForService(ms, svc, r.Meta)); dr != nil {
				dr.TrafficPolicy = svc.TrafficPolicy
			}
		} else if ms.TrafficPolicy != nil {
			ruleForHost(destinationHostForService(ms, svc, r.Meta))
		}
	})

	if len(order) == 0 {
		if host := firstMeshServiceHost(ms, r.Meta); host != "" {
			order = append(order, host)
			rulesByHost[host] = &networking.DestinationRule{
				Host:          host,
				TrafficPolicy: ms.TrafficPolicy,
			}
		}
	}

	out := make([]config.Config, 0, len(order))
	for _, hostname := range order {
		dr := rulesByHost[hostname]
		if dr == nil {
			continue
		}
		cfg := msConfig.DeepCopy()
		cfg.Spec = dr
		out = append(out, cfg)
	}
	return out
}

func appendMeshServiceHTTPRoutes(out *[]*networking.HTTPRoute, ms *networking.MeshService, meta config.Meta, routes []*networking.MeshServiceRoute, matches []*networking.HTTPMatchRequest) {
	httpRoute := &networking.HTTPRoute{
		Match: append([]*networking.HTTPMatchRequest(nil), matches...),
	}
	for _, route := range routes {
		if route == nil {
			continue
		}
		for _, svc := range route.Service {
			if svc == nil {
				continue
			}
			httpRoute.Route = append(httpRoute.Route, &networking.HTTPRouteDestination{
				Destination: &networking.Destination{
					Host:   destinationHostForService(ms, svc, meta),
					Subset: destinationSubsetForService(svc),
				},
				Weight: svc.Weight,
			})
		}
	}
	if len(httpRoute.Route) > 0 {
		*out = append(*out, httpRoute)
	}
}

func forEachMeshServiceDestination(ms *networking.MeshService, visit func(*networking.ServiceDestination)) {
	for _, rule := range ms.Rules {
		if rule == nil {
			continue
		}
		forEachMeshServiceRouteDestination(rule.Route, visit)
		forEachMeshServiceRouteDestination(rule.Routes, visit)
	}
	forEachMeshServiceRouteDestination(ms.Routes, visit)
}

func forEachMeshServiceRouteDestination(routes []*networking.MeshServiceRoute, visit func(*networking.ServiceDestination)) {
	for _, route := range routes {
		if route == nil {
			continue
		}
		for _, svc := range route.Service {
			visit(svc)
		}
	}
}

func meshServiceToDestinationRuleConfig(msConfig config.Config) config.Config {
	rules := meshServiceToDestinationRuleConfigs(msConfig)
	if len(rules) == 0 {
		return msConfig.DeepCopy()
	}
	return rules[0]
}

func firstMeshServiceHost(ms *networking.MeshService, meta config.Meta) string {
	for _, h := range ms.Hosts {
		if h != "" {
			return string(ResolveShortnameToFQDN(h, meta))
		}
	}
	host := ""
	forEachMeshServiceDestination(ms, func(svc *networking.ServiceDestination) {
		if host == "" && svc != nil && svc.Host != "" {
			host = string(ResolveShortnameToFQDN(svc.Host, meta))
		}
	})
	if host != "" {
		return host
	}
	return ""
}

func destinationHostForService(ms *networking.MeshService, svc *networking.ServiceDestination, meta config.Meta) string {
	if svc == nil {
		return firstMeshServiceHost(ms, meta)
	}
	if serviceDestinationUsesLabels(svc) {
		return hostForLabelDestination(ms, svc)
	}
	if svc.Name != "" {
		return string(ResolveShortnameToFQDN(svc.Name, meta))
	}
	if svc.Host != "" {
		return string(ResolveShortnameToFQDN(svc.Host, meta))
	}
	return firstMeshServiceHost(ms, meta)
}

func destinationSubsetForService(svc *networking.ServiceDestination) string {
	if serviceDestinationUsesLabels(svc) {
		return svc.Name
	}
	return ""
}

func hostForLabelDestination(ms *networking.MeshService, svc *networking.ServiceDestination) string {
	if svc != nil && svc.Host != "" {
		return svc.Host
	}
	if len(ms.Hosts) > 0 {
		return ms.Hosts[0]
	}
	return ""
}

func serviceDestinationUsesLabels(svc *networking.ServiceDestination) bool {
	return svc != nil && len(svc.Labels) > 0
}

func hasSubset(dr *networking.DestinationRule, name string) bool {
	for _, subset := range dr.Subsets {
		if subset != nil && subset.Name == name {
			return true
		}
	}
	return false
}

func cloneStringMap(in map[string]string) map[string]string {
	if len(in) == 0 {
		return nil
	}
	out := make(map[string]string, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

func resolveVirtualServiceShortnames(config config.Config) config.Config {
	// values returned from ConfigStore.List are immutable.
	// Therefore, we make a copy
	r := config.DeepCopy()
	rule := r.Spec.(*networking.VirtualService)
	meta := r.Meta

	// resolve top level hosts
	for i, h := range rule.Hosts {
		rule.Hosts[i] = string(ResolveShortnameToFQDN(h, meta))
	}

	// resolve host in http route.subset（dubbo destination）
	for _, d := range rule.Http {
		for _, w := range d.Route {
			if w.Destination != nil {
				w.Destination.Host = string(ResolveShortnameToFQDN(w.Destination.Host, meta))
			}
		}
	}

	return r
}
