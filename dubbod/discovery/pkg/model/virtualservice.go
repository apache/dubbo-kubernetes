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
		Hosts:    append([]string(nil), ms.Hosts...),
		ExportTo: append([]string(nil), ms.VisibleTo...),
	}
	for _, route := range ms.Routes {
		if route == nil || len(route.Service) == 0 {
			continue
		}
		httpRoute := &networking.HTTPRoute{}
		for _, svc := range route.Service {
			if svc == nil {
				continue
			}
			targetHost := svc.Host
			if targetHost == "" && len(ms.Hosts) > 0 {
				targetHost = ms.Hosts[0]
			}
			httpRoute.Route = append(httpRoute.Route, &networking.HTTPRouteDestination{
				Destination: &networking.Destination{
					Host:   targetHost,
					Subset: svc.Name,
				},
				Weight: svc.Weight,
			})
		}
		if len(httpRoute.Route) > 0 {
			vs.Http = append(vs.Http, httpRoute)
		}
	}
	r.Spec = vs
	return resolveVirtualServiceShortnames(r)
}

func meshServiceToDestinationRuleConfig(msConfig config.Config) config.Config {
	r := msConfig.DeepCopy()
	ms, ok := r.Spec.(*networking.MeshService)
	if !ok || ms == nil {
		return r
	}
	dr := &networking.DestinationRule{
		Host:          firstMeshServiceHost(ms, r.Meta),
		TrafficPolicy: ms.TrafficPolicy,
		ExportTo:      append([]string(nil), ms.VisibleTo...),
	}
	seen := setsString{}
	for _, route := range ms.Routes {
		if route == nil {
			continue
		}
		for _, svc := range route.Service {
			if svc == nil || svc.Name == "" || seen.Has(svc.Name) {
				continue
			}
			seen.Insert(svc.Name)
			dr.Subsets = append(dr.Subsets, &networking.Subset{
				Name:          svc.Name,
				Labels:        cloneStringMap(svc.Labels),
				TrafficPolicy: svc.TrafficPolicy,
			})
		}
	}
	r.Spec = dr
	return r
}

func firstMeshServiceHost(ms *networking.MeshService, meta config.Meta) string {
	for _, h := range ms.Hosts {
		if h != "" {
			return string(ResolveShortnameToFQDN(h, meta))
		}
	}
	for _, route := range ms.Routes {
		if route == nil {
			continue
		}
		for _, svc := range route.Service {
			if svc != nil && svc.Host != "" {
				return string(ResolveShortnameToFQDN(svc.Host, meta))
			}
		}
	}
	return ""
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

type setsString map[string]struct{}

func (s setsString) Has(v string) bool {
	_, ok := s[v]
	return ok
}

func (s setsString) Insert(v string) {
	s[v] = struct{}{}
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
