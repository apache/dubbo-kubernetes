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
	core_env "github.com/apache/dubbo-kubernetes/pkg/config/core"
	"github.com/apache/dubbo-kubernetes/pkg/core"
	core_runtime "github.com/apache/dubbo-kubernetes/pkg/core/runtime"
)

var log = core.Log.WithName("mds")

func Setup(rt core_runtime.Runtime) error {
	if rt.Config().DeployMode != core_env.KubernetesMode ||
		!rt.Config().IsFederatedZoneCP() {
		// 非k8s模式以及global控制面不启动该组件
		return nil
	}

	return nil
}
