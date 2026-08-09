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

package grpcgen

import (
	"fmt"
	"sort"
	"strings"

	"github.com/apache/dubbo-kubernetes/dubbod/discovery/pkg/model"
	"github.com/apache/dubbo-kubernetes/pkg/config"
	networking "github.com/kdubbo/api/networking/v1alpha3"
	route "github.com/kdubbo/xds-api/route/v1"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"
)

const (
	dxgateServiceGroup = "networking.dubbo.apache.org"
	dxgateServiceKind  = "DxgateService"
)

func isDxgateServiceBackend(ref gatewayv1.HTTPBackendRef) bool {
	group := ""
	if ref.Group != nil {
		group = string(*ref.Group)
	}
	kind := "Service"
	if ref.Kind != nil {
		kind = string(*ref.Kind)
	}
	return group == dxgateServiceGroup && kind == dxgateServiceKind
}

func ruleUsesDxgateService(rule gatewayv1.HTTPRouteRule) bool {
	for _, ref := range rule.BackendRefs {
		if isDxgateServiceBackend(ref) {
			return true
		}
	}
	return false
}

func buildAgentConfig(push *model.PushContext, httpRoutes []config.Config) *route.AgentConfig {
	if push == nil {
		return nil
	}
	out := &route.AgentConfig{}
	providers := map[string]bool{}
	backends := map[string]bool{}
	policies := map[string]bool{}

	for _, routeConfig := range httpRoutes {
		spec, ok := routeConfig.Spec.(*gatewayv1.HTTPRouteSpec)
		if !ok {
			continue
		}
		for ruleIndex, rule := range spec.Rules {
			if !ruleUsesDxgateService(rule) {
				continue
			}
			if len(rule.BackendRefs) == 0 {
				continue
			}

			var protocol route.AgentProtocol
			var weighted []*route.WeightedBackend
			valid := true
			for _, ref := range rule.BackendRefs {
				if !isDxgateServiceBackend(ref) {
					log.Warnf("HTTPRoute %s/%s rule[%d] mixes Service and DxgateService backends", routeConfig.Namespace, routeConfig.Name, ruleIndex)
					valid = false
					break
				}
				namespace := routeConfig.Namespace
				if ref.Namespace != nil {
					namespace = string(*ref.Namespace)
				}
				if namespace != routeConfig.Namespace {
					log.Warnf("HTTPRoute %s/%s rule[%d] cross-namespace DxgateService reference %s/%s is not supported", routeConfig.Namespace, routeConfig.Name, ruleIndex, namespace, ref.Name)
					valid = false
					break
				}
				serviceConfig, found := push.DxgateService(namespace, string(ref.Name))
				if !found {
					log.Warnf("HTTPRoute %s/%s rule[%d] references missing DxgateService %s/%s", routeConfig.Namespace, routeConfig.Name, ruleIndex, namespace, ref.Name)
					valid = false
					break
				}
				service, ok := serviceConfig.Spec.(*networking.DxgateService)
				if !ok {
					valid = false
					break
				}
				compiled, serviceProtocol, err := compileDxgateService(serviceConfig, service)
				if err != nil {
					log.Warnf("DxgateService %s/%s cannot be compiled: %v", namespace, ref.Name, err)
					valid = false
					break
				}
				if protocol != route.AgentProtocol_AGENT_PROTOCOL_UNSPECIFIED && protocol != serviceProtocol {
					log.Warnf("HTTPRoute %s/%s rule[%d] mixes DxgateService protocols", routeConfig.Namespace, routeConfig.Name, ruleIndex)
					valid = false
					break
				}
				protocol = serviceProtocol
				if compiled.provider != nil && !providers[compiled.provider.Name] {
					out.Providers = append(out.Providers, compiled.provider)
					providers[compiled.provider.Name] = true
				}
				if compiled.policy != nil && !policies[compiled.policy.Name] {
					out.Policies = append(out.Policies, compiled.policy)
					policies[compiled.policy.Name] = true
				}
				weight := uint32(1)
				if ref.Weight != nil && *ref.Weight > 0 {
					weight = uint32(*ref.Weight)
				}
				for _, backend := range compiled.backends {
					if !backends[backend.Name] {
						out.Backends = append(out.Backends, backend)
						backends[backend.Name] = true
					}
					weighted = append(weighted, &route.WeightedBackend{Name: backend.Name, Weight: weight})
				}
			}
			if !valid || len(weighted) == 0 {
				continue
			}
			out.AgentRoutes = append(out.AgentRoutes, &route.AgentRoute{
				Name:             fmt.Sprintf("%s/%s/%d", routeConfig.Namespace, routeConfig.Name, ruleIndex),
				Protocol:         protocol,
				Matches:          compileAgentMatches(spec.Hostnames, rule.Matches),
				WeightedBackends: weighted,
				Policies:         policyNames(weighted, out.Backends),
				Rewrite:          compileAgentRewrite(rule.Filters),
			})
		}
	}
	if len(out.AgentRoutes) == 0 {
		return nil
	}
	sort.Slice(out.Providers, func(i, j int) bool { return out.Providers[i].Name < out.Providers[j].Name })
	sort.Slice(out.Backends, func(i, j int) bool { return out.Backends[i].Name < out.Backends[j].Name })
	sort.Slice(out.Policies, func(i, j int) bool { return out.Policies[i].Name < out.Policies[j].Name })
	sort.Slice(out.AgentRoutes, func(i, j int) bool { return out.AgentRoutes[i].Name < out.AgentRoutes[j].Name })
	return out
}

type compiledDxgateService struct {
	provider *route.AgentProvider
	backends []*route.AgentBackend
	policy   *route.AgentPolicy
}

func compileDxgateService(cfg config.Config, service *networking.DxgateService) (compiledDxgateService, route.AgentProtocol, error) {
	prefix := fmt.Sprintf("%s.%s", cfg.Name, cfg.Namespace)
	policyName := prefix + ".policy"
	policy := compileAgentPolicy(policyName, cfg.Namespace, service.GetPolicies())
	policyRefs := []string(nil)
	if policy != nil {
		policyRefs = []string{policyName}
	}

	switch {
	case service.GetAi() != nil:
		ai := service.GetAi()
		if ai.GetProvider() == nil {
			return compiledDxgateService{}, 0, fmt.Errorf("provider is not set")
		}
		providerName := prefix + ".provider"
		provider := &route.AgentProvider{
			Name:           providerName,
			BaseUrl:        ai.GetEndpoint(),
			Credential:     compileSecretReference(cfg.Namespace, ai.GetProvider().GetCredential()),
			RequestHeaders: compileHeaderValues(ai.GetProvider().GetRequestHeaders()),
		}
		switch {
		case ai.GetProvider().GetOpenai() != nil:
			provider.Kind = route.AgentProviderKind_OPENAI
		case ai.GetProvider().GetAnthropic() != nil:
			provider.Kind = route.AgentProviderKind_ANTHROPIC
		default:
			return compiledDxgateService{}, 0, fmt.Errorf("provider is not set")
		}
		backend := &route.AgentBackend{
			Name: prefix,
			Backend: &route.AgentBackend_Llm{Llm: &route.LLMBackend{
				Provider:      providerName,
				Models:        append([]string(nil), ai.GetModels()...),
				ModelRewrites: cloneStringMap(ai.GetModelRewrites()),
			}},
			Policies: policyRefs,
		}
		return compiledDxgateService{provider: provider, backends: []*route.AgentBackend{backend}, policy: policy}, route.AgentProtocol_LLM, nil
	case service.GetMcp() != nil:
		backends := make([]*route.AgentBackend, 0, len(service.GetMcp().GetTargets()))
		for _, target := range service.GetMcp().GetTargets() {
			static := target.GetStatic()
			if static == nil || static.GetBackendRef() == nil {
				continue
			}
			namespace := static.GetBackendRef().GetNamespace()
			if namespace == "" {
				namespace = cfg.Namespace
			}
			backends = append(backends, &route.AgentBackend{
				Name: fmt.Sprintf("%s.%s", prefix, target.GetName()),
				Backend: &route.AgentBackend_Mcp{Mcp: &route.MCPBackend{
					Endpoint: serviceEndpoint(static.GetBackendRef().GetName(), namespace, static.GetPort(), static.GetPath()),
					Tools:    append([]string(nil), target.GetTools()...),
				}},
				Policies: policyRefs,
			})
		}
		return compiledDxgateService{backends: backends, policy: policy}, route.AgentProtocol_MCP, nil
	case service.GetA2A() != nil:
		a2a := service.GetA2A()
		endpoint := ""
		if ref := a2a.GetBackendRef(); ref != nil {
			namespace := ref.GetNamespace()
			if namespace == "" {
				namespace = cfg.Namespace
			}
			endpoint = serviceEndpoint(ref.GetName(), namespace, a2a.GetPort(), a2a.GetPath())
		} else {
			endpoint = hostEndpoint(a2a.GetHost(), a2a.GetPort(), a2a.GetPath())
		}
		backend := &route.AgentBackend{
			Name: prefix,
			Backend: &route.AgentBackend_A2A{A2A: &route.A2ABackend{
				Endpoint: endpoint,
				Agent:    a2a.GetAgent(),
			}},
			Policies: policyRefs,
		}
		return compiledDxgateService{backends: []*route.AgentBackend{backend}, policy: policy}, route.AgentProtocol_A2A, nil
	default:
		return compiledDxgateService{}, 0, fmt.Errorf("service type is not set")
	}
}

func serviceEndpoint(name, namespace string, port uint32, path string) string {
	return hostEndpoint(fmt.Sprintf("%s.%s.svc.cluster.local", name, namespace), port, path)
}

func hostEndpoint(host string, port uint32, path string) string {
	return fmt.Sprintf("http://%s:%d%s", host, port, normalizeEndpointPath(path))
}

func normalizeEndpointPath(path string) string {
	if path == "" || path == "/" {
		return ""
	}
	return "/" + strings.Trim(path, "/")
}

func compileAgentMatches(hostnames []gatewayv1.Hostname, matches []gatewayv1.HTTPRouteMatch) []*route.AgentRouteMatch {
	hosts := make([]string, 0, len(hostnames))
	for _, hostname := range hostnames {
		hosts = append(hosts, string(hostname))
	}
	if len(hosts) == 0 {
		hosts = []string{""}
	}
	if len(matches) == 0 {
		matches = []gatewayv1.HTTPRouteMatch{{}}
	}
	out := make([]*route.AgentRouteMatch, 0, len(hosts)*len(matches))
	for _, hostname := range hosts {
		for _, match := range matches {
			compiled := &route.AgentRouteMatch{
				Host:    hostname,
				Path:    compileAgentPathMatch(match.Path),
				Headers: make([]*route.AgentHeaderMatch, 0, len(match.Headers)),
			}
			if match.Method != nil {
				compiled.Method = string(*match.Method)
			}
			for _, header := range match.Headers {
				compiled.Headers = append(compiled.Headers, &route.AgentHeaderMatch{
					Name: string(header.Name), Value: header.Value,
				})
			}
			out = append(out, compiled)
		}
	}
	return out
}

func compileAgentPathMatch(match *gatewayv1.HTTPPathMatch) *route.AgentPathMatch {
	if match == nil || match.Value == nil {
		return &route.AgentPathMatch{Match: &route.AgentPathMatch_Prefix{Prefix: "/"}}
	}
	if match.Type != nil && *match.Type == gatewayv1.PathMatchExact {
		return &route.AgentPathMatch{Match: &route.AgentPathMatch_Exact{Exact: *match.Value}}
	}
	return &route.AgentPathMatch{Match: &route.AgentPathMatch_Prefix{Prefix: *match.Value}}
}

func compileAgentRewrite(filters []gatewayv1.HTTPRouteFilter) *route.PathRewrite {
	for _, filter := range filters {
		if filter.Type != gatewayv1.HTTPRouteFilterURLRewrite || filter.URLRewrite == nil || filter.URLRewrite.Path == nil {
			continue
		}
		if filter.URLRewrite.Path.Type == gatewayv1.PrefixMatchHTTPPathModifier && filter.URLRewrite.Path.ReplacePrefixMatch != nil {
			return &route.PathRewrite{ReplacePrefixMatch: *filter.URLRewrite.Path.ReplacePrefixMatch}
		}
	}
	return nil
}

func compileAgentPolicy(name, namespace string, in *networking.DxgateServicePolicies) *route.AgentPolicy {
	if in == nil {
		return nil
	}
	out := &route.AgentPolicy{
		Name:            name,
		Timeout:         in.GetTimeout(),
		MaxBodyBytes:    uint64(max(in.GetMaxBodyBytes(), 0)),
		RequestHeaders:  compileHeaderTransform(in.GetRequestHeaders()),
		ResponseHeaders: compileHeaderTransform(in.GetResponseHeaders()),
	}
	if auth := in.GetAuth(); auth != nil {
		out.Auth = &route.ClientAuthPolicy{
			Header:    auth.GetHeader(),
			SecretRef: compileSecretReference(namespace, auth.GetSecretRef()),
		}
	}
	if limit := in.GetRateLimit(); limit != nil {
		out.RateLimit = &route.AgentRateLimitPolicy{
			Requests: limit.GetRequests(), Window: limit.GetWindow(),
			Key: compileRateLimitKey(limit.GetKey()), Header: limit.GetHeader(),
		}
	}
	if limit := in.GetTokenLimit(); limit != nil {
		out.TokenLimit = &route.AgentTokenLimitPolicy{
			Tokens: limit.GetTokens(), Window: limit.GetWindow(),
			Key: compileRateLimitKey(limit.GetKey()), Header: limit.GetHeader(),
		}
	}
	if retry := in.GetRetry(); retry != nil {
		out.Retry = &route.AgentRetryPolicy{
			Attempts: retry.GetAttempts(), StatusCodes: append([]uint32(nil), retry.GetStatusCodes()...),
		}
	}
	return out
}

func compileSecretReference(namespace string, ref *networking.SecretKeyReference) *route.SecretKeyReference {
	if ref == nil {
		return nil
	}
	return &route.SecretKeyReference{Namespace: namespace, Name: ref.GetName(), Key: ref.GetKey()}
}

func compileHeaderTransform(in *networking.HeaderTransform) *route.AgentHeaderTransform {
	if in == nil {
		return nil
	}
	return &route.AgentHeaderTransform{
		Add:    compileHeaderValues(in.GetAdd()),
		Remove: append([]string(nil), in.GetRemove()...),
	}
}

func compileHeaderValues(in []*networking.HeaderValue) []*route.AgentHeaderValue {
	out := make([]*route.AgentHeaderValue, 0, len(in))
	for _, value := range in {
		if value != nil {
			out = append(out, &route.AgentHeaderValue{Name: value.GetName(), Value: value.GetValue()})
		}
	}
	return out
}

func compileRateLimitKey(in networking.RateLimitKey) route.AgentRateLimitKey {
	switch in {
	case networking.RateLimitKey_BACKEND:
		return route.AgentRateLimitKey_BACKEND
	case networking.RateLimitKey_HEADER:
		return route.AgentRateLimitKey_HEADER
	default:
		return route.AgentRateLimitKey_ROUTE
	}
}

func policyNames(weighted []*route.WeightedBackend, backends []*route.AgentBackend) []string {
	seen := map[string]bool{}
	var out []string
	for _, selected := range weighted {
		for _, backend := range backends {
			if backend.Name != selected.Name {
				continue
			}
			for _, policy := range backend.Policies {
				if !seen[policy] {
					out = append(out, policy)
					seen[policy] = true
				}
			}
		}
	}
	sort.Strings(out)
	return out
}

func cloneStringMap(in map[string]string) map[string]string {
	if len(in) == 0 {
		return nil
	}
	out := make(map[string]string, len(in))
	for key, value := range in {
		out[key] = value
	}
	return out
}
