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

package bootstrap

import (
	"github.com/apache/dubbo-kubernetes/pkg/config"
	config_types "github.com/apache/dubbo-kubernetes/pkg/config/types"
	"github.com/apache/dubbo-kubernetes/pkg/util/files"
	"github.com/pkg/errors"
	"go.uber.org/multierr"
	"net"
	"os"
	"time"
)

var _ config.Config = &BootstrapServerConfig{}

type BootstrapServerConfig struct {
	// Parameters of bootstrap configuration
	Params *BootstrapParamsConfig `json:"params"`
}

func (b *BootstrapServerConfig) Sanitize() {
	b.Params.Sanitize()
}

func (b *BootstrapServerConfig) PostProcess() error {
	return multierr.Combine(b.Params.PostProcess())
}

func (b *BootstrapServerConfig) Validate() error {
	if err := b.Params.Validate(); err != nil {
		return errors.Wrap(err, "Params validation failed")
	}
	return nil
}

func DefaultBootstrapServerConfig() *BootstrapServerConfig {
	return &BootstrapServerConfig{
		Params: DefaultBootstrapParamsConfig(),
	}
}

var _ config.Config = &BootstrapParamsConfig{}

type BootstrapParamsConfig struct {
	config.BaseConfig

	// Address of Envoy Admin
	AdminAddress string `json:"adminAddress" envconfig:"kuma_bootstrap_server_params_admin_address"`
	// Port of Envoy Admin
	AdminPort uint32 `json:"adminPort" envconfig:"kuma_bootstrap_server_params_admin_port"`
	// Path to access log file of Envoy Admin
	AdminAccessLogPath string `json:"adminAccessLogPath" envconfig:"kuma_bootstrap_server_params_admin_access_log_path"`
	// Host of XDS Server. By default it is the same host as the one used by kuma-dp to connect to the control plane
	XdsHost string `json:"xdsHost" envconfig:"kuma_bootstrap_server_params_xds_host"`
	// Port of XDS Server. By default it is autoconfigured from KUMA_XDS_SERVER_GRPC_PORT
	XdsPort uint32 `json:"xdsPort" envconfig:"kuma_bootstrap_server_params_xds_port"`
	// Connection timeout to the XDS Server
	XdsConnectTimeout config_types.Duration `json:"xdsConnectTimeout" envconfig:"kuma_bootstrap_server_params_xds_connect_timeout"`
	// Path to the template of Corefile for data planes to use
	CorefileTemplatePath string `json:"corefileTemplatePath" envconfig:"kuma_bootstrap_server_params_corefile_template_path"`
}

func (b *BootstrapParamsConfig) Validate() error {
	if b.AdminAddress == "" {
		return errors.New("AdminAddress cannot be empty")
	}
	if net.ParseIP(b.AdminAddress) == nil {
		return errors.New("AdminAddress should be a valid IP address")
	}
	if b.AdminPort > 65535 {
		return errors.New("AdminPort must be in the range [0, 65535]")
	}
	if b.AdminAccessLogPath == "" {
		return errors.New("AdminAccessLogPath cannot be empty")
	}
	if b.XdsPort > 65535 {
		return errors.New("AdminPort must be in the range [0, 65535]")
	}
	if b.XdsConnectTimeout.Duration < 0 {
		return errors.New("XdsConnectTimeout cannot be negative")
	}
	if b.CorefileTemplatePath != "" && !files.FileExists(b.CorefileTemplatePath) {
		return errors.New("CorefileTemplatePath must point to an existing file")
	}
	return nil
}

func DefaultBootstrapParamsConfig() *BootstrapParamsConfig {
	return &BootstrapParamsConfig{
		AdminAddress:         "127.0.0.1", // by default, Envoy Admin interface should listen on loopback address
		AdminPort:            9901,
		AdminAccessLogPath:   os.DevNull,
		XdsHost:              "", // by default, it is the same host as the one used by kuma-dp to connect to the control plane
		XdsPort:              0,  // by default, it is autoconfigured from KUMA_XDS_SERVER_GRPC_PORT
		XdsConnectTimeout:    config_types.Duration{Duration: 1 * time.Second},
		CorefileTemplatePath: "", // by default, data plane will use the embedded Corefile to be the template
	}
}
