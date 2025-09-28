package model

import (
	"crypto/md5"
	"fmt"
	"github.com/apache/dubbo-kubernetes/pkg/config"
	"github.com/apache/dubbo-kubernetes/pkg/config/schema/gvk"
	"istio.io/api/security/v1beta1"
	"k8s.io/klog/v2"
	"strings"
	"time"
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

	policy.addRequestAuthentication(sortConfigByCreationTime(env.List(gvk.RequestAuthentication, NamespaceAll)))
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

	foundNamespaceMTLS := make(map[string]v1beta1.PeerAuthentication_MutualTLS_Mode)
	seenNamespaceOrMeshConfig := make(map[string]time.Time)
	versions := []string{}

	for _, config := range configs {
		versions = append(versions, config.UID+"."+config.ResourceVersion)
		spec := config.Spec.(*v1beta1.PeerAuthentication)
		if spec.Selector == nil || len(spec.Selector.MatchLabels) == 0 {
			if t, ok := seenNamespaceOrMeshConfig[config.Namespace]; ok {
				klog.Warningf(
					"Namespace/mesh-level PeerAuthentication is already defined for %q at time %v. Ignore %q which was created at time %v",
					config.Namespace, t, config.Name, config.CreationTimestamp)
				continue
			}
			seenNamespaceOrMeshConfig[config.Namespace] = config.CreationTimestamp

			mode := v1beta1.PeerAuthentication_MutualTLS_UNSET
			if spec.Mtls != nil {
				mode = spec.Mtls.Mode
			}
			if config.Namespace == policy.rootNamespace {
				if mode == v1beta1.PeerAuthentication_MutualTLS_UNSET {
					policy.globalMutualTLSMode = MTLSPermissive
				} else {
					policy.globalMutualTLSMode = ConvertToMutualTLSMode(mode)
				}
			} else {
				foundNamespaceMTLS[config.Namespace] = mode
			}
		}

		policy.peerAuthentications[config.Namespace] = append(policy.peerAuthentications[config.Namespace], config)
	}

	// nolint: gosec
	// Not security sensitive code
	policy.aggregateVersion = fmt.Sprintf("%x", md5.Sum([]byte(strings.Join(versions, ";"))))

	policy.namespaceMutualTLSMode = make(map[string]MutualTLSMode, len(foundNamespaceMTLS))

	inheritedMTLSMode := policy.globalMutualTLSMode
	if inheritedMTLSMode == MTLSUnknown {
		inheritedMTLSMode = MTLSPermissive
	}
	for ns, mtlsMode := range foundNamespaceMTLS {
		if mtlsMode == v1beta1.PeerAuthentication_MutualTLS_UNSET {
			policy.namespaceMutualTLSMode[ns] = inheritedMTLSMode
		} else {
			policy.namespaceMutualTLSMode[ns] = ConvertToMutualTLSMode(mtlsMode)
		}
	}
}

func ConvertToMutualTLSMode(mode v1beta1.PeerAuthentication_MutualTLS_Mode) MutualTLSMode {
	switch mode {
	case v1beta1.PeerAuthentication_MutualTLS_DISABLE:
		return MTLSDisable
	case v1beta1.PeerAuthentication_MutualTLS_PERMISSIVE:
		return MTLSPermissive
	case v1beta1.PeerAuthentication_MutualTLS_STRICT:
		return MTLSStrict
	default:
		return MTLSUnknown
	}
}
