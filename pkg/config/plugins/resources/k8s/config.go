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
	"github.com/apache/dubbo-kubernetes/pkg/config"
)

func DefaultKubernetesStoreConfig() *KubernetesStoreConfig {
	return &KubernetesStoreConfig{
		SystemNamespace: "dubbo-system",
	}
}

var _ config.Config = &KubernetesStoreConfig{}

// KubernetesStoreConfig defines Kubernetes store configuration
type KubernetesStoreConfig struct {
	config.BaseConfig

	// Namespace where Control Plane is installed to.
	SystemNamespace string `json:"systemNamespace" envconfig:"dubbo_store_kubernetes_system_namespace"`
}

func (p *KubernetesStoreConfig) Validate() error {
	if len(p.SystemNamespace) < 1 {
		return errors.New("SystemNamespace should not be empty")
	}
	return nil
}
