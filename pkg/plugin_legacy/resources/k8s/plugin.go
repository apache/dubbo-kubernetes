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

package k8s

import (
	"github.com/pkg/errors"

	core_plugins "github.com/apache/dubbo-kubernetes/pkg/core/plugins"
	core_store "github.com/apache/dubbo-kubernetes/pkg/core/resources/store"
	"github.com/apache/dubbo-kubernetes/pkg/events"
	k8s_runtime "github.com/apache/dubbo-kubernetes/pkg/plugins/extensions/k8s"
	k8s_events "github.com/apache/dubbo-kubernetes/pkg/plugins/resources/k8s/events"
)

var _ core_plugins.ResourceStorePlugin = &plugin{}

type plugin struct{}

func init() {
	core_plugins.Register(core_plugins.Kubernetes, &plugin{})
}

func (p *plugin) NewResourceStore(pc core_plugins.PluginContext, _ core_plugins.PluginConfig) (core_store.ResourceStore, core_store.Transactions, error) {
	mgr, ok := k8s_runtime.FromManagerContext(pc.Extensions())
	if !ok {
		return nil, nil, errors.Errorf("k8s controller runtime Manager hasn't been configured")
	}
	converter, ok := k8s_runtime.FromResourceConverterContext(pc.Extensions())
	if !ok {
		return nil, nil, errors.Errorf("k8s resource converter hasn't been configured")
	}
	store, err := NewStore(mgr.GetClient(), mgr.GetScheme(), converter)
	return store, core_store.NoTransactions{}, err
}

func (p *plugin) Migrate(pc core_plugins.PluginContext, config core_plugins.PluginConfig) (core_plugins.DbVersion, error) {
	return 0, errors.New("migrations are not supported for Kubernetes resource store")
}

func (p *plugin) EventListener(pc core_plugins.PluginContext, writer events.Emitter) error {
	mgr, ok := k8s_runtime.FromManagerContext(pc.Extensions())
	if !ok {
		return errors.Errorf("k8s controller runtime Manager hasn't been configured")
	}
	if err := pc.ComponentManager().Add(k8s_events.NewListener(mgr, writer)); err != nil {
		return err
	}
	return nil
}
