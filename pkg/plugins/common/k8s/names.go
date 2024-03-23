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
	core_model "github.com/apache/dubbo-kubernetes/pkg/core/resources/model"
)

const (
	// K8sMeshDefaultsGenerated identifies that default resources for mesh were successfully generated
	K8sMeshDefaultsGenerated = "k8s.dubbo.io/mesh-defaults-generated"

	// Kubernetes secret type to differentiate dubbo System secrets. Secret is bound to a mesh
	MeshSecretType = "system.dubbo.io/secret" // #nosec G101 -- This is the name not the value

	// Kubernetes secret type to differentiate dubbo System secrets. Secret is bound to a control plane
	GlobalSecretType = "system.dubbo.io/global-secret" // #nosec G101 -- This is the name not the value
)

func ResourceNameExtensions(namespace, name string) core_model.ResourceNameExtensions {
	return core_model.ResourceNameExtensions{
		core_model.K8sNamespaceComponent: namespace,
		core_model.K8sNameComponent:      name,
	}
}
