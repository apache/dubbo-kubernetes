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

package istio

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// ConfigManager handles Istio configuration management
type ConfigManager struct {
	config *IstioConfig
}

// NewConfigManager creates a new configuration manager
func NewConfigManager() *ConfigManager {
	return &ConfigManager{
		config: DefaultConfig(),
	}
}

// LoadFromEnvironment loads configuration from environment variables
func (cm *ConfigManager) LoadFromEnvironment() error {
	if pilotAddr := os.Getenv("ISTIO_PILOT_ADDRESS"); pilotAddr != "" {
		cm.config.PilotAddress = pilotAddr
	}

	if namespace := os.Getenv("ISTIO_NAMESPACE"); namespace != "" {
		cm.config.Namespace = namespace
	}

	if enableDiscovery := os.Getenv("ISTIO_ENABLE_DISCOVERY"); enableDiscovery != "" {
		if enabled, err := strconv.ParseBool(enableDiscovery); err == nil {
			cm.config.EnableDiscovery = enabled
		}
	}

	if tlsEnabled := os.Getenv("ISTIO_TLS_ENABLED"); tlsEnabled != "" {
		if enabled, err := strconv.ParseBool(tlsEnabled); err == nil {
			cm.config.TLSEnabled = enabled
		}
	}

	if certPath := os.Getenv("ISTIO_CERT_PATH"); certPath != "" {
		cm.config.CertPath = certPath
	}

	if keyPath := os.Getenv("ISTIO_KEY_PATH"); keyPath != "" {
		cm.config.KeyPath = keyPath
	}

	if caPath := os.Getenv("ISTIO_CA_PATH"); caPath != "" {
		cm.config.CAPath = caPath
	}

	if syncTimeout := os.Getenv("ISTIO_SYNC_TIMEOUT"); syncTimeout != "" {
		if timeout, err := time.ParseDuration(syncTimeout); err == nil {
			cm.config.SyncTimeout = timeout
		}
	}

	return cm.Validate()
}

// LoadFromConfig loads configuration from a config object
func (cm *ConfigManager) LoadFromConfig(config *IstioConfig) error {
	if config == nil {
		return fmt.Errorf("config cannot be nil")
	}

	cm.config = config
	return cm.Validate()
}

// Validate validates the configuration
func (cm *ConfigManager) Validate() error {
	if cm.config.PilotAddress == "" {
		return fmt.Errorf("pilot address cannot be empty")
	}

	if cm.config.Namespace == "" {
		return fmt.Errorf("namespace cannot be empty")
	}

	if cm.config.SyncTimeout <= 0 {
		return fmt.Errorf("sync timeout must be positive")
	}

	// Validate TLS configuration
	if cm.config.TLSEnabled {
		if cm.config.CertPath != "" || cm.config.KeyPath != "" {
			if cm.config.CertPath == "" || cm.config.KeyPath == "" {
				return fmt.Errorf("both cert path and key path must be provided when using client certificates")
			}

			// Check if certificate files exist
			if _, err := os.Stat(cm.config.CertPath); os.IsNotExist(err) {
				return fmt.Errorf("certificate file does not exist: %s", cm.config.CertPath)
			}

			if _, err := os.Stat(cm.config.KeyPath); os.IsNotExist(err) {
				return fmt.Errorf("key file does not exist: %s", cm.config.KeyPath)
			}
		}

		if cm.config.CAPath != "" {
			if _, err := os.Stat(cm.config.CAPath); os.IsNotExist(err) {
				return fmt.Errorf("CA file does not exist: %s", cm.config.CAPath)
			}
		}
	}

	return nil
}

// GetConfig returns the current configuration
func (cm *ConfigManager) GetConfig() *IstioConfig {
	return cm.config
}

// SetDefaults sets default values for unspecified configuration
func (cm *ConfigManager) SetDefaults() {
	if cm.config.PilotAddress == "" {
		cm.config.PilotAddress = "istiod.istio-system.svc.cluster.local:15010"
	}

	if cm.config.Namespace == "" {
		cm.config.Namespace = "istio-system"
	}

	if cm.config.SyncTimeout <= 0 {
		cm.config.SyncTimeout = 30 * time.Second
	}
}

// String returns a string representation of the configuration
func (cm *ConfigManager) String() string {
	var parts []string
	parts = append(parts, fmt.Sprintf("PilotAddress=%s", cm.config.PilotAddress))
	parts = append(parts, fmt.Sprintf("Namespace=%s", cm.config.Namespace))
	parts = append(parts, fmt.Sprintf("EnableDiscovery=%t", cm.config.EnableDiscovery))
	parts = append(parts, fmt.Sprintf("TLSEnabled=%t", cm.config.TLSEnabled))
	parts = append(parts, fmt.Sprintf("SyncTimeout=%s", cm.config.SyncTimeout))

	if cm.config.CertPath != "" {
		parts = append(parts, fmt.Sprintf("CertPath=%s", cm.config.CertPath))
	}
	if cm.config.KeyPath != "" {
		parts = append(parts, fmt.Sprintf("KeyPath=%s", cm.config.KeyPath))
	}
	if cm.config.CAPath != "" {
		parts = append(parts, fmt.Sprintf("CAPath=%s", cm.config.CAPath))
	}

	return strings.Join(parts, ", ")
}
