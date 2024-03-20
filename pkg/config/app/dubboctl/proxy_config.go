package dubboctl

import (
	"github.com/apache/dubbo-kubernetes/pkg/config"
	config_types "github.com/apache/dubbo-kubernetes/pkg/config/types"
)

type DataplaneRuntime struct {
	config.BaseConfig

	// Path to Envoy binary.
	BinaryPath string `json:"binaryPath,omitempty" envconfig:"dubbo_dataplane_runtime_binary_path"`
	// Dir to store auto-generated Envoy bootstrap config in.
	ConfigDir string `json:"configDir,omitempty" envconfig:"dubbo_dataplane_runtime_config_dir"`
	// Concurrency specifies how to generate the Envoy concurrency flag.
	Concurrency uint32 `json:"concurrency,omitempty" envconfig:"dubbo_dataplane_runtime_concurrency"`
	// Path to a file with dataplane token (use 'dubboctl generate dataplane-token' to get one)
	TokenPath string `json:"dataplaneTokenPath,omitempty" envconfig:"dubbo_dataplane_runtime_token_path"`
	// Token is dataplane token's value provided directly, will be stored to a temporary file before applying
	Token string `json:"dataplaneToken,omitempty" envconfig:"dubbo_dataplane_runtime_token"`
	// Resource is a Dataplane resource that will be applied on Dubbo CP
	Resource string `json:"resource,omitempty" envconfig:"dubbo_dataplane_runtime_resource"`
	// ResourcePath is a path to Dataplane resource that will be applied on Dubbo CP
	ResourcePath string `json:"resourcePath,omitempty" envconfig:"dubbo_dataplane_runtime_resource_path"`
	// ResourceVars are the StringToString values that can fill the Resource template
	ResourceVars map[string]string `json:"resourceVars,omitempty"`
	// EnvoyLogLevel is a level on which Envoy will log.
	// Available values are: [trace][debug][info][warning|warn][error][critical][off]
	// By default it inherits Dubbo DP logging level.
	EnvoyLogLevel string `json:"envoyLogLevel,omitempty" envconfig:"dubbo_dataplane_runtime_envoy_log_level"`
	// EnvoyComponentLogLevel configures Envoy's --component-log-level and uses
	// the exact same syntax: https://www.envoyproxy.io/docs/envoy/latest/operations/cli#cmdoption-component-log-level
	EnvoyComponentLogLevel string `json:"envoyComponentLogLevel,omitempty" envconfig:"dubbo_dataplane_runtime_envoy_component_log_level"`
	// Resources defines the resources for this proxy.
	Resources DataplaneResources `json:"resources,omitempty"`
	// SocketDir dir to store socket used between Envoy and the dp process
	SocketDir string `json:"socketDir,omitempty" envconfig:"dubbo_dataplane_runtime_socket_dir"`
	// Metrics defines properties of metrics
	Metrics Metrics `json:"metrics,omitempty"`
	// DynamicConfiguration defines properties of dataplane dynamic configuration
	DynamicConfiguration DynamicConfiguration `json:"dynamicConfiguration" envconfig:"dubbo_dataplane_runtime_dynamic_configuration"`
}
type ProxyConfig struct {
	DataplaneRuntime DataplaneRuntime
	Dataplane        Dataplane
}

func (p *ProxyConfig) SetDefault() {
	p.DataplaneRuntime.BinaryPath = "/go/src/test/envoy"
	p.DataplaneRuntime.EnvoyLogLevel = "debug"
}

type Metrics struct {
	// CertPath path to the certificate for metrics listener
	CertPath string `json:"metricsCertPath,omitempty" envconfig:"dubbo_dataplane_runtime_metrics_cert_path"`
	// KeyPath path to the key for metrics listener
	KeyPath string `json:"metricsKeyPath,omitempty" envconfig:"dubbo_dataplane_runtime_metrics_key_path"`
}

type DynamicConfiguration struct {
	// RefreshInterval defines how often DPP should refresh dynamic config. Default: 10s
	RefreshInterval config_types.Duration `json:"refreshInterval,omitempty" envconfig:"dubbo_dataplane_runtime_dynamic_configuration_refresh_interval"`
}

// DataplaneResources defines the resources available to a dataplane proxy.
type DataplaneResources struct {
	MaxMemoryBytes uint64 `json:"maxMemoryBytes,omitempty" envconfig:"dubbo_dataplane_resources_max_memory_bytes"`
}
type Dataplane struct {
	config.BaseConfig

	// Mesh name.
	Mesh string `json:"mesh,omitempty" envconfig:"dubbo_dataplane_mesh"`
	// Dataplane name.
	Name string `json:"name,omitempty" envconfig:"dubbo_dataplane_name"`
	// ProxyType defines mode which should be used, supported values: 'dataplane', 'ingress'
	ProxyType string `json:"proxyType,omitempty" envconfig:"dubbo_dataplane_proxy_type"`
	// Drain time for listeners.
	DrainTime config_types.Duration `json:"drainTime,omitempty" envconfig:"dubbo_dataplane_drain_time"`
}
