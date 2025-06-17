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
	config_core "github.com/apache/dubbo-kubernetes/pkg/config/mode"
	core_plugins "github.com/apache/dubbo-kubernetes/pkg/core/plugins"
	core_runtime "github.com/apache/dubbo-kubernetes/pkg/core/runtime"
	"github.com/apache/dubbo-kubernetes/pkg/core/runtime/component"
	plugin_leader "github.com/apache/dubbo-kubernetes/pkg/plugins/leader"
)

var _ core_plugins.BootstrapPlugin = &plugin{}

type plugin struct{}

func init() {
	core_plugins.Register(core_plugins.Universal, &plugin{})
}

func (p *plugin) BeforeBootstrap(b *core_runtime.Builder, _ core_plugins.PluginConfig) error {
	if b.Config().DeployMode == config_core.UniversalMode {
		leaderElector, err := plugin_leader.NewLeaderElector(b)
		if err != nil {
			return err
		}
		b.WithComponentManager(component.NewManager(leaderElector))
	}
	return nil
}

func (p *plugin) AfterBootstrap(b *core_runtime.Builder, _ core_plugins.PluginConfig) error {
	return nil
}

func (p *plugin) Name() core_plugins.PluginName {
	return core_plugins.Universal
}

func (p *plugin) Order() int {
	return core_plugins.EnvironmentPreparingOrder
}
