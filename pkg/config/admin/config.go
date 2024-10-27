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
	"go.uber.org/multierr"
)

import (
	"github.com/apache/dubbo-kubernetes/pkg/config"
	. "github.com/apache/dubbo-kubernetes/pkg/config/observability"
)

type Admin struct {
	config.BaseConfig
	Port             int                    `json:"port" envconfig:"DUBBO_ADMIN_PORT"`
	MetricDashboards *MetricDashboardConfig `json:"metric_dashboards"`
	TraceDashboards  *TraceDashboardConfig  `json:"trace_dashboards"`
	Prometheus       string                 `json:"prometheus"`
}

func (s *Admin) PostProcess() error {
	return multierr.Combine(
		s.MetricDashboards.PostProcess(),
		s.TraceDashboards.PostProcess(),
	)
}

func (s *Admin) Validate() error {
	return multierr.Combine(
		s.MetricDashboards.Validate(),
		s.TraceDashboards.Validate(),
	)
}

func DefaultAdminConfig() *Admin {
	return &Admin{
		Port: 8888,
	}
}
