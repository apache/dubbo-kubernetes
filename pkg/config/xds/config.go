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

package xds

import (
	"time"
)

import (
	"github.com/pkg/errors"
)

import (
	"github.com/apache/dubbo-kubernetes/pkg/config"
	config_types "github.com/apache/dubbo-kubernetes/pkg/config/types"
)

var _ config.Config = &XdsServerConfig{}

// Envoy XDS server configuration
type XdsServerConfig struct {
	config.BaseConfig

	// Interval for re-generating configuration for Dataplanes connected to the Control Plane
	DataplaneConfigurationRefreshInterval config_types.Duration `json:"dataplaneConfigurationRefreshInterval" envconfig:"dubbo_xds_server_dataplane_configuration_refresh_interval"`
	// Interval for flushing status of Dataplanes connected to the Control Plane
	DataplaneStatusFlushInterval config_types.Duration `json:"dataplaneStatusFlushInterval" envconfig:"dubbo_xds_server_dataplane_status_flush_interval"`
	// DataplaneDeregistrationDelay is a delay between proxy terminating a connection and the CP trying to deregister the proxy.
	// It is used only in universal mode when you use direct lifecycle.
	// Setting this setting to 0s disables the delay.
	// Disabling this may cause race conditions that one instance of CP removes proxy object
	// while proxy is connected to another instance of the CP.
	DataplaneDeregistrationDelay config_types.Duration `json:"dataplaneDeregistrationDelay" envconfig:"dubbo_xds_dataplane_deregistration_delay"`
	// Backoff that is executed when Control Plane is sending the response that was previously rejected by Dataplane
	NACKBackoff config_types.Duration `json:"nackBackoff" envconfig:"dubbo_xds_server_nack_backoff"`
}

func (x *XdsServerConfig) Validate() error {
	if x.DataplaneConfigurationRefreshInterval.Duration <= 0 {
		return errors.New("DataplaneConfigurationRefreshInterval must be positive")
	}
	if x.DataplaneStatusFlushInterval.Duration <= 0 {
		return errors.New("DataplaneStatusFlushInterval must be positive")
	}
	return nil
}

func DefaultXdsServerConfig() *XdsServerConfig {
	return &XdsServerConfig{
		DataplaneConfigurationRefreshInterval: config_types.Duration{Duration: 1 * time.Second},
		DataplaneStatusFlushInterval:          config_types.Duration{Duration: 10 * time.Second},
		DataplaneDeregistrationDelay:          config_types.Duration{Duration: 10 * time.Second},
		NACKBackoff:                           config_types.Duration{Duration: 5 * time.Second},
	}
}
