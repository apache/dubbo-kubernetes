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
	"github.com/apache/dubbo-kubernetes/dubbod/discovery/pkg/model"
	"github.com/apache/dubbo-kubernetes/dubbod/discovery/pkg/util/protoconv"
	"github.com/apache/dubbo-kubernetes/pkg/config"
	"github.com/apache/dubbo-kubernetes/pkg/wellknown"
	security "github.com/kdubbo/api/security/v1alpha3"
	jwtv1 "github.com/kdubbo/xds-api/extensions/filters/v1/http/jwt_authn"
	rbacv1 "github.com/kdubbo/xds-api/extensions/filters/v1/http/rbac"
	routerv1 "github.com/kdubbo/xds-api/extensions/filters/v1/http/router"
	hcmv1 "github.com/kdubbo/xds-api/extensions/filters/v1/network/http_connection_manager"
	"google.golang.org/protobuf/proto"
)

func buildInboundHTTPFilters(push *model.PushContext, serviceTarget model.ServiceTarget) []*hcmv1.HttpFilter {
	filters := []*hcmv1.HttpFilter{}
	namespace := serviceTargetNamespace(serviceTarget)
	workloadLabels := workloadLabelsForServiceTarget(serviceTarget)

	if jwt := buildJWTAuthenticationFilter(push.RequestAuthenticationsForWorkload(namespace, workloadLabels)); jwt != nil {
		filters = append(filters, jwt)
	}
	filters = append(filters, buildAuthorizationFilters(push.AuthorizationPoliciesForWorkload(namespace, workloadLabels))...)
	filters = append(filters, routerHTTPFilter())
	return filters
}

func buildJWTAuthenticationFilter(configs []config.Config) *hcmv1.HttpFilter {
	providers := []*jwtv1.JwtProvider{}
	for _, cfg := range configs {
		spec, ok := cfg.Spec.(*security.RequestAuthentication)
		if !ok || spec == nil {
			continue
		}
		for _, rule := range spec.GetJwtRules() {
			if rule == nil || rule.GetIssuer() == "" {
				continue
			}
			providers = append(providers, jwtProviderFromRule(rule))
		}
	}
	if len(providers) == 0 {
		return nil
	}
	return typedHTTPFilter(wellknown.JWTAuthentication, &jwtv1.JwtAuthentication{
		Providers:    providers,
		AllowMissing: true,
	})
}

func jwtProviderFromRule(rule *security.JWTRule) *jwtv1.JwtProvider {
	headers := make([]*jwtv1.JwtHeader, 0, len(rule.GetFromHeaders()))
	for _, header := range rule.GetFromHeaders() {
		if header == nil || header.GetName() == "" {
			continue
		}
		headers = append(headers, &jwtv1.JwtHeader{
			Name:   header.GetName(),
			Prefix: header.GetPrefix(),
		})
	}
	if len(headers) == 0 && len(rule.GetFromParams()) == 0 {
		headers = append(headers, &jwtv1.JwtHeader{Name: "authorization", Prefix: "Bearer "})
	}
	return &jwtv1.JwtProvider{
		Issuer:      rule.GetIssuer(),
		Audiences:   append([]string(nil), rule.GetAudiences()...),
		JwksUri:     rule.GetJwksUri(),
		Jwks:        rule.GetJwks(),
		FromHeaders: headers,
		FromParams:  append([]string(nil), rule.GetFromParams()...),
	}
}

// buildAuthorizationFilters translates AuthorizationPolicies into RBAC filters.
// DENY policies and ALLOW policies must be evaluated independently: a request is
// rejected if it matches any DENY rule, and — when at least one ALLOW policy
// exists — rejected unless it matches an ALLOW rule. Emitting them as two
// filters (DENY first) preserves those semantics; folding both actions into a
// single filter would turn every ALLOW rule into a DENY rule.
func buildAuthorizationFilters(configs []config.Config) []*hcmv1.HttpFilter {
	denyRules := []*rbacv1.Rule{}
	allowRules := []*rbacv1.Rule{}
	hasAllowPolicy := false
	for _, cfg := range configs {
		spec, ok := cfg.Spec.(*security.AuthorizationPolicy)
		if !ok || spec == nil {
			continue
		}
		rules := make([]*rbacv1.Rule, 0, len(spec.GetRules()))
		for _, rule := range spec.GetRules() {
			rules = append(rules, authorizationRuleFromAPI(rule))
		}
		if spec.GetAction() == security.AuthorizationPolicy_DENY {
			denyRules = append(denyRules, rules...)
		} else {
			// An ALLOW policy with no rules matches nothing and therefore
			// rejects every request; track policy presence separately from rules.
			hasAllowPolicy = true
			allowRules = append(allowRules, rules...)
		}
	}
	filters := []*hcmv1.HttpFilter{}
	if len(denyRules) > 0 {
		filters = append(filters, typedHTTPFilter(wellknown.HTTPRoleBasedAccessControl, &rbacv1.RBAC{
			Action: rbacv1.RBAC_DENY,
			Rules:  denyRules,
		}))
	}
	if hasAllowPolicy {
		filters = append(filters, typedHTTPFilter(wellknown.HTTPRoleBasedAccessControl, &rbacv1.RBAC{
			Action: rbacv1.RBAC_ALLOW,
			Rules:  allowRules,
		}))
	}
	return filters
}

func authorizationRuleFromAPI(rule *security.Rule) *rbacv1.Rule {
	if rule == nil {
		return &rbacv1.Rule{}
	}
	sources := make([]*rbacv1.Source, 0, len(rule.GetFrom()))
	for _, from := range rule.GetFrom() {
		if from == nil || from.GetSource() == nil {
			sources = append(sources, &rbacv1.Source{})
			continue
		}
		sources = append(sources, &rbacv1.Source{
			RequestPrincipals: append([]string(nil), from.GetSource().GetRequestPrincipals()...),
		})
	}
	when := make([]*rbacv1.Condition, 0, len(rule.GetWhen()))
	for _, condition := range rule.GetWhen() {
		if condition == nil {
			continue
		}
		when = append(when, &rbacv1.Condition{
			Key:       condition.GetKey(),
			Values:    append([]string(nil), condition.GetValues()...),
			NotValues: append([]string(nil), condition.GetNotValues()...),
		})
	}
	return &rbacv1.Rule{Sources: sources, When: when}
}

func routerHTTPFilter() *hcmv1.HttpFilter {
	return typedHTTPFilter(wellknown.HTTPRouter, &routerv1.Router{})
}

func typedHTTPFilter(name string, cfg proto.Message) *hcmv1.HttpFilter {
	return &hcmv1.HttpFilter{
		Name: name,
		ConfigType: &hcmv1.HttpFilter_TypedConfig{
			TypedConfig: protoconv.MessageToAny(cfg),
		},
	}
}

func serviceTargetNamespace(serviceTarget model.ServiceTarget) string {
	if serviceTarget.Service == nil {
		return ""
	}
	return serviceTarget.Service.Attributes.Namespace
}

func workloadLabelsForServiceTarget(serviceTarget model.ServiceTarget) map[string]string {
	if serviceTarget.Service == nil {
		return nil
	}
	if labels := serviceTarget.Service.Attributes.LabelSelectors; len(labels) > 0 {
		return labels
	}
	return serviceTarget.Service.Attributes.Labels
}
