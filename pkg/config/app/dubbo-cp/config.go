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

	"github.com/apache/dubbo-kubernetes/pkg/config"
	"github.com/apache/dubbo-kubernetes/pkg/config/access"
	"github.com/apache/dubbo-kubernetes/pkg/config/admin"
	"github.com/apache/dubbo-kubernetes/pkg/config/intercp"
	"github.com/asaskevich/govalidator"
	"github.com/pkg/errors"
	"go.uber.org/multierr"

	api_server "github.com/apache/dubbo-kubernetes/pkg/config/api-server"
	"github.com/apache/dubbo-kubernetes/pkg/config/bufman"
	"github.com/apache/dubbo-kubernetes/pkg/config/core"
	"github.com/apache/dubbo-kubernetes/pkg/config/core/resources/store"
	"github.com/apache/dubbo-kubernetes/pkg/config/diagnostics"

	dp_server "github.com/apache/dubbo-kubernetes/pkg/config/dp-server"
	"github.com/apache/dubbo-kubernetes/pkg/config/dubbo"
	"github.com/apache/dubbo-kubernetes/pkg/config/eventbus"
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
	SkipMeshCreation bool `json:"skipMeshCreation" envconfig:"DUBBO_DEFAULTS_SKIP_MESH_CREATION"`
}

type DataplaneMetrics struct {
	config.BaseConfig

	SubscriptionLimit int                   `json:"subscriptionLimit" envconfig:"kuma_metrics_dataplane_subscription_limit"`
	IdleTimeout       config_types.Duration `json:"idleTimeout" envconfig:"kuma_metrics_dataplane_idle_timeout"`
}

type ZoneMetrics struct {
	config.BaseConfig

	SubscriptionLimit int                   `json:"subscriptionLimit" envconfig:"kuma_metrics_zone_subscription_limit"`
	IdleTimeout       config_types.Duration `json:"idleTimeout" envconfig:"kuma_metrics_zone_idle_timeout"`
	// CompactFinishedSubscriptions compacts finished metrics (do not store config and details of KDS exchange).
	CompactFinishedSubscriptions bool `json:"compactFinishedSubscriptions" envconfig:"kuma_metrics_zone_compact_finished_subscriptions"`
}

type MeshMetrics struct {
	config.BaseConfig

	// Deprecated: use MinResyncInterval instead
	MinResyncTimeout config_types.Duration `json:"minResyncTimeout" envconfig:"kuma_metrics_mesh_min_resync_timeout"`
	// Deprecated: use FullResyncInterval instead
	MaxResyncTimeout config_types.Duration `json:"maxResyncTimeout" envconfig:"kuma_metrics_mesh_max_resync_timeout"`
	// BufferSize the size of the buffer between event creation and processing
	BufferSize int `json:"bufferSize" envconfig:"kuma_metrics_mesh_buffer_size"`
	// MinResyncInterval the minimum time between 2 refresh of insights.
	MinResyncInterval config_types.Duration `json:"minResyncInterval" envconfig:"kuma_metrics_mesh_min_resync_interval"`
	// FullResyncInterval time between triggering a full refresh of all the insights
	FullResyncInterval config_types.Duration `json:"fullResyncInterval" envconfig:"kuma_metrics_mesh_full_resync_interval"`
	// EventProcessors is a number of workers that process metrics events.
	EventProcessors int `json:"eventProcessors" envconfig:"kuma_metrics_mesh_event_processors"`
}

type Metrics struct {
	config.BaseConfig

	Dataplane    *DataplaneMetrics    `json:"dataplane"`
	Zone         *ZoneMetrics         `json:"zone"`
	Mesh         *MeshMetrics         `json:"mesh"`
	ControlPlane *ControlPlaneMetrics `json:"controlPlane"`
}

type ControlPlaneMetrics struct {
	// ReportResourcesCount if true will report metrics with the count of resources.
	// Default: true
	ReportResourcesCount bool `json:"reportResourcesCount" envconfig:"kuma_metrics_control_plane_report_resources_count"`
}

type InterCpConfig struct {
	// Catalog configuration. Catalog keeps a record of all live CP instances in the zone.
	Catalog CatalogConfig `json:"catalog"`
	// Intercommunication CP server configuration
	Server InterCpServerConfig `json:"server"`
}

func (i *InterCpConfig) Validate() error {
	if err := i.Server.Validate(); err != nil {
		return errors.Wrap(err, ".Server validation failed")
	}
	if err := i.Catalog.Validate(); err != nil {
		return errors.Wrap(err, ".Catalog validation failed")
	}
	return nil
}

type CatalogConfig struct {
	// InstanceAddress indicates an address on which other control planes can communicate with this CP
	// If empty then it's autoconfigured by taking the first IP of the nonloopback network interface.
	InstanceAddress string `json:"instanceAddress" envconfig:"kuma_inter_cp_catalog_instance_address"`
	// Interval on which CP will send heartbeat to a leader.
	HeartbeatInterval config_types.Duration `json:"heartbeatInterval" envconfig:"kuma_inter_cp_catalog_heartbeat_interval"`
	// Interval on which CP will write all instances to a catalog.
	WriterInterval config_types.Duration `json:"writerInterval" envconfig:"kuma_inter_cp_catalog_writer_interval"`
}

func (i *CatalogConfig) Validate() error {
	if i.InstanceAddress != "" && !govalidator.IsDNSName(i.InstanceAddress) && !govalidator.IsIP(i.InstanceAddress) {
		return errors.New(".InstanceAddress has to be valid IP or DNS address")
	}
	return nil
}

type InterCpServerConfig struct {
	// Port on which Intercommunication CP server will listen
	Port uint16 `json:"port" envconfig:"kuma_inter_cp_server_port"`
	// TlsMinVersion defines the minimum TLS version to be used
	TlsMinVersion string `json:"tlsMinVersion" envconfig:"kuma_inter_cp_server_tls_min_version"`
	// TlsMaxVersion defines the maximum TLS version to be used
	TlsMaxVersion string `json:"tlsMaxVersion" envconfig:"kuma_inter_cp_server_tls_max_version"`
	// TlsCipherSuites defines the list of ciphers to use
	TlsCipherSuites []string `json:"tlsCipherSuites" envconfig:"kuma_inter_cp_server_tls_cipher_suites"`
}

func (i *InterCpServerConfig) Validate() error {
	var errs error
	if i.Port == 0 {
		errs = multierr.Append(errs, errors.New(".Port cannot be zero"))
	}
	if _, err := config_types.TLSVersion(i.TlsMinVersion); err != nil {
		errs = multierr.Append(errs, errors.New(".TlsMinVersion "+err.Error()))
	}
	if _, err := config_types.TLSVersion(i.TlsMaxVersion); err != nil {
		errs = multierr.Append(errs, errors.New(".TlsMaxVersion "+err.Error()))
	}
	if _, err := config_types.TLSCiphers(i.TlsCipherSuites); err != nil {
		errs = multierr.Append(errs, errors.New(".TlsCipherSuites "+err.Error()))
	}
	return errs
}

type GeneralConfig struct {
	config.BaseConfig

	// DNSCacheTTL represents duration for how long Dubbo CP will cache result of resolving dataplane's domain name
	DNSCacheTTL config_types.Duration `json:"dnsCacheTTL" envconfig:"DUBBO_GENERAL_DNS_CACHE_TTL"`
	// TlsCertFile defines a path to a file with PEM-encoded TLS cert that will be used across all the Dubbo Servers.
	TlsCertFile string `json:"tlsCertFile" envconfig:"DUBBO_GENERAL_TLS_CERT_FILE"`
	// TlsKeyFile defines a path to a file with PEM-encoded TLS key that will be used across all the Dubbo Servers.
	TlsKeyFile string `json:"tlsKeyFile" envconfig:"DUBBO_GENERAL_TLS_KEY_FILE"`
	// TlsMinVersion defines the minimum TLS version to be used
	TlsMinVersion string `json:"tlsMinVersion" envconfig:"DUBBO_GENERAL_TLS_MIN_VERSION"`
	// TlsMaxVersion defines the maximum TLS version to be used
	TlsMaxVersion string `json:"tlsMaxVersion" envconfig:"DUBBO_GENERAL_TLS_MAX_VERSION"`
	// TlsCipherSuites defines the list of ciphers to use
	TlsCipherSuites []string `json:"tlsCipherSuites" envconfig:"DUBBO_GENERAL_TLS_CIPHER_SUITES"`
	// WorkDir defines a path to the working directory
	// Dubbo stores in this directory autogenerated entities like certificates.
	// If empty then the working directory is $HOME/.dubbo
	WorkDir string `json:"workDir" envconfig:"DUBBO_GENERAL_WORK_DIR"`
}

type Config struct {
	// General configuration
	General *GeneralConfig `json:"general,omitempty"`
	// DeployMode Type, can be either "kubernetes" or "universal" and "half"
	DeployMode core.DeployMode `json:"deploy_mode,omitempty" envconfig:"DUBBO_DEPLOYMODE"`
	// Mode in which dubbo CP is running. Available values are: "test", "global", "zone"
	Mode core.CpMode `json:"mode" envconfig:"DUBBO_MODE"`
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
	// API Server configuration
	ApiServer *api_server.ApiServerConfig `json:"apiServer,omitempty"`
	// Multizone Config
	Multizone *multizone.MultizoneConfig `json:"multizone,omitempty"`
	// Default dubbo entities configuration
	Defaults *Defaults `json:"defaults,omitempty"`
	// Metrics configuration
	Metrics *Metrics `json:"metrics,omitempty"`
	// Intercommunication CP configuration
	InterCp intercp.InterCpConfig `json:"interCp"`
	// Diagnostics configuration
	Diagnostics *diagnostics.DiagnosticsConfig `json:"diagnostics,omitempty"`
	// Access Control configuration
	Access access.AccessConfig `json:"access"`
	// Proxy holds configuration for proxies
	Proxy xds.Proxy `json:"proxy"`
	// Dataplane Server configuration
	DpServer *dp_server.DpServerConfig `json:"dpServer"`
	// EventBus is a configuration of the event bus which is local to one instance of CP.
	EventBus eventbus.Config `json:"eventBus"`
	// SNP configuration
	DubboConfig           dubbo.DubboConfig     `json:"dubbo_config"`
	Bufman                bufman.Bufman         `json:"bufman"`
	DDSEventBasedWatchdog DDSEventBasedWatchdog `json:"dds_event_based_watchdog"`
}

type DDSEventBasedWatchdog struct {
	// How often we flush changes when experimental event based watchdog is used.
	FlushInterval config_types.Duration `json:"flushInterval" envconfig:"DUBBO_EXPERIMENTAL_DDS_EVENT_BASED_WATCHDOG_FLUSH_INTERVAL"`
	// How often we schedule full DDS resync when experimental event based watchdog is used.
	FullResyncInterval config_types.Duration `json:"fullResyncInterval" envconfig:"DUBBO_EXPERIMENTAL_DDS_EVENT_BASED_WATCHDOG_FULL_RESYNC_INTERVAL"`
	// If true, then initial full resync is going to be delayed by 0 to FullResyncInterval.
	DelayFullResync bool `json:"delayFullResync" envconfig:"DUBBO_EXPERIMENTAL_DDS_EVENT_BASED_WATCHDOG_DELAY_FULL_RESYNC"`
}

func DefaultEventBasedWatchdog() DDSEventBasedWatchdog {
	return DDSEventBasedWatchdog{
		FlushInterval:      config_types.Duration{Duration: 5 * time.Second},
		FullResyncInterval: config_types.Duration{Duration: 1 * time.Minute},
		DelayFullResync:    false,
	}
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
	c.Admin.Sanitize()
}

func (c *Config) PostProcess() error {
	return multierr.Combine(
		c.Store.PostProcess(),
		c.Runtime.PostProcess(),
		c.Defaults.PostProcess(),
		c.Diagnostics.PostProcess(),
		c.Admin.PostProcess(),
	)
}

var DefaultConfig = func() Config {
	return Config{
		BootstrapServer:       bootstrap.DefaultBootstrapServerConfig(),
		DeployMode:            core.UniversalMode,
		Mode:                  core.Zone,
		XdsServer:             xds.DefaultXdsServerConfig(),
		Store:                 store.DefaultStoreConfig(),
		Runtime:               runtime.DefaultRuntimeConfig(),
		Bufman:                bufman.DefaultBufmanConfig(),
		General:               DefaultGeneralConfig(),
		Defaults:              DefaultDefaultsConfig(),
		Multizone:             multizone.DefaultMultizoneConfig(),
		Diagnostics:           diagnostics.DefaultDiagnosticsConfig(),
		DpServer:              dp_server.DefaultDpServerConfig(),
		Admin:                 admin.DefaultAdminConfig(),
		DubboConfig:           dubbo.DefaultServiceNameMappingConfig(),
		EventBus:              eventbus.Default(),
		DDSEventBasedWatchdog: DefaultEventBasedWatchdog(),
		ApiServer:             api_server.DefaultApiServerConfig(),
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
	if err := c.Admin.Validate(); err != nil {
		return errors.Wrap(err, "Admin validation failed")
	}
	return nil
}

func (c Config) GetEnvoyAdminPort() uint32 {
	if c.BootstrapServer == nil || c.BootstrapServer.Params == nil {
		return 0
	}
	return c.BootstrapServer.Params.AdminPort
}
