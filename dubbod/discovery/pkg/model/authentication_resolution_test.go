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
	"testing"

	"github.com/apache/dubbo-kubernetes/pkg/config"
	"github.com/apache/dubbo-kubernetes/pkg/config/schema/gvk"
	security "github.com/kdubbo/api/security/v1alpha3"
	typev1alpha3 "github.com/kdubbo/api/type/v1alpha3"
)

func selectorFromLabels(labels map[string]string) *typev1alpha3.WorkloadSelector {
	if len(labels) == 0 {
		return nil
	}
	return &typev1alpha3.WorkloadSelector{MatchLabels: labels}
}

// peerAuth builds a PeerAuthentication config. A nil selector produces a
// namespace/mesh-level policy; a non-nil one produces a workload-level policy.
func peerAuth(namespace, name string, selector map[string]string, mtls security.PeerAuthentication_MutualTLS_Mode, portLevel map[uint32]security.PeerAuthentication_MutualTLS_Mode) config.Config {
	spec := &security.PeerAuthentication{}
	if mtls != security.PeerAuthentication_MutualTLS_UNSET {
		spec.Mtls = &security.PeerAuthentication_MutualTLS{Mode: mtls}
	}
	if sel := selectorFromLabels(selector); sel != nil {
		spec.Selector = sel
	}
	if len(portLevel) > 0 {
		spec.PortLevelMtls = map[uint32]*security.PeerAuthentication_MutualTLS{}
		for port, mode := range portLevel {
			spec.PortLevelMtls[port] = &security.PeerAuthentication_MutualTLS{Mode: mode}
		}
	}
	return config.Config{
		Meta: config.Meta{GroupVersionKind: gvk.PeerAuthentication, Name: name, Namespace: namespace},
		Spec: spec,
	}
}

func newPolicies(rootNamespace string, configs ...config.Config) *AuthenticationPolicies {
	policy := &AuthenticationPolicies{
		requestAuthentications: map[string][]config.Config{},
		authorizationPolicies:  map[string][]config.Config{},
		peerAuthentications:    map[string][]config.Config{},
		globalMutualTLSMode:    MTLSUnknown,
		rootNamespace:          rootNamespace,
		namespaceMutualTLSMode: map[string]MutualTLSMode{},
	}
	policy.addPeerAuthentication(configs)
	return policy
}

func TestConvertToMutualTLSModeAllModes(t *testing.T) {
	cases := map[security.PeerAuthentication_MutualTLS_Mode]MutualTLSMode{
		security.PeerAuthentication_MutualTLS_DISABLE:    MTLSDisable,
		security.PeerAuthentication_MutualTLS_PERMISSIVE: MTLSPermissive,
		security.PeerAuthentication_MutualTLS_STRICT:     MTLSStrict,
		security.PeerAuthentication_MutualTLS_UNSET:      MTLSUnknown,
	}
	for mode, want := range cases {
		if got := ConvertToMutualTLSMode(mode); got != want {
			t.Errorf("ConvertToMutualTLSMode(%v) = %v, want %v", mode, got, want)
		}
	}
}

func TestEffectiveMutualTLSModePrecedence(t *testing.T) {
	const root = "dubbo-system"
	const ns = "app"
	labels := map[string]string{"app": "httpbin"}

	tests := []struct {
		name    string
		configs []config.Config
		labels  map[string]string
		port    uint32
		want    MutualTLSMode
	}{
		{
			name: "workload selector beats namespace default",
			configs: []config.Config{
				peerAuth(ns, "ns-default", nil, security.PeerAuthentication_MutualTLS_PERMISSIVE, nil),
				peerAuth(ns, "wl", labels, security.PeerAuthentication_MutualTLS_STRICT, nil),
			},
			labels: labels,
			port:   8080,
			want:   MTLSStrict,
		},
		{
			name: "namespace default beats mesh global",
			configs: []config.Config{
				peerAuth(root, "mesh", nil, security.PeerAuthentication_MutualTLS_STRICT, nil),
				peerAuth(ns, "ns-default", nil, security.PeerAuthentication_MutualTLS_DISABLE, nil),
			},
			labels: labels,
			port:   8080,
			want:   MTLSDisable,
		},
		{
			name: "mesh global applies when namespace has none",
			configs: []config.Config{
				peerAuth(root, "mesh", nil, security.PeerAuthentication_MutualTLS_STRICT, nil),
			},
			labels: labels,
			port:   8080,
			want:   MTLSStrict,
		},
		{
			name: "port level override wins over workload spec level",
			configs: []config.Config{
				peerAuth(ns, "wl", labels, security.PeerAuthentication_MutualTLS_STRICT,
					map[uint32]security.PeerAuthentication_MutualTLS_Mode{8080: security.PeerAuthentication_MutualTLS_DISABLE}),
			},
			labels: labels,
			port:   8080,
			want:   MTLSDisable,
		},
		{
			name: "port level ignored for a different port",
			configs: []config.Config{
				peerAuth(ns, "wl", labels, security.PeerAuthentication_MutualTLS_STRICT,
					map[uint32]security.PeerAuthentication_MutualTLS_Mode{9090: security.PeerAuthentication_MutualTLS_DISABLE}),
			},
			labels: labels,
			port:   8080,
			want:   MTLSStrict,
		},
		{
			name:    "no policy resolves to unknown",
			configs: nil,
			labels:  labels,
			port:    8080,
			want:    MTLSUnknown,
		},
		{
			name: "empty namespace falls through to mesh global",
			configs: []config.Config{
				peerAuth(root, "mesh", nil, security.PeerAuthentication_MutualTLS_PERMISSIVE, nil),
			},
			labels: labels,
			port:   0,
			want:   MTLSPermissive,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			namespace := ns
			if tt.name == "empty namespace falls through to mesh global" {
				namespace = ""
			}
			policy := newPolicies(root, tt.configs...)
			if got := policy.EffectiveMutualTLSMode(namespace, tt.labels, tt.port); got != tt.want {
				t.Fatalf("EffectiveMutualTLSMode = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestEffectiveMutualTLSModeRootNamespaceWorkloadSelector(t *testing.T) {
	const root = "dubbo-system"
	labels := map[string]string{"app": "httpbin"}
	// A workload-selector PeerAuthentication living in the root namespace applies
	// to matching workloads in other namespaces when their own namespace has no
	// matching policy.
	policy := newPolicies(root,
		peerAuth(root, "root-wl", labels, security.PeerAuthentication_MutualTLS_STRICT, nil),
	)
	if got := policy.EffectiveMutualTLSMode("app", labels, 8080); got != MTLSStrict {
		t.Fatalf("effective mode = %v, want MTLSStrict (root ns workload selector)", got)
	}
	// A workload in the same namespaces whose labels do not match falls through
	// to unknown.
	if got := policy.EffectiveMutualTLSMode("app", map[string]string{"app": "other"}, 8080); got != MTLSUnknown {
		t.Fatalf("non-matching mode = %v, want MTLSUnknown", got)
	}
}

func TestEffectiveMutualTLSModeNilPolicy(t *testing.T) {
	var policy *AuthenticationPolicies
	if got := policy.EffectiveMutualTLSMode("app", nil, 80); got != MTLSUnknown {
		t.Fatalf("nil policy mode = %v, want MTLSUnknown", got)
	}
}

func TestNamespaceLevelInheritsGlobalWhenUnset(t *testing.T) {
	const root = "dubbo-system"
	// A namespace-level PeerAuthentication with no mtls mode (UNSET) inherits
	// the mesh-global mode rather than resetting to unknown.
	policy := newPolicies(root,
		peerAuth(root, "mesh", nil, security.PeerAuthentication_MutualTLS_STRICT, nil),
		peerAuth("app", "ns-unset", nil, security.PeerAuthentication_MutualTLS_UNSET, nil),
	)
	if got := policy.namespaceMutualTLSMode["app"]; got != MTLSStrict {
		t.Fatalf("inherited namespace mode = %v, want MTLSStrict", got)
	}
}

func TestDuplicateNamespaceLevelPeerAuthIgnored(t *testing.T) {
	const ns = "app"
	// Two namespace-level policies in the same namespace: the first by creation
	// order wins and the second must be ignored.
	first := peerAuth(ns, "first", nil, security.PeerAuthentication_MutualTLS_STRICT, nil)
	second := peerAuth(ns, "second", nil, security.PeerAuthentication_MutualTLS_DISABLE, nil)
	policy := newPolicies("dubbo-system", first, second)

	if got := policy.EffectiveMutualTLSMode(ns, nil, 0); got != MTLSStrict {
		t.Fatalf("effective mode = %v, want MTLSStrict (first policy wins)", got)
	}
}

func TestPeerAuthenticationModeForPort(t *testing.T) {
	specStrict := &security.PeerAuthentication{
		Mtls: &security.PeerAuthentication_MutualTLS{Mode: security.PeerAuthentication_MutualTLS_STRICT},
		PortLevelMtls: map[uint32]*security.PeerAuthentication_MutualTLS{
			8080: {Mode: security.PeerAuthentication_MutualTLS_DISABLE},
			9090: {Mode: security.PeerAuthentication_MutualTLS_UNSET},
		},
	}

	tests := []struct {
		name string
		spec *security.PeerAuthentication
		port uint32
		want MutualTLSMode
	}{
		{"nil spec", nil, 8080, MTLSUnknown},
		{"port level override", specStrict, 8080, MTLSDisable},
		{"port level unset falls back to spec", specStrict, 9090, MTLSStrict},
		{"unmapped port uses spec level", specStrict, 7070, MTLSStrict},
		{"port zero ignores port level", specStrict, 0, MTLSStrict},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := peerAuthenticationModeForPort(tt.spec, tt.port); got != tt.want {
				t.Fatalf("peerAuthenticationModeForPort = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSelectorMatchesWorkload(t *testing.T) {
	tests := []struct {
		name     string
		selector map[string]string
		labels   map[string]string
		want     bool
	}{
		{"empty selector matches all", nil, map[string]string{"app": "x"}, true},
		{"empty labels with selector fails", map[string]string{"app": "x"}, nil, false},
		{"exact match", map[string]string{"app": "x"}, map[string]string{"app": "x", "v": "1"}, true},
		{"value mismatch", map[string]string{"app": "x"}, map[string]string{"app": "y"}, false},
		{"missing key", map[string]string{"app": "x", "tier": "t"}, map[string]string{"app": "x"}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := selectorMatchesWorkload(selectorFromLabels(tt.selector), tt.labels); got != tt.want {
				t.Fatalf("selectorMatchesWorkload = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestHasPeerAuthSelector(t *testing.T) {
	if hasPeerAuthSelector(nil) {
		t.Fatal("nil spec should have no selector")
	}
	noSel := &security.PeerAuthentication{}
	if hasPeerAuthSelector(noSel) {
		t.Fatal("spec without selector should report false")
	}
	withSel := &security.PeerAuthentication{Selector: selectorFromLabels(map[string]string{"app": "x"})}
	if !hasPeerAuthSelector(withSel) {
		t.Fatal("spec with selector should report true")
	}
}
