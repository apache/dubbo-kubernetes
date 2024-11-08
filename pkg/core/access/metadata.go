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
	"github.com/apache/dubbo-kubernetes/pkg/core/user"
)

type ControlPlaneMetadataAccess interface {
	ValidateView(ctx context.Context, user user.User) error
}

type staticMetadataAccess struct {
	usernames map[string]struct{}
	groups    map[string]struct{}
}

func NewStaticControlPlaneMetadataAccess(cfg config_access.ControlPlaneMetadataStaticAccessConfig) ControlPlaneMetadataAccess {
	s := &staticMetadataAccess{
		usernames: make(map[string]struct{}, len(cfg.Users)),
		groups:    make(map[string]struct{}, len(cfg.Groups)),
	}
	for _, u := range cfg.Users {
		s.usernames[u] = struct{}{}
	}
	for _, g := range cfg.Groups {
		s.groups[g] = struct{}{}
	}
	return s
}

func (s staticMetadataAccess) ValidateView(ctx context.Context, user user.User) error {
	return Validate(s.usernames, s.groups, user, "control-plane metadata")
}
