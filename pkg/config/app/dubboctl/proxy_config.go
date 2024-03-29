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

package dubboctl

import (
	mesh_proto "github.com/apache/dubbo-kubernetes/api/mesh/v1alpha1"
	"time"
)

import (
	"github.com/apache/dubbo-kubernetes/pkg/config"
	config_types "github.com/apache/dubbo-kubernetes/pkg/config/types"
)

var DefaultConfig = func() Config {
	return Config{
		ControlPlane: ControlPlane{
			URL: "https://localhost:5678",
			Retry: CpRetry{
				Backoff:     config_types.Duration{Duration: 3 * time.Second},
				MaxDuration: config_types.Duration{Duration: 5 * time.Minute}, // this value can be fairy long since what will happen when there is a connection error is that the Dataplane will be restarted (by process manager like systemd/K8S etc.) and will try to connect again.
			},
		},
		Dataplane: Dataplane{
			Mesh:      "",
			Name:      "", // Dataplane name must be set explicitly
			DrainTime: config_types.Duration{Duration: 30 * time.Second},
			ProxyType: "dataplane",
		},
		DataplaneRuntime: DataplaneRuntime{
			BinaryPath: "envoy",
			ConfigDir:  "", // if left empty, a temporary directory will be generated automatically
			DynamicConfiguration: DynamicConfiguration{
				RefreshInterval: config_types.Duration{Duration: 10 * time.Second},
			},
		},
		DNS: DNS{
			Enabled:                   true,
			CoreDNSPort:               15053,
			EnvoyDNSPort:              15054,
			CoreDNSEmptyPort:          15055,
			CoreDNSBinaryPath:         "coredns",
			CoreDNSConfigTemplatePath: "",
			ConfigDir:                 "", // if left empty, a temporary directory will be generated automatically
			PrometheusPort:            19153,
			CoreDNSLogging:            false,
		},
	}
}

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

type Config struct {
	ControlPlane ControlPlane `json:"controlPlane,omitempty"`
	// Dataplane defines bootstrap configuration of the dataplane (Envoy).
	Dataplane Dataplane `json:"dataplane,omitempty"`
	// DataplaneRuntime defines the context in which dataplane (Envoy) runs.
	DataplaneRuntime DataplaneRuntime `json:"dataplaneRuntime,omitempty"`
	// DNS defines a configuration for builtin DNS in Dubbo DP
	DNS DNS `json:"dns,omitempty"`
}
type ControlPlane struct {
	// URL defines the address of Control Plane DP server.
	URL string `json:"url,omitempty" envconfig:"dubbo_control_plane_url"`
	// Retry settings for Control Plane communication
	Retry CpRetry `json:"retry,omitempty"`
	// CaCert defines Certificate Authority that will be used to verify connection to the Control Plane. It takes precedence over CaCertFile.
	CaCert string `json:"caCert" envconfig:"dubbo_control_plane_ca_cert"`
	// CaCertFile defines a file for Certificate Authority that will be used to verify connection to the Control Plane.
	CaCertFile string `json:"caCertFile" envconfig:"dubbo_control_plane_ca_cert_file"`
}
type CpRetry struct {
	config.BaseConfig

	// Duration to wait between retries
	Backoff config_types.Duration `json:"backoff,omitempty" envconfig:"dubbo_control_plane_retry_backoff"`
	// Max duration for retries (this is not exact time for execution, the check is done between retries)
	MaxDuration config_types.Duration `json:"maxDuration,omitempty" envconfig:"dubbo_control_plane_retry_max_duration"`
}
type DNS struct {
	config.BaseConfig

	// If true then builtin DNS functionality is enabled and CoreDNS server is started
	Enabled bool `json:"enabled,omitempty" envconfig:"dubbo_dns_enabled"`
	// CoreDNSPort defines a port that handles DNS requests. When transparent proxy is enabled then iptables will redirect DNS traffic to this port.
	CoreDNSPort uint32 `json:"coreDnsPort,omitempty" envconfig:"dubbo_dns_core_dns_port"`
	// CoreDNSEmptyPort defines a port that always responds with empty NXDOMAIN respond. It is required to implement a fallback to a real DNS
	CoreDNSEmptyPort uint32 `json:"coreDnsEmptyPort,omitempty" envconfig:"dubbo_dns_core_dns_empty_port"`
	// EnvoyDNSPort defines a port that handles Virtual IP resolving by Envoy. CoreDNS should be configured that it first tries to use this DNS resolver and then the real one.
	EnvoyDNSPort uint32 `json:"envoyDnsPort,omitempty" envconfig:"dubbo_dns_envoy_dns_port"`
	// CoreDNSBinaryPath defines a path to CoreDNS binary.
	CoreDNSBinaryPath string `json:"coreDnsBinaryPath,omitempty" envconfig:"dubbo_dns_core_dns_binary_path"`
	// CoreDNSConfigTemplatePath defines a path to a CoreDNS config template.
	CoreDNSConfigTemplatePath string `json:"coreDnsConfigTemplatePath,omitempty" envconfig:"dubbo_dns_core_dns_config_template_path"`
	// Dir to store auto-generated DNS Server config in.
	ConfigDir string `json:"configDir,omitempty" envconfig:"dubbo_dns_config_dir"`
	// PrometheusPort where Prometheus stats will be exposed for the DNS Server
	PrometheusPort uint32 `json:"prometheusPort,omitempty" envconfig:"dubbo_dns_prometheus_port"`
	// If true then CoreDNS logging is enabled
	CoreDNSLogging bool `json:"coreDNSLogging,omitempty" envconfig:"dubbo_dns_enable_logging"`
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

func (d *Dataplane) IsZoneProxy() bool {
	return d.ProxyType == string(mesh_proto.IngressProxyType) ||
		d.ProxyType == string(mesh_proto.EgressProxyType)
}
