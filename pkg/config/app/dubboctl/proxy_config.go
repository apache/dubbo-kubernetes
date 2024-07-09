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
	"time"
)

import (
	mesh_proto "github.com/apache/dubbo-kubernetes/api/mesh/v1alpha1"
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
	BinaryPath string `json:"binaryPath,omitempty" envconfig:"DUBBO_DATAPLANE_RUNTIME_BINARY_PATH"`
	// Dir to store auto-generated Envoy bootstrap config in.
	ConfigDir string `json:"configDir,omitempty" envconfig:"DUBBO_DATAPLANE_RUNTIME_CONFIG_DIR"`
	// Concurrency specifies how to generate the Envoy concurrency flag.
	Concurrency uint32 `json:"concurrency,omitempty" envconfig:"DUBBO_DATAPLANE_RUNTIME_CONCURRENCY"`
	// Path to a file with dataplane token (use 'dubboctl generate dataplane-token' to get one)
	TokenPath string `json:"dataplaneTokenPath,omitempty" envconfig:"DUBBO_DATAPLANE_RUNTIME_TOKEN_PATH"`
	// Token is dataplane token's value provided directly, will be stored to a temporary file before applying
	Token string `json:"dataplaneToken,omitempty" envconfig:"DUBBO_DATAPLANE_RUNTIME_TOKEN"`
	// Resource is a Dataplane resource that will be applied on Dubbo CP
	Resource string `json:"resource,omitempty" envconfig:"DUBBO_DATAPLANE_RUNTIME_RESOURCE"`
	// ResourcePath is a path to Dataplane resource that will be applied on Dubbo CP
	ResourcePath string `json:"resourcePath,omitempty" envconfig:"DUBBO_DATAPLANE_RUNTIME_RESOURCE_PATH"`
	// ResourceVars are the StringToString values that can fill the Resource template
	ResourceVars map[string]string `json:"resourceVars,omitempty"`
	// EnvoyLogLevel is a level on which Envoy will log.
	// Available values are: [trace][debug][info][warning|warn][error][critical][off]
	// By default it inherits Dubbo DP logging level.
	EnvoyLogLevel string `json:"envoyLogLevel,omitempty" envconfig:"DUBBO_DATAPLANE_RUNTIME_ENVOY_LOG_LEVEL"`
	// EnvoyComponentLogLevel configures Envoy's --component-log-level and uses
	// the exact same syntax: https://www.envoyproxy.io/docs/envoy/latest/operations/cli#cmdoption-component-log-level
	EnvoyComponentLogLevel string `json:"envoyComponentLogLevel,omitempty" envconfig:"DUBBO_DATAPLANE_RUNTIME_ENVOY_COMPONENT_LOG_LEVEL"`
	// Resources defines the resources for this proxy.
	Resources DataplaneResources `json:"resources,omitempty"`
	// SocketDir dir to store socket used between Envoy and the dp process
	SocketDir string `json:"socketDir,omitempty" envconfig:"DUBBO_DATAPLANE_RUNTIME_SOCKET_DIR"`
	// Metrics defines properties of metrics
	Metrics Metrics `json:"metrics,omitempty"`
	// DynamicConfiguration defines properties of dataplane dynamic configuration
	DynamicConfiguration DynamicConfiguration `json:"dynamicConfiguration" envconfig:"DUBBO_DATAPLANE_RUNTIME_DYNAMIC_CONFIGURATION"`
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
	URL string `json:"url,omitempty" envconfig:"DUBBO_CONTROL_PLANE_URL"`
	// Retry settings for Control Plane communication
	Retry CpRetry `json:"retry,omitempty"`
	// CaCert defines Certificate Authority that will be used to verify connection to the Control Plane. It takes precedence over CaCertFile.
	CaCert string `json:"caCert" envconfig:"DUBBO_CONTROL_PLANE_CA_CERT"`
	// CaCertFile defines a file for Certificate Authority that will be used to verify connection to the Control Plane.
	CaCertFile string `json:"caCertFile" envconfig:"DUBBO_CONTROL_PLANE_CA_CERT_FILE"`
}
type CpRetry struct {
	config.BaseConfig

	// Duration to wait between retries
	Backoff config_types.Duration `json:"backoff,omitempty" envconfig:"DUBBO_CONTROL_PLANE_RETRY_BACKOFF"`
	// Max duration for retries (this is not exact time for execution, the check is done between retries)
	MaxDuration config_types.Duration `json:"maxDuration,omitempty" envconfig:"DUBBO_CONTROL_PLANE_RETRY_MAX_DURATION"`
}
type DNS struct {
	config.BaseConfig

	// If true then builtin DNS functionality is enabled and CoreDNS server is started
	Enabled bool `json:"enabled,omitempty" envconfig:"DUBBO_DNS_ENABLED"`
	// CoreDNSPort defines a port that handles DNS requests. When transparent proxy is enabled then iptables will redirect DNS traffic to this port.
	CoreDNSPort uint32 `json:"coreDnsPort,omitempty" envconfig:"DUBBO_DNS_CORE_DNS_PORT"`
	// CoreDNSEmptyPort defines a port that always responds with empty NXDOMAIN respond. It is required to implement a fallback to a real DNS
	CoreDNSEmptyPort uint32 `json:"coreDnsEmptyPort,omitempty" envconfig:"DUBBO_DNS_CORE_DNS_EMPTY_PORT"`
	// EnvoyDNSPort defines a port that handles Virtual IP resolving by Envoy. CoreDNS should be configured that it first tries to use this DNS resolver and then the real one.
	EnvoyDNSPort uint32 `json:"envoyDnsPort,omitempty" envconfig:"DUBBO_DNS_ENVOY_DNS_PORT"`
	// CoreDNSBinaryPath defines a path to CoreDNS binary.
	CoreDNSBinaryPath string `json:"coreDnsBinaryPath,omitempty" envconfig:"DUBBO_DNS_CORE_DNS_BINARY_PATH"`
	// CoreDNSConfigTemplatePath defines a path to a CoreDNS config template.
	CoreDNSConfigTemplatePath string `json:"coreDnsConfigTemplatePath,omitempty" envconfig:"DUBBO_DNS_CORE_DNS_CONFIG_TEMPLATE_PATH"`
	// Dir to store auto-generated DNS Server config in.
	ConfigDir string `json:"configDir,omitempty" envconfig:"DUBBO_DNS_CONFIG_DIR"`
	// PrometheusPort where Prometheus stats will be exposed for the DNS Server
	PrometheusPort uint32 `json:"prometheusPort,omitempty" envconfig:"DUBBO_DNS_PROMETHEUS_PORT"`
	// If true then CoreDNS logging is enabled
	CoreDNSLogging bool `json:"coreDNSLogging,omitempty" envconfig:"DUBBO_DNS_ENABLE_LOGGING"`
}

type Metrics struct {
	// CertPath path to the certificate for metrics listener
	CertPath string `json:"metricsCertPath,omitempty" envconfig:"DUBBO_DATAPLANE_RUNTIME_METRICS_CERT_PATH"`
	// KeyPath path to the key for metrics listener
	KeyPath string `json:"metricsKeyPath,omitempty" envconfig:"DUBBO_DATAPLANE_RUNTIME_METRICS_KEY_PATH"`
}

type DynamicConfiguration struct {
	// RefreshInterval defines how often DPP should refresh dynamic config. Default: 10s
	RefreshInterval config_types.Duration `json:"refreshInterval,omitempty" envconfig:"DUBBO_DATAPLANE_RUNTIME_DYNAMIC_CONFIGURATION_REFRESH_INTERVAL"`
}

// DataplaneResources defines the resources available to a dataplane proxy.
type DataplaneResources struct {
	MaxMemoryBytes uint64 `json:"maxMemoryBytes,omitempty" envconfig:"DUBBO_DATAPLANE_RESOURCES_MAX_MEMORY_BYTES"`
}

type Dataplane struct {
	config.BaseConfig

	// Mesh name.
	Mesh string `json:"mesh,omitempty" envconfig:"DUBBO_DATAPLANE_MESH"`
	// Dataplane name.
	Name string `json:"name,omitempty" envconfig:"DUBBO_DATAPLANE_NAME"`
	// ProxyType defines mode which should be used, supported values: 'dataplane', 'ingress'
	ProxyType string `json:"proxyType,omitempty" envconfig:"DUBBO_DATAPLANE_PROXY_TYPE"`
	// Drain time for listeners.
	DrainTime config_types.Duration `json:"drainTime,omitempty" envconfig:"DUBBO_DATAPLANE_DRAIN_TIME"`
}

func (d *Dataplane) IsZoneProxy() bool {
	return d.ProxyType == string(mesh_proto.IngressProxyType) ||
		d.ProxyType == string(mesh_proto.EgressProxyType)
}
