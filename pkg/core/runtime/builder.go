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

package runtime

import (
	"context"
	"fmt"
	"github.com/apache/dubbo-kubernetes/pkg/envoy/admin"
	"github.com/apache/dubbo-kubernetes/pkg/metrics"
	"github.com/apache/dubbo-kubernetes/pkg/xds/secrets"
	"os"
	"sync"
	"time"
)

import (
	"dubbo.apache.org/dubbo-go/v3/config_center"
	"dubbo.apache.org/dubbo-go/v3/metadata/report"
	dubboRegistry "dubbo.apache.org/dubbo-go/v3/registry"

	"github.com/pkg/errors"
)

import (
	api_server "github.com/apache/dubbo-kubernetes/pkg/api-server/customization"
	dubbo_cp "github.com/apache/dubbo-kubernetes/pkg/config/app/dubbo-cp"
	"github.com/apache/dubbo-kubernetes/pkg/core"
	core_ca "github.com/apache/dubbo-kubernetes/pkg/core/ca"
	config_manager "github.com/apache/dubbo-kubernetes/pkg/core/config/manager"
	"github.com/apache/dubbo-kubernetes/pkg/core/datasource"
	"github.com/apache/dubbo-kubernetes/pkg/core/dns/lookup"
	"github.com/apache/dubbo-kubernetes/pkg/core/governance"
	"github.com/apache/dubbo-kubernetes/pkg/core/reg_client"
	"github.com/apache/dubbo-kubernetes/pkg/core/registry"
	core_manager "github.com/apache/dubbo-kubernetes/pkg/core/resources/manager"
	core_store "github.com/apache/dubbo-kubernetes/pkg/core/resources/store"
	"github.com/apache/dubbo-kubernetes/pkg/core/runtime/component"
	dds_context "github.com/apache/dubbo-kubernetes/pkg/dds/context"
	dp_server "github.com/apache/dubbo-kubernetes/pkg/dp-server/server"
	"github.com/apache/dubbo-kubernetes/pkg/events"
	"github.com/apache/dubbo-kubernetes/pkg/xds/cache/mesh"
)

// BuilderContext provides access to Builder's interim state.
type BuilderContext interface {
	ComponentManager() component.Manager
	ResourceStore() core_store.CustomizableResourceStore
	Transactions() core_store.Transactions
	ConfigStore() core_store.ResourceStore
	DataSourceLoader() datasource.Loader
	ResourceManager() core_manager.CustomizableResourceManager
	Config() dubbo_cp.Config
	RegistryCenter() dubboRegistry.Registry
	RegClient() reg_client.RegClient
	MetadataReportCenter() report.MetadataReport
	AdminRegistry() *registry.Registry
	ServiceDiscovery() dubboRegistry.ServiceDiscovery
	ConfigCenter() config_center.DynamicConfiguration
	Governance() governance.GovernanceConfig
	Extensions() context.Context
	ConfigManager() config_manager.ConfigManager
	LeaderInfo() component.LeaderInfo
	EventBus() events.EventBus
	DpServer() *dp_server.DpServer
	DataplaneCache() *sync.Map
	CAProvider() secrets.CaProvider
	DDSContext() *dds_context.Context
	ResourceValidators() ResourceValidators
}

var _ BuilderContext = &Builder{}

// Builder represents a multi-step initialization process.
type Builder struct {
	cfg                  dubbo_cp.Config
	cm                   component.Manager
	rs                   core_store.CustomizableResourceStore
	cs                   core_store.ResourceStore
	txs                  core_store.Transactions
	rm                   core_manager.CustomizableResourceManager
	rom                  core_manager.ReadOnlyResourceManager
	ext                  context.Context
	meshCache            *mesh.Cache
	eac                  admin.EnvoyAdminClient
	lif                  lookup.LookupIPFunc
	configm              config_manager.ConfigManager
	leadInfo             component.LeaderInfo
	erf                  events.EventBus
	apim                 api_server.APIManager
	cam                  core_ca.Managers
	dsl                  datasource.Loader
	dps                  *dp_server.DpServer
	registryCenter       dubboRegistry.Registry
	metadataReportCenter report.MetadataReport
	configCenter         config_center.DynamicConfiguration
	adminRegistry        *registry.Registry
	governance           governance.GovernanceConfig
	rv                   ResourceValidators
	ddsctx               *dds_context.Context
	appCtx               context.Context
	dCache               *sync.Map
	cap                  secrets.CaProvider
	metrics              metrics.Metrics
	regClient            reg_client.RegClient
	serviceDiscover      dubboRegistry.ServiceDiscovery
	*runtimeInfo
}

func (b *Builder) CAProvider() secrets.CaProvider {
	return b.cap
}

func BuilderFor(appCtx context.Context, cfg dubbo_cp.Config) (*Builder, error) {
	hostname, err := os.Hostname()
	if err != nil {
		return nil, errors.Wrap(err, "could not get hostname")
	}
	suffix := core.NewUUID()[0:4]
	return &Builder{
		cfg: cfg,
		ext: context.Background(),
		runtimeInfo: &runtimeInfo{
			instanceId: fmt.Sprintf("%s-%s", hostname, suffix),
			startTime:  time.Now(),
			mode:       cfg.Mode,
			deployMode: cfg.DeployMode,
		},
		appCtx: appCtx,
	}, nil
}

func (b *Builder) WithComponentManager(cm component.Manager) *Builder {
	b.cm = cm
	return b
}

func (b *Builder) WithResourceStore(rs core_store.CustomizableResourceStore) *Builder {
	b.rs = rs
	return b
}

func (b *Builder) WithTransactions(txs core_store.Transactions) *Builder {
	b.txs = txs
	return b
}

func (b *Builder) WithEnvoyAdminClient(eac admin.EnvoyAdminClient) *Builder {
	b.eac = eac
	return b
}

func (b *Builder) WithConfigStore(cs core_store.ResourceStore) *Builder {
	b.cs = cs
	return b
}

func (b *Builder) WithResourceManager(rm core_manager.CustomizableResourceManager) *Builder {
	b.rm = rm
	return b
}

func (b *Builder) WithReadOnlyResourceManager(rom core_manager.ReadOnlyResourceManager) *Builder {
	b.rom = rom
	return b
}

func (b *Builder) WithExtensions(ext context.Context) *Builder {
	b.ext = ext
	return b
}

func (b *Builder) WithExtension(key interface{}, value interface{}) *Builder {
	b.ext = context.WithValue(b.ext, key, value)
	return b
}

func (b *Builder) WithConfigManager(configm config_manager.ConfigManager) *Builder {
	b.configm = configm
	return b
}

func (b *Builder) WithLeaderInfo(leadInfo component.LeaderInfo) *Builder {
	b.leadInfo = leadInfo
	return b
}

func (b *Builder) WithDataplaneCache(cache *sync.Map) *Builder {
	b.dCache = cache
	return b
}

func (b *Builder) WithLookupIP(lif lookup.LookupIPFunc) *Builder {
	b.lif = lif
	return b
}

func (b *Builder) WithMeshCache(meshCache *mesh.Cache) *Builder {
	b.meshCache = meshCache
	return b
}

func (b *Builder) WithEventBus(erf events.EventBus) *Builder {
	b.erf = erf
	return b
}

func (b *Builder) WithDataSourceLoader(loader datasource.Loader) *Builder {
	b.dsl = loader
	return b
}

func (b *Builder) WithDpServer(dps *dp_server.DpServer) *Builder {
	b.dps = dps
	return b
}

func (b *Builder) MeshCache() *mesh.Cache {
	return b.meshCache
}

func (b *Builder) WithCAProvider(cap secrets.CaProvider) *Builder {
	b.cap = cap
	return b
}

func (b *Builder) CaManagers() core_ca.Managers {
	return b.cam
}

func (b *Builder) Metrics() metrics.Metrics {
	return b.metrics
}

func (b *Builder) WithDDSContext(ddsctx *dds_context.Context) *Builder {
	b.ddsctx = ddsctx
	return b
}

func (b *Builder) WithResourceValidators(rv ResourceValidators) *Builder {
	b.rv = rv
	return b
}

func (b *Builder) WithRegClient(regClient reg_client.RegClient) *Builder {
	b.regClient = regClient
	return b
}

func (b *Builder) WithCaManagers(cam core_ca.Managers) *Builder {
	b.cam = cam
	return b
}

func (b *Builder) WithCaManager(name string, cam core_ca.Manager) *Builder {
	b.cam[name] = cam
	return b
}

func (b *Builder) WithRegistryCenter(rg dubboRegistry.Registry) *Builder {
	b.registryCenter = rg
	return b
}

func (b *Builder) WithGovernanceConfig(gc governance.GovernanceConfig) *Builder {
	b.governance = gc
	return b
}

func (b *Builder) WithMetadataReport(mr report.MetadataReport) *Builder {
	b.metadataReportCenter = mr
	return b
}

func (b *Builder) WithConfigCenter(cc config_center.DynamicConfiguration) *Builder {
	b.configCenter = cc
	return b
}

func (b *Builder) WithAdminRegistry(ag *registry.Registry) *Builder {
	b.adminRegistry = ag
	return b
}

func (b *Builder) WithServiceDiscovery(discovery dubboRegistry.ServiceDiscovery) *Builder {
	b.serviceDiscover = discovery
	return b
}

func (b *Builder) APIManager() api_server.APIManager {
	return b.apim
}

func (b *Builder) Build() (Runtime, error) {
	if b.cm == nil {
		return nil, errors.Errorf("ComponentManager has not been configured")
	}
	if b.rs == nil {
		return nil, errors.Errorf("ResourceStore has not been configured")
	}
	if b.txs == nil {
		return nil, errors.Errorf("Transactions has not been configured")
	}
	if b.rm == nil {
		return nil, errors.Errorf("ResourceManager has not been configured")
	}
	if b.rom == nil {
		return nil, errors.Errorf("ReadOnlyResourceManager has not been configured")
	}
	if b.ext == nil {
		return nil, errors.Errorf("Extensions have been misconfigured")
	}
	if b.leadInfo == nil {
		return nil, errors.Errorf("LeaderInfo has not been configured")
	}
	if b.erf == nil {
		return nil, errors.Errorf("EventReaderFactory has not been configured")
	}
	if b.dps == nil {
		return nil, errors.Errorf("DpServer has not been configured")
	}
	if b.meshCache == nil {
		return nil, errors.Errorf("MeshCache has not been configured")
	}

	return &runtime{
		RuntimeInfo: b.runtimeInfo,
		RuntimeContext: &runtimeContext{
			cfg:                  b.cfg,
			rm:                   b.rm,
			rom:                  b.rom,
			txs:                  b.txs,
			ddsctx:               b.ddsctx,
			ext:                  b.ext,
			configm:              b.configm,
			registryCenter:       b.registryCenter,
			metadataReportCenter: b.metadataReportCenter,
			configCenter:         b.configCenter,
			adminRegistry:        b.adminRegistry,
			governance:           b.governance,
			leadInfo:             b.leadInfo,
			erf:                  b.erf,
			dCache:               b.dCache,
			dps:                  b.dps,
			serviceDiscovery:     b.serviceDiscover,
			rv:                   b.rv,
			appCtx:               b.appCtx,
			meshCache:            b.meshCache,
			regClient:            b.regClient,
		},
		Manager: b.cm,
	}, nil
}

func (b *Builder) RegClient() reg_client.RegClient {
	return b.regClient
}

func (b *Builder) DataplaneCache() *sync.Map {
	return b.dCache
}

func (b *Builder) Governance() governance.GovernanceConfig {
	return b.governance
}

func (b *Builder) AdminRegistry() *registry.Registry {
	return b.adminRegistry
}

func (b *Builder) ConfigCenter() config_center.DynamicConfiguration {
	return b.configCenter
}

func (b *Builder) ServiceDiscovery() dubboRegistry.ServiceDiscovery {
	return b.serviceDiscover
}

func (b *Builder) RegistryCenter() dubboRegistry.Registry {
	return b.registryCenter
}

func (b *Builder) MetadataReportCenter() report.MetadataReport {
	return b.metadataReportCenter
}

func (b *Builder) ComponentManager() component.Manager {
	return b.cm
}

func (b *Builder) ResourceStore() core_store.CustomizableResourceStore {
	return b.rs
}

func (b *Builder) Transactions() core_store.Transactions {
	return b.txs
}

func (b *Builder) ConfigStore() core_store.ResourceStore {
	return b.cs
}

func (b *Builder) ResourceManager() core_manager.CustomizableResourceManager {
	return b.rm
}

func (b *Builder) ReadOnlyResourceManager() core_manager.ReadOnlyResourceManager {
	return b.rom
}

func (b *Builder) LookupIP() lookup.LookupIPFunc {
	return b.lif
}

func (b *Builder) Config() dubbo_cp.Config {
	return b.cfg
}

func (b *Builder) DataSourceLoader() datasource.Loader {
	return b.dsl
}

func (b *Builder) DDSContext() *dds_context.Context {
	return b.ddsctx
}

func (b *Builder) Extensions() context.Context {
	return b.ext
}

func (b *Builder) ConfigManager() config_manager.ConfigManager {
	return b.configm
}

func (b *Builder) LeaderInfo() component.LeaderInfo {
	return b.leadInfo
}

func (b *Builder) EventBus() events.EventBus {
	return b.erf
}

func (b *Builder) DpServer() *dp_server.DpServer {
	return b.dps
}

func (b *Builder) ResourceValidators() ResourceValidators {
	return b.rv
}

func (b *Builder) AppCtx() context.Context {
	return b.appCtx
}
