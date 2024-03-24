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

package zone

import (
	"context"
	core_manager "github.com/apache/dubbo-kubernetes/pkg/core/resources/manager"
	"github.com/apache/dubbo-kubernetes/pkg/core/resources/model"
	core_store "github.com/apache/dubbo-kubernetes/pkg/core/resources/store"
)

func NewZoneManager(store core_store.ResourceStore, validator Validator, unsafeDelete bool) core_manager.ResourceManager {
	return &zoneManager{
		ResourceManager: core_manager.NewResourceManager(store),
		store:           store,
		validator:       validator,
		unsafeDelete:    unsafeDelete,
	}
}

type zoneManager struct {
	core_manager.ResourceManager
	store        core_store.ResourceStore
	validator    Validator
	unsafeDelete bool
}

func (z *zoneManager) Delete(ctx context.Context, r model.Resource, opts ...core_store.DeleteOptionsFunc) error {
	options := core_store.NewDeleteOptions(opts...)
	if !z.unsafeDelete {
		if err := z.validator.ValidateDelete(ctx, options.Name); err != nil {
			return err
		}
	}
	return z.ResourceManager.Delete(ctx, r, opts...)
}
