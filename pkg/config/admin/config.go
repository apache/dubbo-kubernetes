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

package admin

import (
	"errors"
	"github.com/apache/dubbo-kubernetes/pkg/config/admin/auth"
	. "github.com/apache/dubbo-kubernetes/pkg/config/admin/observability"
	"go.uber.org/multierr"
)

import (
	"github.com/apache/dubbo-kubernetes/pkg/config"
)

type Admin struct {
	config.BaseConfig
	Port             int                    `json:"port" envconfig:"DUBBO_ADMIN_PORT"`
	MetricDashboards *MetricDashboardConfig `json:"metricDashboards"`
	TraceDashboards  *TraceDashboardConfig  `json:"traceDashboards"`
	Prometheus       string                 `json:"prometheus"`
	Grafana          string                 `json:"grafana"`
	Auth             *auth.AuthConfig       `json:"auth"`
}

func (s *Admin) PostProcess() error {
	return multierr.Combine(
		s.MetricDashboards.PostProcess(),
		s.TraceDashboards.PostProcess(),
	)
}

func (s *Admin) Validate() error {
	if s.MetricDashboards != nil {
		if err := s.MetricDashboards.Validate(); err != nil {
			return err
		}
	}
	if s.TraceDashboards != nil {
		if err := s.TraceDashboards.Validate(); err != nil {
			return err
		}
	}
	if s.Auth == nil {
		return errors.New("auth config is needed, but found empty")
	}
	if err := s.Auth.Validate(); err != nil {
		return err
	}
	return nil
}

func DefaultAdminConfig() *Admin {
	return &Admin{
		Port: 8888,
	}
}
