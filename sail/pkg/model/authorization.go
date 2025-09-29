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

package model

import (
	"github.com/apache/dubbo-kubernetes/pkg/config/schema/gvk"
	authpb "istio.io/api/security/v1beta1"
)

type AuthorizationPolicy struct {
	Name        string                      `json:"name"`
	Namespace   string                      `json:"namespace"`
	Annotations map[string]string           `json:"annotations"`
	Spec        *authpb.AuthorizationPolicy `json:"spec"`
}

type AuthorizationPolicies struct {
	NamespaceToPolicies map[string][]AuthorizationPolicy `json:"namespace_to_policies"`
	RootNamespace       string                           `json:"root_namespace"`
}

func GetAuthorizationPolicies(env *Environment) *AuthorizationPolicies {
	policy := &AuthorizationPolicies{
		NamespaceToPolicies: map[string][]AuthorizationPolicy{},
		RootNamespace:       env.Mesh().GetRootNamespace(),
	}

	policies := env.List(gvk.AuthorizationPolicy, NamespaceAll)
	sortConfigByCreationTime(policies)

	policyCount := make(map[string]int)
	for _, config := range policies {
		policyCount[config.Namespace]++
	}

	for _, config := range policies {
		authzConfig := AuthorizationPolicy{
			Name:        config.Name,
			Namespace:   config.Namespace,
			Annotations: config.Annotations,
			Spec:        config.Spec.(*authpb.AuthorizationPolicy),
		}
		if _, ok := policy.NamespaceToPolicies[config.Namespace]; !ok {
			policy.NamespaceToPolicies[config.Namespace] = make([]AuthorizationPolicy, 0, policyCount[config.Namespace])
		}
		policy.NamespaceToPolicies[config.Namespace] = append(policy.NamespaceToPolicies[config.Namespace], authzConfig)
	}

	return policy
}
