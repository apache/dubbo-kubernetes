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

package kubemesh

import (
	"github.com/apache/dubbo-kubernetes/pkg/kube"
	"github.com/apache/dubbo-kubernetes/pkg/kube/krt"
	"github.com/apache/dubbo-kubernetes/pkg/mesh/meshwatcher"
	v1 "k8s.io/api/core/v1"
)

const (
	MeshConfigKey   = "mesh"
	MeshNetworksKey = "meshNetworks"
)

func NewConfigMapSource(client kube.Client, namespace, name, key string, opts krt.OptionsBuilder) meshwatcher.MeshConfigResource {
	return meshwatcher.MeshConfigResource{}
}

func meshConfigMapData(cm *v1.ConfigMap, key string) *string {
	if cm == nil {
		return nil
	}

	cfgYaml, exists := cm.Data[key]
	if !exists {
		return nil
	}

	return &cfgYaml
}
