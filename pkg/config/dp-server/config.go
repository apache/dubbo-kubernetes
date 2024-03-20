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

package dp_server

import (
	"time"
)

import (
	"github.com/pkg/errors"

	"go.uber.org/multierr"
)

import (
	"github.com/apache/dubbo-kubernetes/pkg/config"
	config_types "github.com/apache/dubbo-kubernetes/pkg/config/types"
)

var _ config.Config = &DpServerConfig{}

// DpServerConfig defines the data plane Server configuration that serves API
// like Bootstrap/XDS.
type DpServerConfig struct {
	config.BaseConfig
	// ReadHeaderTimeout defines the amount of time DP server will be
	// allowed to read request headers. The connection's read deadline is reset
	// after reading the headers and the Handler can decide what is considered
	// too slow for the body. If ReadHeaderTimeout is zero there is no timeout.
	//
	// The timeout is configurable as in rare cases, when Dubbo CP was restarting,
	// 1s which is explicitly set in other servers was insufficient and DPs
	// were failing to reconnect (we observed this in Projected Service Account
	// Tokens e2e tests, which started flaking a lot after introducing explicit
	// 1s timeout)
	ReadHeaderTimeout config_types.Duration `json:"readHeaderTimeout" envconfig:"dubbo_dp_server_read_header_timeout"`
	// Port of the DP Server
	Port int `json:"port" envconfig:"dubbo_dp_server_port"`
	// Hds defines a Health Discovery Service configuration
	Hds *HdsConfig `json:"hds"`
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
		Port:              5678,
		Hds:               DefaultHdsConfig(),
		ReadHeaderTimeout: config_types.Duration{Duration: 5 * time.Second},
	}
}

func DefaultHdsConfig() *HdsConfig {
	return &HdsConfig{
		Enabled:         true,
		Interval:        config_types.Duration{Duration: 5 * time.Second},
		RefreshInterval: config_types.Duration{Duration: 10 * time.Second},
		CheckDefaults: &HdsCheck{
			Timeout:            config_types.Duration{Duration: 2 * time.Second},
			Interval:           config_types.Duration{Duration: 1 * time.Second},
			NoTrafficInterval:  config_types.Duration{Duration: 1 * time.Second},
			HealthyThreshold:   1,
			UnhealthyThreshold: 1,
		},
	}
}

type HdsConfig struct {
	config.BaseConfig

	// Enabled if true then Envoy will actively check application's ports, but only on Universal.
	// On Kubernetes this feature disabled for now regardless the flag value
	Enabled bool `json:"enabled" envconfig:"dubbo_dp_server_hds_enabled"`
	// Interval for Envoy to send statuses for HealthChecks
	Interval config_types.Duration `json:"interval" envconfig:"dubbo_dp_server_hds_interval"`
	// RefreshInterval is an interval for re-genarting configuration for Dataplanes connected to the Control Plane
	RefreshInterval config_types.Duration `json:"refreshInterval" envconfig:"dubbo_dp_server_hds_refresh_interval"`
	// CheckDefaults defines a HealthCheck configuration
	CheckDefaults *HdsCheck `json:"checkDefaults"`
}

func (h *HdsConfig) PostProcess() error {
	return multierr.Combine(h.CheckDefaults.PostProcess())
}

func (h *HdsConfig) Validate() error {
	if h.Interval.Duration <= 0 {
		return errors.New("Interval must be greater than 0s")
	}
	if err := h.CheckDefaults.Validate(); err != nil {
		return errors.Wrap(err, "Check is invalid")
	}
	return nil
}

type HdsCheck struct {
	config.BaseConfig

	// Timeout is a time to wait for a health check response. If the timeout is reached the
	// health check attempt will be considered a failure.
	Timeout config_types.Duration `json:"timeout" envconfig:"dubbo_dp_server_hds_check_timeout"`
	// Interval between health checks.
	Interval config_types.Duration `json:"interval" envconfig:"dubbo_dp_server_hds_check_interval"`
	// NoTrafficInterval is a special health check interval that is used when a cluster has
	// never had traffic routed to it.
	NoTrafficInterval config_types.Duration `json:"noTrafficInterval" envconfig:"dubbo_dp_server_hds_check_no_traffic_interval"`
	// HealthyThreshold is a number of healthy health checks required before a host is marked
	// healthy.
	HealthyThreshold uint32 `json:"healthyThreshold" envconfig:"dubbo_dp_server_hds_check_healthy_threshold"`
	// UnhealthyThreshold is a number of unhealthy health checks required before a host is marked
	// unhealthy.
	UnhealthyThreshold uint32 `json:"unhealthyThreshold" envconfig:"dubbo_dp_server_hds_check_unhealthy_threshold"`
}

func (h *HdsCheck) Validate() error {
	if h.Timeout.Duration <= 0 {
		return errors.New("Timeout must be greater than 0s")
	}
	if h.Interval.Duration <= 0 {
		return errors.New("Interval must be greater than 0s")
	}
	if h.NoTrafficInterval.Duration <= 0 {
		return errors.New("NoTrafficInterval must be greater than 0s")
	}
	return nil
}
