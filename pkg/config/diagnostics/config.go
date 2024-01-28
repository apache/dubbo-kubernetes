package diagnostics

import (
	"github.com/apache/dubbo-kubernetes/pkg/config"
)

type DiagnosticsConfig struct {
	config.BaseConfig

	// Port of Diagnostic Server for checking health and readiness of the Control Plane
	ServerPort uint32 `json:"serverPort" envconfig:"dubbo_diagnostics_server_port"`
}

var _ config.Config = &DiagnosticsConfig{}

func (d *DiagnosticsConfig) Validate() error {
	return nil
}

func DefaultDiagnosticsConfig() *DiagnosticsConfig {
	return &DiagnosticsConfig{
		ServerPort: 5680,
	}
}
