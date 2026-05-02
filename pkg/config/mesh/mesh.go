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

package mesh

import (
	"fmt"
	"os"
	"time"

	"github.com/apache/dubbo-kubernetes/pkg/config/constants"
	"github.com/apache/dubbo-kubernetes/pkg/util/protomarshal"
	"github.com/apache/dubbo-kubernetes/pkg/util/sets"
	"github.com/hashicorp/go-multierror"
	meshv1alpha1 "github.com/kdubbo/api/mesh/v1alpha1"
	"google.golang.org/protobuf/types/known/durationpb"
	"sigs.k8s.io/yaml"
)

func ReadMeshConfig(filename string) (*meshv1alpha1.MeshConfig, error) {
	yaml, err := os.ReadFile(filename)
	if err != nil {
		return nil, multierror.Prefix(err, "cannot read mesh config file")
	}
	return ApplyMeshConfigDefaults(string(yaml))
}

func ApplyMeshConfig(yaml string, defaultConfig *meshv1alpha1.MeshConfig) (*meshv1alpha1.MeshConfig, error) {
	prevProxyConfig := defaultConfig.DefaultConfig
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

	defaultConfig.TrustDomainAliases = sets.SortedList(sets.New(append(defaultConfig.TrustDomainAliases, prevTrustDomainAliases...)...))
	// TODO ValidationMeshConfig
	return defaultConfig, nil
}

func ApplyMeshConfigDefaults(yaml string) (*meshv1alpha1.MeshConfig, error) {
	return ApplyMeshConfig(yaml, DefaultMeshConfig())
}

func ApplyProxyConfig(yaml string, meshConfig *meshv1alpha1.MeshConfig) (*meshv1alpha1.MeshConfig, error) {
	mc := protomarshal.Clone(meshConfig)
	pc, err := MergeProxyConfig(yaml, mc.DefaultConfig)
	if err != nil {
		return nil, err
	}
	mc.DefaultConfig = pc
	return mc, nil
}

func MergeProxyConfig(yaml string, proxyConfig *meshv1alpha1.ProxyConfig) (*meshv1alpha1.ProxyConfig, error) {
	origMetadata := proxyConfig.ProxyMetadata
	if err := protomarshal.ApplyYAML(yaml, proxyConfig); err != nil {
		return nil, fmt.Errorf("could not parse proxy config: %v", err)
	}
	newMetadata := proxyConfig.ProxyMetadata
	proxyConfig.ProxyMetadata = mergeMap(origMetadata, newMetadata)
	return proxyConfig, nil
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

func DefaultMeshConfig() *meshv1alpha1.MeshConfig {
	proxyConfig := DefaultProxyConfig()
	return &meshv1alpha1.MeshConfig{
		TrustDomain:        constants.DefaultClusterLocalDomain,
		TrustDomainAliases: []string{},
		Certificates:       []*meshv1alpha1.Certificate{},
		DefaultConfig:      proxyConfig,

		RootNamespace:                  constants.DubboSystemNamespace,
		ConnectTimeout:                 durationpb.New(10 * time.Second),
		DefaultServiceExportTo:         []string{"*"},
		DefaultVirtualServiceExportTo:  []string{"*"},
		DefaultDestinationRuleExportTo: []string{"*"},
		DnsRefreshRate:                 durationpb.New(60 * time.Second),
	}
}

func DefaultProxyConfig() *meshv1alpha1.ProxyConfig {
	return &meshv1alpha1.ProxyConfig{
		ConfigPath:             constants.ConfigPathDir,
		DiscoveryAddress:       "dubbod.dubbo-system.svc:26012",
		ControlPlaneAuthPolicy: meshv1alpha1.AuthenticationPolicy_MUTUAL_TLS,
		StatusPort:             26020,
	}
}
