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
	"testing"

	"github.com/apache/dubbo-kubernetes/pkg/config"
	"github.com/apache/dubbo-kubernetes/pkg/config/schema/gvk"
	security "github.com/kdubbo/api/security/v1alpha3"
	typev1alpha3 "github.com/kdubbo/api/type/v1alpha3"
)

func TestConvertToMutualTLSModeIncludesPermissive(t *testing.T) {
	if got := ConvertToMutualTLSMode(security.PeerAuthentication_MutualTLS_PERMISSIVE); got != MTLSPermissive {
		t.Fatalf("mode = %v, want MTLSPermissive", got)
	}
}

func TestGlobalPeerAuthenticationFromRootNamespaceAppliesToWorkloadNamespace(t *testing.T) {
	policy := &AuthenticationPolicies{
		peerAuthentications:    map[string][]config.Config{},
		requestAuthentications: map[string][]config.Config{},
		authorizationPolicies:  map[string][]config.Config{},
		globalMutualTLSMode:    MTLSUnknown,
		rootNamespace:          "dubbo-system",
		namespaceMutualTLSMode: map[string]MutualTLSMode{},
	}
	policy.addPeerAuthentication([]config.Config{
		{
			Meta: config.Meta{
				GroupVersionKind: gvk.PeerAuthentication,
				Name:             "default",
				Namespace:        "dubbo-system",
			},
			Spec: &security.PeerAuthentication{
				Mtls: &security.PeerAuthentication_MutualTLS{
					Mode: security.PeerAuthentication_MutualTLS_STRICT,
				},
			},
		},
	})

	if got := policy.EffectiveMutualTLSMode("app", nil, 80); got != MTLSStrict {
		t.Fatalf("effective mode = %v, want MTLSStrict", got)
	}
}

func TestRequestAuthenticationSelectorMatchesWorkload(t *testing.T) {
	policy := &AuthenticationPolicies{
		requestAuthentications: map[string][]config.Config{},
		authorizationPolicies:  map[string][]config.Config{},
		peerAuthentications:    map[string][]config.Config{},
		rootNamespace:          "dubbo-system",
	}
	policy.addRequestAuthentication([]config.Config{{
		Meta: config.Meta{
			GroupVersionKind: gvk.RequestAuthentication,
			Name:             "jwt-example",
			Namespace:        "foo",
		},
		Spec: &security.RequestAuthentication{
			Selector: newWorkloadSelector("app", "httpbin"),
			JwtRules: []*security.JWTRule{{
				Issuer:  "testing@secure.dubbo.apache.org",
				JwksUri: "tools/jwt/samples/jwks.json",
			}},
		},
	}})

	if got := policy.RequestAuthenticationsForWorkload("foo", map[string]string{"app": "httpbin"}); len(got) != 1 {
		t.Fatalf("matching request authentications = %d, want 1", len(got))
	}
	if got := policy.RequestAuthenticationsForWorkload("foo", map[string]string{"app": "reviews"}); len(got) != 0 {
		t.Fatalf("non-matching request authentications = %d, want 0", len(got))
	}
}

func TestAuthorizationPolicySelectorMatchesWorkload(t *testing.T) {
	policy := &AuthenticationPolicies{
		requestAuthentications: map[string][]config.Config{},
		authorizationPolicies:  map[string][]config.Config{},
		peerAuthentications:    map[string][]config.Config{},
		rootNamespace:          "dubbo-system",
	}
	policy.addAuthorizationPolicy([]config.Config{{
		Meta: config.Meta{
			GroupVersionKind: gvk.AuthorizationPolicy,
			Name:             "require-jwt",
			Namespace:        "foo",
		},
		Spec: &security.AuthorizationPolicy{
			Selector: newWorkloadSelector("app", "httpbin"),
			Action:   security.AuthorizationPolicy_ALLOW,
			Rules: []*security.Rule{{
				From: []*security.From{{
					Source: &security.Source{
						RequestPrincipals: []string{"testing@secure.dubbo.apache.org/testing@secure.dubbo.apache.org"},
					},
				}},
			}},
		},
	}})

	if got := policy.AuthorizationPoliciesForWorkload("foo", map[string]string{"app": "httpbin"}); len(got) != 1 {
		t.Fatalf("matching authorization policies = %d, want 1", len(got))
	}
	if got := policy.AuthorizationPoliciesForWorkload("bar", map[string]string{"app": "httpbin"}); len(got) != 0 {
		t.Fatalf("other namespace authorization policies = %d, want 0", len(got))
	}
}

func newWorkloadSelector(key, value string) *typev1alpha3.WorkloadSelector {
	return &typev1alpha3.WorkloadSelector{MatchLabels: map[string]string{key: value}}
}
