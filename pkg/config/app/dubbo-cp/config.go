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

package dubbo_cp

import (
	"time"
)

import (
	"github.com/pkg/errors"

	"go.uber.org/multierr"
)

import (
	"github.com/apache/dubbo-kubernetes/pkg/config"
	"github.com/apache/dubbo-kubernetes/pkg/config/admin"
	"github.com/apache/dubbo-kubernetes/pkg/config/bufman"
	"github.com/apache/dubbo-kubernetes/pkg/config/core"
	"github.com/apache/dubbo-kubernetes/pkg/config/core/resources/store"
	"github.com/apache/dubbo-kubernetes/pkg/config/diagnostics"
	dp_server "github.com/apache/dubbo-kubernetes/pkg/config/dp-server"
	"github.com/apache/dubbo-kubernetes/pkg/config/dubbo"
	"github.com/apache/dubbo-kubernetes/pkg/config/eventbus"
	"github.com/apache/dubbo-kubernetes/pkg/config/intercp"
	"github.com/apache/dubbo-kubernetes/pkg/config/multizone"
	"github.com/apache/dubbo-kubernetes/pkg/config/plugins/runtime"
	config_types "github.com/apache/dubbo-kubernetes/pkg/config/types"
	"github.com/apache/dubbo-kubernetes/pkg/config/xds"
	"github.com/apache/dubbo-kubernetes/pkg/config/xds/bootstrap"
)

var _ config.Config = &Config{}

var _ config.Config = &Defaults{}

type Defaults struct {
	config.BaseConfig

	// If true, it skips creating the default Mesh
	SkipMeshCreation bool `json:"skipMeshCreation" envconfig:"dubbo_defaults_skip_mesh_creation"`
}

type GeneralConfig struct {
	config.BaseConfig

	// DNSCacheTTL represents duration for how long Dubbo CP will cache result of resolving dataplane's domain name
	DNSCacheTTL config_types.Duration `json:"dnsCacheTTL" envconfig:"dubbo_general_dns_cache_ttl"`
	// TlsCertFile defines a path to a file with PEM-encoded TLS cert that will be used across all the Dubbo Servers.
	TlsCertFile string `json:"tlsCertFile" envconfig:"dubbo_general_tls_cert_file"`
	// TlsKeyFile defines a path to a file with PEM-encoded TLS key that will be used across all the Dubbo Servers.
	TlsKeyFile string `json:"tlsKeyFile" envconfig:"dubbo_general_tls_key_file"`
	// TlsMinVersion defines the minimum TLS version to be used
	TlsMinVersion string `json:"tlsMinVersion" envconfig:"dubbo_general_tls_min_version"`
	// TlsMaxVersion defines the maximum TLS version to be used
	TlsMaxVersion string `json:"tlsMaxVersion" envconfig:"dubbo_general_tls_max_version"`
	// TlsCipherSuites defines the list of ciphers to use
	TlsCipherSuites []string `json:"tlsCipherSuites" envconfig:"dubbo_general_tls_cipher_suites"`
	// WorkDir defines a path to the working directory
	// Dubbo stores in this directory autogenerated entities like certificates.
	// If empty then the working directory is $HOME/.dubbo
	WorkDir string `json:"workDir" envconfig:"dubbo_general_work_dir"`
}

type Config struct {
	// General configuration
	General *GeneralConfig `json:"general,omitempty"`
	// DeployMode Type, can be either "kubernetes" or "universal" and "half"
	DeployMode core.DeployMode `json:"deploy_mode,omitempty" envconfig:"dubbo_deploymode"`
	// Mode in which dubbo CP is running. Available values are: "test", "global", "zone"
	Mode core.CpMode `json:"mode" envconfig:"dubbo_mode"`
	// Configuration of Bootstrap Server, which provides bootstrap config to Dataplanes
	BootstrapServer *bootstrap.BootstrapServerConfig `json:"bootstrapServer,omitempty"`
	// Resource Store configuration
	Store *store.StoreConfig `json:"store,omitempty"`
	// Envoy XDS server configuration
	XdsServer *xds.XdsServerConfig `json:"xdsServer,omitempty"`
	// admin console configuration
	Admin *admin.Admin `json:"admin"`
	// DeployMode-specific configuration
	Runtime *runtime.RuntimeConfig `json:"runtime,omitempty"`
	// Multizone Config
	Multizone *multizone.MultizoneConfig `json:"multizone,omitempty"`
	// Default dubbo entities configuration
	Defaults *Defaults `json:"defaults,omitempty"`
	// Diagnostics configuration
	Diagnostics *diagnostics.DiagnosticsConfig `json:"diagnostics,omitempty"`
	// Proxy holds configuration for proxies
	Proxy xds.Proxy `json:"proxy"`
	// Dataplane Server configuration
	DpServer *dp_server.DpServerConfig `json:"dpServer"`
	// EventBus is a configuration of the event bus which is local to one instance of CP.
	EventBus eventbus.Config `json:"eventBus"`
	// Intercommunication CP configuration
	InterCp intercp.InterCpConfig `json:"interCp"`
	// SNP configuration
	DubboConfig           dubbo.DubboConfig     `json:"dubbo_config"`
	Bufman                bufman.Bufman         `json:"bufman"`
	DDSEventBasedWatchdog DDSEventBasedWatchdog `json:"dds_event_based_watchdog"`
}

type DDSEventBasedWatchdog struct {
	// How often we flush changes when experimental event based watchdog is used.
	FlushInterval config_types.Duration `json:"flushInterval" envconfig:"DUBBO_EXPERIMENTAL_KDS_EVENT_BASED_WATCHDOG_FLUSH_INTERVAL"`
	// How often we schedule full KDS resync when experimental event based watchdog is used.
	FullResyncInterval config_types.Duration `json:"fullResyncInterval" envconfig:"DUBBO_EXPERIMENTAL_KDS_EVENT_BASED_WATCHDOG_FULL_RESYNC_INTERVAL"`
	// If true, then initial full resync is going to be delayed by 0 to FullResyncInterval.
	DelayFullResync bool `json:"delayFullResync" envconfig:"DUBBO_EXPERIMENTAL_KDS_EVENT_BASED_WATCHDOG_DELAY_FULL_RESYNC"`
}

func (c Config) IsFederatedZoneCP() bool {
	return c.Mode == core.Zone && c.Multizone.Zone.GlobalAddress != "" && c.Multizone.Zone.Name != ""
}

func (c Config) IsNonFederatedZoneCP() bool {
	return c.Mode == core.Zone && !c.IsFederatedZoneCP()
}

func (c *Config) Sanitize() {
	c.Store.Sanitize()

	c.Runtime.Sanitize()
	c.Defaults.Sanitize()

	c.Diagnostics.Sanitize()
}

func (c *Config) PostProcess() error {
	return multierr.Combine(
		c.Store.PostProcess(),
		c.Runtime.PostProcess(),
		c.Defaults.PostProcess(),
		c.Diagnostics.PostProcess(),
	)
}

var DefaultConfig = func() Config {
	return Config{
		BootstrapServer: bootstrap.DefaultBootstrapServerConfig(),
		DeployMode:      core.UniversalMode,
		Mode:            core.Zone,
		XdsServer:       xds.DefaultXdsServerConfig(),
		Store:           store.DefaultStoreConfig(),
		Runtime:         runtime.DefaultRuntimeConfig(),
		Bufman:          bufman.DefaultBufmanConfig(),
		General:         DefaultGeneralConfig(),
		Defaults:        DefaultDefaultsConfig(),
		Multizone:       multizone.DefaultMultizoneConfig(),
		Diagnostics:     diagnostics.DefaultDiagnosticsConfig(),
		DpServer:        dp_server.DefaultDpServerConfig(),
		Admin:           admin.DefaultAdminConfig(),
		InterCp:         intercp.DefaultInterCpConfig(),
		DubboConfig:     dubbo.DefaultServiceNameMappingConfig(),
		EventBus:        eventbus.Default(),
	}
}

func DefaultGeneralConfig() *GeneralConfig {
	return &GeneralConfig{
		DNSCacheTTL:     config_types.Duration{Duration: 10 * time.Second},
		WorkDir:         "",
		TlsCipherSuites: []string{},
		TlsMinVersion:   "TLSv1_2",
	}
}

func DefaultDefaultsConfig() *Defaults {
	return &Defaults{
		SkipMeshCreation: false,
	}
}

func (c *Config) Validate() error {
	if err := core.ValidateCpMode(c.Mode); err != nil {
		return errors.Wrap(err, "Mode validation failed")
	}
	switch c.Mode {
	case core.Global:
	case core.Zone:
		if c.DeployMode != core.KubernetesMode && c.DeployMode != core.UniversalMode && c.DeployMode != core.HalfHostMode {
			return errors.Errorf("DeployMode should be either %s or %s or %s", core.KubernetesMode, core.UniversalMode, core.HalfHostMode)
		}
		if err := c.Runtime.Validate(c.DeployMode); err != nil {
			return errors.Wrap(err, "Runtime validation failed")
		}
	}
	if err := c.Store.Validate(); err != nil {
		return errors.Wrap(err, "Store validation failed")
	}
	if err := c.Defaults.Validate(); err != nil {
		return errors.Wrap(err, "Defaults validation failed")
	}
	if err := c.Diagnostics.Validate(); err != nil {
		return errors.Wrap(err, "Diagnostics validation failed")
	}

	return nil
}

func (c Config) GetEnvoyAdminPort() uint32 {
	if c.BootstrapServer == nil || c.BootstrapServer.Params == nil {
		return 0
	}
	return c.BootstrapServer.Params.AdminPort
}
