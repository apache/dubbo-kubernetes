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

package dubbo

import (
	"time"
)

import (
	"github.com/pkg/errors"

	"go.uber.org/multierr"
)

func DefaultServiceNameMappingConfig() DubboConfig {
	return DubboConfig{}
}

type DubboConfig struct {
	Debounce Debounce `json:"debounce"`
}

func (s *DubboConfig) Validate() error {
	var errs error
	if err := s.Debounce.Validate(); err != nil {
		errs = multierr.Append(errs, errors.Wrap(err, ".Debounce validation failed"))
	}

	return errs
}

type Debounce struct {
	After  time.Duration `yaml:"after"`
	Max    time.Duration `yaml:"max"`
	Enable bool          `yaml:"enable"`
}

func (s *Debounce) Validate() error {
	var errs error

	afterThreshold := time.Second * 10
	if s.After > afterThreshold {
		errs = multierr.Append(errs, errors.New(".After can not greater than "+afterThreshold.String()))
	}

	maxThreshold := time.Second * 10
	if s.Max > maxThreshold {
		errs = multierr.Append(errs, errors.New(".Max can not greater than "+maxThreshold.String()))
	}

	return errs
}
