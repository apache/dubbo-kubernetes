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
