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
	config_core "github.com/apache/dubbo-kubernetes/pkg/config/core"
	"github.com/apache/dubbo-kubernetes/pkg/core"
	"github.com/apache/dubbo-kubernetes/pkg/core/resources/model"
	"github.com/apache/dubbo-kubernetes/pkg/core/resources/registry"
	"github.com/apache/dubbo-kubernetes/pkg/core/runtime"
	dds_server "github.com/apache/dubbo-kubernetes/pkg/dds/server"
	sync_store "github.com/apache/dubbo-kubernetes/pkg/dds/store"
	resources_k8s "github.com/apache/dubbo-kubernetes/pkg/plugins/resources/k8s"
)

var (
	ddsDeltaGlobalLog = core.Log.WithName("dds-delta-global")
)

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
	resourceSyncer, err := sync_store.NewResourceSyncer(ddsDeltaGlobalLog, rt.ResourceStore(), rt.Transactions(), rt.Extensions())
	if err != nil {
		return err
	}
	kubeFactory := resources_k8s.NewSimpleKubeFactory()
}
