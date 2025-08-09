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

package kubetypes

import (
	"github.com/apache/dubbo-kubernetes/operator/pkg/config"
	"github.com/apache/dubbo-kubernetes/pkg/config/schema/gvk"
	istioioapimeshv1alpha1 "istio.io/api/mesh/v1alpha1"
	k8sioapicorev1 "k8s.io/api/core/v1"
)

func getGvk(obj any) (config.GroupVersionKind, bool) {
	switch obj.(type) {
	case *k8sioapicorev1.ConfigMap:
		return gvk.ConfigMap, true
	case *istioioapimeshv1alpha1.MeshConfig:
		return gvk.MeshConfig, true
	default:
		return config.GroupVersionKind{}, false
	}

}
