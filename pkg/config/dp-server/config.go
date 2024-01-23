package dp_server

import (
	"github.com/pkg/errors"

	"go.uber.org/multierr"
)

import (
	"github.com/apache/dubbo-kubernetes/pkg/config"
)

var _ config.Config = &DpServerConfig{}

// DpServerConfig defines the data plane Server configuration that serves API
// like Bootstrap/XDS.
type DpServerConfig struct {
	config.BaseConfig

	// Port of the DP Server
	Port int `json:"port" envconfig:"dubbo_dp_server_port"`
}

func (a *DpServerConfig) PostProcess() error {
	return nil
}

func (a *DpServerConfig) Validate() error {
	var errs error
	if a.Port < 0 {
		errs = multierr.Append(errs, errors.New(".Port cannot be negative"))
	}
	return errs
}

func DefaultDpServerConfig() *DpServerConfig {
	return &DpServerConfig{
		Port: 5678,
	}
}
