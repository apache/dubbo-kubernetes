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
	"crypto/md5"
	"fmt"
	"strings"
	"time"

	"github.com/apache/dubbo-kubernetes/pkg/config/schema/gvk"
	"github.com/kdubbo/api/security/v1alpha3"

	"github.com/apache/dubbo-kubernetes/pkg/config"
	typev1alpha3 "github.com/kdubbo/api/type/v1alpha3"
)

type MutualTLSMode int

const (
	MTLSUnknown MutualTLSMode = iota
	MTLSDisable
	MTLSPermissive
	MTLSStrict
)

type AuthenticationPolicies struct {
	requestAuthentications map[string][]config.Config
	authorizationPolicies  map[string][]config.Config
	peerAuthentications    map[string][]config.Config
	globalMutualTLSMode    MutualTLSMode
	rootNamespace          string
	namespaceMutualTLSMode map[string]MutualTLSMode
	aggregateVersion       string
}

func initAuthenticationPolicies(env *Environment) *AuthenticationPolicies {
	policy := &AuthenticationPolicies{
		requestAuthentications: map[string][]config.Config{},
		authorizationPolicies:  map[string][]config.Config{},
		peerAuthentications:    map[string][]config.Config{},
		globalMutualTLSMode:    MTLSUnknown,
		rootNamespace:          env.Mesh().GetRootNamespace(),
	}

	requestAuthentications := sortConfigByCreationTime(env.List(gvk.RequestAuthentication, NamespaceAll))
	authorizationPolicies := sortConfigByCreationTime(env.List(gvk.AuthorizationPolicy, NamespaceAll))
	peerAuthentications := sortConfigByCreationTime(env.List(gvk.PeerAuthentication, NamespaceAll))

	policy.addRequestAuthentication(requestAuthentications)
	policy.addAuthorizationPolicy(authorizationPolicies)
	policy.addPeerAuthentication(peerAuthentications)
	policy.aggregateVersion = authPolicyAggregateVersion(requestAuthentications, authorizationPolicies, peerAuthentications)

	return policy
}

func (policy *AuthenticationPolicies) addRequestAuthentication(configs []config.Config) {
	for _, config := range configs {
		if _, ok := config.Spec.(*v1alpha3.RequestAuthentication); ok {
			policy.requestAuthentications[config.Namespace] = append(policy.requestAuthentications[config.Namespace], config)
		}
	}
}

func (policy *AuthenticationPolicies) addAuthorizationPolicy(configs []config.Config) {
	for _, config := range configs {
		if _, ok := config.Spec.(*v1alpha3.AuthorizationPolicy); ok {
			policy.authorizationPolicies[config.Namespace] = append(policy.authorizationPolicies[config.Namespace], config)
		}
	}
}

func (policy *AuthenticationPolicies) addPeerAuthentication(configs []config.Config) {
	sortConfigByCreationTime(configs)

	foundNamespaceMTLS := make(map[string]v1alpha3.PeerAuthentication_MutualTLS_Mode)
	seenNamespaceOrMeshConfig := make(map[string]time.Time)

	for _, config := range configs {
		spec := config.Spec.(*v1alpha3.PeerAuthentication)
		selector := spec.GetSelector()
		if selector == nil || len(selector.MatchLabels) == 0 {
			if t, ok := seenNamespaceOrMeshConfig[config.Namespace]; ok {
				log.Warnf(
					"Namespace/mesh-level PeerAuthentication is already defined for %q at time %v. Ignore %q which was created at time %v",
					config.Namespace, t, config.Name, config.CreationTimestamp)
				continue
			}
			seenNamespaceOrMeshConfig[config.Namespace] = config.CreationTimestamp

			mode := v1alpha3.PeerAuthentication_MutualTLS_UNSET
			if spec.Mtls != nil {
				mode = spec.Mtls.Mode
			}
			if config.Namespace == policy.rootNamespace {
				policy.globalMutualTLSMode = ConvertToMutualTLSMode(mode)
			} else {
				foundNamespaceMTLS[config.Namespace] = mode
			}
		}

		policy.peerAuthentications[config.Namespace] = append(policy.peerAuthentications[config.Namespace], config)
	}

	policy.namespaceMutualTLSMode = make(map[string]MutualTLSMode, len(foundNamespaceMTLS))

	inheritedMTLSMode := policy.globalMutualTLSMode

	for ns, mtlsMode := range foundNamespaceMTLS {
		if mtlsMode == v1alpha3.PeerAuthentication_MutualTLS_UNSET {
			policy.namespaceMutualTLSMode[ns] = inheritedMTLSMode
		} else {
			policy.namespaceMutualTLSMode[ns] = ConvertToMutualTLSMode(mtlsMode)
		}
	}
}

func authPolicyAggregateVersion(groups ...[]config.Config) string {
	versions := []string{}
	for _, configs := range groups {
		for _, cfg := range configs {
			versions = append(versions, cfg.GroupVersionKind.String()+"."+cfg.Namespace+"."+cfg.Name+"."+cfg.UID+"."+cfg.ResourceVersion)
		}
	}
	// Not security sensitive code.
	return fmt.Sprintf("%x", md5.Sum([]byte(strings.Join(versions, ";"))))
}

func ConvertToMutualTLSMode(mode v1alpha3.PeerAuthentication_MutualTLS_Mode) MutualTLSMode {
	switch mode {
	case v1alpha3.PeerAuthentication_MutualTLS_DISABLE:
		return MTLSDisable
	case v1alpha3.PeerAuthentication_MutualTLS_PERMISSIVE:
		return MTLSPermissive
	case v1alpha3.PeerAuthentication_MutualTLS_STRICT:
		return MTLSStrict
	default:
		return MTLSUnknown
	}
}

func (policy *AuthenticationPolicies) EffectiveMutualTLSMode(namespace string, workloadLabels map[string]string, port uint32) MutualTLSMode {
	if policy == nil {
		return MTLSUnknown
	}

	if namespace != "" {
		if mode := policy.matchingPeerAuthentication(namespace, workloadLabels, port); mode != MTLSUnknown {
			return mode
		}
		if mode, ok := policy.namespaceMutualTLSMode[namespace]; ok && mode != MTLSUnknown {
			return mode
		}
	}

	if policy.rootNamespace != "" {
		if mode := policy.matchingPeerAuthentication(policy.rootNamespace, workloadLabels, port); mode != MTLSUnknown {
			return mode
		}
	}

	if policy.globalMutualTLSMode != MTLSUnknown {
		return policy.globalMutualTLSMode
	}

	return MTLSUnknown
}

func (policy *AuthenticationPolicies) RequestAuthenticationsForWorkload(namespace string, workloadLabels map[string]string) []config.Config {
	if policy == nil {
		return nil
	}
	return policy.matchingWorkloadSecurityConfigs(policy.requestAuthentications, namespace, workloadLabels, func(spec any) *typev1alpha3.WorkloadSelector {
		if req, ok := spec.(*v1alpha3.RequestAuthentication); ok {
			return req.GetSelector()
		}
		return nil
	})
}

func (policy *AuthenticationPolicies) AuthorizationPoliciesForWorkload(namespace string, workloadLabels map[string]string) []config.Config {
	if policy == nil {
		return nil
	}
	return policy.matchingWorkloadSecurityConfigs(policy.authorizationPolicies, namespace, workloadLabels, func(spec any) *typev1alpha3.WorkloadSelector {
		if authz, ok := spec.(*v1alpha3.AuthorizationPolicy); ok {
			return authz.GetSelector()
		}
		return nil
	})
}

func (policy *AuthenticationPolicies) matchingWorkloadSecurityConfigs(configsByNamespace map[string][]config.Config, namespace string, workloadLabels map[string]string, selector func(any) *typev1alpha3.WorkloadSelector) []config.Config {
	var out []config.Config
	appendMatches := func(configs []config.Config) {
		for _, cfg := range configs {
			if selectorMatchesWorkload(selector(cfg.Spec), workloadLabels) {
				out = append(out, cfg)
			}
		}
	}
	if namespace != "" {
		appendMatches(configsByNamespace[namespace])
	}
	if policy.rootNamespace != "" && policy.rootNamespace != namespace {
		appendMatches(configsByNamespace[policy.rootNamespace])
	}
	return out
}

func (policy *AuthenticationPolicies) matchingPeerAuthentication(namespace string, workloadLabels map[string]string, port uint32) MutualTLSMode {
	configs := policy.peerAuthentications[namespace]
	if len(configs) == 0 {
		return MTLSUnknown
	}

	for _, cfg := range configs {
		spec := cfg.Spec.(*v1alpha3.PeerAuthentication)
		if hasPeerAuthSelector(spec) && selectorMatchesWorkload(spec.GetSelector(), workloadLabels) {
			if mode := peerAuthenticationModeForPort(spec, port); mode != MTLSUnknown {
				return mode
			}
		}
	}

	for _, cfg := range configs {
		spec := cfg.Spec.(*v1alpha3.PeerAuthentication)
		if !hasPeerAuthSelector(spec) {
			if mode := peerAuthenticationModeForPort(spec, port); mode != MTLSUnknown {
				return mode
			}
		}
	}

	return MTLSUnknown
}

func hasPeerAuthSelector(spec *v1alpha3.PeerAuthentication) bool {
	if spec == nil {
		return false
	}
	selector := spec.GetSelector()
	return selector != nil && len(selector.MatchLabels) > 0
}

func selectorMatchesWorkload(selector *typev1alpha3.WorkloadSelector, workloadLabels map[string]string) bool {
	if selector == nil || len(selector.MatchLabels) == 0 {
		return true
	}
	if len(workloadLabels) == 0 {
		return false
	}
	for k, v := range selector.MatchLabels {
		if workloadLabels[k] != v {
			return false
		}
	}
	return true
}

func peerAuthenticationModeForPort(spec *v1alpha3.PeerAuthentication, port uint32) MutualTLSMode {
	if spec == nil {
		return MTLSUnknown
	}
	if port != 0 && spec.PortLevelMtls != nil {
		if mtls, ok := spec.PortLevelMtls[port]; ok && mtls != nil {
			if mtls.Mode != v1alpha3.PeerAuthentication_MutualTLS_UNSET {
				return ConvertToMutualTLSMode(mtls.Mode)
			}
		}
	}
	if spec.Mtls != nil && spec.Mtls.Mode != v1alpha3.PeerAuthentication_MutualTLS_UNSET {
		return ConvertToMutualTLSMode(spec.Mtls.Mode)
	}
	return MTLSUnknown
}
