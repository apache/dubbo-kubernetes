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

package global

import (
	"context"
	"time"
)

import (
	"github.com/go-logr/logr"

	"github.com/pkg/errors"
)

import (
	mesh_proto "github.com/apache/dubbo-kubernetes/api/mesh/v1alpha1"
	system_proto "github.com/apache/dubbo-kubernetes/api/system/v1alpha1"
	config_core "github.com/apache/dubbo-kubernetes/pkg/config/core"
	store_config "github.com/apache/dubbo-kubernetes/pkg/config/core/resources/store"
	"github.com/apache/dubbo-kubernetes/pkg/core"
	"github.com/apache/dubbo-kubernetes/pkg/core/resources/apis/system"
	core_manager "github.com/apache/dubbo-kubernetes/pkg/core/resources/manager"
	"github.com/apache/dubbo-kubernetes/pkg/core/resources/model"
	"github.com/apache/dubbo-kubernetes/pkg/core/resources/registry"
	"github.com/apache/dubbo-kubernetes/pkg/core/resources/store"
	"github.com/apache/dubbo-kubernetes/pkg/core/runtime"
	"github.com/apache/dubbo-kubernetes/pkg/core/runtime/component"
	dds_client "github.com/apache/dubbo-kubernetes/pkg/dds/client"
	"github.com/apache/dubbo-kubernetes/pkg/dds/mux"
	dds_server "github.com/apache/dubbo-kubernetes/pkg/dds/server"
	"github.com/apache/dubbo-kubernetes/pkg/dds/service"
	dds_sync_store "github.com/apache/dubbo-kubernetes/pkg/dds/store"
	sync_store "github.com/apache/dubbo-kubernetes/pkg/dds/store"
	"github.com/apache/dubbo-kubernetes/pkg/dds/util"
	dubbo_log "github.com/apache/dubbo-kubernetes/pkg/log"
	resources_k8s "github.com/apache/dubbo-kubernetes/pkg/plugins/resources/k8s"
	util_proto "github.com/apache/dubbo-kubernetes/pkg/util/proto"
)

var ddsDeltaGlobalLog = core.Log.WithName("dds-delta-global")

func Setup(rt runtime.Runtime) error {
	if rt.Config().Mode != config_core.Global {
		return nil
	}
	reg := registry.Global()
	ddsServer, err := dds_server.New(
		ddsDeltaGlobalLog,
		rt,
		reg.ObjectTypes(model.HasDDSFlag(model.GlobalToZoneSelector)),
		"global",
		rt.Config().Multizone.Global.DDS.RefreshInterval.Duration,
		rt.DDSContext().GlobalProvidedFilter,
		rt.DDSContext().GlobalResourceMapper,
		rt.Config().Multizone.Global.DDS.NackBackoff.Duration,
	)
	if err != nil {
		return err
	}
	resourceSyncer, err := sync_store.NewResourceSyncer(ddsDeltaGlobalLog, rt.ResourceManager(), rt.Transactions(), rt.Extensions())
	if err != nil {
		return err
	}
	kubeFactory := resources_k8s.NewSimpleKubeFactory()

	onGlobalToZoneSyncConnect := mux.OnGlobalToZoneSyncConnectFunc(func(stream mesh_proto.DDSSyncService_GlobalToZoneSyncServer, errChan chan error) {
		zoneID, err := util.ClientIDFromIncomingCtx(stream.Context())
		if err != nil {
			errChan <- err
		}
		log := ddsDeltaGlobalLog.WithValues("peer-id", zoneID)
		log = dubbo_log.AddFieldsFromCtx(log, stream.Context(), rt.Extensions())
		log.Info("Global To Zone new session created")
		if err := createZoneIfAbsent(stream.Context(), log, zoneID, rt.ResourceManager()); err != nil {
			errChan <- errors.Wrap(err, "Global CP could not create a zone")
		}
		if err := ddsServer.GlobalToZoneSync(stream); err != nil {
			errChan <- err
		} else {
			log.V(1).Info("GlobalToZoneSync finished gracefully")
		}
	})

	onZoneToGlobalSyncConnect := mux.OnZoneToGlobalSyncConnectFunc(func(stream mesh_proto.DDSSyncService_ZoneToGlobalSyncServer, errChan chan error) {
		zoneID, err := util.ClientIDFromIncomingCtx(stream.Context())
		if err != nil {
			errChan <- err
		}
		log := ddsDeltaGlobalLog.WithValues("peer-id", zoneID)
		log = dubbo_log.AddFieldsFromCtx(log, stream.Context(), rt.Extensions())
		ddsStream := dds_client.NewDeltaDDSStream(stream, zoneID, rt, "")
		sink := dds_client.NewDDSSyncClient(
			log,
			reg.ObjectTypes(model.HasDDSFlag(model.ZoneToGlobalFlag)),
			ddsStream,
			dds_sync_store.GlobalSyncCallback(stream.Context(), resourceSyncer, rt.Config().Store.Type == store_config.KubernetesStore, kubeFactory, rt.Config().Store.Kubernetes.SystemNamespace),
			rt.Config().Multizone.Global.DDS.ResponseBackoff.Duration,
		)
		go func() {
			if err := sink.Receive(); err != nil {
				errChan <- errors.Wrap(err, "DDSSyncClient finished with an error")
			} else {
				log.V(1).Info("DDSSyncClient finished gracefully")
			}
		}()
	})

	var streamInterceptors []service.StreamInterceptor
	for _, filter := range rt.DDSContext().GlobalServerFilters {
		streamInterceptors = append(streamInterceptors, filter)
	}

	if rt.Config().Multizone.Global.DDS.ZoneHealthCheck.Timeout.Duration > time.Duration(0) {
		zwLog := ddsDeltaGlobalLog.WithName("zone-watch")
		zw, err := mux.NewZoneWatch(
			zwLog,
			rt.Config().Multizone.Global.DDS.ZoneHealthCheck,
			rt.EventBus(),
			rt.ReadOnlyResourceManager(),
			rt.Extensions(),
		)
		if err != nil {
			return errors.Wrap(err, "couldn't create ZoneWatch")
		}
		if err := rt.Add(component.NewResilientComponent(zwLog, zw)); err != nil {
			return err
		}
	}
	return rt.Add(mux.NewServer(
		rt.DDSContext().GlobalServerFilters,
		rt.DDSContext().ServerStreamInterceptors,
		rt.DDSContext().ServerUnaryInterceptor,
		*rt.Config().Multizone.Global.DDS,
		service.NewGlobalDDSServiceServer(
			rt.AppContext(),
			rt.ResourceManager(),
			rt.GetInstanceId(),
			streamInterceptors,
			rt.Extensions(),
			rt.Config().Store.Upsert,
			rt.EventBus(),
			rt.Config().Multizone.Global.DDS.ZoneHealthCheck.PollInterval.Duration,
		),
		mux.NewDDSSyncServiceServer(
			rt.AppContext(),
			onGlobalToZoneSyncConnect,
			onZoneToGlobalSyncConnect,
			rt.DDSContext().GlobalServerFilters,
			rt.Extensions(),
			rt.EventBus(),
		),
	))
}

func createZoneIfAbsent(ctx context.Context, log logr.Logger, name string, resManager core_manager.ResourceManager) error {
	if err := resManager.Get(ctx, system.NewZoneResource(), store.GetByKey(name, model.NoMesh)); err != nil {
		if !store.IsResourceNotFound(err) {
			return err
		}
		log.Info("creating Zone", "name", name)
		zone := &system.ZoneResource{
			Spec: &system_proto.Zone{
				Enabled: util_proto.Bool(true),
			},
		}
		if err := resManager.Create(ctx, zone, store.CreateByKey(name, model.NoMesh)); err != nil {
			return err
		}
	}
	return nil
}
