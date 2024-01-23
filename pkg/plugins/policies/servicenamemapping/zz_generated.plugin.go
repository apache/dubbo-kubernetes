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

package servicenamemapping

import (
	"github.com/apache/dubbo-kubernetes/pkg/plugins/policies/core"
	api_v1alpha1 "github.com/apache/dubbo-kubernetes/pkg/plugins/policies/servicenamemapping/api/v1alpha1"
	k8s_v1alpha1 "github.com/apache/dubbo-kubernetes/pkg/plugins/policies/servicenamemapping/k8s/v1alpha1"
	plugin_v1alpha1 "github.com/apache/dubbo-kubernetes/pkg/plugins/policies/servicenamemapping/plugin/v1alpha1"
)

func init() {
	core.Register(
		api_v1alpha1.ServiceNameMappingResourceTypeDescriptor,
		k8s_v1alpha1.AddToScheme,
		plugin_v1alpha1.NewPlugin(),
	)
}
