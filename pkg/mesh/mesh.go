package mesh

import (
	"fmt"
	"github.com/apache/dubbo-kubernetes/pkg/util/pointer"
	"github.com/apache/dubbo-kubernetes/pkg/util/sets"
	"github.com/hashicorp/go-multierror"
	meshconfig "istio.io/api/mesh/v1alpha1"
	"istio.io/istio/pkg/config/validation/agent"
	"istio.io/istio/pkg/util/protomarshal"
	"os"
	"sigs.k8s.io/yaml"
)

func DefaultProxyConfig() *meshconfig.ProxyConfig {
	return &meshconfig.ProxyConfig{}
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
	// We want to keep semantics that all fields are overrides, except proxy config is a merge. This allows
	// decent customization while also not requiring users to redefine the entire proxy config if they want to override
	// Note: if we want to add more structure in the future, we will likely need to revisit this idea.

	// Store the current set proxy config so we don't wipe it out, we will configure this later
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
	// Get just the proxy config yaml
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

	warn, err := agent.ValidateMeshConfig(defaultConfig)
	if err != nil {
		return nil, err
	}
	if warn != nil {
		fmt.Printf("warnings occurred during mesh validation: %v", warn)
	}

	return defaultConfig, nil
}

func DefaultMeshConfig() *meshconfig.MeshConfig {
	proxyConfig := DefaultProxyConfig()

	// Defaults matching the standard install
	// order matches the generated mesh config.
	return &meshconfig.MeshConfig{
		DefaultConfig: proxyConfig,
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
