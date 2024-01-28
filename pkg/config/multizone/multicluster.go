package multizone

import (
	"github.com/apache/dubbo-kubernetes/pkg/config"
	config_types "github.com/apache/dubbo-kubernetes/pkg/config/types"
	"go.uber.org/multierr"
	"time"
)

var _ config.Config = &MultizoneConfig{}

// GlobalConfig defines Global configuration
type GlobalConfig struct {
	// DDS Configuration
	DDS *DdsServerConfig `json:"kds,omitempty"`
}

func (g *GlobalConfig) Sanitize() {
	g.DDS.Sanitize()
}

func (g *GlobalConfig) PostProcess() error {
	return multierr.Combine(g.DDS.PostProcess())
}

func (g *GlobalConfig) Validate() error {
	return g.DDS.Validate()
}

func DefaultGlobalConfig() *GlobalConfig {
	return &GlobalConfig{
		DDS: &DdsServerConfig{
			GrpcPort:        5685,
			RefreshInterval: config_types.Duration{Duration: 1 * time.Second},
		},
	}
}

var _ config.Config = &ZoneConfig{}

// ZoneConfig defines zone configuration
type ZoneConfig struct {
	// Dubbo Zone name used to mark the zone dataplane resources
	Name string `json:"name,omitempty" envconfig:"kuma_multizone_zone_name"`
	// GlobalAddress URL of Global Kuma CP
	GlobalAddress string `json:"globalAddress,omitempty" envconfig:"kuma_multizone_zone_global_address"`
	// DisableOriginLabelValidation disables validation of the origin label when applying resources on Zone CP
	DisableOriginLabelValidation bool `json:"disableOriginLabelValidation,omitempty" envconfig:"kuma_multizone_zone_disable_origin_label_validation"`
}

func (r *ZoneConfig) Sanitize() {
}

func (r *ZoneConfig) PostProcess() error {
	return nil
}

func (r *ZoneConfig) Validate() error {
	return nil
}

func DefaultZoneConfig() *ZoneConfig {
	return &ZoneConfig{
		GlobalAddress:                "",
		Name:                         "default",
		DisableOriginLabelValidation: false,
	}
}

// MultizoneConfig defines multizone configuration
type MultizoneConfig struct {
	Global *GlobalConfig `json:"global,omitempty"`
	Zone   *ZoneConfig   `json:"zone,omitempty"`
}

func (m *MultizoneConfig) Sanitize() {
	m.Global.Sanitize()
	m.Zone.Sanitize()
}

func (m *MultizoneConfig) PostProcess() error {
	return multierr.Combine(
		m.Global.PostProcess(),
		m.Zone.PostProcess(),
	)
}

func (m *MultizoneConfig) Validate() error {
	panic("not implemented. Call Global and Zone validators as needed.")
}

func DefaultMultizoneConfig() *MultizoneConfig {
	return &MultizoneConfig{
		Global: DefaultGlobalConfig(),
		Zone:   DefaultZoneConfig(),
	}
}
