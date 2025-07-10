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

package zone

import (
	"github.com/pkg/errors"
)

import (
	mesh_proto "github.com/apache/dubbo-kubernetes/api/mesh/v1alpha1"
	"github.com/apache/dubbo-kubernetes/pkg/config"
	"github.com/apache/dubbo-kubernetes/pkg/config/mode/resources/store"
	"github.com/apache/dubbo-kubernetes/pkg/core"
	"github.com/apache/dubbo-kubernetes/pkg/core/resources/model"
	"github.com/apache/dubbo-kubernetes/pkg/core/resources/registry"
	core_runtime "github.com/apache/dubbo-kubernetes/pkg/core/runtime"
	"github.com/apache/dubbo-kubernetes/pkg/core/runtime/component"
	dds_client "github.com/apache/dubbo-kubernetes/pkg/dds/client"
	"github.com/apache/dubbo-kubernetes/pkg/dds/mux"
	dds_server "github.com/apache/dubbo-kubernetes/pkg/dds/server"
	dds_sync_store "github.com/apache/dubbo-kubernetes/pkg/dds/store"
	resources_k8s "github.com/apache/dubbo-kubernetes/pkg/plugins/resources/k8s"
)

var ddsDeltaZoneLog = core.Log.WithName("dds-delta-zone")

func Setup(rt core_runtime.Runtime) error {
	if !rt.Config().IsFederatedZoneCP() {
		return nil
	}
	zone := rt.Config().Multizone.Zone.Name
	reg := registry.Global()
	ddsCtx := rt.DDSContext()
	ddsServer, err := dds_server.New(
		ddsDeltaZoneLog,
		rt,
		reg.ObjectTypes(model.HasDDSFlag(model.ZoneToGlobalFlag)),
		zone,
		rt.Config().Multizone.Zone.DDS.RefreshInterval.Duration,
		ddsCtx.ZoneProvidedFilter,
		ddsCtx.ZoneResourceMapper,
		rt.Config().Multizone.Zone.DDS.NackBackoff.Duration,
	)
	if err != nil {
		return err
	}
	resourceSyncer, err := dds_sync_store.NewResourceSyncer(ddsDeltaZoneLog, rt.ResourceManager(), rt.Transactions(), rt.Extensions())
	if err != nil {
		return err
	}
	kubeFactory := resources_k8s.NewSimpleKubeFactory()
	cfg := rt.Config()
	cfgForDisplay, err := config.ConfigForDisplay(&cfg)
	if err != nil {
		return errors.Wrap(err, "could not construct config for display")
	}
	cfgJson, err := config.ToJson(cfgForDisplay)
	if err != nil {
		return errors.Wrap(err, "could not marshall config to json")
	}

	onGlobalToZoneSyncStarted := mux.OnGlobalToZoneSyncStartedFunc(func(stream mesh_proto.DDSSyncService_GlobalToZoneSyncClient, errChan chan error) {
		log := ddsDeltaZoneLog.WithValues("dds-version", "v2")
		syncClient := dds_client.NewDDSSyncClient(
			log,
			reg.ObjectTypes(model.HasDDSFlag(model.GlobalToZoneSelector)),
			dds_client.NewDeltaDDSStream(stream, zone, rt, string(cfgJson)),
			dds_sync_store.ZoneSyncCallback(
				stream.Context(),
				rt.DDSContext().Configs,
				resourceSyncer,
				rt.Config().Store.Type == store.KubernetesStore,
				zone,
				kubeFactory,
				rt.Config().Store.Kubernetes.SystemNamespace,
			),
			rt.Config().Multizone.Zone.DDS.ResponseBackoff.Duration,
		)
		go func() {
			if err := syncClient.Receive(); err != nil {
				errChan <- errors.Wrap(err, "GlobalToZoneSyncClient finished with an error")
			} else {
				log.V(1).Info("GlobalToZoneSyncClient finished gracefully")
			}
		}()
	})

	onZoneToGlobalSyncStarted := mux.OnZoneToGlobalSyncStartedFunc(func(stream mesh_proto.DDSSyncService_ZoneToGlobalSyncClient, errChan chan error) {
		log := ddsDeltaZoneLog.WithValues("dds-version", "v2", "peer-id", "global")
		log.Info("ZoneToGlobalSync new session created")
		session := dds_server.NewServerStream(stream)
		go func() {
			if err := ddsServer.ZoneToGlobal(session); err != nil {
				errChan <- errors.Wrap(err, "ZoneToGlobalSync finished with an error")
			} else {
				log.V(1).Info("ZoneToGlobalSync finished gracefully")
			}
		}()
	})

	muxClient := mux.NewClient(
		rt.DDSContext().ZoneClientCtx,
		rt.Config().Multizone.Zone.GlobalAddress,
		zone,
		onGlobalToZoneSyncStarted,
		onZoneToGlobalSyncStarted,
		*rt.Config().Multizone.Zone.DDS,
	)
	return rt.Add(component.NewResilientComponent(ddsDeltaZoneLog.WithName("dds-mux-client"), muxClient))
}
