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

package access

import (
	"context"

	config_access "github.com/apache/dubbo-kubernetes/pkg/config/access"
	"github.com/apache/dubbo-kubernetes/pkg/core/access"
	"github.com/apache/dubbo-kubernetes/pkg/core/user"
)

type staticEnvoyAdminAccess struct {
	configDump accessMaps
	stats      accessMaps
	clusters   accessMaps
}

type accessMaps struct {
	usernames map[string]struct{}
	groups    map[string]struct{}
}

func (am accessMaps) Validate(user user.User) error {
	return access.Validate(am.usernames, am.groups, user, "envoy proxy info")
}

var _ EnvoyAdminAccess = &staticEnvoyAdminAccess{}

func NewStaticEnvoyAdminAccess(
	configDumpCfg config_access.ViewConfigDumpStaticAccessConfig,
	statsCfg config_access.ViewStatsStaticAccessConfig,
	clustersCfg config_access.ViewClustersStaticAccessConfig,
) EnvoyAdminAccess {
	return &staticEnvoyAdminAccess{
		configDump: mapAccess(configDumpCfg.Users, configDumpCfg.Groups),
		stats:      mapAccess(statsCfg.Users, statsCfg.Groups),
		clusters:   mapAccess(clustersCfg.Users, clustersCfg.Groups),
	}
}

func mapAccess(users []string, groups []string) accessMaps {
	m := accessMaps{
		usernames: make(map[string]struct{}, len(users)),
		groups:    make(map[string]struct{}, len(groups)),
	}
	for _, usr := range users {
		m.usernames[usr] = struct{}{}
	}
	for _, group := range groups {
		m.groups[group] = struct{}{}
	}
	return m
}

func (s *staticEnvoyAdminAccess) ValidateViewConfigDump(ctx context.Context, user user.User) error {
	return s.configDump.Validate(user)
}

func (s *staticEnvoyAdminAccess) ValidateViewStats(ctx context.Context, user user.User) error {
	return s.stats.Validate(user)
}

func (s *staticEnvoyAdminAccess) ValidateViewClusters(ctx context.Context, user user.User) error {
	return s.clusters.Validate(user)
}
