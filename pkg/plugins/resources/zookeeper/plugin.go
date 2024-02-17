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

package zookeeper

import (
	"github.com/apache/dubbo-kubernetes/pkg/core"
	core_plugins "github.com/apache/dubbo-kubernetes/pkg/core/plugins"
	core_store "github.com/apache/dubbo-kubernetes/pkg/core/resources/store"
	"github.com/apache/dubbo-kubernetes/pkg/events"
	"github.com/pkg/errors"
)

var (
	log                                  = core.Log.WithName("plugins").WithName("resources").WithName("zookeeper")
	_   core_plugins.ResourceStorePlugin = &plugin{}
)

type plugin struct{}

func init() {
	core_plugins.Register(core_plugins.Zookeeper, &plugin{})
}

func (p *plugin) NewResourceStore(pc core_plugins.PluginContext, _ core_plugins.PluginConfig) (core_store.ResourceStore, core_store.Transactions, error) {
	log.Info("dubbo-cp runs with an in-zookeeper database and its state isn't preserved between restarts. Keep in mind that an in-memory database cannot be used with multiple instances of the control plane.")
	return NewStore(), core_store.NoTransactions{}, nil
}

func (p *plugin) Migrate(pc core_plugins.PluginContext, config core_plugins.PluginConfig) (core_plugins.DbVersion, error) {
	return 0, errors.New("migrations are not supported for Memory resource store")
}

func (p *plugin) EventListener(context core_plugins.PluginContext, writer events.Emitter) error {
	context.ResourceStore().DefaultResourceStore().(*zookeeperStore).SetEventWriter(writer)
	return nil
}
