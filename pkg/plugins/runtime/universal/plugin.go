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

package universal

import (
	config_core "github.com/apache/dubbo-kubernetes/pkg/config/core"
	"github.com/apache/dubbo-kubernetes/pkg/core"
	"github.com/apache/dubbo-kubernetes/pkg/core/logger"
	core_plugins "github.com/apache/dubbo-kubernetes/pkg/core/plugins"
	core_runtime "github.com/apache/dubbo-kubernetes/pkg/core/runtime"
)

var log = core.Log.WithName("plugin").WithName("runtime").WithName("universal")

type plugin struct{}

func init() {
	core_plugins.Register(core_plugins.Universal, &plugin{})
}

func (p *plugin) Customize(rt core_runtime.Runtime) error {
	// 半托管和纯VM模式都应该用这个插件
	if rt.Config().DeployMode == config_core.KubernetesMode {
		return nil
	}

	if err := rt.AdminRegistry().Subscribe(
		rt.MetadataReportCenter(),
		rt.ResourceManager(),
		rt.DataplaneCache(),
		rt.ServiceDiscovery(),
		rt.EventBus(),
		rt.Config().Store.Kubernetes.SystemNamespace,
	); err != nil {
		logger.Errorf("Failed to subscribe to registry, error msg is %s.", err.Error())
		return err
	}

	defer func() {
		if err := rt.AdminRegistry().Destroy(); err != nil {
			logger.Errorf("Failed to subscribe to registry, error msg is %s.", err.Error())
			return
		}
	}()
	return nil
}
