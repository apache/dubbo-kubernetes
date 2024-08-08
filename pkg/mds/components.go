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

package mds

import (
	"time"
)

import (
	"github.com/pkg/errors"
)

import (
	mesh_proto "github.com/apache/dubbo-kubernetes/api/mesh/v1alpha1"
	core_env "github.com/apache/dubbo-kubernetes/pkg/config/core"
	"github.com/apache/dubbo-kubernetes/pkg/core"
	core_mesh "github.com/apache/dubbo-kubernetes/pkg/core/resources/apis/mesh"
	core_model "github.com/apache/dubbo-kubernetes/pkg/core/resources/model"
	core_runtime "github.com/apache/dubbo-kubernetes/pkg/core/runtime"
	"github.com/apache/dubbo-kubernetes/pkg/mds/pusher"
	"github.com/apache/dubbo-kubernetes/pkg/mds/server"
	k8s_extensions "github.com/apache/dubbo-kubernetes/pkg/plugins/extensions/k8s"
)

var log = core.Log.WithName("mds")

func Setup(rt core_runtime.Runtime) error {
	if rt.Config().DeployMode != core_env.KubernetesMode ||
		!rt.Config().IsNonFederatedZoneCP() {
		// 非k8s模式以及zone控制面不启动该组件
		return nil
	}
	cfg := rt.Config().DubboConfig

	mgr, ok := k8s_extensions.FromManagerContext(rt.Extensions())
	if !ok {
		return errors.Errorf("k8s controller runtime Manager hasn't been configured")
	}

	converter, ok := k8s_extensions.FromResourceConverterContext(rt.Extensions())
	if !ok {
		return errors.Errorf("k8s resource converter hasn't been configured")
	}

	dubboPusher := pusher.NewPusher(rt.ResourceManager(), rt.EventBus(), func() *time.Ticker {
		// todo: should configured by config in the future
		return time.NewTicker(time.Minute * 10)
	}, []core_model.ResourceType{
		core_mesh.MappingType,
		core_mesh.MetaDataType,
	})

	mdsServer := server.NewMdsServer(
		rt.AppContext(),
		cfg,
		dubboPusher,
		mgr,
		converter,
		rt.ResourceManager(),
		rt.Transactions(),
		rt.Config().Multizone.Zone.Name,
		rt.Config().Store.Kubernetes.SystemNamespace,
	)
	mesh_proto.RegisterMDSSyncServiceServer(rt.DpServer().GrpcServer(), mdsServer)

	return rt.Add(dubboPusher, mdsServer)
}
