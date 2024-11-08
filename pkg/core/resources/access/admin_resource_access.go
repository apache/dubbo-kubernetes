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
	"fmt"

	config_access "github.com/apache/dubbo-kubernetes/pkg/config/access"
	"github.com/apache/dubbo-kubernetes/pkg/core/access"
	"github.com/apache/dubbo-kubernetes/pkg/core/resources/model"
	"github.com/apache/dubbo-kubernetes/pkg/core/user"
)

type adminResourceAccess struct {
	usernames map[string]struct{}
	groups    map[string]struct{}
}

func NewAdminResourceAccess(cfg config_access.AdminResourcesStaticAccessConfig) ResourceAccess {
	a := &adminResourceAccess{
		usernames: make(map[string]struct{}),
		groups:    make(map[string]struct{}),
	}
	for _, user := range cfg.Users {
		a.usernames[user] = struct{}{}
	}

	for _, group := range cfg.Groups {
		a.groups[group] = struct{}{}
	}
	return a
}

var _ ResourceAccess = &adminResourceAccess{}

func (a *adminResourceAccess) ValidateCreate(ctx context.Context, _ model.ResourceKey, _ model.ResourceSpec, descriptor model.ResourceTypeDescriptor, user user.User) error {
	return a.validateAdminAccess(ctx, user, descriptor)
}

func (a *adminResourceAccess) ValidateUpdate(ctx context.Context, _ model.ResourceKey, _ model.ResourceSpec, _ model.ResourceSpec, descriptor model.ResourceTypeDescriptor, user user.User) error {
	return a.validateAdminAccess(ctx, user, descriptor)
}

func (a *adminResourceAccess) ValidateDelete(ctx context.Context, _ model.ResourceKey, _ model.ResourceSpec, descriptor model.ResourceTypeDescriptor, user user.User) error {
	return a.validateAdminAccess(ctx, user, descriptor)
}

func (a *adminResourceAccess) ValidateList(ctx context.Context, _ string, descriptor model.ResourceTypeDescriptor, user user.User) error {
	return a.validateAdminAccess(ctx, user, descriptor)
}

func (a *adminResourceAccess) ValidateGet(ctx context.Context, _ model.ResourceKey, descriptor model.ResourceTypeDescriptor, user user.User) error {
	return a.validateAdminAccess(ctx, user, descriptor)
}

func (r *adminResourceAccess) validateAdminAccess(_ context.Context, u user.User, descriptor model.ResourceTypeDescriptor) error {
	if !descriptor.AdminOnly {
		return nil
	}
	return access.Validate(r.usernames, r.groups, u, fmt.Sprintf("the resource of type %q", descriptor.Name))
}
