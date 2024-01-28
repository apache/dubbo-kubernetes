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

package plugins

import (
	"github.com/pkg/errors"
)

import (
	core_store "github.com/apache/dubbo-kubernetes/pkg/core/resources/store"
	core_runtime "github.com/apache/dubbo-kubernetes/pkg/core/runtime"
	core_xds "github.com/apache/dubbo-kubernetes/pkg/core/xds"
	"github.com/apache/dubbo-kubernetes/pkg/events"
	xds_context "github.com/apache/dubbo-kubernetes/pkg/xds/context"
)

type Plugin interface{}

type PluginConfig interface{}

type PluginContext = core_runtime.BuilderContext

type MutablePluginContext = core_runtime.Builder

// EnvironmentPreparingOrder describes an order at which base environment plugins (Universal/Kubernetes) configure the control plane.
var EnvironmentPreparingOrder = 0

// EnvironmentPreparedOrder describes an order at which you can put a plugin and expect that
// the base environment is already configured by Universal/Kubernetes plugins.
var EnvironmentPreparedOrder = EnvironmentPreparingOrder + 1

// BootstrapPlugin is responsible for environment-specific initialization at start up,
// e.g. Kubernetes-specific part of configuration.
// Unlike other plugins, can mutate plugin context directly.
type BootstrapPlugin interface {
	Plugin
	BeforeBootstrap(*MutablePluginContext, PluginConfig) error
	AfterBootstrap(*MutablePluginContext, PluginConfig) error
	Name() PluginName
	// Order defines an order in which plugins are applied on the control plane.
	// If you don't have specific need, consider using EnvironmentPreparedOrder
	Order() int
}

// ResourceStorePlugin is responsible for instantiating a particular ResourceStore.
type (
	DbVersion           = uint
	ResourceStorePlugin interface {
		Plugin
		NewResourceStore(PluginContext, PluginConfig) (core_store.ResourceStore, core_store.Transactions, error)
		Migrate(PluginContext, PluginConfig) (DbVersion, error)
		EventListener(PluginContext, events.Emitter) error
	}
)

var AlreadyMigrated = errors.New("database already migrated")

// ConfigStorePlugin is responsible for instantiating a particular ConfigStore.
type ConfigStorePlugin interface {
	Plugin
	NewConfigStore(PluginContext, PluginConfig) (core_store.ResourceStore, error)
}

// RuntimePlugin is responsible for registering environment-specific components,
// e.g. Kubernetes admission web hooks.
type RuntimePlugin interface {
	Plugin
	Customize(core_runtime.Runtime) error
}

// PolicyPlugin a plugin to add a Policy to dubbo
type PolicyPlugin interface {
	Plugin
	// Apply to `rs` using the `ctx` and `proxy` the mutation for all policies of the type this plugin implements.
	// You can access matching policies by using `proxy.Policies.Dynamic`.
	Apply(rs *core_xds.ResourceSet, ctx xds_context.Context, proxy *core_xds.Proxy) error
}
