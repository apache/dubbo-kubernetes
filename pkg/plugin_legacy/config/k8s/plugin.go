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
)

import (
	core_plugins "github.com/apache/dubbo-kubernetes/pkg/core/plugins"
	core_store "github.com/apache/dubbo-kubernetes/pkg/core/resources/store"
	k8s_extensions "github.com/apache/dubbo-kubernetes/pkg/plugins/extensions/k8s"
)

var _ core_plugins.ConfigStorePlugin = &plugin{}

type plugin struct{}

func init() {
	core_plugins.Register(core_plugins.Kubernetes, &plugin{})
}

func (p *plugin) NewConfigStore(pc core_plugins.PluginContext, _ core_plugins.PluginConfig) (core_store.ResourceStore, error) {
	mgr, ok := k8s_extensions.FromManagerContext(pc.Extensions())
	if !ok {
		return nil, errors.Errorf("k8s controller runtime Manager hasn't been configured")
	}
	converter, ok := k8s_extensions.FromResourceConverterContext(pc.Extensions())
	if !ok {
		return nil, errors.Errorf("k8s resource converter hasn't been configured")
	}
	return NewStore(mgr.GetClient(), pc.Config().Store.Kubernetes.SystemNamespace, mgr.GetScheme(), converter)
}
