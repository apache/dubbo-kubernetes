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

package bootstrap

import (
	"context"
	"net"
)

import (
	"github.com/pkg/errors"
)

import (
	dubbo_cp "github.com/apache/dubbo-kubernetes/pkg/config/app/dubbo-cp"
	config_core "github.com/apache/dubbo-kubernetes/pkg/config/core"
	"github.com/apache/dubbo-kubernetes/pkg/config/core/resources/store"
	"github.com/apache/dubbo-kubernetes/pkg/core"
	config_manager "github.com/apache/dubbo-kubernetes/pkg/core/config/manager"
	"github.com/apache/dubbo-kubernetes/pkg/core/datasource"
	"github.com/apache/dubbo-kubernetes/pkg/core/dns/lookup"
	core_plugins "github.com/apache/dubbo-kubernetes/pkg/core/plugins"
	"github.com/apache/dubbo-kubernetes/pkg/core/resources/apis/system"
	core_manager "github.com/apache/dubbo-kubernetes/pkg/core/resources/manager"
	core_store "github.com/apache/dubbo-kubernetes/pkg/core/resources/store"
	core_runtime "github.com/apache/dubbo-kubernetes/pkg/core/runtime"
	"github.com/apache/dubbo-kubernetes/pkg/core/runtime/component"
	dds_context "github.com/apache/dubbo-kubernetes/pkg/dds/context"
	"github.com/apache/dubbo-kubernetes/pkg/dp-server/server"
	"github.com/apache/dubbo-kubernetes/pkg/dubbo"
	"github.com/apache/dubbo-kubernetes/pkg/events"
	"github.com/apache/dubbo-kubernetes/pkg/intercp"
	"github.com/apache/dubbo-kubernetes/pkg/intercp/catalog"
	"github.com/apache/dubbo-kubernetes/pkg/intercp/envoyadmin"
	mesh_cache "github.com/apache/dubbo-kubernetes/pkg/xds/cache/mesh"
	xds_context "github.com/apache/dubbo-kubernetes/pkg/xds/context"
	xds_server "github.com/apache/dubbo-kubernetes/pkg/xds/server"
)

var log = core.Log.WithName("bootstrap")

func buildRuntime(appCtx context.Context, cfg dubbo_cp.Config) (core_runtime.Runtime, error) {
	if err := autoconfigure(&cfg); err != nil {
		return nil, err
	}
	builder, err := core_runtime.BuilderFor(appCtx, cfg)
	if err != nil {
		return nil, err
	}
	for _, plugin := range core_plugins.Plugins().BootstrapPlugins() {
		if err := plugin.BeforeBootstrap(builder, cfg); err != nil {
			return nil, errors.Wrapf(err, "failed to run beforeBootstrap plugin:'%s'", plugin.Name())
		}
	}
	if err := initializeResourceStore(cfg, builder); err != nil {
		return nil, err
	}
	if err := initializeConfigStore(cfg, builder); err != nil {
		return nil, err
	}

	builder.ResourceStore().Customize(system.ConfigType, builder.ConfigStore())

	initializeConfigManager(builder)

	if err := initializeResourceManager(cfg, builder); err != nil { //nolint:contextcheck
		return nil, err
	}

	builder.WithDataSourceLoader(datasource.NewDataSourceLoader(builder.ReadOnlyResourceManager()))

	leaderInfoComponent := &component.LeaderInfoComponent{}
	builder.WithLeaderInfo(leaderInfoComponent)

	builder.WithLookupIP(lookup.CacheLookupIP(net.LookupIP, cfg.General.DNSCacheTTL.Duration))
	builder.WithDpServer(server.NewDpServer(*cfg.DpServer))

	resourceManager := builder.ResourceManager()
	kdsContext := dds_context.DefaultContext(appCtx, resourceManager, cfg)
	builder.WithDDSContext(kdsContext)

	if cfg.Mode == config_core.Global {
		kdsEnvoyAdminClient := dubbo.NewDDSEnvoyAdminClient(
			builder.DDSContext().EnvoyAdminRPCs,
			cfg.Store.Type == store.KubernetesStore,
		)
		forwardingClient := envoyadmin.NewForwardingEnvoyAdminClient(
			builder.ReadOnlyResourceManager(),
			catalog.NewConfigCatalog(resourceManager),
			builder.GetInstanceId(),
			intercp.PooledEnvoyAdminClientFn(builder.InterCPClientPool()),
			kdsEnvoyAdminClient,
		)
		builder.WithEnvoyAdminClient(forwardingClient)
	} else {
		builder.WithEnvoyAdminClient(dubbo.NewEnvoyAdminClient(
			resourceManager,
			builder.Config().GetEnvoyAdminPort(),
		))
	}

	if err := initializeMeshCache(builder); err != nil {
		return nil, err
	}

	for _, plugin := range core_plugins.Plugins().BootstrapPlugins() {
		if err := plugin.AfterBootstrap(builder, cfg); err != nil {
			return nil, errors.Wrapf(err, "failed to run afterBootstrap plugin:'%s'", plugin.Name())
		}
	}

	rt, err := builder.Build()
	if err != nil {
		return nil, err
	}

	if err := rt.Add(leaderInfoComponent); err != nil {
		return nil, err
	}

	for name, plugin := range core_plugins.Plugins().RuntimePlugins() {
		if err := plugin.Customize(rt); err != nil {
			return nil, errors.Wrapf(err, "failed to configure runtime plugin:'%s'", name)
		}
	}
	return rt, nil
}

func Bootstrap(appCtx context.Context, cfg dubbo_cp.Config) (core_runtime.Runtime, error) {
	runtime, err := buildRuntime(appCtx, cfg)
	if err != nil {
		return nil, err
	}

	return runtime, nil
}

func initializeResourceStore(cfg dubbo_cp.Config, builder *core_runtime.Builder) error {
	var pluginName core_plugins.PluginName
	var pluginConfig core_plugins.PluginConfig
	switch cfg.Store.Type {
	case store.KubernetesStore:
		pluginName = core_plugins.Kubernetes
		pluginConfig = nil
	case store.MemoryStore:
		pluginName = core_plugins.Memory
		pluginConfig = nil
	default:
		return errors.Errorf("unknown store type %s", cfg.Store.Type)
	}
	plugin, err := core_plugins.Plugins().ResourceStore(pluginName)
	if err != nil {
		return errors.Wrapf(err, "could not retrieve store %s plugin", pluginName)
	}

	rs, transactions, err := plugin.NewResourceStore(builder, pluginConfig)
	if err != nil {
		return err
	}
	builder.WithResourceStore(core_store.NewCustomizableResourceStore(rs))
	builder.WithTransactions(transactions)
	eventBus, err := events.NewEventBus(cfg.EventBus.BufferSize)
	if err != nil {
		return err
	}
	if err := plugin.EventListener(builder, eventBus); err != nil {
		return err
	}
	builder.WithEventBus(eventBus)
	return nil
}

func initializeConfigStore(cfg dubbo_cp.Config, builder *core_runtime.Builder) error {
	var pluginName core_plugins.PluginName
	var pluginConfig core_plugins.PluginConfig
	switch cfg.Store.Type {
	case store.KubernetesStore:
		pluginName = core_plugins.Kubernetes
	case store.MemoryStore:
		pluginName = core_plugins.Universal
	default:
		return errors.Errorf("unknown store type %s", cfg.Store.Type)
	}
	plugin, err := core_plugins.Plugins().ConfigStore(pluginName)
	if err != nil {
		return errors.Wrapf(err, "could not retrieve secret store %s plugin", pluginName)
	}
	if cs, err := plugin.NewConfigStore(builder, pluginConfig); err != nil {
		return err
	} else {
		builder.WithConfigStore(cs)
		return nil
	}
}

func initializeResourceManager(cfg dubbo_cp.Config, builder *core_runtime.Builder) error {
	defaultManager := core_manager.NewResourceManager(builder.ResourceStore())
	customizableManager := core_manager.NewCustomizableResourceManager(defaultManager, nil)

	builder.WithResourceManager(customizableManager)

	if builder.Config().Store.Cache.Enabled {
		cachedManager, err := core_manager.NewCachedManager(
			customizableManager,
			builder.Config().Store.Cache.ExpirationTime.Duration,
		)
		if err != nil {
			return err
		}
		builder.WithReadOnlyResourceManager(cachedManager)
	} else {
		builder.WithReadOnlyResourceManager(customizableManager)
	}
	return nil
}

func initializeConfigManager(builder *core_runtime.Builder) {
	builder.WithConfigManager(config_manager.NewConfigManager(builder.ConfigStore()))
}

func initializeMeshCache(builder *core_runtime.Builder) error {
	meshContextBuilder := xds_context.NewMeshContextBuilder(
		builder.ReadOnlyResourceManager(),
		xds_server.MeshResourceTypes(),
		builder.LookupIP(),
		builder.Config().Multizone.Zone.Name)

	meshSnapshotCache, err := mesh_cache.NewCache(
		builder.Config().Store.Cache.ExpirationTime.Duration,
		meshContextBuilder)
	if err != nil {
		return err
	}

	builder.WithMeshCache(meshSnapshotCache)
	return nil
}
