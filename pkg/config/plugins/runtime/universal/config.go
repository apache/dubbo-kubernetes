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
