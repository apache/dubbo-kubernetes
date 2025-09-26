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

package mesh

import (
	"fmt"
	"github.com/apache/dubbo-kubernetes/pkg/config/constants"
	"github.com/apache/dubbo-kubernetes/pkg/ptr"
	"github.com/apache/dubbo-kubernetes/pkg/util/pointer"
	"github.com/apache/dubbo-kubernetes/pkg/util/protomarshal"
	"github.com/apache/dubbo-kubernetes/pkg/util/sets"
	"github.com/hashicorp/go-multierror"
	"google.golang.org/protobuf/types/known/durationpb"
	wrappers "google.golang.org/protobuf/types/known/wrapperspb"
	meshconfig "istio.io/api/mesh/v1alpha1"
	"istio.io/api/networking/v1alpha3"
	"os"
	"sigs.k8s.io/yaml"
	"time"
)

// DefaultMeshNetworks returns a default meshnetworks configuration.
// By default, it is empty.
func DefaultMeshNetworks() *meshconfig.MeshNetworks {
	return ptr.Of(EmptyMeshNetworks())
}

func DefaultProxyConfig() *meshconfig.ProxyConfig {
	return &meshconfig.ProxyConfig{
		ConfigPath:               constants.ConfigPathDir,
		ClusterName:              &meshconfig.ProxyConfig_ServiceCluster{ServiceCluster: constants.ServiceClusterName},
		DrainDuration:            durationpb.New(45 * time.Second),
		TerminationDrainDuration: durationpb.New(5 * time.Second),
		ProxyAdminPort:           15000,
		// TODO authpolicy
		DiscoveryAddress:       "dubbod.dubbo-system.svc:15012",
		ControlPlaneAuthPolicy: meshconfig.AuthenticationPolicy_MUTUAL_TLS,
		StatNameLength:         189,
		StatusPort:             15020,
	}
}

func ReadMeshConfig(filename string) (*meshconfig.MeshConfig, error) {
	yaml, err := os.ReadFile(filename)
	if err != nil {
		return nil, multierror.Prefix(err, "cannot read mesh config file")
	}
	return ApplyMeshConfigDefaults(string(yaml))
}

func ApplyMeshConfigDefaults(yaml string) (*meshconfig.MeshConfig, error) {
	return ApplyMeshConfig(yaml, DefaultMeshConfig())
}
func ApplyMeshConfig(yaml string, defaultConfig *meshconfig.MeshConfig) (*meshconfig.MeshConfig, error) {
	prevProxyConfig := defaultConfig.DefaultConfig
	prevDefaultProvider := defaultConfig.DefaultProviders
	prevExtensionProviders := defaultConfig.ExtensionProviders
	prevTrustDomainAliases := defaultConfig.TrustDomainAliases

	defaultConfig.DefaultConfig = DefaultProxyConfig()
	if err := protomarshal.ApplyYAML(yaml, defaultConfig); err != nil {
		return nil, multierror.Prefix(err, "failed to convert to proto.")
	}
	defaultConfig.DefaultConfig = prevProxyConfig

	raw, err := toMap(yaml)
	if err != nil {
		return nil, err
	}
	pc, err := extractYamlField("defaultConfig", raw)
	if err != nil {
		return nil, multierror.Prefix(err, "failed to extract proxy config")
	}
	if pc != "" {
		pc, err := MergeProxyConfig(pc, defaultConfig.DefaultConfig)
		if err != nil {
			return nil, err
		}
		defaultConfig.DefaultConfig = pc
	}

	defaultConfig.DefaultProviders = prevDefaultProvider
	dp, err := extractYamlField("defaultProviders", raw)
	if err != nil {
		return nil, multierror.Prefix(err, "failed to extract default providers")
	}
	if dp != "" {
		if err := protomarshal.ApplyYAML(dp, defaultConfig.DefaultProviders); err != nil {
			return nil, fmt.Errorf("could not parse default providers: %v", err)
		}
	}

	newExtensionProviders := defaultConfig.ExtensionProviders
	defaultConfig.ExtensionProviders = prevExtensionProviders
	for _, p := range newExtensionProviders {
		found := false
		for _, e := range defaultConfig.ExtensionProviders {
			if p.Name == e.Name {
				e.Provider = p.Provider
				found = true
				break
			}
		}
		if !found {
			defaultConfig.ExtensionProviders = append(defaultConfig.ExtensionProviders, p)
		}
	}

	defaultConfig.TrustDomainAliases = sets.SortedList(sets.New(append(defaultConfig.TrustDomainAliases, prevTrustDomainAliases...)...))
	// TODO ValidationMeshConfig
	return defaultConfig, nil
}

func ApplyProxyConfig(yaml string, meshConfig *meshconfig.MeshConfig) (*meshconfig.MeshConfig, error) {
	mc := protomarshal.Clone(meshConfig)
	pc, err := MergeProxyConfig(yaml, mc.DefaultConfig)
	if err != nil {
		return nil, err
	}
	mc.DefaultConfig = pc
	return mc, nil
}

// EmptyMeshNetworks configuration with no networks
func EmptyMeshNetworks() meshconfig.MeshNetworks {
	return meshconfig.MeshNetworks{
		Networks: map[string]*meshconfig.Network{},
	}
}

// ParseMeshNetworks returns a new MeshNetworks decoded from the
// input YAML.
func ParseMeshNetworks(yaml string) (*meshconfig.MeshNetworks, error) {
	out := EmptyMeshNetworks()
	if err := protomarshal.ApplyYAML(yaml, &out); err != nil {
		return nil, multierror.Prefix(err, "failed to convert to proto.")
	}

	// if err := agent.ValidateMeshNetworks(&out); err != nil {
	// 	return nil, err
	// }
	return &out, nil
}

func DefaultMeshConfig() *meshconfig.MeshConfig {
	proxyConfig := DefaultProxyConfig()
	return &meshconfig.MeshConfig{
		EnableTracing:               true,
		AccessLogFile:               "",
		AccessLogEncoding:           meshconfig.MeshConfig_TEXT,
		AccessLogFormat:             "",
		EnableEnvoyAccessLogService: false,
		ProtocolDetectionTimeout:    durationpb.New(0),
		TrustDomain:                 constants.DefaultClusterLocalDomain,
		TrustDomainAliases:          []string{},
		EnableAutoMtls:              wrappers.Bool(true),
		OutboundTrafficPolicy:       &meshconfig.MeshConfig_OutboundTrafficPolicy{Mode: meshconfig.MeshConfig_OutboundTrafficPolicy_ALLOW_ANY},
		InboundTrafficPolicy:        &meshconfig.MeshConfig_InboundTrafficPolicy{Mode: meshconfig.MeshConfig_InboundTrafficPolicy_PASSTHROUGH},
		LocalityLbSetting: &v1alpha3.LocalityLoadBalancerSetting{
			Enabled: wrappers.Bool(true),
		},
		Certificates:  []*meshconfig.Certificate{},
		DefaultConfig: proxyConfig,

		RootNamespace:                  constants.DubboSystemNamespace,
		ProxyListenPort:                15001,
		ProxyInboundListenPort:         15006,
		ConnectTimeout:                 durationpb.New(10 * time.Second),
		DefaultServiceExportTo:         []string{"*"},
		DefaultVirtualServiceExportTo:  []string{"*"},
		DefaultDestinationRuleExportTo: []string{"*"},
		DnsRefreshRate:                 durationpb.New(60 * time.Second),
		DefaultProviders:               &meshconfig.MeshConfig_DefaultProviders{},
	}
}

func MergeProxyConfig(yaml string, proxyConfig *meshconfig.ProxyConfig) (*meshconfig.ProxyConfig, error) {
	origMetadata := proxyConfig.ProxyMetadata
	origProxyHeaders := proxyConfig.ProxyHeaders
	if err := protomarshal.ApplyYAML(yaml, proxyConfig); err != nil {
		return nil, fmt.Errorf("could not parse proxy config: %v", err)
	}
	newMetadata := proxyConfig.ProxyMetadata
	proxyConfig.ProxyMetadata = mergeMap(origMetadata, newMetadata)
	correctProxyHeaders(proxyConfig, origProxyHeaders)
	return proxyConfig, nil
}

func correctProxyHeaders(proxyConfig *meshconfig.ProxyConfig, orig *meshconfig.ProxyConfig_ProxyHeaders) {
	ph := proxyConfig.ProxyHeaders
	if ph != nil && orig != nil {
		ph.ForwardedClientCert = pointer.NonEmptyOrDefault(ph.ForwardedClientCert, orig.ForwardedClientCert)
		ph.RequestId = pointer.NonEmptyOrDefault(ph.RequestId, orig.RequestId)
		ph.AttemptCount = pointer.NonEmptyOrDefault(ph.AttemptCount, orig.AttemptCount)
		ph.Server = pointer.NonEmptyOrDefault(ph.Server, orig.Server)
		ph.EnvoyDebugHeaders = pointer.NonEmptyOrDefault(ph.EnvoyDebugHeaders, orig.EnvoyDebugHeaders)
	}
}
func extractYamlField(key string, mp map[string]any) (string, error) {
	proxyConfig := mp[key]
	if proxyConfig == nil {
		return "", nil
	}
	bytes, err := yaml.Marshal(proxyConfig)
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}

func toMap(yamlText string) (map[string]any, error) {
	mp := map[string]any{}
	if err := yaml.Unmarshal([]byte(yamlText), &mp); err != nil {
		return nil, err
	}
	return mp, nil
}

func mergeMap(original map[string]string, merger map[string]string) map[string]string {
	if original == nil && merger == nil {
		return nil
	}
	if original == nil {
		original = map[string]string{}
	}
	for k, v := range merger {
		original[k] = v
	}
	return original
}
