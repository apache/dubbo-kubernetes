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
	"github.com/apache/dubbo-kubernetes/pkg/core/kubeclient/client"
	"sync"
	"time"
)

import (
	dubbo_cp "github.com/apache/dubbo-kubernetes/pkg/config/app/dubbo-cp"
	"github.com/apache/dubbo-kubernetes/pkg/config/core"
	config_manager "github.com/apache/dubbo-kubernetes/pkg/core/config/manager"
	core_manager "github.com/apache/dubbo-kubernetes/pkg/core/resources/manager"
	core_store "github.com/apache/dubbo-kubernetes/pkg/core/resources/store"
	"github.com/apache/dubbo-kubernetes/pkg/core/runtime/component"
	dp_server "github.com/apache/dubbo-kubernetes/pkg/dp-server/server"
	"github.com/apache/dubbo-kubernetes/pkg/events"
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
}

type RuntimeContext interface {
	Config() dubbo_cp.Config
	ResourceManager() core_manager.ResourceManager
	ResourceStore() core_store.ResourceStore
	Transactions() core_store.Transactions
	ReadOnlyResourceManager() core_manager.ReadOnlyResourceManager
	ConfigStore() core_store.ResourceStore
	Extensions() context.Context
	ConfigManager() config_manager.ConfigManager
	LeaderInfo() component.LeaderInfo
	EventBus() events.EventBus
	DpServer() *dp_server.DpServer
	ResourceValidators() ResourceValidators
	// AppContext returns a context.Context which tracks the lifetime of the apps, it gets cancelled when the app is starting to shutdown.
	AppContext() context.Context
	KubeClient() *client.KubeClient
	XDS() xds_runtime.XDSRuntimeContext
}

type ResourceValidators struct {
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

var _ RuntimeContext = &runtimeContext{}

type runtimeContext struct {
	cfg        dubbo_cp.Config
	rm         core_manager.ResourceManager
	rs         core_store.ResourceStore
	txs        core_store.Transactions
	cs         core_store.ResourceStore
	rom        core_manager.ReadOnlyResourceManager
	ext        context.Context
	configm    config_manager.ConfigManager
	xds        xds_runtime.XDSRuntimeContext
	leadInfo   component.LeaderInfo
	erf        events.EventBus
	dps        *dp_server.DpServer
	kubeclient *client.KubeClient
	rv         ResourceValidators
	appCtx     context.Context
}

func (rc *runtimeContext) KubeClient() *client.KubeClient {
	return rc.kubeclient
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

func (rc *runtimeContext) ResourceStore() core_store.ResourceStore {
	return rc.rs
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
