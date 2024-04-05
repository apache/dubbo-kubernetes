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
	"sync"
	"time"
)

import (
	"dubbo.apache.org/dubbo-go/v3/config_center"
	"dubbo.apache.org/dubbo-go/v3/metadata/report"
	dubboRegistry "dubbo.apache.org/dubbo-go/v3/registry"
)

import (
	dubbo_cp "github.com/apache/dubbo-kubernetes/pkg/config/app/dubbo-cp"
	"github.com/apache/dubbo-kubernetes/pkg/config/core"
	"github.com/apache/dubbo-kubernetes/pkg/core/admin"
	config_manager "github.com/apache/dubbo-kubernetes/pkg/core/config/manager"
	"github.com/apache/dubbo-kubernetes/pkg/core/governance"
	managers_dataplane "github.com/apache/dubbo-kubernetes/pkg/core/managers/apis/dataplane"
	managers_mesh "github.com/apache/dubbo-kubernetes/pkg/core/managers/apis/mesh"
	"github.com/apache/dubbo-kubernetes/pkg/core/reg_client"
	"github.com/apache/dubbo-kubernetes/pkg/core/registry"
	core_manager "github.com/apache/dubbo-kubernetes/pkg/core/resources/manager"
	core_store "github.com/apache/dubbo-kubernetes/pkg/core/resources/store"
	"github.com/apache/dubbo-kubernetes/pkg/core/runtime/component"
	dds_context "github.com/apache/dubbo-kubernetes/pkg/dds/context"
	dp_server "github.com/apache/dubbo-kubernetes/pkg/dp-server/server"
	"github.com/apache/dubbo-kubernetes/pkg/events"
	"github.com/apache/dubbo-kubernetes/pkg/xds/cache/mesh"
	xds_runtime "github.com/apache/dubbo-kubernetes/pkg/xds/runtime"
)

// Runtime represents initialized application state.
type Runtime interface {
	RuntimeInfo
	RuntimeContext
	component.Manager
}

type RuntimeInfo interface {
	GetInstanceId() string
	SetClusterId(clusterId string)
	GetClusterId() string
	GetStartTime() time.Time
	GetMode() core.CpMode
	GetDeployMode() core.DeployMode
}

type RuntimeContext interface {
	Config() dubbo_cp.Config
	ResourceManager() core_manager.ResourceManager
	Transactions() core_store.Transactions
	ReadOnlyResourceManager() core_manager.ReadOnlyResourceManager
	ConfigStore() core_store.ResourceStore
	Extensions() context.Context
	EnvoyAdminClient() admin.EnvoyAdminClient
	ConfigManager() config_manager.ConfigManager
	LeaderInfo() component.LeaderInfo
	EventBus() events.EventBus
	DpServer() *dp_server.DpServer
	DataplaneCache() *sync.Map
	DDSContext() *dds_context.Context
	RegistryCenter() dubboRegistry.Registry
	ServiceDiscovery() dubboRegistry.ServiceDiscovery
	MetadataReportCenter() report.MetadataReport
	Governance() governance.GovernanceConfig
	ConfigCenter() config_center.DynamicConfiguration
	AdminRegistry() *registry.Registry
	RegClient() reg_client.RegClient
	ResourceValidators() ResourceValidators
	// AppContext returns a context.Context which tracks the lifetime of the apps, it gets cancelled when the app is starting to shutdown.
	AppContext() context.Context
	XDS() xds_runtime.XDSRuntimeContext
	MeshCache() *mesh.Cache
}

type ResourceValidators struct {
	Dataplane managers_dataplane.Validator
	Mesh      managers_mesh.MeshValidator
}

type ExtraReportsFn func(Runtime) (map[string]string, error)

var _ Runtime = &runtime{}

type runtime struct {
	RuntimeInfo
	RuntimeContext
	component.Manager
}

var _ RuntimeInfo = &runtimeInfo{}

type runtimeInfo struct {
	mtx sync.RWMutex

	instanceId string
	clusterId  string
	startTime  time.Time
	mode       core.CpMode
	deployMode core.DeployMode
}

func (i *runtimeInfo) GetInstanceId() string {
	return i.instanceId
}

func (i *runtimeInfo) SetClusterId(clusterId string) {
	i.mtx.Lock()
	defer i.mtx.Unlock()
	i.clusterId = clusterId
}

func (i *runtimeInfo) GetClusterId() string {
	i.mtx.RLock()
	defer i.mtx.RUnlock()
	return i.clusterId
}

func (i *runtimeInfo) GetStartTime() time.Time {
	return i.startTime
}

func (i *runtimeInfo) GetMode() core.CpMode {
	return i.mode
}

func (i *runtimeInfo) GetDeployMode() core.DeployMode {
	return i.deployMode
}

var _ RuntimeContext = &runtimeContext{}

type runtimeContext struct {
	cfg                  dubbo_cp.Config
	rm                   core_manager.ResourceManager
	txs                  core_store.Transactions
	cs                   core_store.ResourceStore
	rom                  core_manager.ReadOnlyResourceManager
	ext                  context.Context
	eac                  admin.EnvoyAdminClient
	configm              config_manager.ConfigManager
	xds                  xds_runtime.XDSRuntimeContext
	leadInfo             component.LeaderInfo
	erf                  events.EventBus
	dps                  *dp_server.DpServer
	dCache               *sync.Map
	rv                   ResourceValidators
	ddsctx               *dds_context.Context
	registryCenter       dubboRegistry.Registry
	metadataReportCenter report.MetadataReport
	configCenter         config_center.DynamicConfiguration
	adminRegistry        *registry.Registry
	governance           governance.GovernanceConfig
	appCtx               context.Context
	meshCache            *mesh.Cache
	regClient            reg_client.RegClient
	serviceDiscovery     dubboRegistry.ServiceDiscovery
}

func (b *runtimeContext) RegClient() reg_client.RegClient {
	return b.regClient
}

func (b *runtimeContext) ServiceDiscovery() dubboRegistry.ServiceDiscovery {
	return b.serviceDiscovery
}

func (b *runtimeContext) DataplaneCache() *sync.Map {
	return b.dCache
}

func (b *runtimeContext) Governance() governance.GovernanceConfig {
	return b.governance
}

func (b *runtimeContext) ConfigCenter() config_center.DynamicConfiguration {
	return b.configCenter
}

func (b *runtimeContext) AdminRegistry() *registry.Registry {
	return b.adminRegistry
}

func (b *runtimeContext) RegistryCenter() dubboRegistry.Registry {
	return b.registryCenter
}

func (b *runtimeContext) MetadataReportCenter() report.MetadataReport {
	return b.metadataReportCenter
}

func (b *runtimeContext) EnvoyAdminClient() admin.EnvoyAdminClient {
	return b.eac
}

func (b *runtimeContext) MeshCache() *mesh.Cache {
	return b.meshCache
}

func (rc *runtimeContext) DDSContext() *dds_context.Context {
	return rc.ddsctx
}

func (rc *runtimeContext) XDS() xds_runtime.XDSRuntimeContext {
	return rc.xds
}

func (rc *runtimeContext) EventBus() events.EventBus {
	return rc.erf
}

func (rc *runtimeContext) Config() dubbo_cp.Config {
	return rc.cfg
}

func (rc *runtimeContext) ResourceManager() core_manager.ResourceManager {
	return rc.rm
}

func (rc *runtimeContext) Transactions() core_store.Transactions {
	return rc.txs
}

func (rc *runtimeContext) ConfigStore() core_store.ResourceStore {
	return rc.cs
}

func (rc *runtimeContext) ReadOnlyResourceManager() core_manager.ReadOnlyResourceManager {
	return rc.rom
}

func (rc *runtimeContext) Extensions() context.Context {
	return rc.ext
}

func (rc *runtimeContext) ConfigManager() config_manager.ConfigManager {
	return rc.configm
}

func (rc *runtimeContext) LeaderInfo() component.LeaderInfo {
	return rc.leadInfo
}

func (rc *runtimeContext) DpServer() *dp_server.DpServer {
	return rc.dps
}

func (rc *runtimeContext) ResourceValidators() ResourceValidators {
	return rc.rv
}

func (rc *runtimeContext) AppContext() context.Context {
	return rc.appCtx
}
