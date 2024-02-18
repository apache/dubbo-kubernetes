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
	"os"
	"time"
)

import (
	"github.com/pkg/errors"
)

import (
	dubbo_cp "github.com/apache/dubbo-kubernetes/pkg/config/app/dubbo-cp"
	"github.com/apache/dubbo-kubernetes/pkg/core"
	config_manager "github.com/apache/dubbo-kubernetes/pkg/core/config/manager"
	"github.com/apache/dubbo-kubernetes/pkg/core/datasource"
	"github.com/apache/dubbo-kubernetes/pkg/core/dns/lookup"
	core_manager "github.com/apache/dubbo-kubernetes/pkg/core/resources/manager"
	core_store "github.com/apache/dubbo-kubernetes/pkg/core/resources/store"
	"github.com/apache/dubbo-kubernetes/pkg/core/runtime/component"
	dds_context "github.com/apache/dubbo-kubernetes/pkg/dds/context"
	dp_server "github.com/apache/dubbo-kubernetes/pkg/dp-server/server"
	"github.com/apache/dubbo-kubernetes/pkg/dubbo"
	"github.com/apache/dubbo-kubernetes/pkg/events"
	"github.com/apache/dubbo-kubernetes/pkg/intercp/client"
	"github.com/apache/dubbo-kubernetes/pkg/xds/cache/mesh"
)

// BuilderContext provides access to Builder's interim state.
type BuilderContext interface {
	ComponentManager() component.Manager
	ResourceStore() core_store.CustomizableResourceStore
	Transactions() core_store.Transactions
	ConfigStore() core_store.ResourceStore
	ResourceManager() core_manager.CustomizableResourceManager
	Config() dubbo_cp.Config
	Extensions() context.Context
	ConfigManager() config_manager.ConfigManager
	LeaderInfo() component.LeaderInfo
	EventBus() events.EventBus
	DpServer() *dp_server.DpServer
	InterCPClientPool() *client.Pool
	DDSContext() *dds_context.Context
	ResourceValidators() ResourceValidators
}

var _ BuilderContext = &Builder{}

// Builder represents a multi-step initialization process.
type Builder struct {
	cfg dubbo_cp.Config
	cm  component.Manager
	rs  core_store.CustomizableResourceStore
	cs  core_store.ResourceStore
	txs core_store.Transactions
	rm  core_manager.CustomizableResourceManager
	rom core_manager.ReadOnlyResourceManager

	lif         lookup.LookupIPFunc
	eac         dubbo.EnvoyAdminClient
	ext         context.Context
	meshCache   *mesh.Cache
	configm     config_manager.ConfigManager
	leadInfo    component.LeaderInfo
	erf         events.EventBus
	dsl         datasource.Loader
	interCpPool *client.Pool
	dps         *dp_server.DpServer
	rv          ResourceValidators
	ddsctx      *dds_context.Context
	appCtx      context.Context
	*runtimeInfo
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

func (b *Builder) WithEnvoyAdminClient(eac dubbo.EnvoyAdminClient) *Builder {
	b.eac = eac
	return b
}

func (b *Builder) WithDDSContext(ddsctx *dds_context.Context) *Builder {
	b.ddsctx = ddsctx
	return b
}

func (b *Builder) WithResourceValidators(rv ResourceValidators) *Builder {
	b.rv = rv
	return b
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

	return &runtime{
		RuntimeInfo: b.runtimeInfo,
		RuntimeContext: &runtimeContext{
			cfg:      b.cfg,
			rm:       b.rm,
			rom:      b.rom,
			rs:       b.rs,
			txs:      b.txs,
			ddsctx:   b.ddsctx,
			ext:      b.ext,
			configm:  b.configm,
			leadInfo: b.leadInfo,
			erf:      b.erf,
			dps:      b.dps,
			eac:      b.eac,
			rv:       b.rv,
			appCtx:   b.appCtx,
		},
		Manager: b.cm,
	}, nil
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

func (b *Builder) InterCPClientPool() *client.Pool {
	return b.interCpPool
}

func (b *Builder) LookupIP() lookup.LookupIPFunc {
	return b.lif
}

func (b *Builder) Config() dubbo_cp.Config {
	return b.cfg
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
