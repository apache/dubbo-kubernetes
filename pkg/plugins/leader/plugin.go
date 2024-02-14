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

package leader

import (
	"github.com/pkg/errors"
)

import (
	"github.com/apache/dubbo-kubernetes/pkg/config/core/resources/store"
	core_runtime "github.com/apache/dubbo-kubernetes/pkg/core/runtime"
	"github.com/apache/dubbo-kubernetes/pkg/core/runtime/component"
	leader_memory "github.com/apache/dubbo-kubernetes/pkg/plugins/leader/memory"
	leader_nacos "github.com/apache/dubbo-kubernetes/pkg/plugins/leader/nacos"
	leader_zookeeper "github.com/apache/dubbo-kubernetes/pkg/plugins/leader/zookeeper"
)

func NewLeaderElector(b *core_runtime.Builder) (component.LeaderElector, error) {
	switch b.Config().Store.Type {
	case store.MemoryStore:
		return leader_memory.NewAlwaysLeaderElector(), nil
	case store.ZookeeperStore:
		return leader_zookeeper.NewZookeeperLeaderElector(), nil
	case store.NacosStore:
		return leader_nacos.NewNacosLeaderElector(), nil
	// In case of Kubernetes, Leader Elector is embedded in a Kubernetes ComponentManager
	default:
		return nil, errors.Errorf("no election leader for storage of type %s", b.Config().Store.Type)
	}
}
