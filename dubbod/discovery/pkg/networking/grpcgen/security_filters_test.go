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
	"testing"
	"time"

	"github.com/apache/dubbo-kubernetes/dubbod/discovery/pkg/model"
	"github.com/apache/dubbo-kubernetes/pkg/config"
	"github.com/apache/dubbo-kubernetes/pkg/config/schema/gvk"
	"github.com/apache/dubbo-kubernetes/pkg/wellknown"
	mesh "github.com/kdubbo/api/mesh/v1alpha1"
	security "github.com/kdubbo/api/security/v1alpha3"
	typev1alpha3 "github.com/kdubbo/api/type/v1alpha3"
	extauthzv1 "github.com/kdubbo/xds-api/extensions/filters/v1/http/ext_authz"
	jwtv1 "github.com/kdubbo/xds-api/extensions/filters/v1/http/jwt_authn"
	rbacv1 "github.com/kdubbo/xds-api/extensions/filters/v1/http/rbac"
	"google.golang.org/protobuf/types/known/durationpb"
)

func TestBuildInboundHTTPFiltersAddsJWTAndAuthorizationBeforeRouter(t *testing.T) {
	push := newRDSTestPushContext(t, []config.Config{
		newRequestAuthenticationConfig(),
		newAuthorizationPolicyConfig(),
	}, []*model.Service{
		newRDSTestService("httpbin", "foo", "httpbin.foo.svc.cluster.local", 8000),
	})
	serviceTarget := model.ServiceTarget{
		Service: &model.Service{
			Attributes: model.ServiceAttributes{
				Namespace:      "foo",
				LabelSelectors: map[string]string{"app": "httpbin"},
			},
		},
	}

	filters := buildInboundHTTPFilters(push, serviceTarget)
	if len(filters) != 3 {
		t.Fatalf("filters = %d, want jwt, rbac, router", len(filters))
	}
	if filters[0].GetName() != wellknown.JWTAuthentication {
		t.Fatalf("first filter = %q, want JWT authn", filters[0].GetName())
	}
	if filters[1].GetName() != wellknown.HTTPRoleBasedAccessControl {
		t.Fatalf("second filter = %q, want RBAC", filters[1].GetName())
	}
	if filters[2].GetName() != wellknown.HTTPRouter {
		t.Fatalf("last filter = %q, want router", filters[2].GetName())
	}

	jwtConfig := &jwtv1.JwtAuthentication{}
	if err := filters[0].GetTypedConfig().UnmarshalTo(jwtConfig); err != nil {
		t.Fatalf("unmarshal jwt filter: %v", err)
	}
	if !jwtConfig.GetAllowMissing() {
		t.Fatalf("allowMissing = false, want true")
	}
	if len(jwtConfig.GetProviders()) != 1 {
		t.Fatalf("jwt providers = %d, want 1", len(jwtConfig.GetProviders()))
	}
	if got := jwtConfig.GetProviders()[0].GetIssuer(); got != "testing@secure.dubbo.apache.org" {
		t.Fatalf("issuer = %q, want testing issuer", got)
	}
	if got := jwtConfig.GetProviders()[0].GetJwksUri(); got != "https://secure.dubbo.apache.org/jwt/samples/jwks.json" {
		t.Fatalf("jwksUri = %q, want local sample path", got)
	}
	provider := jwtConfig.GetProviders()[0]
	if got := provider.GetFromCookies(); len(got) != 1 || got[0] != "session" {
		t.Fatalf("fromCookies = %v, want [session]", got)
	}
	if !provider.GetForwardOriginalToken() || provider.GetOutputPayloadToHeader() != "x-jwt-payload" {
		t.Fatalf("JWT forwarding fields = %+v", provider)
	}
	if got := provider.GetOutputClaimToHeaders(); len(got) != 1 || got[0].GetClaim() != "sub" || got[0].GetHeader() != "x-jwt-sub" {
		t.Fatalf("claim headers = %v, want sub -> x-jwt-sub", got)
	}

	rbacConfig := &rbacv1.RBAC{}
	if err := filters[1].GetTypedConfig().UnmarshalTo(rbacConfig); err != nil {
		t.Fatalf("unmarshal rbac filter: %v", err)
	}
	if rbacConfig.GetAction() != rbacv1.RBAC_ALLOW {
		t.Fatalf("action = %v, want ALLOW", rbacConfig.GetAction())
	}
	if len(rbacConfig.GetRules()) != 1 {
		t.Fatalf("rbac rules = %d, want 1", len(rbacConfig.GetRules()))
	}
	rule := rbacConfig.GetRules()[0]
	if got := rule.GetSources()[0].GetRequestPrincipals()[0]; got != "testing@secure.dubbo.apache.org/testing@secure.dubbo.apache.org" {
		t.Fatalf("request principal = %q, want testing principal", got)
	}
	if got := rule.GetWhen()[0].GetKey(); got != "request.auth.claims[groups]" {
		t.Fatalf("claim key = %q, want groups claim", got)
	}
	if got := rule.GetWhen()[0].GetValues()[0]; got != "group1" {
		t.Fatalf("claim value = %q, want group1", got)
	}
	if got := rule.GetOperations()[0].GetMethods(); len(got) != 1 || got[0] != "GET" {
		t.Fatalf("methods = %v, want [GET]", got)
	}
	if got := rule.GetSources()[0].GetRemoteIpBlocks(); len(got) != 1 || got[0] != "10.0.0.0/8" {
		t.Fatalf("remote IP blocks = %v, want [10.0.0.0/8]", got)
	}
}

func newRequestAuthenticationConfig() config.Config {
	return config.Config{
		Meta: config.Meta{
			GroupVersionKind: gvk.RequestAuthentication,
			Name:             "jwt-example",
			Namespace:        "foo",
		},
		Spec: &security.RequestAuthentication{
			Selector: &typev1alpha3.WorkloadSelector{MatchLabels: map[string]string{"app": "httpbin"}},
			JwtRules: []*security.JWTRule{{
				Issuer:                "testing@secure.dubbo.apache.org",
				JwksUri:               "https://secure.dubbo.apache.org/jwt/samples/jwks.json",
				FromCookies:           []string{"session"},
				ForwardOriginalToken:  true,
				OutputPayloadToHeader: "x-jwt-payload",
				OutputClaimToHeaders: []*security.ClaimToHeader{{
					Claim:  "sub",
					Header: "x-jwt-sub",
				}},
			}},
		},
	}
}

func newAuthorizationPolicyConfig() config.Config {
	return config.Config{
		Meta: config.Meta{
			GroupVersionKind: gvk.AuthorizationPolicy,
			Name:             "require-jwt",
			Namespace:        "foo",
		},
		Spec: &security.AuthorizationPolicy{
			Selector: &typev1alpha3.WorkloadSelector{MatchLabels: map[string]string{"app": "httpbin"}},
			Action:   security.AuthorizationPolicy_ALLOW,
			Rules: []*security.Rule{{
				From: []*security.From{{
					Source: &security.Source{
						RequestPrincipals: []string{"testing@secure.dubbo.apache.org/testing@secure.dubbo.apache.org"},
						RemoteIpBlocks:    []string{"10.0.0.0/8"},
					},
				}},
				To: []*security.To{{
					Operation: &security.Operation{Methods: []string{"GET"}, Paths: []string{"/headers*"}},
				}},
				When: []*security.Condition{{
					Key:    "request.auth.claims[groups]",
					Values: []string{"group1"},
				}},
			}},
		},
	}
}

func TestBuildAuthorizationFiltersCustomDryRunAndTrustDomainAlias(t *testing.T) {
	custom := config.Config{
		Meta: config.Meta{GroupVersionKind: gvk.AuthorizationPolicy, Name: "external-check", Namespace: "foo"},
		Spec: &security.AuthorizationPolicy{
			Action:   security.AuthorizationPolicy_CUSTOM,
			Provider: &security.ExtensionProvider{Name: "opa"},
			Rules: []*security.Rule{{
				From: []*security.From{{Source: &security.Source{
					Principals: []string{"spiffe://cluster.local/ns/foo/sa/client"},
				}}},
			}},
		},
	}
	audit := config.Config{
		Meta: config.Meta{GroupVersionKind: gvk.AuthorizationPolicy, Name: "audit-admin", Namespace: "foo"},
		Spec: &security.AuthorizationPolicy{
			Action: security.AuthorizationPolicy_AUDIT,
			Rules:  []*security.Rule{{To: []*security.To{{Operation: &security.Operation{Paths: []string{"/admin*"}}}}}},
		},
	}
	push := newRDSTestPushContext(t, nil, nil)
	push.Mesh.TrustDomainAliases = []string{"old.local"}
	push.Mesh.ExtensionProviders = []*mesh.MeshExtensionProvider{{
		Name: "opa",
		Provider: &mesh.MeshExtensionProvider_EnvoyExtAuthzHttp{
			EnvoyExtAuthzHttp: &mesh.ExternalAuthorizationProvider{
				Service:                      "opa.foo.svc.cluster.local",
				Port:                         9191,
				PathPrefix:                   "/check",
				IncludeRequestHeadersInCheck: []string{"authorization"},
				HeadersToUpstreamOnAllow:     []string{"x-user"},
				Timeout:                      durationpb.New(2 * time.Second),
				FailOpen:                     true,
			},
		},
	}}

	filters := buildAuthorizationFilters(push, []config.Config{audit, custom})
	if len(filters) != 2 {
		t.Fatalf("filters = %d, want audit + external authorization", len(filters))
	}
	auditConfig := &rbacv1.RBAC{}
	if err := filters[0].GetTypedConfig().UnmarshalTo(auditConfig); err != nil {
		t.Fatalf("unmarshal audit filter: %v", err)
	}
	if !auditConfig.GetShadow() || auditConfig.GetPolicyName() != "audit-admin" {
		t.Fatalf("audit filter = %+v, want shadow audit-admin", auditConfig)
	}
	external := &extauthzv1.ExtAuthz{}
	if err := filters[1].GetTypedConfig().UnmarshalTo(external); err != nil {
		t.Fatalf("unmarshal external authorization filter: %v", err)
	}
	if external.GetService() != "opa.foo.svc.cluster.local" || external.GetPort() != 9191 || !external.GetFailOpen() {
		t.Fatalf("external authorization = %+v", external)
	}
	principals := external.GetRules()[0].GetSources()[0].GetPrincipals()
	if len(principals) != 2 || principals[1] != "spiffe://old.local/ns/foo/sa/client" {
		t.Fatalf("expanded principals = %v, want cluster.local and old.local", principals)
	}
}

func TestBuildAuthorizationFiltersSplitsDenyAndAllow(t *testing.T) {
	deny := config.Config{
		Meta: config.Meta{GroupVersionKind: gvk.AuthorizationPolicy, Name: "deny-bad", Namespace: "foo"},
		Spec: &security.AuthorizationPolicy{
			Action: security.AuthorizationPolicy_DENY,
			Rules: []*security.Rule{{
				From: []*security.From{{
					Source: &security.Source{RequestPrincipals: []string{"issuer/bad-subject"}},
				}},
			}},
		},
	}
	allow := newAuthorizationPolicyConfig()

	filters := buildAuthorizationFilters(nil, []config.Config{deny, allow})
	if len(filters) != 2 {
		t.Fatalf("filters = %d, want deny + allow", len(filters))
	}

	denyConfig := &rbacv1.RBAC{}
	if err := filters[0].GetTypedConfig().UnmarshalTo(denyConfig); err != nil {
		t.Fatalf("unmarshal deny filter: %v", err)
	}
	if denyConfig.GetAction() != rbacv1.RBAC_DENY {
		t.Fatalf("first filter action = %v, want DENY", denyConfig.GetAction())
	}
	if len(denyConfig.GetRules()) != 1 {
		t.Fatalf("deny rules = %d, want 1", len(denyConfig.GetRules()))
	}

	allowConfig := &rbacv1.RBAC{}
	if err := filters[1].GetTypedConfig().UnmarshalTo(allowConfig); err != nil {
		t.Fatalf("unmarshal allow filter: %v", err)
	}
	if allowConfig.GetAction() != rbacv1.RBAC_ALLOW {
		t.Fatalf("second filter action = %v, want ALLOW", allowConfig.GetAction())
	}
	if len(allowConfig.GetRules()) != 1 {
		t.Fatalf("allow rules = %d, want 1", len(allowConfig.GetRules()))
	}
}

func TestBuildAuthorizationFiltersEmptyAllowPolicyDeniesAll(t *testing.T) {
	// An ALLOW policy with no rules matches nothing, which must still emit an
	// ALLOW filter (with zero rules) so every request is rejected.
	allow := config.Config{
		Meta: config.Meta{GroupVersionKind: gvk.AuthorizationPolicy, Name: "deny-all", Namespace: "foo"},
		Spec: &security.AuthorizationPolicy{Action: security.AuthorizationPolicy_ALLOW},
	}
	filters := buildAuthorizationFilters(nil, []config.Config{allow})
	if len(filters) != 1 {
		t.Fatalf("filters = %d, want 1", len(filters))
	}
	rbacConfig := &rbacv1.RBAC{}
	if err := filters[0].GetTypedConfig().UnmarshalTo(rbacConfig); err != nil {
		t.Fatalf("unmarshal filter: %v", err)
	}
	if rbacConfig.GetAction() != rbacv1.RBAC_ALLOW {
		t.Fatalf("action = %v, want ALLOW", rbacConfig.GetAction())
	}
	if len(rbacConfig.GetRules()) != 0 {
		t.Fatalf("rules = %d, want 0", len(rbacConfig.GetRules()))
	}
}
