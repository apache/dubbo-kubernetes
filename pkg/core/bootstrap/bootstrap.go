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
	"net/url"
	"strings"
	"sync"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/extension"
	"dubbo.apache.org/dubbo-go/v3/config/instance"
	"dubbo.apache.org/dubbo-go/v3/config_center"

	"github.com/pkg/errors"
)

import (
	dubbo_cp "github.com/apache/dubbo-kubernetes/pkg/config/app/dubbo-cp"
	config_core "github.com/apache/dubbo-kubernetes/pkg/config/core"
	"github.com/apache/dubbo-kubernetes/pkg/config/core/resources/store"
	"github.com/apache/dubbo-kubernetes/pkg/core"
	admin2 "github.com/apache/dubbo-kubernetes/pkg/core/admin"
	config_manager "github.com/apache/dubbo-kubernetes/pkg/core/config/manager"
	"github.com/apache/dubbo-kubernetes/pkg/core/consts"
	"github.com/apache/dubbo-kubernetes/pkg/core/datasource"
	"github.com/apache/dubbo-kubernetes/pkg/core/extensions"
	"github.com/apache/dubbo-kubernetes/pkg/core/governance"
	"github.com/apache/dubbo-kubernetes/pkg/core/logger"
	"github.com/apache/dubbo-kubernetes/pkg/core/managers/apis/condition_route"
	dataplane_managers "github.com/apache/dubbo-kubernetes/pkg/core/managers/apis/dataplane"
	"github.com/apache/dubbo-kubernetes/pkg/core/managers/apis/dynamic_config"
	mapping_managers "github.com/apache/dubbo-kubernetes/pkg/core/managers/apis/mapping"
	mesh_managers "github.com/apache/dubbo-kubernetes/pkg/core/managers/apis/mesh"
	metadata_managers "github.com/apache/dubbo-kubernetes/pkg/core/managers/apis/metadata"
	"github.com/apache/dubbo-kubernetes/pkg/core/managers/apis/tag_route"
	core_plugins "github.com/apache/dubbo-kubernetes/pkg/core/plugins"
	dubbo_registry "github.com/apache/dubbo-kubernetes/pkg/core/registry"
	"github.com/apache/dubbo-kubernetes/pkg/core/resources/apis/mesh"
	core_manager "github.com/apache/dubbo-kubernetes/pkg/core/resources/manager"
	"github.com/apache/dubbo-kubernetes/pkg/core/resources/registry"
	core_store "github.com/apache/dubbo-kubernetes/pkg/core/resources/store"
	core_runtime "github.com/apache/dubbo-kubernetes/pkg/core/runtime"
	"github.com/apache/dubbo-kubernetes/pkg/core/runtime/component"
	dds_context "github.com/apache/dubbo-kubernetes/pkg/dds/context"
	"github.com/apache/dubbo-kubernetes/pkg/dp-server/server"
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
	// 定义store的状态
	if cfg.DeployMode == config_core.UniversalMode || cfg.DeployMode == config_core.HalfHostMode {
		cfg.Store.Type = store.Traditional
	} else {
		cfg.Store.Type = store.KubernetesStore
	}
	// 初始化传统微服务体系所需要的组件
	if err := initializeTraditional(cfg, builder); err != nil {
		return nil, err
	}
	if err := initializeResourceStore(cfg, builder); err != nil {
		return nil, err
	}

	// 隐蔽了configStore, 后期再补全

	builder.WithResourceValidators(core_runtime.ResourceValidators{})

	if err := initializeResourceManager(cfg, builder); err != nil { //nolint:contextcheck
		return nil, err
	}

	builder.WithDataSourceLoader(datasource.NewDataSourceLoader(builder.ReadOnlyResourceManager()))

	leaderInfoComponent := &component.LeaderInfoComponent{}
	builder.WithLeaderInfo(leaderInfoComponent)

	builder.WithDataplaneCache(&sync.Map{})
	builder.WithDpServer(server.NewDpServer(*cfg.DpServer))

	resourceManager := builder.ResourceManager()
	kdsContext := dds_context.DefaultContext(appCtx, resourceManager, cfg)
	builder.WithDDSContext(kdsContext)

	if cfg.Mode == config_core.Global {
		kdsEnvoyAdminClient := admin2.NewDDSEnvoyAdminClient(
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
		builder.WithEnvoyAdminClient(admin2.NewEnvoyAdminClient(
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

func initializeTraditional(cfg dubbo_cp.Config, builder *core_runtime.Builder) error {
	// 如果是k8s环境模式直接返回, 这里针对传统的微服务体系(包括纯vm和半托管)
	if cfg.DeployMode == config_core.KubernetesMode {
		return nil
	}
	address := cfg.Store.Traditional.ConfigCenter
	registryAddress := cfg.Store.Traditional.Registry.Address
	metadataReportAddress := cfg.Store.Traditional.MetadataReport.Address
	c, addrUrl := getValidAddressConfig(address, registryAddress)
	configCenter := newConfigCenter(c, addrUrl)
	properties, err := configCenter.GetProperties(consts.DubboPropertyKey)
	if err != nil {
		logger.Info("No configuration found in config center.")
	}
	if len(properties) > 0 {
		logger.Infof("Loaded remote configuration from config center:\n %s", properties)
		for _, property := range strings.Split(properties, "\n") {
			if strings.HasPrefix(property, consts.RegistryAddressKey) {
				registryAddress = strings.Split(property, "=")[1]
			}
			if strings.HasPrefix(property, consts.MetadataReportAddressKey) {
				metadataReportAddress = strings.Split(property, "=")[1]
			}
		}
	}
	if len(registryAddress) > 0 {
		logger.Infof("Valid registry address is %s", registryAddress)
		c := newAddressConfig(registryAddress)
		addrUrl, err := c.ToURL()
		if err != nil {
			panic(err)
		}

		fac := extensions.GetRegClientFactory(addrUrl.Protocol)
		if fac != nil {
			regClient := fac.CreateRegClient(addrUrl)
			builder.WithRegClient(regClient)
		} else {
			logger.Sugar().Infof("Metadata of type %v not registered.", addrUrl.Protocol)
		}

		registryCenter, err := extension.GetRegistry(c.GetProtocol(), addrUrl)
		if err != nil {
			return err
		}
		builder.WithGovernanceConfig(governance.NewGovernanceConfig(configCenter, registryCenter, c.GetProtocol()))
		builder.WithRegistryCenter(registryCenter)
		delegate, err := extension.GetRegistry(addrUrl.Protocol, addrUrl)
		if err != nil {
			logger.Error("Error initialize registry instance.")
			return err
		}

		sdUrl := addrUrl.Clone()
		sdUrl.AddParam("registry", addrUrl.Protocol)
		sdUrl.Protocol = "service-discovery"
		sdDelegate, err := extension.GetServiceDiscovery(sdUrl)
		if err != nil {
			logger.Error("Error initialize service discovery instance.")
			return err
		}
		builder.WithServiceDiscovery(sdDelegate)
		adminRegistry := dubbo_registry.NewRegistry(delegate, sdDelegate)
		builder.WithAdminRegistry(adminRegistry)
	}
	if len(metadataReportAddress) > 0 {
		logger.Infof("Valid meta center address is %s", metadataReportAddress)
		c := newAddressConfig(metadataReportAddress)
		addrUrl, err := c.ToURL()
		if err != nil {
			panic(err)
		}
		factory := extension.GetMetadataReportFactory(c.GetProtocol())
		metadataReport := factory.CreateMetadataReport(addrUrl)
		builder.WithMetadataReport(metadataReport)
	}
	// 设置MetadataReportUrl
	instance.SetMetadataReportUrl(addrUrl)
	// 设置MetadataReportInstance
	instance.SetMetadataReportInstanceByReg(addrUrl)

	return nil
}

func getValidAddressConfig(address string, registryAddress string) (store.AddressConfig, *common.URL) {
	if len(address) <= 0 && len(registryAddress) <= 0 {
		panic("Must at least specify `admin.config-center.address` or `admin.registry.address`!")
	}

	var c store.AddressConfig
	if len(address) > 0 {
		logger.Infof("Specified config center address is %s", address)
		c = newAddressConfig(address)
	} else {
		logger.Info("Using registry address as default config center address")
		c = newAddressConfig(registryAddress)
	}

	configUrl, err := c.ToURL()
	if err != nil {
		panic(err)
	}
	return c, configUrl
}

func newAddressConfig(address string) store.AddressConfig {
	cfg := store.AddressConfig{}
	cfg.Address = address
	var err error
	cfg.Url, err = url.Parse(address)
	if err != nil {
		panic(err)
	}
	return cfg
}

func newConfigCenter(c store.AddressConfig, url *common.URL) config_center.DynamicConfiguration {
	factory, err := extension.GetConfigCenterFactory(c.GetProtocol())
	if err != nil {
		logger.Info(err.Error())
		panic(err)
	}

	configCenter, err := factory.GetDynamicConfiguration(url)
	if err != nil {
		logger.Info("Failed to init config center, error msg is %s.", err.Error())
		panic(err)
	}
	return configCenter
}

func initializeResourceStore(cfg dubbo_cp.Config, builder *core_runtime.Builder) error {
	var pluginName core_plugins.PluginName
	var pluginConfig core_plugins.PluginConfig
	switch cfg.Store.Type {
	case store.KubernetesStore:
		pluginName = core_plugins.Kubernetes
		pluginConfig = nil
	case store.Traditional:
		pluginName = core_plugins.Traditional
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
	case store.Traditional:
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

	customizableManager.Customize(
		mesh.DataplaneType,
		dataplane_managers.NewDataplaneManager(
			builder.ResourceStore(),
			cfg.Multizone.Zone.Name,
		))

	customizableManager.Customize(
		mesh.MappingType,
		mapping_managers.NewMappingManager(
			builder.ResourceStore(),
		))

	customizableManager.Customize(
		mesh.MetaDataType,
		metadata_managers.NewMetadataManager(
			builder.ResourceStore(),
		))

	customizableManager.Customize(
		mesh.ConditionRouteType,
		condition_route.NewConditionRouteManager(
			builder.ResourceStore(),
		))

	customizableManager.Customize(
		mesh.TagRouteType,
		tag_route.NewTagRouteManager(
			builder.ResourceStore(),
		))

	customizableManager.Customize(
		mesh.DynamicConfigType,
		dynamic_config.NewDynamicConfigManager(
			builder.ResourceStore(),
		))

	customizableManager.Customize(
		mesh.MeshType,
		mesh_managers.NewMeshManager(
			builder.ResourceStore(),
			customizableManager,
			registry.Global(),
			builder.ResourceValidators().Mesh,
			builder.Extensions(),
			cfg,
		),
	)

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
