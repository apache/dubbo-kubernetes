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

package memory

import (
	"fmt"

	"github.com/apache/dubbo-kubernetes/dubbod/planet/pkg/model"
	"github.com/apache/dubbo-kubernetes/pkg/config"
	"github.com/apache/dubbo-kubernetes/pkg/config/schema/collection"
	"github.com/apache/dubbo-kubernetes/pkg/slices"
	"k8s.io/apimachinery/pkg/types"
)

type Controller struct {
	monitor     Monitor
	configStore model.ConfigStore
	hasSynced   func() bool

	namespacesFilter func(obj interface{}) bool
}

func NewController(cs model.ConfigStore) *Controller {
	out := &Controller{
		configStore: cs,
		// TODO monitor
	}
	return out
}

func (c *Controller) Run(stop <-chan struct{}) {
	c.monitor.Run(stop)
}

func (c *Controller) HasSynced() bool {
	if c.hasSynced != nil {
		return c.hasSynced()
	}
	return true
}

func (c *Controller) Schemas() collection.Schemas {
	return c.configStore.Schemas()
}

func (c *Controller) Get(kind config.GroupVersionKind, key, namespace string) *config.Config {
	if c.namespacesFilter != nil && !c.namespacesFilter(namespace) {
		return nil
	}
	return c.configStore.Get(kind, key, namespace)
}

func (c *Controller) Create(config config.Config) (revision string, err error) {
	if revision, err = c.configStore.Create(config); err == nil {
		c.monitor.ScheduleProcessEvent(ConfigEvent{
			config: config,
			event:  model.EventAdd,
		})
	}
	return
}

func (c *Controller) Update(config config.Config) (newRevision string, err error) {
	oldconfig := c.configStore.Get(config.GroupVersionKind, config.Name, config.Namespace)
	if newRevision, err = c.configStore.Update(config); err == nil {
		c.monitor.ScheduleProcessEvent(ConfigEvent{
			old:    *oldconfig,
			config: config,
			event:  model.EventUpdate,
		})
	}
	return
}

func (c *Controller) UpdateStatus(config config.Config) (newRevision string, err error) {
	oldconfig := c.configStore.Get(config.GroupVersionKind, config.Name, config.Namespace)
	if newRevision, err = c.configStore.UpdateStatus(config); err == nil {
		c.monitor.ScheduleProcessEvent(ConfigEvent{
			old:    *oldconfig,
			config: config,
			event:  model.EventUpdate,
		})
	}
	return
}

func (c *Controller) Patch(orig config.Config, patchFn config.PatchFunc) (newRevision string, err error) {
	cfg, typ := patchFn(orig.DeepCopy())
	switch typ {
	case types.MergePatchType:
	case types.JSONPatchType:
	default:
		return "", fmt.Errorf("unsupported merge type: %s", typ)
	}
	if newRevision, err = c.configStore.Patch(cfg, patchFn); err == nil {
		c.monitor.ScheduleProcessEvent(ConfigEvent{
			old:    orig,
			config: cfg,
			event:  model.EventUpdate,
		})
	}
	return
}

func (c *Controller) Delete(kind config.GroupVersionKind, key, namespace string, resourceVersion *string) error {
	if config := c.Get(kind, key, namespace); config != nil {
		if err := c.configStore.Delete(kind, key, namespace, resourceVersion); err != nil {
			return err
		}
		c.monitor.ScheduleProcessEvent(ConfigEvent{
			config: *config,
			event:  model.EventDelete,
		})
		return nil
	}
	return fmt.Errorf("delete: config %v/%v/%v does not exist", kind, namespace, key)
}

func (c *Controller) List(kind config.GroupVersionKind, namespace string) []config.Config {
	configs := c.configStore.List(kind, namespace)
	if c.namespacesFilter != nil {
		return slices.Filter(configs, func(config config.Config) bool {
			return c.namespacesFilter(config)
		})
	}
	return configs
}

func (c *Controller) RegisterEventHandler(kind config.GroupVersionKind, f model.EventHandler) {
	c.monitor.AppendEventHandler(kind, f)
}

func (c *Controller) RegisterHasSyncedHandler(cb func() bool) {
	c.hasSynced = cb
}
