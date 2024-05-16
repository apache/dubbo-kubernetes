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

package manager

import (
	"context"
	"time"
)

import (
	config_model "github.com/apache/dubbo-kubernetes/pkg/core/resources/apis/system"
	"github.com/apache/dubbo-kubernetes/pkg/core/resources/model"
	core_store "github.com/apache/dubbo-kubernetes/pkg/core/resources/store"
)

const ClusterIdConfigKey = "dubbo-cluster-id"

type ConfigManager interface {
	Create(context.Context, *config_model.ConfigResource, ...core_store.CreateOptionsFunc) error
	Update(context.Context, *config_model.ConfigResource, ...core_store.UpdateOptionsFunc) error
	Delete(context.Context, *config_model.ConfigResource, ...core_store.DeleteOptionsFunc) error
	DeleteAll(context.Context, ...core_store.DeleteAllOptionsFunc) error
	Get(context.Context, *config_model.ConfigResource, ...core_store.GetOptionsFunc) error
	List(context.Context, *config_model.ConfigResourceList, ...core_store.ListOptionsFunc) error
}

func NewConfigManager(configStore core_store.ResourceStore) ConfigManager {
	return &configManager{
		configStore: configStore,
	}
}

var _ ConfigManager = &configManager{}

type configManager struct {
	configStore core_store.ResourceStore
}

func (s *configManager) Get(ctx context.Context, config *config_model.ConfigResource, fs ...core_store.GetOptionsFunc) error {
	return s.configStore.Get(ctx, config, fs...)
}

func (s *configManager) List(ctx context.Context, configs *config_model.ConfigResourceList, fs ...core_store.ListOptionsFunc) error {
	return s.configStore.List(ctx, configs, fs...)
}

func (s *configManager) Create(ctx context.Context, config *config_model.ConfigResource, fs ...core_store.CreateOptionsFunc) error {
	return s.configStore.Create(ctx, config, append(fs, core_store.CreatedAt(time.Now()))...)
}

func (s *configManager) Update(ctx context.Context, config *config_model.ConfigResource, fs ...core_store.UpdateOptionsFunc) error {
	return s.configStore.Update(ctx, config, append(fs, core_store.ModifiedAt(time.Now()))...)
}

func (s *configManager) Delete(ctx context.Context, config *config_model.ConfigResource, fs ...core_store.DeleteOptionsFunc) error {
	return s.configStore.Delete(ctx, config, fs...)
}

func (s *configManager) DeleteAll(ctx context.Context, fs ...core_store.DeleteAllOptionsFunc) error {
	list := &config_model.ConfigResourceList{}
	opts := core_store.NewDeleteAllOptions(fs...)
	if err := s.configStore.List(ctx, list, core_store.ListByMesh(opts.Mesh)); err != nil {
		return err
	}
	for _, item := range list.Items {
		if err := s.Delete(ctx, item, core_store.DeleteBy(model.MetaToResourceKey(item.Meta))); err != nil && !core_store.IsResourceNotFound(err) {
			return err
		}
	}
	return nil
}
