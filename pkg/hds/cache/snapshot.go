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

package cache

import (
	util_xds_v3 "github.com/apache/dubbo-kubernetes/pkg/util/xds/v3"
	envoy_service_health_v3 "github.com/envoyproxy/go-control-plane/envoy/service/health/v3"
	envoy_types "github.com/envoyproxy/go-control-plane/pkg/cache/types"
	"github.com/envoyproxy/go-control-plane/pkg/cache/v3"
	"github.com/pkg/errors"
)

const HealthCheckSpecifierType = "envoy.service.health.v3.HealthCheckSpecifier"

func NewSnapshot(version string, hcs *envoy_service_health_v3.HealthCheckSpecifier) util_xds_v3.Snapshot {
	return &Snapshot{
		HealthChecks: cache.Resources{
			Version: version,
			Items: map[string]envoy_types.ResourceWithTTL{
				"hcs": {Resource: hcs},
			},
		},
	}
}

// Snapshot is an internally consistent snapshot of HDS resources.
type Snapshot struct {
	HealthChecks cache.Resources
}

func (s *Snapshot) GetSupportedTypes() []string {
	return []string{HealthCheckSpecifierType}
}

func (s *Snapshot) Consistent() error {
	if s == nil {
		return errors.New("nil Snapshot")
	}
	return nil
}

func (s *Snapshot) GetResources(typ string) map[string]envoy_types.Resource {
	if s == nil || typ != HealthCheckSpecifierType {
		return nil
	}
	withoutTtl := make(map[string]envoy_types.Resource, len(s.HealthChecks.Items))
	for name, res := range s.HealthChecks.Items {
		withoutTtl[name] = res.Resource
	}
	return withoutTtl
}

func (s *Snapshot) GetVersion(typ string) string {
	if s == nil || typ != HealthCheckSpecifierType {
		return ""
	}
	return s.HealthChecks.Version
}

func (s *Snapshot) WithVersion(typ string, version string) util_xds_v3.Snapshot {
	if s == nil {
		return nil
	}
	if s.GetVersion(typ) == version || typ != HealthCheckSpecifierType {
		return s
	}
	n := cache.Resources{
		Version: version,
		Items:   s.HealthChecks.Items,
	}
	return &Snapshot{HealthChecks: n}
}
