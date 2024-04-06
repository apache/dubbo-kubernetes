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
	"errors"
	"fmt"
	"net"
	"net/http"
	"time"
)

import (
	dubbo_cp "github.com/apache/dubbo-kubernetes/pkg/config/app/dubbo-cp"
	config_core "github.com/apache/dubbo-kubernetes/pkg/config/core"
	config_manager "github.com/apache/dubbo-kubernetes/pkg/core/config/manager"
	"github.com/apache/dubbo-kubernetes/pkg/core/datasource"
	core_mesh "github.com/apache/dubbo-kubernetes/pkg/core/resources/apis/mesh"
	core_manager "github.com/apache/dubbo-kubernetes/pkg/core/resources/manager"
	core_model "github.com/apache/dubbo-kubernetes/pkg/core/resources/model"
	"github.com/apache/dubbo-kubernetes/pkg/core/resources/store"
	core_runtime "github.com/apache/dubbo-kubernetes/pkg/core/runtime"
	"github.com/apache/dubbo-kubernetes/pkg/core/runtime/component"
	"github.com/apache/dubbo-kubernetes/pkg/dp-server/server"
	"github.com/apache/dubbo-kubernetes/pkg/events"
	leader_memory "github.com/apache/dubbo-kubernetes/pkg/plugins/leader/memory"
	resources_memory "github.com/apache/dubbo-kubernetes/pkg/plugins/resources/memory"
	mesh_cache "github.com/apache/dubbo-kubernetes/pkg/xds/cache/mesh"
	xds_context "github.com/apache/dubbo-kubernetes/pkg/xds/context"
	xds_server "github.com/apache/dubbo-kubernetes/pkg/xds/server"
)

var _ core_runtime.RuntimeInfo = &TestRuntimeInfo{}

type TestRuntimeInfo struct {
	InstanceId string
	ClusterId  string
	StartTime  time.Time
	Mode       config_core.CpMode
	DeployMode config_core.DeployMode
}

func (i *TestRuntimeInfo) GetMode() config_core.CpMode {
	return i.Mode
}

func (i *TestRuntimeInfo) GetInstanceId() string {
	return i.InstanceId
}

func (i *TestRuntimeInfo) SetClusterId(clusterId string) {
	i.ClusterId = clusterId
}

func (i *TestRuntimeInfo) GetClusterId() string {
	return i.ClusterId
}

func (i *TestRuntimeInfo) GetStartTime() time.Time {
	return i.StartTime
}

func (i *TestRuntimeInfo) GetDeployMode() config_core.DeployMode {
	return i.DeployMode
}

func BuilderFor(appCtx context.Context, cfg dubbo_cp.Config) (*core_runtime.Builder, error) {
	builder, err := core_runtime.BuilderFor(appCtx, cfg)
	if err != nil {
		return nil, err
	}

	builder.
		WithComponentManager(component.NewManager(leader_memory.NewAlwaysLeaderElector())).
		WithResourceStore(store.NewCustomizableResourceStore(store.NewPaginationStore(resources_memory.NewStore()))).
		WithTransactions(store.NoTransactions{})

	rm := newResourceManager(builder) //nolint:contextcheck
	builder.WithResourceManager(rm).
		WithReadOnlyResourceManager(rm)

	builder.WithDataSourceLoader(datasource.NewDataSourceLoader(builder.ResourceManager()))
	builder.WithLeaderInfo(&component.LeaderInfoComponent{})
	builder.WithLookupIP(func(s string) ([]net.IP, error) {
		return nil, errors.New("LookupIP not set, set one in your test to resolve things")
	})
	eventBus, err := events.NewEventBus(10)
	if err != nil {
		return nil, err
	}
	builder.WithEventBus(eventBus)
	builder.WithDpServer(server.NewDpServer(*cfg.DpServer, func(writer http.ResponseWriter, request *http.Request) bool {
		return true
	}))

	err = initializeMeshCache(builder)
	if err != nil {
		return nil, err
	}

	return builder, nil
}

func initializeConfigManager(builder *core_runtime.Builder) {
	configm := config_manager.NewConfigManager(builder.ResourceStore())
	builder.WithConfigManager(configm)
}

func newResourceManager(builder *core_runtime.Builder) core_manager.CustomizableResourceManager {
	defaultManager := core_manager.NewResourceManager(builder.ResourceStore())
	customManagers := map[core_model.ResourceType]core_manager.ResourceManager{}
	customizableManager := core_manager.NewCustomizableResourceManager(defaultManager, customManagers)
	return customizableManager
}

func initializeMeshCache(builder *core_runtime.Builder) error {
	meshContextBuilder := xds_context.NewMeshContextBuilder(
		builder.ReadOnlyResourceManager(),
		xds_server.MeshResourceTypes(),
		builder.LookupIP(),
		builder.Config().Multizone.Zone.Name,
	)

	meshSnapshotCache, err := mesh_cache.NewCache(
		builder.Config().Store.Cache.ExpirationTime.Duration,
		meshContextBuilder,
	)
	if err != nil {
		return err
	}

	builder.WithMeshCache(meshSnapshotCache)
	return nil
}

type DummyEnvoyAdminClient struct {
	PostQuitCalled   *int
	ConfigDumpCalled int
	StatsCalled      int
	ClustersCalled   int
}

func (d *DummyEnvoyAdminClient) Stats(ctx context.Context, proxy core_model.ResourceWithAddress) ([]byte, error) {
	d.StatsCalled++
	return []byte("server.live: 1\n"), nil
}

func (d *DummyEnvoyAdminClient) Clusters(ctx context.Context, proxy core_model.ResourceWithAddress) ([]byte, error) {
	d.ClustersCalled++
	return []byte("dubbo:envoy:admin\n"), nil
}

func (d *DummyEnvoyAdminClient) GenerateAPIToken(dp *core_mesh.DataplaneResource) (string, error) {
	return "token", nil
}

func (d *DummyEnvoyAdminClient) PostQuit(ctx context.Context, dataplane *core_mesh.DataplaneResource) error {
	if d.PostQuitCalled != nil {
		*d.PostQuitCalled++
	}

	return nil
}

func (d *DummyEnvoyAdminClient) ConfigDump(ctx context.Context, proxy core_model.ResourceWithAddress) ([]byte, error) {
	d.ConfigDumpCalled++
	return []byte(fmt.Sprintf(`{"envoyAdminAddress": "%s"}`, proxy.AdminAddress(9901))), nil
}
