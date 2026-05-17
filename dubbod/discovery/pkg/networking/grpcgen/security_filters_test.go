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

	"github.com/apache/dubbo-kubernetes/dubbod/discovery/pkg/model"
	"github.com/apache/dubbo-kubernetes/pkg/config"
	"github.com/apache/dubbo-kubernetes/pkg/config/schema/gvk"
	"github.com/apache/dubbo-kubernetes/pkg/wellknown"
	security "github.com/kdubbo/api/security/v1alpha3"
	typev1alpha3 "github.com/kdubbo/api/type/v1alpha3"
	jwtv1 "github.com/kdubbo/xds-api/extensions/filters/v1/http/jwt_authn"
	rbacv1 "github.com/kdubbo/xds-api/extensions/filters/v1/http/rbac"
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
	if got := jwtConfig.GetProviders()[0].GetJwksUri(); got != "tools/jwt/samples/jwks.json" {
		t.Fatalf("jwksUri = %q, want local sample path", got)
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
				Issuer:  "testing@secure.dubbo.apache.org",
				JwksUri: "tools/jwt/samples/jwks.json",
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
					},
				}},
				When: []*security.Condition{{
					Key:    "request.auth.claims[groups]",
					Values: []string{"group1"},
				}},
			}},
		},
	}
}
