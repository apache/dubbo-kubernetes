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

	"github.com/apache/dubbo-kubernetes/pkg/core/resources/model"
	"github.com/apache/dubbo-kubernetes/pkg/core/user"
)

type ResourceAccess interface {
	ValidateCreate(ctx context.Context, key model.ResourceKey, spec model.ResourceSpec, desc model.ResourceTypeDescriptor, user user.User) error
	ValidateUpdate(ctx context.Context, key model.ResourceKey, currentSpec model.ResourceSpec, newSpec model.ResourceSpec, desc model.ResourceTypeDescriptor, user user.User) error
	ValidateDelete(ctx context.Context, key model.ResourceKey, spec model.ResourceSpec, desc model.ResourceTypeDescriptor, user user.User) error
	ValidateList(ctx context.Context, mesh string, desc model.ResourceTypeDescriptor, user user.User) error
	ValidateGet(ctx context.Context, key model.ResourceKey, desc model.ResourceTypeDescriptor, user user.User) error
}
