package universal

import (
	"github.com/apache/dubbo-kubernetes/pkg/config"
	config_types "github.com/apache/dubbo-kubernetes/pkg/config/types"
	"github.com/pkg/errors"
	"go.uber.org/multierr"
	"time"
)

func DefaultUniversalRuntimeConfig() *UniversalRuntimeConfig {
	return &UniversalRuntimeConfig{
		DataplaneCleanupAge: config_types.Duration{Duration: 3 * 24 * time.Hour},
	}
}

var _ config.Config = &UniversalRuntimeConfig{}

// UniversalRuntimeConfig defines Universal-specific configuration
type UniversalRuntimeConfig struct {
	config.BaseConfig

	// DataplaneCleanupAge defines how long Dataplane should be offline to be cleaned up by GC
	DataplaneCleanupAge config_types.Duration `json:"dataplaneCleanupAge" envconfig:"kuma_runtime_universal_dataplane_cleanup_age"`
}

func (u *UniversalRuntimeConfig) Validate() error {
	var errs error
	if u.DataplaneCleanupAge.Duration <= 0 {
		errs = multierr.Append(errs, errors.Errorf(".DataplaneCleanupAge must be positive"))
	}
	return errs
}
