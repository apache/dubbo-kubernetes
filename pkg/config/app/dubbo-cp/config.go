package dubbo_cp

import (
	dubbogo "dubbo.apache.org/dubbo-go/v3/config"
	"github.com/apache/dubbo-kubernetes/pkg/config/admin"
	"github.com/apache/dubbo-kubernetes/pkg/config/bufman"
	"github.com/pkg/errors"

	"go.uber.org/multierr"
)

import (
	"github.com/apache/dubbo-kubernetes/pkg/config"
	"github.com/apache/dubbo-kubernetes/pkg/config/core"
	"github.com/apache/dubbo-kubernetes/pkg/config/core/resources/store"
	"github.com/apache/dubbo-kubernetes/pkg/config/diagnostics"
	dp_server "github.com/apache/dubbo-kubernetes/pkg/config/dp-server"
	"github.com/apache/dubbo-kubernetes/pkg/config/eventbus"
	"github.com/apache/dubbo-kubernetes/pkg/config/plugins/runtime"
)

var _ config.Config = &Config{}

var _ config.Config = &Defaults{}

type Defaults struct {
	config.BaseConfig
}

type Config struct {
	// Environment Type, can be either "kubernetes" or "universal"
	Environment core.EnvironmentType `json:"environment,omitempty" envconfig:"dubbo_environment"`
	// Mode in which dubbo CP is running. Available values are: "standalone", "global", "zone"
	Mode core.CpMode `json:"mode" envconfig:"dubbo_mode"`
	// Resource Store configuration
	Store *store.StoreConfig `json:"store,omitempty"`
	// Environment-specific configuration
	Runtime *runtime.RuntimeConfig `json:"runtime,omitempty"`
	// Default dubbo entities configuration
	Defaults *Defaults `json:"defaults,omitempty"`
	// Diagnostics configuration
	Diagnostics *diagnostics.DiagnosticsConfig `json:"diagnostics,omitempty"`
	// Dataplane Server configuration
	DpServer *dp_server.DpServerConfig `json:"dpServer"`
	// EventBus is a configuration of the event bus which is local to one instance of CP.
	EventBus eventbus.Config    `json:"eventBus"`
	Bufman   bufman.Bufman      `json:"bufman"`
	Admin    admin.Admin        `json:"admin"`
	Dubbo    dubbogo.RootConfig `json:"dubbo"`
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
		Environment: core.UniversalEnvironment,
		Mode:        core.Zone,
		Store:       store.DefaultStoreConfig(),
		Runtime:     runtime.DefaultRuntimeConfig(),
		Defaults:    &Defaults{},
		Diagnostics: diagnostics.DefaultDiagnosticsConfig(),
		DpServer:    dp_server.DefaultDpServerConfig(),
		EventBus:    eventbus.Default(),
	}
}

func (c *Config) Validate() error {
	if err := core.ValidateCpMode(c.Mode); err != nil {
		return errors.Wrap(err, "Mode validation failed")
	}
	switch c.Mode {
	case core.Global:
	case core.Zone:
		if c.Environment != core.KubernetesEnvironment && c.Environment != core.UniversalEnvironment {
			return errors.Errorf("Environment should be either %s or %s", core.KubernetesEnvironment, core.UniversalEnvironment)
		}
		if err := c.Runtime.Validate(c.Environment); err != nil {
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
