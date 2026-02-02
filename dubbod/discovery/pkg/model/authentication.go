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
	"github.com/apache/dubbo-kubernetes/api/security/v1alpha3"
	"github.com/apache/dubbo-kubernetes/pkg/config/schema/gvk"
	"strings"
	"time"

	typev1alpha3 "github.com/apache/dubbo-kubernetes/api/type/v1alpha3"
	"github.com/apache/dubbo-kubernetes/pkg/config"
)

type MutualTLSMode int

const (
	MTLSUnknown MutualTLSMode = iota
	MTLSDisable
	MTLSStrict
)

type AuthenticationPolicies struct {
	requestAuthentications map[string][]config.Config
	peerAuthentications    map[string][]config.Config
	globalMutualTLSMode    MutualTLSMode
	rootNamespace          string
	namespaceMutualTLSMode map[string]MutualTLSMode
	aggregateVersion       string
}

func initAuthenticationPolicies(env *Environment) *AuthenticationPolicies {
	policy := &AuthenticationPolicies{
		requestAuthentications: map[string][]config.Config{},
		peerAuthentications:    map[string][]config.Config{},
		globalMutualTLSMode:    MTLSUnknown,
		rootNamespace:          env.Mesh().GetRootNamespace(),
	}

	policy.addPeerAuthentication(sortConfigByCreationTime(env.List(gvk.PeerAuthentication, NamespaceAll)))

	return policy
}

func (policy *AuthenticationPolicies) addRequestAuthentication(configs []config.Config) {
	for _, config := range configs {
		policy.requestAuthentications[config.Namespace] = append(policy.requestAuthentications[config.Namespace], config)
	}
}

func (policy *AuthenticationPolicies) addPeerAuthentication(configs []config.Config) {
	sortConfigByCreationTime(configs)

	foundNamespaceMTLS := make(map[string]v1alpha3.PeerAuthentication_MutualTLS_Mode)
	seenNamespaceOrMeshGlobalConfig := make(map[string]time.Time)
	versions := []string{}

	for _, config := range configs {
		versions = append(versions, config.UID+"."+config.ResourceVersion)
		spec := config.Spec.(*v1alpha3.PeerAuthentication)
		selector := spec.GetSelector()
		if selector == nil || len(selector.MatchLabels) == 0 {
			if t, ok := seenNamespaceOrMeshGlobalConfig[config.Namespace]; ok {
				log.Warnf(
					"Namespace/mesh-level PeerAuthentication is already defined for %q at time %v. Ignore %q which was created at time %v",
					config.Namespace, t, config.Name, config.CreationTimestamp)
				continue
			}
			seenNamespaceOrMeshGlobalConfig[config.Namespace] = config.CreationTimestamp

			mode := v1alpha3.PeerAuthentication_MutualTLS_UNSET
			if spec.Mtls != nil {
				mode = spec.Mtls.Mode
			}
			if config.Namespace == policy.rootNamespace {
				if mode == v1alpha3.PeerAuthentication_MutualTLS_UNSET {
					policy.globalMutualTLSMode = ConvertToMutualTLSMode(mode)
				}
			} else {
				foundNamespaceMTLS[config.Namespace] = mode
			}
		}

		policy.peerAuthentications[config.Namespace] = append(policy.peerAuthentications[config.Namespace], config)
	}

	// Not security sensitive code
	policy.aggregateVersion = fmt.Sprintf("%x", md5.Sum([]byte(strings.Join(versions, ";"))))

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

func ConvertToMutualTLSMode(mode v1alpha3.PeerAuthentication_MutualTLS_Mode) MutualTLSMode {
	switch mode {
	case v1alpha3.PeerAuthentication_MutualTLS_DISABLE:
		return MTLSDisable
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
